//go:build !windows && !darwin

package installer

import (
	"os/exec"
	"path/filepath"
)

// Open shows the folder that holds the extension and opens chrome://extensions
// if Chrome is on PATH. Best effort.
func Open(extensionDir string) {
	_ = exec.Command("xdg-open", filepath.Dir(extensionDir)).Start()
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			_ = exec.Command(p, "chrome://extensions").Start()
			return
		}
	}
}
