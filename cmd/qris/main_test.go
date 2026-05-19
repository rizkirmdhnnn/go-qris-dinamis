package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rizkirmdhn/go-qris-dinamis/qris"
)

// validStatic mirrors qris/helpers_test.go but lives here because that helper
// is package-private to qris and unavailable from main_test.
func validStatic() string {
	parts := []string{
		"000201", "010211", "52045812", "5303360",
		"5802ID", "5905STORE", "6007JAKARTA",
	}
	payload := strings.Join(parts, "") + "6304"
	return payload + qris.CRC16(payload)
}

func TestRun_HelpExitsZero(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-h"}, strings.NewReader(""), &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit=%d, want 0; stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("help output missing 'Usage:': %s", out.String())
	}
}

func TestRun_NoInputExitsTwo(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-a", "50000"}, strings.NewReader(""), &out, &errBuf)
	if code != 2 {
		t.Fatalf("exit=%d, want 2; stderr=%s", code, errBuf.String())
	}
	msg := errBuf.String()
	if !strings.Contains(msg, "no input") && !strings.Contains(msg, "empty input") {
		t.Errorf("stderr missing 'no input' or 'empty input': %s", msg)
	}
}

func TestRun_StdinHappyPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	in := strings.NewReader(validStatic())
	code := run([]string{"-a", "50000"}, in, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit=%d, want 0; stderr=%s", code, errBuf.String())
	}
	result := strings.TrimSpace(out.String())
	if err := qris.Validate(result); err != nil {
		t.Fatalf("output invalid: %v", err)
	}
	d, err := qris.Parse(result)
	if err != nil || d.Method != qris.MethodDynamic {
		t.Errorf("Parse(out).Method=%q err=%v, want dynamic", d.Method, err)
	}
}

func TestRun_FlagInputHappyPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-i", validStatic(), "-a", "50000", "--fee", "1000"}, strings.NewReader(""), &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit=%d, want 0; stderr=%s", code, errBuf.String())
	}
	if err := qris.Validate(strings.TrimSpace(out.String())); err != nil {
		t.Fatalf("output invalid: %v", err)
	}
}

func TestRun_MultipleSourcesExitsTwo(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-i", "x", "-f", "y", "-a", "1"}, strings.NewReader(""), &out, &errBuf)
	if code != 2 {
		t.Fatalf("exit=%d, want 2; stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "multiple input sources") {
		t.Errorf("stderr missing 'multiple input sources': %s", errBuf.String())
	}
}

func TestRun_BadAmountExitsTwo(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-i", "x", "-a", "0"}, strings.NewReader(""), &out, &errBuf)
	if code != 2 {
		t.Fatalf("exit=%d, want 2", code)
	}
}

func TestRun_CRCMismatchExitsOne(t *testing.T) {
	q := validStatic()
	bad := q[:len(q)-1] + "0"
	var out, errBuf bytes.Buffer
	code := run([]string{"-i", bad, "-a", "50000"}, strings.NewReader(""), &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit=%d, want 1; stderr=%s", code, errBuf.String())
	}
}
