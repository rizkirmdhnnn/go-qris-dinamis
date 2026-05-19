package qris

import (
	"bytes"
	"testing"
)

// pngSignature is the 8-byte magic header every PNG file starts with.
var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

func TestRenderPNG_ValidQRIS(t *testing.T) {
	in := validStaticQRIS()

	got, err := RenderPNG(in, 512)
	if err != nil {
		t.Fatalf("RenderPNG returned error: %v", err)
	}
	if !bytes.HasPrefix(got, pngSignature) {
		t.Errorf("output does not start with PNG signature; first 8 bytes = % x", got[:min(8, len(got))])
	}
	if len(got) < 100 {
		t.Errorf("PNG suspiciously small: %d bytes", len(got))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestRenderPNG_EmptyInput(t *testing.T) {
	_, err := RenderPNG("", 256)
	if err == nil {
		t.Fatal("RenderPNG(\"\", 256) returned nil error, want error")
	}
}
