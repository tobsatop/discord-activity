package appicon

import (
	"bytes"
	"image/png"
	"os"
	"testing"
)

func TestRender(t *testing.T) {
	if _, err := png.Decode(bytes.NewReader(PNG(Render(64, App(0))))); err != nil {
		t.Fatal(err)
	}
}

// TestPreview writes a preview when APPICON_PREVIEW names a file.
func TestPreview(t *testing.T) {
	out := os.Getenv("APPICON_PREVIEW")
	if out == "" {
		t.Skip("set APPICON_PREVIEW=file.png")
	}
	if err := os.WriteFile(out, PNG(Render(512, App(0.06))), 0o644); err != nil {
		t.Fatal(err)
	}
}
