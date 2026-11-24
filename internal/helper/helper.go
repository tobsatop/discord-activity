// Package helper is the native messaging host. Chrome starts it while a
// YouTube Music tab is open. It reads songs from the extension on stdin and
// shows them in the Discord app. When Chrome closes stdin, it clears the
// status and exits.
//
// stdout belongs to Chrome: nothing but protocol frames may ever be written there.
package helper

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/tobsatop/discord-activity/internal/activity"
	"github.com/tobsatop/discord-activity/internal/discordipc"
	"github.com/tobsatop/discord-activity/internal/nativemsg"
)

const maxLogBytes = 1024 * 1024

// ClientIDPattern is a Discord Application ID.
var ClientIDPattern = regexp.MustCompile(`^\d{17,20}$`)

// Options for Run.
type Options struct {
	Version string
	DataDir string // holds config.json and host.log
	Stdin   io.Reader
	Stdout  io.Writer
	Getenv  func(string) string
}

// exit is os.Exit; Chrome going away mid-write ends the helper like the Node version did.
var exit = os.Exit

type host struct {
	logFile string
	stdout  io.Writer
	outMu   sync.Mutex

	mu    sync.Mutex
	track *activity.Track
	basic bool
}

// Run serves Chrome until stdin closes and returns the exit code.
func Run(opts Options) int {
	h := &host{logFile: filepath.Join(opts.DataDir, "host.log"), stdout: opts.Stdout}
	configFile := filepath.Join(opts.DataDir, "config.json")

	h.trimLog()
	h.captureCrashes()
	h.log(fmt.Sprintf("host started (ytm-discord %s, %s %s/%s, pid %d)", opts.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, os.Getpid()))
	defer func() {
		if r := recover(); r != nil {
			h.log(fmt.Sprintf("crash: %v\n%s", r, debug.Stack()))
			exit(1)
		}
	}()

	var discord *discordipc.Client
	if clientID := loadClientID(opts.Getenv, configFile); clientID != "" {
		discord = discordipc.New(discordipc.Options{
			ClientID: clientID,
			Log:      h.log,
			OnStatus: func(s discordipc.Status) { h.send(statusMessage{Type: "status", State: s.State, Detail: s.Detail}) },
			OnActivityError: func(message string) {
				h.log("Discord rejected the activity: " + message)
				h.mu.Lock()
				if !h.basic {
					h.basic = true // retry without the newer optional fields
					a := activity.ToActivity(h.track, true)
					h.mu.Unlock()
					discord.SetActivity(a)
					return
				}
				h.mu.Unlock()
				h.send(statusMessage{Type: "status", State: "error", Detail: "Discord ไม่รับสถานะ: " + message})
			},
		})
		discord.Start()
	} else {
		h.log("no valid Discord Application ID in " + configFile)
		h.send(statusMessage{Type: "status", State: "error", Detail: "ยังไม่ได้ใส่ Discord Application ID — รันตัวติดตั้ง (README ขั้นที่ 3)"})
	}

	_ = nativemsg.Read(opts.Stdin, func(message any) {
		msg, _ := message.(map[string]any)
		if msg == nil || msg["type"] != "update" {
			return
		}
		track := activity.ParseTrack(msg["track"])
		if track != nil && track.Playing {
			h.log(fmt.Sprintf("playing: %s — %s", track.Title, track.Artist))
		} else {
			h.log("nothing playing")
		}
		h.mu.Lock()
		h.track = track
		a := activity.ToActivity(track, h.basic)
		h.mu.Unlock()
		if discord != nil {
			discord.SetActivity(a)
		}
	}, func(err error) {
		h.log("bad message from Chrome: " + err.Error())
	})

	h.log("Chrome disconnected; clearing the status")
	if discord != nil {
		discord.Stop()
	}
	return 0
}

type statusMessage struct {
	Type   string `json:"type"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

func (h *host) send(message any) {
	frame, err := nativemsg.Encode(message)
	if err != nil {
		h.log("cannot encode a message: " + err.Error())
		return
	}
	h.outMu.Lock()
	defer h.outMu.Unlock()
	if _, err := h.stdout.Write(frame); err != nil {
		exit(0) // Chrome went away mid-write
	}
}

// log appends a line to host.log, and echoes it to stderr only when that is
// a terminal: under Chrome nobody may be reading stderr, and a full pipe
// would block the helper.
func (h *host) log(text string) {
	line := time.Now().UTC().Format("2006-01-02T15:04:05.000Z") + " " + text + "\n"
	if fi, err := os.Stderr.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		_, _ = os.Stderr.WriteString(line)
	}
	f, err := os.OpenFile(h.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return // logging must never take the helper down
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func (h *host) trimLog() {
	if fi, err := os.Stat(h.logFile); err == nil && fi.Size() > maxLogBytes {
		_ = os.WriteFile(h.logFile, nil, 0o644)
	}
}

// captureCrashes sends a crash in any goroutine to host.log instead of the
// stderr nobody reads.
func (h *host) captureCrashes() {
	f, err := os.OpenFile(h.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_ = debug.SetCrashOutput(f, debug.CrashOptions{})
}

// loadClientID reads DISCORD_CLIENT_ID, or else clientId from config.json.
// It returns "" unless the result is a valid Application ID.
func loadClientID(getenv func(string) string, configFile string) string {
	id := getenv("DISCORD_CLIENT_ID")
	if id == "" {
		id = ReadConfigClientID(configFile)
	}
	if !ClientIDPattern.MatchString(id) {
		return ""
	}
	return id
}

// ReadConfigClientID is clientId from a config.json, or "".
func ReadConfigClientID(configFile string) string {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return ""
	}
	var config struct {
		ClientID json.RawMessage `json:"clientId"`
	}
	if json.Unmarshal(data, &config) != nil || len(config.ClientID) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(config.ClientID, &s) == nil {
		return strings.TrimSpace(s)
	}
	return string(config.ClientID) // a bare number
}
