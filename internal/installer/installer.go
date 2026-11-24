// Package installer sets the helper up on this computer: it copies the
// program into the per-user data folder, registers it as Chrome's native
// messaging host, and unpacks the extension next to it.
//
//	ytm-discord                     ask for the Discord Application ID (double-click)
//	ytm-discord <Application ID>    same, without asking
//	ytm-discord --uninstall         remove the registration and the data folder
package installer

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tobsatop/discord-activity/internal/helper"
)

// Options for Run. main fills them from the real environment; tests use temp folders.
type Options struct {
	Args        []string // command-line arguments (no program name)
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Getenv      func(string) string
	GOOS        string
	Home        string
	Executable  string // the running program, copied into the data folder
	Extension   fs.FS  // the extension's files, manifest.json at the root
	Interactive bool   // double-clicked: ask, open the folder, wait for Enter
	// Open shows the extension folder in Explorer/Finder.
	// Called only when Interactive.
	Open func(extensionDir string)
}

type run struct {
	Options
	in *bufio.Reader
}

// Run installs or uninstalls and returns the exit code.
func Run(opts Options) int {
	r := &run{Options: opts, in: bufio.NewReader(opts.Stdin)}
	code := r.dispatch()
	if r.Interactive {
		// A double-clicked window closes when the program ends; keep it readable.
		fmt.Fprintln(r.Stdout)
		fmt.Fprint(r.Stdout, "กด Enter เพื่อปิดหน้าต่างนี้")
		_, _ = r.in.ReadString('\n')
	}
	return code
}

func (r *run) dispatch() int {
	var positional []string
	for _, arg := range r.Args {
		switch {
		case arg == "--uninstall":
			return r.uninstall()
		case !strings.HasPrefix(arg, "--"):
			positional = append(positional, arg)
		}
	}
	arg := ""
	if len(positional) > 0 {
		arg = positional[0]
	}
	if err := r.install(arg); err != nil {
		var quiet exitError
		if !errors.As(err, &quiet) {
			fmt.Fprintf(r.Stderr, "ติดตั้งไม่สำเร็จ: %v\n", err)
		}
		return 1
	}
	return 0
}

// exitError: the message was already printed.
type exitError struct{}

func (exitError) Error() string { return "exit" }

func (r *run) dataDir() string { return DataDir(r.GOOS, r.Home, r.Getenv) }

func (r *run) install(arg string) error {
	dataDir := r.dataDir()
	configFile := filepath.Join(dataDir, "config.json")

	clientID, err := r.resolveClientID(arg, configFile, dataDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	config, _ := json.MarshalIndent(map[string]string{"clientId": clientID}, "", "  ")
	if err := os.WriteFile(configFile, append(config, '\n'), 0o644); err != nil {
		return err
	}

	helperPath := filepath.Join(dataDir, HelperName(r.GOOS))
	if err := installHelper(r.Executable, helperPath, r.GOOS); err != nil {
		return fmt.Errorf("copy %s: %w", helperPath, err)
	}
	legacy := removeLegacyFiles(dataDir)

	manifestJSON, err := fs.ReadFile(r.Extension, "manifest.json")
	if err != nil {
		return err
	}
	id, err := ExtensionID(manifestJSON)
	if err != nil {
		return err
	}
	hostManifest := HostManifest(helperPath, id)

	var done []string
	if r.GOOS == "windows" {
		file := filepath.Join(dataDir, "native-host.json")
		if err := os.WriteFile(file, hostManifest, 0o644); err != nil {
			return err
		}
		for _, t := range targets(r.GOOS, r.Home, r.Getenv) {
			if err := setRegistryDefault(t.RegistryKey, file); err != nil {
				fmt.Fprintf(r.Stderr, "ลงทะเบียนกับ %s ไม่สำเร็จ: %v\n", t.Browser, err)
				continue
			}
			done = append(done, t.Browser)
		}
	} else {
		for _, t := range targets(r.GOOS, r.Home, r.Getenv) {
			browserDir := filepath.Dir(filepath.Dir(t.File))
			if _, err := os.Stat(browserDir); err != nil {
				continue // browser not installed
			}
			if err := os.MkdirAll(filepath.Dir(t.File), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(t.File, hostManifest, 0o644); err != nil {
				return err
			}
			done = append(done, t.Browser)
		}
	}
	if len(done) == 0 {
		fmt.Fprintln(r.Stderr, "ไม่พบ Chrome บนเครื่องนี้ — ติดตั้ง Chrome ก่อนแล้วรันใหม่")
		return exitError{}
	}

	extensionDir := filepath.Join(dataDir, "extension")
	_, statErr := os.Stat(filepath.Join(extensionDir, "manifest.json"))
	update := statErr == nil
	if err := Extract(r.Extension, extensionDir); err != nil {
		return fmt.Errorf("extension: %w", err)
	}

	out := r.Stdout
	fmt.Fprintf(out, "ลงทะเบียนโปรแกรมเสริมกับ %s แล้ว\n", strings.Join(done, ", "))
	fmt.Fprintf(out, "ID ส่วนขยาย: %s\n", id)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "ขั้นต่อไป:")
	if update {
		fmt.Fprintln(out, "1. เปิด chrome://extensions แล้วกด ↻ บนการ์ด YouTube Music → Discord")
		fmt.Fprintf(out, "   (ถ้ายังไม่มีการ์ดนี้: เปิด Developer mode กด Load unpacked แล้วเลือกโฟลเดอร์ %s)\n", extensionDir)
		fmt.Fprintln(out, "2. refresh แท็บ YouTube Music")
	} else {
		fmt.Fprintln(out, "1. เปิด chrome://extensions แล้วเปิด Developer mode")
		if legacy {
			fmt.Fprintln(out, "   (การ์ด YouTube Music → Discord เดิมที่โหลดจากโฟลเดอร์โปรเจกต์: กด Remove ก่อน)")
		}
		fmt.Fprintf(out, "2. กด Load unpacked แล้วเลือกโฟลเดอร์ %s\n", extensionDir)
		fmt.Fprintln(out, "3. เปิดแอป Discord แล้วเล่นเพลงที่ music.youtube.com")
	}
	if legacy {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "ไม่ต้องใช้ Node.js และโฟลเดอร์โปรเจกต์เดิมแล้ว ลบทิ้งได้หลังทำขั้นต่อไปเสร็จ")
	}
	if r.Interactive && r.Open != nil {
		r.Open(extensionDir)
	}
	return nil
}

func (r *run) resolveClientID(arg, configFile, dataDir string) (string, error) {
	if arg != "" {
		id := strings.TrimSpace(arg)
		if !helper.ClientIDPattern.MatchString(id) {
			r.badClientID()
			return "", exitError{}
		}
		return id, nil
	}
	for _, existing := range []string{helper.ReadConfigClientID(configFile), legacyClientID(dataDir)} {
		if helper.ClientIDPattern.MatchString(existing) {
			fmt.Fprintf(r.Stdout, "ใช้ Application ID เดิม: %s\n", existing)
			return existing, nil
		}
	}
	for {
		fmt.Fprint(r.Stdout, "วาง Application ID จาก Discord Developer Portal: ")
		line, err := r.in.ReadString('\n')
		id := strings.TrimSpace(line)
		if helper.ClientIDPattern.MatchString(id) {
			return id, nil
		}
		if err != nil || !r.Interactive {
			if err != nil {
				fmt.Fprintln(r.Stdout)
			}
			r.badClientID()
			return "", exitError{}
		}
		r.badClientID() // double-clicked: let them paste again
	}
}

func (r *run) badClientID() {
	fmt.Fprintln(r.Stderr, "Application ID ต้องเป็นตัวเลข 17–20 หลัก (Discord Developer Portal → General Information)")
}

func (r *run) uninstall() int {
	for _, t := range targets(r.GOOS, r.Home, r.Getenv) {
		if t.RegistryKey != "" {
			_ = deleteRegistryKey(t.RegistryKey) // was not registered
		} else {
			_ = os.Remove(t.File)
		}
	}
	dataDir := r.dataDir()
	_ = os.RemoveAll(dataDir)
	fmt.Fprintln(r.Stdout, "ถอนการลงทะเบียนแล้ว (ลบส่วนขยายเองได้ที่ chrome://extensions)")
	if _, err := os.Stat(dataDir); err == nil {
		// Windows can't delete a program that is running (Chrome may be using the helper).
		fmt.Fprintf(r.Stdout, "ลบโฟลเดอร์ %s ไม่หมด — ปิด Chrome แล้วลบโฟลเดอร์นี้เองได้\n", dataDir)
	}
	return 0
}

// ExtensionID is the id Chrome gives an unpacked extension, derived from the
// "key" in its manifest.
func ExtensionID(manifestJSON []byte) (string, error) {
	var m struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return "", err
	}
	der, err := base64.StdEncoding.DecodeString(m.Key)
	if err != nil || len(der) == 0 {
		return "", fmt.Errorf("manifest.json has no valid key")
	}
	sum := sha256.Sum256(der)
	hexID := hex.EncodeToString(sum[:16])
	id := make([]byte, len(hexID))
	for i, c := range hexID {
		v := strings.IndexRune("0123456789abcdef", c)
		id[i] = byte('a' + v)
	}
	return string(id), nil
}

// HostManifest is the native messaging host manifest Chrome reads.
func HostManifest(helperPath, extensionID string) []byte {
	m := struct {
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		Path           string   `json:"path"`
		Type           string   `json:"type"`
		AllowedOrigins []string `json:"allowed_origins"`
	}{
		Name:           HostName,
		Description:    "YouTube Music to Discord Rich Presence",
		Path:           helperPath,
		Type:           "stdio",
		AllowedOrigins: []string{"chrome-extension://" + extensionID + "/"},
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(m)
	return b.Bytes()
}

// installHelper copies the running program to dst. A helper Chrome is still
// running can't be overwritten on Windows, but it can be renamed out of the way.
func installHelper(src, dst, goos string) error {
	if src == "" {
		return errors.New("cannot find this program's own path")
	}
	if same(src, dst) {
		return nil // running the installed copy itself
	}
	olds, _ := filepath.Glob(dst + ".old-*")
	for _, old := range olds {
		_ = os.Remove(old) // left by an earlier install; fails if still running
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	_ = os.Chmod(tmp, 0o755)
	clearQuarantine(tmp)
	if err := os.Rename(tmp, dst); err == nil {
		return nil
	}
	old := fmt.Sprintf("%s.old-%d", dst, time.Now().UnixNano())
	if err := os.Rename(dst, old); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Rename(old, dst)
		return err
	}
	_ = os.Remove(old)
	return nil
}

func same(a, b string) bool {
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	return err == nil && os.SameFile(fa, fb)
}

// removeLegacyFiles deletes the launchers the Node.js installer wrote and
// reports whether there were any.
func removeLegacyFiles(dataDir string) bool {
	found := false
	for _, name := range []string{"run-host.bat", "run-host.sh"} {
		if os.Remove(filepath.Join(dataDir, name)) == nil {
			found = true
		}
	}
	return found
}

var legacyMain = regexp.MustCompile(`"([^"]+)[\\/]main\.ts"`)

// legacyClientID finds the Application ID of a Node.js install: its launcher
// names <project>/host/main.ts, and the ID is in <project>/host/config.json.
func legacyClientID(dataDir string) string {
	for _, name := range []string{"run-host.bat", "run-host.sh"} {
		data, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			continue
		}
		if m := legacyMain.FindSubmatch(data); m != nil {
			if id := helper.ReadConfigClientID(filepath.Join(string(m[1]), "config.json")); id != "" {
				return id
			}
		}
	}
	return ""
}

// Extract writes the extension into dir and removes files that are no longer
// part of it, so an update doesn't leave old scripts behind.
func Extract(files fs.FS, dir string) error {
	keep := map[string]bool{}
	err := fs.WalkDir(files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		keep[p] = true
		target := filepath.Join(dir, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(files, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return err
	}
	var stale []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if !keep[filepath.ToSlash(rel)] {
			stale = append(stale, p)
			if d.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
	for _, p := range stale {
		_ = os.RemoveAll(p)
	}
	return nil
}
