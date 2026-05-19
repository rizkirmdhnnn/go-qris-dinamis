package qris

import (
	"fmt"
	"strconv"
)

// TLV represents a single Tag-Length-Value element from a QRIS payload.
type TLV struct {
	Tag    string // 2-char ASCII
	Length int    // value length in bytes
	Value  string // raw value
}

// parseTLV decodes a flat (non-nested) TLV string into a slice of TLV.
// Returns ErrInvalidFormat on any structural error.
func parseTLV(s string) ([]TLV, error) {
	var out []TLV
	for i := 0; i < len(s); {
		if i+4 > len(s) {
			return nil, fmt.Errorf("%w: truncated header at offset %d", ErrInvalidFormat, i)
		}
		tag := s[i : i+2]
		length, err := strconv.Atoi(s[i+2 : i+4])
		if err != nil || length < 0 {
			return nil, fmt.Errorf("%w: bad length at offset %d", ErrInvalidFormat, i)
		}
		if i+4+length > len(s) {
			return nil, fmt.Errorf("%w: value exceeds payload at offset %d", ErrInvalidFormat, i)
		}
		out = append(out, TLV{Tag: tag, Length: length, Value: s[i+4 : i+4+length]})
		i += 4 + length
	}
	return out, nil
}

// serializeTLV encodes a slice of TLV back to a flat QRIS string.
// Length is recomputed from len(Value); the TLV.Length field is ignored.
func serializeTLV(tlvs []TLV) string {
	var n int
	for _, t := range tlvs {
		n += 4 + len(t.Value)
	}
	b := make([]byte, 0, n)
	for _, t := range tlvs {
		b = append(b, t.Tag...)
		b = append(b, fmt.Sprintf("%02d", len(t.Value))...)
		b = append(b, t.Value...)
	}
	return string(b)
}
