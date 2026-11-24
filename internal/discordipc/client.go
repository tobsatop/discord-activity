package discordipc

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tobsatop/discord-activity/internal/activity"
)

// Status is what the helper reports to the extension about Discord.
// State is connecting, connected, unavailable or error.
type Status struct {
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

// Options configure a Client. Only ClientID, Log, OnStatus and OnActivityError are required.
type Options struct {
	ClientID string
	Log      func(line string)
	OnStatus func(Status)
	// OnActivityError: Discord answered SET_ACTIVITY with an error (usually a
	// field it doesn't accept). It runs on the client's goroutine and may call
	// SetActivity.
	OnActivityError func(message string)

	RetryInterval time.Duration // default 15s
	RateWindow    time.Duration // default 20s
	RateMax       int           // default 5 SET_ACTIVITY per RateWindow, Discord's limit

	PathFor func(i int) string                  // default Path with the real environment
	Dial    func(path string) (net.Conn, error) // default Dial
}

// Client keeps one connection to Discord, reconnects on its own and respects
// the SET_ACTIVITY rate limit. Only the latest activity is sent.
//
// All state lives on one goroutine (loop); every event (a call, a frame from
// Discord, a timer) is queued to it as a function, like an event loop.
type Client struct {
	opts Options

	mu    sync.Mutex
	queue []func()
	wake  chan struct{}

	// Owned by the loop goroutine.
	conn        *link
	ready       bool
	connecting  bool
	stopped     bool
	closeReason string
	lastStatus  string
	retryTimer  *time.Timer
	flushTimer  *time.Timer
	desired     *activity.Activity
	desiredKey  string
	appliedKey  string // what Discord shows from this connection
	sendTimes   []time.Time
	stopNonce   string
	stopDone    func()
}

// link is one connection; closed tells our own Close apart from a failure.
type link struct {
	net.Conn
	closed atomic.Bool
}

func (l *link) close() {
	l.closed.Store(true)
	_ = l.Conn.Close()
}

// New creates a client. Call Start to connect.
func New(opts Options) *Client {
	if opts.RetryInterval == 0 {
		opts.RetryInterval = 15 * time.Second
	}
	if opts.RateWindow == 0 {
		opts.RateWindow = 20 * time.Second
	}
	if opts.RateMax == 0 {
		opts.RateMax = 5
	}
	if opts.PathFor == nil {
		opts.PathFor = func(i int) string { return Path(i, runtime.GOOS, os.Getenv) }
	}
	if opts.Dial == nil {
		opts.Dial = Dial
	}
	c := &Client{opts: opts, wake: make(chan struct{}, 1), desiredKey: "null", appliedKey: "null"}
	go c.loop()
	return c
}

func (c *Client) loop() {
	for range c.wake {
		for {
			c.mu.Lock()
			queue := c.queue
			c.queue = nil
			c.mu.Unlock()
			if len(queue) == 0 {
				break
			}
			for _, f := range queue {
				f()
			}
		}
	}
}

// post queues f to run on the loop goroutine. Never blocks, so it is safe
// from any goroutine, including the loop itself.
func (c *Client) post(f func()) {
	c.mu.Lock()
	c.queue = append(c.queue, f)
	c.mu.Unlock()
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// Start connects in the background.
func (c *Client) Start() { c.post(c.connect) }

// SetActivity shows this activity (nil clears it). Only the latest value is sent.
func (c *Client) SetActivity(a *activity.Activity) {
	key, err := json.Marshal(a)
	if err != nil {
		return
	}
	c.post(func() {
		c.desired = a
		c.desiredKey = string(key)
		c.scheduleFlush()
	})
}

// Stop clears the activity and disconnects, waiting at most about a second.
func (c *Client) Stop() {
	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }
	c.post(func() { c.stop(finish) })
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func (c *Client) stop(finish func()) {
	c.stopped = true
	stopTimer(&c.retryTimer)
	stopTimer(&c.flushTimer)
	l := c.conn
	if l == nil {
		finish()
		return
	}
	if !c.ready || c.appliedKey == "null" {
		l.close()
		finish()
		return
	}
	// Clear the status, then wait for Discord's answer (or a second) so the
	// clear arrives before the connection goes away.
	c.stopNonce = newNonce()
	c.stopDone = func() {
		l.close()
		finish()
	}
	c.write(l, EncodeFrame(OpFrame, command{Cmd: "SET_ACTIVITY", Args: activityArgs{Pid: os.Getpid()}, Nonce: c.stopNonce}))
	time.AfterFunc(time.Second, func() { c.post(c.finishStop) })
}

func (c *Client) finishStop() {
	if done := c.stopDone; done != nil {
		c.stopDone = nil
		done()
	}
}

func stopTimer(t **time.Timer) {
	if *t != nil {
		(*t).Stop()
		*t = nil
	}
}

func (c *Client) status(s Status) {
	key := s.State + "\x00" + s.Detail
	if key == c.lastStatus {
		return
	}
	c.lastStatus = key
	line := "discord: " + s.State
	if s.Detail != "" {
		line += " " + s.Detail
	}
	c.opts.Log(line)
	c.opts.OnStatus(s)
}

func (c *Client) connect() {
	if c.stopped || c.conn != nil || c.connecting {
		return
	}
	c.connecting = true
	if c.lastStatus == "" {
		c.status(Status{State: "connecting"})
	}
	go func() {
		var conn net.Conn
		for i := 0; i < PipeCount && conn == nil; i++ {
			if cn, err := c.opts.Dial(c.opts.PathFor(i)); err == nil {
				conn = cn
			}
		}
		c.post(func() {
			c.connecting = false
			if c.stopped {
				if conn != nil {
					_ = conn.Close()
				}
				return
			}
			if conn == nil {
				c.status(Status{State: "unavailable", Detail: "Discord app is not running"})
				c.retry(c.opts.RetryInterval)
				return
			}
			c.attach(conn)
		})
	}()
}

func (c *Client) attach(conn net.Conn) {
	l := &link{Conn: conn}
	c.conn = l
	c.ready = false
	c.appliedKey = "null"
	c.closeReason = ""
	go c.read(l)
	c.write(l, EncodeFrame(OpHandshake, handshake{V: 1, ClientID: c.opts.ClientID}))
}

func (c *Client) read(l *link) {
	var dec Decoder
	buf := make([]byte, 64*1024)
	for {
		n, err := l.Read(buf)
		for _, f := range dec.Push(buf[:n]) {
			c.post(func() { c.handle(l, f) })
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !l.closed.Load() {
				msg := err.Error()
				c.post(func() { c.opts.Log("discord socket error: " + msg) })
			}
			c.post(func() { c.onClose(l) })
			return
		}
	}
}

func (c *Client) write(l *link, b []byte) {
	_ = l.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := l.Write(b); err != nil {
		if !l.closed.Load() {
			c.opts.Log("discord socket error: " + err.Error())
		}
		l.close() // the reader sees it and reports the close
	}
}

func (c *Client) handle(l *link, f Frame) {
	if c.conn != l {
		return
	}
	data, _ := f.Data.(map[string]any)
	if c.stopped {
		if c.stopNonce != "" && data["nonce"] == c.stopNonce {
			c.finishStop()
		}
		return
	}
	switch f.Op {
	case OpPing:
		c.write(l, EncodeFrame(OpPong, f.Data))
	case OpClose:
		c.closeReason = "Discord ไม่รับการเชื่อมต่อ: " + jsString(data["message"], "unknown") +
			" — ตรวจ Application ID (README หัวข้อเปลี่ยน Application ID)"
		l.close()
	case OpFrame:
		cmd, evt := data["cmd"], data["evt"]
		if cmd == "DISPATCH" && evt == "READY" {
			c.ready = true
			// Deliberately not logging the Discord user name: people paste host.log into issues.
			c.status(Status{State: "connected"})
			c.scheduleFlush()
		} else if cmd == "SET_ACTIVITY" && evt == "ERROR" {
			inner, _ := data["data"].(map[string]any)
			c.opts.OnActivityError(jsString(inner["message"], "unknown error"))
		}
	}
}

func (c *Client) onClose(l *link) {
	if c.conn != l {
		return
	}
	c.conn = nil
	c.ready = false
	c.appliedKey = "null"
	stopTimer(&c.flushTimer)
	if c.stopped {
		c.finishStop()
		return
	}
	if c.closeReason != "" {
		c.status(Status{State: "error", Detail: c.closeReason})
		c.retry(c.opts.RetryInterval * 4)
	} else {
		c.status(Status{State: "unavailable", Detail: "Discord app closed the connection"})
		c.retry(c.opts.RetryInterval)
	}
}

func (c *Client) retry(d time.Duration) {
	stopTimer(&c.retryTimer)
	var t *time.Timer
	t = time.AfterFunc(d, func() {
		c.post(func() {
			if c.retryTimer == t {
				c.retryTimer = nil
				c.connect()
			}
		})
	})
	c.retryTimer = t
}

func (c *Client) scheduleFlush() {
	if c.stopped || !c.ready || c.flushTimer != nil || c.desiredKey == c.appliedKey {
		return
	}
	now := time.Now()
	recent := c.sendTimes[:0]
	for _, at := range c.sendTimes {
		if now.Sub(at) < c.opts.RateWindow {
			recent = append(recent, at)
		}
	}
	c.sendTimes = recent
	var wait time.Duration
	if len(recent) >= c.opts.RateMax {
		wait = c.opts.RateWindow - now.Sub(recent[0]) + 50*time.Millisecond
	}
	var t *time.Timer
	t = time.AfterFunc(wait, func() {
		c.post(func() {
			if c.flushTimer == t {
				c.flushTimer = nil
				c.flush()
			}
		})
	})
	c.flushTimer = t
}

func (c *Client) flush() {
	l := c.conn
	if c.stopped || !c.ready || l == nil || c.desiredKey == c.appliedKey {
		return
	}
	c.sendTimes = append(c.sendTimes, time.Now())
	c.appliedKey = c.desiredKey
	args := activityArgs{Pid: os.Getpid(), Activity: c.desired}
	c.write(l, EncodeFrame(OpFrame, command{Cmd: "SET_ACTIVITY", Args: args, Nonce: newNonce()}))
}

type handshake struct {
	V        int    `json:"v"`
	ClientID string `json:"client_id"`
}

type command struct {
	Cmd   string       `json:"cmd"`
	Args  activityArgs `json:"args"`
	Nonce string       `json:"nonce"`
}

type activityArgs struct {
	Pid      int                `json:"pid"`
	Activity *activity.Activity `json:"activity,omitempty"`
}

// newNonce is a random UUID v4.
func newNonce() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// jsString turns a JSON value into text the way a JavaScript template string
// would, with def for null or missing.
func jsString(v any, def string) string {
	switch x := v.(type) {
	case nil:
		return def
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "true"
		}
		return "false"
	case map[string]any:
		return "[object Object]"
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}
