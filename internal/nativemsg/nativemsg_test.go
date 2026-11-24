package nativemsg

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"reflect"
	"testing"
)

func frame(t *testing.T, v any) []byte {
	t.Helper()
	b, err := Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestEncode(t *testing.T) {
	got := frame(t, map[string]any{"type": "status", "detail": "ไทย <&>"})
	body := `{"detail":"ไทย <&>","type":"status"}`
	if n := binary.LittleEndian.Uint32(got); int(n) != len(body) {
		t.Fatalf("length %d, want %d", n, len(body))
	}
	if string(got[4:]) != body {
		t.Fatalf("body %s", got[4:])
	}
}

// oneByte delivers a byte at a time, like a slow pipe.
type oneByte struct{ r io.Reader }

func (o oneByte) Read(p []byte) (int, error) { return o.r.Read(p[:1]) }

func TestReadSplitAndJunk(t *testing.T) {
	var stream bytes.Buffer
	stream.Write(frame(t, map[string]any{"type": "update", "n": 1}))
	bad := []byte("{nope")
	var h [4]byte
	binary.LittleEndian.PutUint32(h[:], uint32(len(bad)))
	stream.Write(h[:])
	stream.Write(bad)
	binary.LittleEndian.PutUint32(h[:], MaxIncoming+1)
	stream.Write(h[:])
	stream.Write(make([]byte, MaxIncoming+1))
	stream.Write(frame(t, "last"))

	var got []any
	var errs []error
	err := Read(oneByte{&stream}, func(m any) { got = append(got, m) }, func(e error) { errs = append(errs, e) })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != "last" {
		t.Fatalf("messages %v", got)
	}
	if n := got[0].(map[string]any)["n"]; !reflect.DeepEqual(n, any(json.Number("1"))) {
		t.Fatalf("number decoded as %#v", n)
	}
	if len(errs) != 2 || errs[1].Error() != "message too large: 1048577 bytes" {
		t.Fatalf("errors %v", errs)
	}
}
