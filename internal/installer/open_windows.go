package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// Open shows the extension folder (selected) in Explorer and opens
// chrome://extensions in Chrome if Chrome is installed. Best effort.
func Open(extensionDir string) {
	explorer := exec.Command("explorer.exe")
	// explorer.exe parses its own command line; Go's quoting breaks /select with spaces.
	explorer.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + extensionDir + `"`}
	_ = explorer.Start()

	if chrome := findChrome(); chrome != "" {
		_ = exec.Command(chrome, "chrome://extensions").Start()
	}
}

// findChrome looks Chrome up without "start chrome", which pops up an error
// dialog when Chrome is missing.
func findChrome() string {
	const appPath = `Software\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe`
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		if k, err := registry.OpenKey(root, appPath, registry.QUERY_VALUE); err == nil {
			p, _, err := k.GetStringValue("")
			k.Close()
			if err == nil && exists(p) {
				return p
			}
		}
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		if base := os.Getenv(env); base != "" {
			p := filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe")
			if exists(p) {
				return p
			}
		}
	}
	return ""
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
