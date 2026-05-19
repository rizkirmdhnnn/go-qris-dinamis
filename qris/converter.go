package qris

import (
	"fmt"
	"strconv"
)

// Options configures Convert.
type Options struct {
	Amount int // required, must be > 0 and at most 13 digits
	Fee    int // optional fixed fee; 0 = none; must be >= 0 and at most 13 digits
}

// Convert turns a static QRIS into a dynamic one by switching tag 01 from
// "11" to "12", injecting tag 54 (amount), optionally tags 55/56 (fixed fee),
// and recomputing the CRC16 at tag 63.
func Convert(s string, opts Options) (string, error) {
	if opts.Amount <= 0 || len(strconv.Itoa(opts.Amount)) > 13 {
		return "", ErrInvalidAmount
	}
	if opts.Fee < 0 || len(strconv.Itoa(opts.Fee)) > 13 {
		return "", ErrInvalidFee
	}
	if err := Validate(s); err != nil {
		return "", err
	}

	tlvs, err := parseTLV(s[:len(s)-8])
	if err != nil {
		return "", err
	}

	methodIdx := -1
	for i, t := range tlvs {
		if t.Tag == "01" {
			methodIdx = i
			break
		}
	}
	if methodIdx == -1 {
		return "", fmt.Errorf("%w: missing tag 01", ErrInvalidFormat)
	}
	if tlvs[methodIdx].Value == "12" {
		return "", ErrAlreadyDynamic
	}
	tlvs[methodIdx].Value = "12"

	newTLVs := []TLV{{Tag: "54", Value: strconv.Itoa(opts.Amount)}}
	if opts.Fee > 0 {
		newTLVs = append(newTLVs,
			TLV{Tag: "55", Value: "02"},
			TLV{Tag: "56", Value: strconv.Itoa(opts.Fee)},
		)
	}

	insertAt := len(tlvs)
	for i, t := range tlvs {
		if t.Tag >= "58" {
			insertAt = i
			break
		}
	}
	merged := make([]TLV, 0, len(tlvs)+len(newTLVs))
	merged = append(merged, tlvs[:insertAt]...)
	merged = append(merged, newTLVs...)
	merged = append(merged, tlvs[insertAt:]...)

	payload := serializeTLV(merged) + "6304"
	return payload + CRC16(payload), nil
}
