//go:build !windows && !darwin

package installer

import (
	"os/exec"
	"path/filepath"
)

// Open shows the folder that holds the extension. Best effort. Chrome ignores
// chrome:// URLs passed on its command line, so users type chrome://extensions.
func Open(extensionDir string) {
	_ = exec.Command("xdg-open", filepath.Dir(extensionDir)).Start()
}
