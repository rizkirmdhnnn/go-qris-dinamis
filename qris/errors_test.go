package qris

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	for _, err := range []error{
		ErrInvalidFormat,
		ErrCRCMismatch,
		ErrAlreadyDynamic,
		ErrInvalidAmount,
		ErrInvalidFee,
	} {
		if err == nil {
			t.Fatalf("sentinel error is nil")
		}
		if err.Error() == "" {
			t.Fatalf("sentinel error has empty message")
		}
	}

	wrapped := errors.New("wrap: " + ErrCRCMismatch.Error())
	if errors.Is(wrapped, ErrCRCMismatch) {
		t.Fatalf("wrapped-by-string should NOT match via errors.Is (sanity check)")
	}
}
