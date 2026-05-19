package qris

import (
	"errors"
	"strings"
	"testing"
)

func tagValue(t *testing.T, qrisStr, tag string) (string, bool) {
	t.Helper()
	if err := Validate(qrisStr); err != nil {
		t.Fatalf("output not valid: %v", err)
	}
	tlvs, err := parseTLV(qrisStr[:len(qrisStr)-8])
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	for _, x := range tlvs {
		if x.Tag == tag {
			return x.Value, true
		}
	}
	return "", false
}

func TestConvert_AmountOnly(t *testing.T) {
	out, err := Convert(validStaticQRIS(), Options{Amount: 50000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(out); err != nil {
		t.Fatalf("output invalid: %v", err)
	}

	if v, ok := tagValue(t, out, "01"); !ok || v != "12" {
		t.Errorf("tag 01 = %q present=%v, want %q", v, ok, "12")
	}
	if v, ok := tagValue(t, out, "54"); !ok || v != "50000" {
		t.Errorf("tag 54 = %q present=%v, want %q", v, ok, "50000")
	}
	if _, ok := tagValue(t, out, "55"); ok {
		t.Errorf("tag 55 should not be present when no fee")
	}

	d, err := Parse(out)
	if err != nil || d.Method != MethodDynamic {
		t.Errorf("Parse(out).Method = %q (err=%v), want %q", d.Method, err, MethodDynamic)
	}
}

func TestConvert_AmountAndFee(t *testing.T) {
	out, err := Convert(validStaticQRIS(), Options{Amount: 50000, Fee: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Validate(out); err != nil {
		t.Fatalf("output invalid: %v", err)
	}
	if v, _ := tagValue(t, out, "54"); v != "50000" {
		t.Errorf("tag 54 = %q, want %q", v, "50000")
	}
	if v, _ := tagValue(t, out, "55"); v != "02" {
		t.Errorf("tag 55 = %q, want %q", v, "02")
	}
	if v, _ := tagValue(t, out, "56"); v != "1000" {
		t.Errorf("tag 56 = %q, want %q", v, "1000")
	}

	// 54, 55, 56 must appear in order and BEFORE 58.
	payload := out[:len(out)-8]
	i54 := strings.Index(payload, "5405")
	i55 := strings.Index(payload, "5502")
	i56 := strings.Index(payload, "5604")
	i58 := strings.Index(payload, "5802ID")
	if !(i54 >= 0 && i55 > i54 && i56 > i55 && i58 > i56) {
		t.Errorf("tag order wrong: i54=%d i55=%d i56=%d i58=%d", i54, i55, i56, i58)
	}
}

func TestConvert_AlreadyDynamic(t *testing.T) {
	_, err := Convert(validDynamicQRIS(), Options{Amount: 50000})
	if !errors.Is(err, ErrAlreadyDynamic) {
		t.Fatalf("got %v, want ErrAlreadyDynamic", err)
	}
}

func TestConvert_BadAmount(t *testing.T) {
	cases := []int{0, -1, 99999999999999} // last has 14 digits
	for _, a := range cases {
		_, err := Convert(validStaticQRIS(), Options{Amount: a})
		if !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("amount=%d: got %v, want ErrInvalidAmount", a, err)
		}
	}
}

func TestConvert_BadFee(t *testing.T) {
	cases := []int{-1, 99999999999999}
	for _, f := range cases {
		_, err := Convert(validStaticQRIS(), Options{Amount: 1, Fee: f})
		if !errors.Is(err, ErrInvalidFee) {
			t.Errorf("fee=%d: got %v, want ErrInvalidFee", f, err)
		}
	}
}

func TestConvert_PropagatesValidate(t *testing.T) {
	q := validStaticQRIS()
	bad := q[:len(q)-1] + "0" // break CRC
	_, err := Convert(bad, Options{Amount: 50000})
	if !errors.Is(err, ErrCRCMismatch) {
		t.Fatalf("got %v, want ErrCRCMismatch", err)
	}
}
