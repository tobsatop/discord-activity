//go:build !windows

package fakediscord

import "net"

// Listen listens on a socket path (or pipe name on Windows).
func Listen(path string) (net.Listener, error) { return net.Listen("unix", path) }
