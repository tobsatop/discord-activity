package discordipc_test

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tobsatop/discord-activity/internal/activity"
	"github.com/tobsatop/discord-activity/internal/discordipc"
	"github.com/tobsatop/discord-activity/internal/fakediscord"
)

func TestMain(m *testing.M) {
	// Safety net: anything that forgets PathFor still never finds the real Discord.
	dir, err := os.MkdirTemp("", "ytmd-")
	if err != nil {
		panic(err)
	}
	os.Setenv(discordipc.EnvIPC, filepath.Join(dir, "nothing"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func TestPathPerPlatform(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	cases := []struct {
		i    int
		goos string
		env  map[string]string
		want string
	}{
		{0, "windows", nil, `\\.\pipe\discord-ipc-0`},
		{3, "darwin", map[string]string{"TMPDIR": "/var/folders/x/T/"}, "/var/folders/x/T/discord-ipc-3"},
		{0, "linux", map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000", "TMPDIR": "/tmp"}, "/run/user/1000/discord-ipc-0"},
		{0, "linux", nil, "/tmp/discord-ipc-0"},
		{2, "windows", map[string]string{discordipc.EnvIPC: `\\.\pipe\test`}, `\\.\pipe\test-2`},
		{1, "darwin", map[string]string{discordipc.EnvIPC: "/x/discord-ipc", "TMPDIR": "/y"}, "/x/discord-ipc-1"},
	}
	for _, c := range cases {
		if got := discordipc.Path(c.i, c.goos, env(c.env)); got != c.want {
			t.Errorf("Path(%d, %s, %v) = %s, want %s", c.i, c.goos, c.env, got, c.want)
		}
	}
}

func TestFramesSplitAndGlued(t *testing.T) {
	bytes := append(discordipc.EncodeFrame(1, map[string]any{"a": "ไทย"}), discordipc.EncodeFrame(3, map[string]any{"b": 2})...)
	var d discordipc.Decoder
	var frames []discordipc.Frame
	frames = append(frames, d.Push(bytes[:5])...)
	frames = append(frames, d.Push(bytes[5:20])...)
	frames = append(frames, d.Push(bytes[20:])...)
	if len(frames) != 2 || frames[0].Op != 1 || frames[1].Op != 3 {
		t.Fatalf("frames %+v", frames)
	}
	if got := frames[0].Data.(map[string]any)["a"]; got != "ไทย" {
		t.Fatalf("a = %v", got)
	}
	if got := fmt.Sprint(frames[1].Data.(map[string]any)["b"]); got != "2" {
		t.Fatalf("b = %v", got)
	}
}

func TestCorruptLengthDropsBuffer(t *testing.T) {
	var d discordipc.Decoder
	bad := []byte{1, 0, 0, 0, 0xff, 0xff, 0xff, 0x7f}
	if frames := d.Push(bad); len(frames) != 0 {
		t.Fatalf("frames %+v", frames)
	}
	if frames := d.Push(discordipc.EncodeFrame(1, "ok")); len(frames) != 1 {
		t.Fatalf("decoder did not recover: %+v", frames)
	}
}

type recorder struct {
	mu       sync.Mutex
	statuses []discordipc.Status
	errors   []string
}

func (r *recorder) onStatus(s discordipc.Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statuses = append(r.statuses, s)
}

func (r *recorder) has(state, detailPart string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.statuses {
		if s.State == state && strings.Contains(s.Detail, detailPart) {
			return true
		}
	}
	return false
}

func (r *recorder) count(state string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, s := range r.statuses {
		if s.State == state {
			n++
		}
	}
	return n
}

func until(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func newClient(t *testing.T, base string, r *recorder, tweak func(*discordipc.Options)) *discordipc.Client {
	opts := discordipc.Options{
		ClientID:        "123456789012345678",
		Log:             func(string) {},
		OnStatus:        r.onStatus,
		OnActivityError: func(m string) { r.mu.Lock(); r.errors = append(r.errors, m); r.mu.Unlock() },
		PathFor:         func(i int) string { return fmt.Sprintf("%s-%d", base, i) },
	}
	if tweak != nil {
		tweak(&opts)
	}
	c := discordipc.New(opts)
	t.Cleanup(c.Stop)
	return c
}

func details(list []*activity.Activity) []string {
	out := make([]string, len(list))
	for i, a := range list {
		if a == nil {
			out[i] = "<nil>"
		} else {
			out[i] = a.Details
		}
	}
	return out
}

func TestRateLimitWaitsAndSendsOnlyTheLatest(t *testing.T) {
	discord := fakediscord.Start(t, fakediscord.Options{})
	r := &recorder{}
	ipc := newClient(t, discord.Base, r, func(o *discordipc.Options) {
		o.RateWindow = 400 * time.Millisecond
		o.RateMax = 2
	})
	ipc.Start()
	act := func(d string) *activity.Activity { return &activity.Activity{Type: 2, Details: d} }

	ipc.SetActivity(act("one"))
	discord.WaitFor(t, func(l []*activity.Activity) bool { return len(l) == 1 })
	firstAt := time.Now()
	ipc.SetActivity(act("two"))
	discord.WaitFor(t, func(l []*activity.Activity) bool { return len(l) == 2 })
	ipc.SetActivity(act("three")) // over the limit: waits for the window
	ipc.SetActivity(act("four"))  // replaces "three" before it is sent
	discord.WaitFor(t, func(l []*activity.Activity) bool { return len(l) == 3 })
	if waited := time.Since(firstAt); waited < 300*time.Millisecond {
		t.Fatalf("third update only waited %v", waited)
	}
	if got := details(discord.Activities()); !reflect.DeepEqual(got, []string{"one", "two", "four"}) {
		t.Fatalf("got %v", got)
	}
	if !r.has("connecting", "") || !r.has("connected", "") {
		t.Fatalf("statuses %+v", r.statuses)
	}
}

func TestStopClearsTheActivity(t *testing.T) {
	discord := fakediscord.Start(t, fakediscord.Options{})
	ipc := newClient(t, discord.Base, &recorder{}, nil)
	ipc.Start()
	ipc.SetActivity(&activity.Activity{Type: 2, Details: "song"})
	discord.WaitFor(t, func(l []*activity.Activity) bool { return len(l) == 1 })
	start := time.Now()
	ipc.Stop()
	if got := details(discord.Activities()); !reflect.DeepEqual(got, []string{"song", "<nil>"}) {
		t.Fatalf("got %v", got)
	}
	if time.Since(start) > 900*time.Millisecond {
		t.Fatalf("Stop took %v; it should return once Discord answers", time.Since(start))
	}
}

func TestSameActivityIsNotResent(t *testing.T) {
	discord := fakediscord.Start(t, fakediscord.Options{})
	ipc := newClient(t, discord.Base, &recorder{}, nil)
	ipc.Start()
	ipc.SetActivity(&activity.Activity{Type: 2, Details: "song"})
	discord.WaitFor(t, func(l []*activity.Activity) bool { return len(l) == 1 })
	ipc.SetActivity(&activity.Activity{Type: 2, Details: "song"})
	ipc.SetActivity(nil)
	discord.WaitFor(t, func(l []*activity.Activity) bool { return len(l) == 2 })
	time.Sleep(50 * time.Millisecond)
	if got := details(discord.Activities()); !reflect.DeepEqual(got, []string{"song", "<nil>"}) {
		t.Fatalf("got %v", got)
	}
}

func TestReportsUnavailableAndReconnects(t *testing.T) {
	r := &recorder{}
	ipc := newClient(t, fakediscord.NewBase(t), r, func(o *discordipc.Options) { o.RetryInterval = 50 * time.Millisecond })
	ipc.Start()
	until(t, "unavailable", func() bool { return r.has("unavailable", "Discord app is not running") })
	if r.count("unavailable") != 1 {
		t.Fatalf("the same status is reported once: %+v", r.statuses)
	}
}

// scripted is a one-connection fake that runs the given script on the server side.
func scripted(t *testing.T, script func(c net.Conn, d *discordipc.Decoder)) string {
	t.Helper()
	base := fakediscord.NewBase(t)
	ln, err := fakediscord.Listen(base + "-0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				script(c, &discordipc.Decoder{})
			}()
		}
	}()
	return base
}

func readFrame(c net.Conn, d *discordipc.Decoder) (discordipc.Frame, bool) {
	buf := make([]byte, 4096)
	for {
		n, err := c.Read(buf)
		if frames := d.Push(buf[:n]); len(frames) > 0 {
			return frames[0], true
		}
		if err != nil {
			return discordipc.Frame{}, false
		}
	}
}

func TestCloseFromDiscordIsAnError(t *testing.T) {
	base := scripted(t, func(c net.Conn, d *discordipc.Decoder) {
		if f, ok := readFrame(c, d); ok && f.Op == discordipc.OpHandshake {
			c.Write(discordipc.EncodeFrame(discordipc.OpClose, map[string]any{"code": 4000, "message": "Invalid Client ID"}))
			readFrame(c, d) // until the client hangs up
		}
	})
	r := &recorder{}
	ipc := newClient(t, base, r, nil)
	ipc.Start()
	want := "Discord ไม่รับการเชื่อมต่อ: Invalid Client ID — ตรวจ Application ID (README หัวข้อเปลี่ยน Application ID)"
	until(t, "error status", func() bool { return r.has("error", want) })
}

func TestPingGetsPongAndClosedConnectionReconnects(t *testing.T) {
	pongs := make(chan any, 4)
	var mu sync.Mutex
	conns := 0
	base := scripted(t, func(c net.Conn, d *discordipc.Decoder) {
		mu.Lock()
		conns++
		n := conns
		mu.Unlock()
		if _, ok := readFrame(c, d); !ok {
			return
		}
		c.Write(discordipc.EncodeFrame(discordipc.OpFrame, map[string]any{"cmd": "DISPATCH", "evt": "READY"}))
		if n > 1 {
			readFrame(c, d)
			return
		}
		c.Write(discordipc.EncodeFrame(discordipc.OpPing, map[string]any{"x": 1}))
		if f, ok := readFrame(c, d); ok && f.Op == discordipc.OpPong {
			pongs <- f.Data
		}
		// return: closes the connection, as when Discord quits
	})
	r := &recorder{}
	ipc := newClient(t, base, r, func(o *discordipc.Options) { o.RetryInterval = 50 * time.Millisecond })
	ipc.Start()
	select {
	case data := <-pongs:
		if fmt.Sprint(data) != "map[x:1]" {
			t.Fatalf("pong data %v", data)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no pong")
	}
	until(t, "closed", func() bool { return r.has("unavailable", "Discord app closed the connection") })
	until(t, "reconnected", func() bool { return r.count("connected") == 2 })
}
