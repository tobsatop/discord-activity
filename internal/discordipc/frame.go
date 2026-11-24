// Package discordipc is a minimal client for the Discord desktop app's local
// IPC, the channel that Rich Presence libraries use. Each frame is an int32 LE
// opcode, an int32 LE byte length, then UTF-8 JSON.
package discordipc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
)

// Opcodes.
const (
	OpHandshake = 0
	OpFrame     = 1
	OpClose     = 2
	OpPing      = 3
	OpPong      = 4
)

const maxFrame = 16 * 1024 * 1024

// Frame is one decoded message. Data is generic JSON (maps, strings, json.Number).
type Frame struct {
	Op   int32
	Data any
}

// EncodeFrame builds one frame.
func EncodeFrame(op int32, payload any) []byte {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		// Only our own payload types reach here; they always encode.
		panic(err)
	}
	data := bytes.TrimSuffix(body.Bytes(), []byte("\n"))
	out := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint32(out[0:], uint32(op))
	binary.LittleEndian.PutUint32(out[4:], uint32(len(data)))
	copy(out[8:], data)
	return out
}

// Decoder collects bytes and returns whole frames.
type Decoder struct {
	buf []byte
}

// Push adds a chunk and returns every frame completed by it.
func (d *Decoder) Push(chunk []byte) []Frame {
	d.buf = append(d.buf, chunk...)
	var frames []Frame
	for len(d.buf) >= 8 {
		op := int32(binary.LittleEndian.Uint32(d.buf[0:]))
		length := int32(binary.LittleEndian.Uint32(d.buf[4:]))
		if length < 0 || length > maxFrame {
			d.buf = nil // corrupt stream; drop it
			break
		}
		if len(d.buf) < 8+int(length) {
			break
		}
		body := d.buf[8 : 8+int(length)]
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.UseNumber()
		var data any
		err := dec.Decode(&data)
		d.buf = d.buf[8+int(length):]
		if err != nil {
			continue // skip a malformed frame
		}
		frames = append(frames, Frame{Op: op, Data: data})
	}
	if len(d.buf) == 0 {
		d.buf = nil // let the backing array go
	}
	return frames
}
