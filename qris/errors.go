package qris

import "errors"

var (
	ErrInvalidFormat  = errors.New("invalid QRIS format")
	ErrCRCMismatch    = errors.New("CRC mismatch")
	ErrAlreadyDynamic = errors.New("QRIS is already dynamic")
	ErrInvalidAmount  = errors.New("amount must be > 0 and have at most 13 digits")
	ErrInvalidFee     = errors.New("fee must be >= 0 and have at most 13 digits")
)
