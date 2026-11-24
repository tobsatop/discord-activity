// Package fakediscord is a stand-in for the Discord app's IPC socket, for tests.
// It listens on a private path (a socket in a temp folder, or a uniquely named
// pipe on Windows), never on Discord's real one.
package fakediscord

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/tobsatop/discord-activity/internal/activity"
	"github.com/tobsatop/discord-activity/internal/discordipc"
)

// Options change how the fake answers.
type Options struct {
	// RejectDetailsURL answers an activity that has details_url with an error,
	// like Discord versions that don't know the field.
	RejectDetailsURL bool
}

// Server records every SET_ACTIVITY it receives.
type Server struct {
	// Base goes in YTM_DISCORD_IPC (or discordipc.Options.PathFor) so the
	// client finds this fake at index 0.
	Base string

	opts Options
	ln   net.Listener

	mu         sync.Mutex
	activities []*activity.Activity
	conns      map[net.Conn]bool
}

// ShortTempDir makes a temp folder with a short path: Unix socket paths are
// limited to about 104 bytes.
func ShortTempDir(t testing.TB) string {
	t.Helper()
	base := os.TempDir()
	if runtime.GOOS != "windows" && len(base) >= 60 {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "ytmd-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// NewBase is a fresh IPC base (see Server.Base) where nothing listens yet.
func NewBase(t testing.TB) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return `\\.\pipe\ytmd-test-` + randomName() + `-discord-ipc`
	}
	return ShortTempDir(t) + "/discord-ipc"
}

// Start listens and stops when the test ends.
func Start(t testing.TB, opts Options) *Server {
	t.Helper()
	s := &Server{Base: NewBase(t), opts: opts, conns: map[net.Conn]bool{}}
	var err error
	s.ln, err = Listen(s.Base + "-0")
	if err != nil {
		t.Fatal(err)
	}
	go s.accept()
	t.Cleanup(s.Close)
	return s
}

// Close stops listening and drops every connection.
func (s *Server) Close() {
	_ = s.ln.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.conns {
		_ = c.Close()
	}
}

// Activities is every SET_ACTIVITY so far, in order (nil = cleared).
func (s *Server) Activities() []*activity.Activity {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*activity.Activity(nil), s.activities...)
}

// WaitFor waits until ready returns true for the activities received so far.
func (s *Server) WaitFor(t testing.TB, ready func([]*activity.Activity) bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		list := s.Activities()
		if ready(list) {
			return
		}
		if time.Now().After(deadline) {
			b, _ := json.Marshal(list)
			t.Fatalf("timed out; got %s", b)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *Server) accept() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[c] = true
		s.mu.Unlock()
		go s.serve(c)
	}
}

func (s *Server) serve(c net.Conn) {
	defer func() {
		_ = c.Close()
		s.mu.Lock()
		delete(s.conns, c)
		s.mu.Unlock()
	}()
	var dec discordipc.Decoder
	buf := make([]byte, 64*1024)
	for {
		n, err := c.Read(buf)
		for _, f := range dec.Push(buf[:n]) {
			s.handle(c, f)
		}
		if err != nil {
			return
		}
	}
}

func (s *Server) handle(c net.Conn, f discordipc.Frame) {
	data, _ := f.Data.(map[string]any)
	switch {
	case f.Op == discordipc.OpHandshake:
		_, _ = c.Write(discordipc.EncodeFrame(discordipc.OpFrame, map[string]any{
			"cmd": "DISPATCH", "evt": "READY", "data": map[string]any{"user": map[string]any{"username": "tester"}},
		}))
	case f.Op == discordipc.OpFrame && data["cmd"] == "SET_ACTIVITY":
		var a *activity.Activity
		if args, ok := data["args"].(map[string]any); ok && args["activity"] != nil {
			raw, _ := json.Marshal(args["activity"])
			a = new(activity.Activity)
			_ = json.Unmarshal(raw, a)
		}
		s.mu.Lock()
		s.activities = append(s.activities, a)
		s.mu.Unlock()
		reply := map[string]any{"cmd": "SET_ACTIVITY", "evt": nil, "data": a, "nonce": data["nonce"]}
		if s.opts.RejectDetailsURL && a != nil && a.DetailsURL != "" {
			reply["evt"] = "ERROR"
			reply["data"] = map[string]any{"code": 4000, "message": "child 'details_url' not allowed"}
		}
		_, _ = c.Write(discordipc.EncodeFrame(discordipc.OpFrame, reply))
	}
}

func randomName() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
