package qris

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseTLV(t *testing.T) {
	in := "000201" + "010211" + "5905STORE"
	got, err := parseTLV(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TLV{
		{Tag: "00", Length: 2, Value: "01"},
		{Tag: "01", Length: 2, Value: "11"},
		{Tag: "59", Length: 5, Value: "STORE"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTLV() = %#v, want %#v", got, want)
	}
}

func TestParseTLV_Errors(t *testing.T) {
	cases := []struct {
		name, in string
	}{
		{"truncated header", "0002"},        // only 4 chars, no value bytes after length=02
		{"length exceeds remainder", "00050"}, // tag 00 length 05 but value bytes absent
		{"non-digit length", "00XX01"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseTLV(tc.in)
			if !errors.Is(err, ErrInvalidFormat) {
				t.Fatalf("got err=%v, want ErrInvalidFormat", err)
			}
		})
	}
}

func TestSerializeTLV(t *testing.T) {
	in := []TLV{
		{Tag: "00", Length: 2, Value: "01"},
		{Tag: "54", Length: 5, Value: "50000"},
	}
	got := serializeTLV(in)
	want := "000201" + "540550000"
	if got != want {
		t.Fatalf("serializeTLV() = %q, want %q", got, want)
	}
}

func TestSerializeTLV_RecomputesLength(t *testing.T) {
	// If caller passes wrong Length, serialize must still use len(Value).
	in := []TLV{{Tag: "00", Length: 99, Value: "01"}}
	got := serializeTLV(in)
	want := "000201"
	if got != want {
		t.Fatalf("serializeTLV() = %q, want %q (must recompute length)", got, want)
	}
}
