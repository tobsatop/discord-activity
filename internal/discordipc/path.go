package discordipc

import (
	"fmt"
	"strings"
)

// PipeCount: Discord listens on the first free of discord-ipc-0..9.
const PipeCount = 10

// EnvIPC overrides where to look for Discord: a path without the "-N" suffix,
// e.g. /tmp/test/discord-ipc or \\.\pipe\ytmd-test-discord-ipc. Tests use it
// so they never reach the real Discord app.
const EnvIPC = "YTM_DISCORD_IPC"

// Path is where Discord listens for index i on the given OS.
func Path(i int, goos string, getenv func(string) string) string {
	if base := getenv(EnvIPC); base != "" {
		return fmt.Sprintf("%s-%d", base, i)
	}
	if goos == "windows" {
		return fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
	}
	base := "/tmp"
	for _, name := range []string{"XDG_RUNTIME_DIR", "TMPDIR", "TMP", "TEMP"} {
		if v := getenv(name); v != "" {
			base = v
			break
		}
	}
	return fmt.Sprintf("%s/discord-ipc-%d", strings.TrimRight(base, "/"), i)
}
