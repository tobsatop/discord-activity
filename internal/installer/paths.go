package installer

import (
	"path/filepath"
)

// HostName must match HOST_NAME in extension/background.js.
const HostName = "local.ytm_discord_presence"

// Overrides for tests and CI, so they never touch the real user folders.
const (
	// EnvDataDir replaces the per-user data folder (installer and helper).
	EnvDataDir = "YTM_DISCORD_DATA_DIR"
	// EnvBrowserHome replaces the home folder when looking for browsers (macOS/Linux).
	EnvBrowserHome = "YTM_DISCORD_BROWSER_HOME"
	// EnvRegistryKey replaces HKCU\Software as the root of the browser keys (Windows).
	EnvRegistryKey = "YTM_DISCORD_REGISTRY_KEY"
)

const appFolder = "ytm-discord-presence"

// DataDir is the per-user folder for the helper, its config, log and the
// extension. It is outside Documents/Desktop/Downloads on purpose: macOS won't
// let Chrome start a program that lives there.
func DataDir(goos, home string, getenv func(string) string) string {
	if dir := getenv(EnvDataDir); dir != "" {
		return dir
	}
	switch goos {
	case "windows":
		base := getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, appFolder)
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", appFolder)
	default:
		base := getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(base, appFolder)
	}
}

// HelperName is the file name of the installed helper.
func HelperName(goos string) string {
	if goos == "windows" {
		return "ytm-discord-host.exe"
	}
	return "ytm-discord-host"
}

// target is where one browser looks for the native host manifest: a file
// (macOS/Linux) or a registry key under HKCU (Windows).
type target struct {
	Browser     string
	File        string
	RegistryKey string
}

func targets(goos, home string, getenv func(string) string) []target {
	if goos == "windows" {
		root := getenv(EnvRegistryKey)
		if root == "" {
			root = `Software`
		}
		return []target{
			{Browser: "Chrome", RegistryKey: root + `\Google\Chrome\NativeMessagingHosts\` + HostName},
			{Browser: "Edge", RegistryKey: root + `\Microsoft\Edge\NativeMessagingHosts\` + HostName},
		}
	}
	if h := getenv(EnvBrowserHome); h != "" {
		home = h
	}
	type profile struct{ browser, dir string }
	profiles := []profile{
		{"Chrome", ".config/google-chrome"},
		{"Edge", ".config/microsoft-edge"},
		{"Brave", ".config/BraveSoftware/Brave-Browser"},
		{"Chromium", ".config/chromium"},
	}
	if goos == "darwin" {
		profiles = []profile{
			{"Chrome", "Library/Application Support/Google/Chrome"},
			{"Edge", "Library/Application Support/Microsoft Edge"},
			{"Brave", "Library/Application Support/BraveSoftware/Brave-Browser"},
			{"Chromium", "Library/Application Support/Chromium"},
		}
	}
	out := make([]target, len(profiles))
	for i, p := range profiles {
		out[i] = target{
			Browser: p.browser,
			File:    filepath.Join(home, filepath.FromSlash(p.dir), "NativeMessagingHosts", HostName+".json"),
		}
	}
	return out
}
