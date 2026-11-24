package discordipc

import (
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// Dial opens a Discord named pipe. go-winio uses overlapped I/O, so one
// goroutine can read while another writes; a plain os.OpenFile pipe deadlocks.
func Dial(path string) (net.Conn, error) {
	timeout := 2 * time.Second
	return winio.DialPipe(path, &timeout)
}
