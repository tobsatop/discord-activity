package installer

import "golang.org/x/sys/unix"

// clearQuarantine drops the "downloaded from the internet" mark, so Gatekeeper
// doesn't stop Chrome from starting the installed copy. Best effort.
func clearQuarantine(path string) {
	_ = unix.Removexattr(path, "com.apple.quarantine")
}
