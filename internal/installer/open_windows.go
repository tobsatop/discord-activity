package installer

import (
	"os/exec"
	"syscall"
)

// Open shows the extension folder (selected) in Explorer, ready to drag onto
// chrome://extensions or pick with Load unpacked. Best effort.
//
// It does not open chrome://extensions itself: Chrome ignores chrome:// URLs
// passed on its command line, so users type the address (README step 4).
func Open(extensionDir string) {
	explorer := exec.Command("explorer.exe")
	// explorer.exe parses its own command line; Go's quoting breaks /select with spaces.
	explorer.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + extensionDir + `"`}
	_ = explorer.Start()
}
