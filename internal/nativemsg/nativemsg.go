// Package nativemsg implements Chrome's native messaging framing: each message
// is a 4-byte little-endian length followed by that many bytes of UTF-8 JSON,
// on stdin/stdout.
package nativemsg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MaxIncoming caps messages from Chrome; the extension only sends small song updates.
const MaxIncoming = 1024 * 1024

// Encode frames one message.
func Encode(message any) ([]byte, error) {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(message); err != nil {
		return nil, err
	}
	data := bytes.TrimSuffix(body.Bytes(), []byte("\n"))
	out := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(out, uint32(len(data)))
	copy(out[4:], data)
	return out, nil
}

// Read calls onMessage for each message until r ends (returns nil) or fails.
// A message that isn't valid JSON, or is too large, goes to onError and the
// stream continues with the next frame.
func Read(r io.Reader, onMessage func(any), onError func(error)) error {
	var header [4]byte
	for {
		if _, err := io.ReadFull(r, header[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}
		length := binary.LittleEndian.Uint32(header[:])
		if length > MaxIncoming {
			onError(fmt.Errorf("message too large: %d bytes", length))
			// Skip the body so the next frame starts in the right place.
			if _, err := io.CopyN(io.Discard, r, int64(length)); err != nil {
				return nil
			}
			continue
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil
		}
		d := json.NewDecoder(bytes.NewReader(body))
		d.UseNumber()
		var message any
		if err := d.Decode(&message); err != nil {
			onError(err)
			continue
		}
		if d.More() {
			onError(errors.New("unexpected data after the JSON value"))
			continue
		}
		onMessage(message)
	}
}
