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

func dialogAsk(note string) (string, bool) {
	msg := "วาง Application ID จาก Discord Developer Portal"
	if note != "" {
		msg = note + "\n\n" + msg
	}
	out, err := osa(`text returned of (display dialog ` + quote(msg) +
		` default answer "" buttons {"ยกเลิก", "ติดตั้ง"} default button "ติดตั้ง" cancel button "ยกเลิก" with title ` +
		quote(dialogTitle) + iconClause() + `)`)
	if err != nil {
		return "", false
	}
	return out, true
}

func dialogFinish(output string, code int) {
	output = strings.TrimSpace(output)
	if output == "ยกเลิกการติดตั้ง" {
		return // the person pressed Cancel; nothing more to say
	}
	head := "ติดตั้งเสร็จแล้ว"
	if code != 0 {
		head = "ติดตั้งไม่สำเร็จ"
	}
	_, _ = osa(`display dialog ` + quote(head+"\n\n"+output) +
		` buttons {"ตกลง"} default button "ตกลง" with title ` + quote(dialogTitle) + iconClause())
}
