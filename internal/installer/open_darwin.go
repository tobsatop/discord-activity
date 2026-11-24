package installer

import (
	"os/exec"
)

// Open reveals the extension folder in Finder, ready to drag onto
// chrome://extensions or pick with Load unpacked. Best effort.
//
// It does not open chrome://extensions itself: Chrome ignores chrome:// URLs
// handed to it from outside, so users type the address (README step 4).
func Open(extensionDir string) {
	_ = exec.Command("open", "-R", extensionDir).Start()
}
