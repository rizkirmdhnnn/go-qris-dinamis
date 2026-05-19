package qris

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

// RenderPNG renders an arbitrary QRIS string as a PNG QR code of the given
// pixel size (width = height = size). It does not parse or validate the input;
// callers pass in whatever string they want encoded (typically the output of
// Convert). Error-correction level is fixed at Medium.
//
// Returns the PNG-encoded bytes, suitable for writing to a file or io.Writer.
func RenderPNG(qrisString string, size int) ([]byte, error) {
	png, err := qrcode.Encode(qrisString, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("qris: render PNG: %w", err)
	}
	return png, nil
}
