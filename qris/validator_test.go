package qris

import (
	"errors"
	"testing"
)

func TestValidate_OK(t *testing.T) {
	if err := Validate(validStaticQRIS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(validDynamicQRIS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_CRCMismatch(t *testing.T) {
	q := validStaticQRIS()
	// Flip the last hex char of the CRC.
	bad := q[:len(q)-1] + flipHex(q[len(q)-1])
	err := Validate(bad)
	if !errors.Is(err, ErrCRCMismatch) {
		t.Fatalf("got %v, want ErrCRCMismatch", err)
	}
}

func TestValidate_TooShort(t *testing.T) {
	err := Validate("1234")
	if !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("got %v, want ErrInvalidFormat", err)
	}
}

func TestValidate_MissingCRCTag(t *testing.T) {
	// 12 chars, last 4 are CRC value, preceding 4 should be "6304" but won't be.
	err := Validate("00020101021100AB")
	if !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("got %v, want ErrInvalidFormat", err)
	}
}

func TestValidate_LowercaseCRC(t *testing.T) {
	q := validStaticQRIS()
	lower := q[:len(q)-4] + toLower(q[len(q)-4:])
	if err := Validate(lower); err != nil {
		t.Fatalf("lowercase CRC should be accepted: %v", err)
	}
}

func flipHex(c byte) string {
	if c == '0' {
		return "1"
	}
	return "0"
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i, c := range []byte(s) {
		if c >= 'A' && c <= 'F' {
			c = c + 32
		}
		b[i] = c
	}
	return string(b)
}
