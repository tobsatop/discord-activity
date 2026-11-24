//go:build !windows

package discordipc

import (
	"net"
	"time"
)

// Dial opens a Discord Unix socket.
func Dial(path string) (net.Conn, error) {
	return net.DialTimeout("unix", path, 2*time.Second)
}
