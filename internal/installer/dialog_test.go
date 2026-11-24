package installer

import (
	"strings"
	"testing"
)

// The macOS app asks and finishes through callbacks instead of the console.
func (e *env) runWithDialogs(ask func(note string) (string, bool)) (code, finished int) {
	e.stdout.Reset()
	e.stderr.Reset()
	finished = -1
	code = Run(Options{
		Stdin:       strings.NewReader(""), // must not be read
		Stdout:      &e.stdout,
		Stderr:      &e.stderr,
		Getenv:      e.getenv,
		GOOS:        e.goos,
		Home:        e.home,
		Executable:  e.exe,
		Extension:   testExtension(),
		Interactive: true,
		Open:        func(dir string) { e.opened = append(e.opened, dir) },
		Ask:         ask,
		Finish:      func(c int) { finished = c },
	})
	return code, finished
}

func TestDialogAsksAgainAfterBadID(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	var notes []string
	answers := []string{"not a number", " 123456789012345678 "}
	code, finished := e.runWithDialogs(func(note string) (string, bool) {
		notes = append(notes, note)
		a := answers[0]
		answers = answers[1:]
		return a, true
	})
	if code != 0 || finished != 0 {
		t.Fatalf("code %d, finished %d\n%s%s", code, finished, e.stdout.String(), e.stderr.String())
	}
	if len(notes) != 2 || notes[0] != "" || notes[1] != badClientIDMessage {
		t.Fatalf("notes %q", notes)
	}
	if !strings.Contains(e.stdout.String(), "ลงทะเบียนโปรแกรมเสริมกับ Chrome แล้ว") {
		t.Fatalf("missing success line:\n%s", e.stdout.String())
	}
	if strings.Contains(e.stdout.String(), "กด Enter") {
		t.Fatal("dialog mode must not ask to press Enter")
	}
	if len(e.opened) != 1 {
		t.Fatalf("extension folder opened %d times", len(e.opened))
	}
}

func TestDialogCancel(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	code, finished := e.runWithDialogs(func(string) (string, bool) { return "", false })
	if code != 1 || finished != 1 {
		t.Fatalf("code %d, finished %d", code, finished)
	}
	if !strings.Contains(e.stderr.String(), "ยกเลิกการติดตั้ง") {
		t.Fatalf("stderr: %s", e.stderr.String())
	}
}

func TestDialogReusesSavedID(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	if code := e.run("", "123456789012345678"); code != 0 {
		t.Fatal("first install failed")
	}
	code, finished := e.runWithDialogs(func(string) (string, bool) {
		t.Fatal("must not ask when an ID is saved")
		return "", false
	})
	if code != 0 || finished != 0 {
		t.Fatalf("code %d, finished %d", code, finished)
	}
}
