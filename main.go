// ytm-discord: one program that is both the installer and the native
// messaging helper for the YouTube Music → Discord extension.
//
//   - Chrome starts it with the extension's origin (chrome-extension://…/,
//     plus --parent-window= on Windows): helper mode, see internal/helper.
//   - Double-clicked, or given an Application ID or --uninstall: installer
//     mode, see internal/installer.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tobsatop/discord-activity/internal/helper"
	"github.com/tobsatop/discord-activity/internal/installer"
)

// The extension the installer unpacks for Chrome's "Load unpacked".
// Only the files Chrome needs (no TypeScript config or type declarations).
//
//go:embed extension/manifest.json extension/*.js extension/icons/*.png
var embedded embed.FS

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if isHelperLaunch(args) {
		return helper.Run(helper.Options{
			Version: version,
			DataDir: helperDataDir(),
			Stdin:   os.Stdin,
			Stdout:  os.Stdout,
			Getenv:  os.Getenv,
		})
	}
	for _, arg := range args {
		switch arg {
		case "--version":
			fmt.Println(version)
			return 0
		case "--help", "-h":
			fmt.Println("ytm-discord " + version)
			fmt.Println("  ytm-discord                     ask for the Discord Application ID, then install")
			fmt.Println("  ytm-discord <Application ID>    install without asking")
			fmt.Println("  ytm-discord --uninstall         remove the helper")
			return 0
		}
	}
	extension, err := fs.Sub(embedded, "extension")
	if err != nil {
		panic(err)
	}
	home, _ := os.UserHomeDir()
	return installer.Run(installer.Options{
		Args:        args,
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Getenv:      os.Getenv,
		GOOS:        runtime.GOOS,
		Home:        home,
		Executable:  executable(),
		Extension:   extension,
		Interactive: len(args) == 0,
		Open:        installer.Open,
	})
}

// isHelperLaunch: Chrome passes the caller's origin, and on Windows also
// --parent-window=<handle>.
func isHelperLaunch(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "chrome-extension://") || strings.HasPrefix(arg, "--parent-window=") {
			return true
		}
	}
	return false
}

// helperDataDir: the installer puts the helper in the data folder, so its
// config and log live next to it.
func helperDataDir() string {
	if dir := os.Getenv(installer.EnvDataDir); dir != "" {
		return dir
	}
	return filepath.Dir(executable())
}

func executable() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}
