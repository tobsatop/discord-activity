//go:build darwin

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// When the installer runs as "YouTube Music Discord Setup.app" (double-clicked in
// Finder, so there is no Terminal window), it talks through dialog boxes
// shown by the system's osascript.

const dialogTitle = "YouTube Music → Discord"

var errCanceled = errors.New("canceled")

func useDialogs(args []string) bool {
	exe, err := os.Executable()
	if err != nil || !strings.Contains(exe, ".app/Contents/MacOS/") || len(args) > 0 {
		return false
	}
	// Opened from Finder, stdin is /dev/null: a character device but not a
	// terminal, so ask the terminal driver rather than checking the file mode.
	_, err = unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	return err != nil // no terminal attached
}

func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// iconClause uses the app's own icon in the dialogs when it is there.
func iconClause() string {
	if exe, err := os.Executable(); err == nil {
		icns := filepath.Join(filepath.Dir(filepath.Dir(exe)), "Resources", "AppIcon.icns")
		if _, err := os.Stat(icns); err == nil {
			return " with icon POSIX file " + quote(icns)
		}
	}
	return " with icon note"
}

func osa(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", "activate", "-e", script).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && strings.Contains(string(ee.Stderr), "-128") {
			return "", errCanceled
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// dialogErr remembers a dialog that failed for a reason other than Cancel,
// so the end of the run can still tell the person what happened.
var dialogErr error

// osaDialog runs a dialog script with the app icon, and once more with the
// standard icon if that fails for a reason other than Cancel.
func osaDialog(script func(icon string) string) (string, error) {
	out, err := osa(script(iconClause()))
	if err != nil && !errors.Is(err, errCanceled) {
		out, err = osa(script(" with icon note"))
	}
	return out, err
}

func dialogAsk(note string) (string, bool) {
	msg := "วาง Application ID จาก Discord Developer Portal"
	if note != "" {
		msg = note + "\n\n" + msg
	}
	out, err := osaDialog(func(icon string) string {
		return `text returned of (display dialog ` + quote(msg) +
			` default answer "" buttons {"ยกเลิก", "ติดตั้ง"} default button "ติดตั้ง" cancel button "ยกเลิก" with title ` +
			quote(dialogTitle) + icon + `)`
	})
	if err != nil {
		if !errors.Is(err, errCanceled) {
			dialogErr = err
		}
		return "", false
	}
	return out, true
}

func dialogFinish(output string, code int) {
	output = strings.TrimSpace(output)
	if output == "ยกเลิกการติดตั้ง" && dialogErr == nil {
		return // the person pressed Cancel; nothing more to say
	}
	head := "ติดตั้งเสร็จแล้ว"
	if code != 0 {
		head = "ติดตั้งไม่สำเร็จ"
	}
	if dialogErr != nil {
		head = "ติดตั้งไม่สำเร็จ: เปิดกล่องข้อความไม่ได้ (" + dialogErr.Error() + ")"
	}
	text := head + "\n\n" + output
	_, err := osaDialog(func(icon string) string {
		return `display dialog ` + quote(text) + ` buttons {"ตกลง"} default button "ตกลง" with title ` + quote(dialogTitle) + icon
	})
	if err != nil && !errors.Is(err, errCanceled) {
		// No dialogs at all: put the result in a text file and open it.
		file := filepath.Join(os.TempDir(), "YouTube Music Discord Setup.txt")
		if os.WriteFile(file, []byte(text+"\n"), 0o644) == nil {
			_ = exec.Command("open", "-e", file).Run()
		}
	}
}
