package qris

import (
	"fmt"
	"strings"
)

// Validate checks structural correctness and verifies the CRC16-CCITT
// checksum stored at tag 63 (which must be the last TLV).
func Validate(s string) error {
	if len(s) < 8 {
		return fmt.Errorf("%w: too short", ErrInvalidFormat)
	}
	if s[len(s)-8:len(s)-4] != "6304" {
		return fmt.Errorf("%w: missing or misplaced CRC tag", ErrInvalidFormat)
	}
	want := strings.ToUpper(s[len(s)-4:])
	got := CRC16(s[:len(s)-4])
	if got != want {
		return fmt.Errorf("%w: have %s, calculated %s", ErrCRCMismatch, want, got)
	}
	tlvs, err := parseTLV(s[:len(s)-8])
	if err != nil {
		return err
	}
	hasMethod := false
	for _, t := range tlvs {
		if t.Tag == "01" {
			hasMethod = true
			break
		}
	}
	if !hasMethod {
		return fmt.Errorf("%w: missing tag 01", ErrInvalidFormat)
	}
	return nil
}
