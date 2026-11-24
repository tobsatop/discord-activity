package fakediscord

import (
	"net"

	"github.com/Microsoft/go-winio"
)

// Listen listens on a socket path (or pipe name on Windows).
func Listen(path string) (net.Listener, error) { return winio.ListenPipe(path, nil) }
