package installer

import (
	"os/exec"
)

// Open reveals the extension folder in Finder and brings up chrome://extensions
// in Chrome if it can. Best effort: failures are ignored.
func Open(extensionDir string) {
	_ = exec.Command("open", "-R", extensionDir).Start()
	_ = exec.Command("open", "-a", "Google Chrome", "chrome://extensions").Start()
}
