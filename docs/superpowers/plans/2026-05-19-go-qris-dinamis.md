# go-qris-dinamis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a zero-dep Go CLI `qris` that converts a static QRIS string into a dynamic QRIS with injected amount and optional fixed fee, plus a public library package `qris` for programmatic use.

**Architecture:** Two-layer split. `cmd/qris/main.go` is a thin CLI adapter (flag parsing + I/O + exit codes). `qris/` package holds pure logic (TLV parse/serialize, CRC16-CCITT, Parse, Validate, Convert). Logic functions take/return strings — no I/O inside the package.

**Tech Stack:** Go 1.22+ stdlib only (`flag`, `os`, `io`, `errors`, `strings`, `strconv`, `fmt`, `testing`). No third-party deps.

**Working directory:** `/Users/rizkirmdhn/Documents/Code/05-Lain-lain/qris/` (currently empty, no git repo yet).

**Module path:** `github.com/rizkirmdhn/go-qris-dinamis`

**Spec:** `docs/superpowers/specs/2026-05-19-go-qris-dinamis-cli-design.md`

**Spec deviation:** spec section 6.7 lists `qris/testdata/static_valid.txt` and `dynamic_valid.txt` as test fixtures. This plan uses programmatic fixtures (helper functions in `qris/helpers_test.go`) instead — same coverage with no separate files to maintain or to recompute CRC for. Re-add testdata files later if a use case appears (e.g. fuzzing seeds).

---

### Task 1: Project bootstrap

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `LICENSE`
- Create: `README.md` (skeleton; finalized in Task 10)

- [ ] **Step 1: Initialize git repository**

Run:
```bash
git init
```
Expected: `Initialized empty Git repository in .../qris/.git/`

- [ ] **Step 2: Initialize Go module**

Run:
```bash
go mod init github.com/rizkirmdhn/go-qris-dinamis
```
Expected: creates `go.mod` with `module github.com/rizkirmdhn/go-qris-dinamis` and `go 1.22` (or current). If Go reports a version < 1.22, edit `go.mod` to bump `go 1.22`.

- [ ] **Step 3: Create `.gitignore`**

Write `.gitignore`:
```
# Binaries
/qris
/cmd/qris/qris

# Test artifacts
*.test
*.out

# OS
.DS_Store

# Editors
.idea/
.vscode/
```

- [ ] **Step 4: Create `LICENSE` (MIT)**

Write `LICENSE`:
```
MIT License

Copyright (c) 2026 Achmad Rizki Ramadhan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 5: Create skeleton `README.md`**

Write `README.md`:
```markdown
# go-qris-dinamis

CLI Go untuk mengubah QRIS statis menjadi dinamis (inject nominal + opsional fee tetap, recompute CRC16).

Status: under development. See `docs/superpowers/specs/` for the design spec.
```

- [ ] **Step 6: Verify build + initial commit**

Run:
```bash
go build ./...
```
Expected: no output (nothing to build yet, just module init). No error.

Run:
```bash
git add go.mod .gitignore LICENSE README.md docs/
git commit -m "chore: bootstrap go-qris-dinamis module + spec"
```
Expected: one commit created. `git status` shows clean tree.

---

### Task 2: CRC16-CCITT (TDD)

**Files:**
- Create: `qris/crc16.go`
- Create: `qris/crc16_test.go`

- [ ] **Step 1: Write the failing test**

Write `qris/crc16_test.go`:
```go
package qris

import "testing"

func TestCRC16(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"standard vector", "123456789", "29B1"},
		{"empty string", "", "FFFF"},
		{"single byte A", "A", "B915"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CRC16(tc.in)
			if got != tc.want {
				t.Fatalf("CRC16(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./qris/...
```
Expected: FAIL with `undefined: CRC16`.

- [ ] **Step 3: Implement `CRC16`**

Write `qris/crc16.go`:
```go
// Package qris implements parsing and conversion of QRIS (Quick Response Code
// Indonesian Standard) data per EMVCo TLV format.
package qris

import "fmt"

// CRC16 computes CRC-16/CCITT-FALSE (poly 0x1021, init 0xFFFF, no XOR-out,
// no reflection) and returns a 4-character uppercase hexadecimal string.
func CRC16(s string) string {
	crc := uint16(0xFFFF)
	for _, b := range []byte(s) {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test ./qris/...
```
Expected: PASS — three subtests pass.

- [ ] **Step 5: Commit**

```bash
git add qris/crc16.go qris/crc16_test.go
git commit -m "feat(qris): add CRC16-CCITT implementation"
```

---

### Task 3: Sentinel errors

**Files:**
- Create: `qris/errors.go`
- Create: `qris/errors_test.go`

- [ ] **Step 1: Write the failing test**

Write `qris/errors_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./qris/...
```
Expected: FAIL — `undefined: ErrInvalidFormat` etc.

- [ ] **Step 3: Implement sentinel errors**

Write `qris/errors.go`:
```go
package qris

import "errors"

var (
	ErrInvalidFormat  = errors.New("invalid QRIS format")
	ErrCRCMismatch    = errors.New("CRC mismatch")
	ErrAlreadyDynamic = errors.New("QRIS is already dynamic")
	ErrInvalidAmount  = errors.New("amount must be > 0 and have at most 13 digits")
	ErrInvalidFee     = errors.New("fee must be >= 0 and have at most 13 digits")
)
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test ./qris/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add qris/errors.go qris/errors_test.go
git commit -m "feat(qris): add sentinel errors"
```

---

### Task 4: TLV parse + serialize (TDD)

**Files:**
- Create: `qris/tlv.go`
- Create: `qris/tlv_test.go`

- [ ] **Step 1: Write the failing test**

Write `qris/tlv_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./qris/...
```
Expected: FAIL — `undefined: TLV`, `undefined: parseTLV`, `undefined: serializeTLV`.

- [ ] **Step 3: Implement TLV type, parser, and serializer**

Write `qris/tlv.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test ./qris/...
```
Expected: PASS — all parseTLV and serializeTLV subtests pass.

- [ ] **Step 5: Commit**

```bash
git add qris/tlv.go qris/tlv_test.go
git commit -m "feat(qris): add TLV parse + serialize"
```

---

### Task 5: Test helper + `Parse` (TDD)

**Files:**
- Create: `qris/helpers_test.go`
- Create: `qris/parser.go`
- Create: `qris/parser_test.go`

- [ ] **Step 1: Write shared test helper**

Write `qris/helpers_test.go`:
```go
package qris

import "strings"

// validStaticQRIS returns a freshly constructed valid static QRIS string
// with a real CRC16. Used across tests; depends only on CRC16 (Task 2).
func validStaticQRIS() string {
	parts := []string{
		"000201",      // 00 02 "01"     PayloadFormatIndicator
		"010211",      // 01 02 "11"     PointOfInitiationMethod (static)
		"52045812",    // 52 04 "5812"   MerchantCategoryCode
		"5303360",     // 53 03 "360"    Currency (IDR)
		"5802ID",      // 58 02 "ID"     CountryCode
		"5905STORE",   // 59 05 "STORE"  MerchantName
		"6007JAKARTA", // 60 07 "JAKARTA" MerchantCity
	}
	payload := strings.Join(parts, "") + "6304"
	return payload + CRC16(payload)
}

// validDynamicQRIS returns the static fixture mutated to dynamic with a fresh
// CRC. Used to test Parse() and the already-dynamic path of Convert().
func validDynamicQRIS() string {
	parts := []string{
		"000201",
		"010212", // dynamic
		"52045812",
		"5303360",
		"540550000", // amount 50000
		"5802ID",
		"5905STORE",
		"6007JAKARTA",
	}
	payload := strings.Join(parts, "") + "6304"
	return payload + CRC16(payload)
}
```

- [ ] **Step 2: Write the failing test for `Parse`**

Write `qris/parser_test.go`:
```go
package qris

import "testing"

func TestParse_Static(t *testing.T) {
	d, err := Parse(validStaticQRIS())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Version != "01" {
		t.Errorf("Version = %q, want %q", d.Version, "01")
	}
	if d.Method != MethodStatic {
		t.Errorf("Method = %q, want %q", d.Method, MethodStatic)
	}
	if d.MerchantName != "STORE" {
		t.Errorf("MerchantName = %q, want %q", d.MerchantName, "STORE")
	}
	if d.MerchantCity != "JAKARTA" {
		t.Errorf("MerchantCity = %q, want %q", d.MerchantCity, "JAKARTA")
	}
	if d.Currency != "360" {
		t.Errorf("Currency = %q, want %q", d.Currency, "360")
	}
	if d.CountryCode != "ID" {
		t.Errorf("CountryCode = %q, want %q", d.CountryCode, "ID")
	}
	if d.CRC == "" {
		t.Errorf("CRC empty")
	}
	if len(d.Raw) == 0 {
		t.Errorf("Raw empty")
	}
}

func TestParse_Dynamic(t *testing.T) {
	d, err := Parse(validDynamicQRIS())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Method != MethodDynamic {
		t.Errorf("Method = %q, want %q", d.Method, MethodDynamic)
	}
}

func TestParse_Garbage(t *testing.T) {
	if _, err := Parse("not a qris"); err == nil {
		t.Fatal("expected error for garbage input, got nil")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run:
```bash
go test ./qris/...
```
Expected: FAIL — `undefined: Parse`, `undefined: MethodStatic`, etc.

- [ ] **Step 4: Implement `Parse`, `Data`, `Method`**

Write `qris/parser.go`:
```go
package qris

import "fmt"

// Method is the QRIS point-of-initiation indicator (tag 01).
type Method string

const (
	MethodStatic  Method = "static"
	MethodDynamic Method = "dynamic"
)

// Data is the structured view of a QRIS string (top-level fields only).
type Data struct {
	Version      string
	Method       Method
	MerchantName string
	MerchantCity string
	Currency     string
	CountryCode  string
	CRC          string
	Raw          []TLV
}

// Parse decodes a QRIS string into Data. It does not verify the CRC; use
// Validate for that.
func Parse(s string) (Data, error) {
	tlvs, err := parseTLV(s)
	if err != nil {
		return Data{}, err
	}
	d := Data{Raw: tlvs, Currency: "360", CountryCode: "ID"}
	for _, t := range tlvs {
		switch t.Tag {
		case "00":
			d.Version = t.Value
		case "01":
			switch t.Value {
			case "11":
				d.Method = MethodStatic
			case "12":
				d.Method = MethodDynamic
			default:
				return Data{}, fmt.Errorf("%w: unknown method %q", ErrInvalidFormat, t.Value)
			}
		case "53":
			d.Currency = t.Value
		case "58":
			d.CountryCode = t.Value
		case "59":
			d.MerchantName = t.Value
		case "60":
			d.MerchantCity = t.Value
		case "63":
			d.CRC = t.Value
		}
	}
	return d, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run:
```bash
go test ./qris/...
```
Expected: PASS — all `TestParse_*` subtests pass.

- [ ] **Step 6: Commit**

```bash
git add qris/parser.go qris/parser_test.go qris/helpers_test.go
git commit -m "feat(qris): add Parse + Data + test helpers"
```

---

### Task 6: `Validate` (TDD)

**Files:**
- Create: `qris/validator.go`
- Create: `qris/validator_test.go`

- [ ] **Step 1: Write the failing test**

Write `qris/validator_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./qris/...
```
Expected: FAIL — `undefined: Validate`.

- [ ] **Step 3: Implement `Validate`**

Write `qris/validator.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test ./qris/...
```
Expected: PASS — all `TestValidate_*` subtests pass.

- [ ] **Step 5: Commit**

```bash
git add qris/validator.go qris/validator_test.go
git commit -m "feat(qris): add Validate with CRC check"
```

---

### Task 7: `Convert` + `Options` (TDD)

**Files:**
- Create: `qris/converter.go`
- Create: `qris/converter_test.go`

- [ ] **Step 1: Write the failing test**

Write `qris/converter_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test ./qris/...
```
Expected: FAIL — `undefined: Options`, `undefined: Convert`.

- [ ] **Step 3: Implement `Convert` and `Options`**

Write `qris/converter.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test ./qris/...
```
Expected: PASS — all `TestConvert_*` subtests pass; previous tests still green.

- [ ] **Step 5: Commit**

```bash
git add qris/converter.go qris/converter_test.go
git commit -m "feat(qris): add Convert (static to dynamic)"
```

---

### Task 8: CLI entry (`cmd/qris/main.go`)

**Files:**
- Create: `cmd/qris/main.go`

- [ ] **Step 1: Write `main.go`**

Write `cmd/qris/main.go`:
```go
// Command qris converts a static QRIS string into a dynamic one.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/rizkirmdhn/go-qris-dinamis/qris"
)

var version = "dev"

const usage = `Usage: qris [flags]

Convert a static QRIS string into a dynamic one with an injected amount and
optional fixed fee. Input may come from -i, -f, or stdin (piped).

Flags:
  -i, --input    string  QRIS source string
  -f, --file     string  path to a file containing the QRIS string
  -a, --amount   int     transaction amount in rupiah (required, > 0)
      --fee      int     optional fixed service fee in rupiah (default 0)
  -h, --help             show this message
  -v, --version          print version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var (
		input   string
		file    string
		amount  int
		fee     int
		showVer bool
	)

	fs := flag.NewFlagSet("qris", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // suppress default error output; we print our own
	fs.StringVar(&input, "i", "", "")
	fs.StringVar(&input, "input", "", "")
	fs.StringVar(&file, "f", "", "")
	fs.StringVar(&file, "file", "", "")
	fs.IntVar(&amount, "a", 0, "")
	fs.IntVar(&amount, "amount", 0, "")
	fs.IntVar(&fee, "fee", 0, "")
	fs.BoolVar(&showVer, "v", false, "")
	fs.BoolVar(&showVer, "version", false, "")
	help := fs.Bool("h", false, "")
	helpLong := fs.Bool("help", false, "")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "qris: %v\n%s", err, usage)
		return 2
	}
	if *help || *helpLong {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if showVer {
		fmt.Fprintf(stdout, "qris %s\n", version)
		return 0
	}

	// Validate amount/fee at the CLI layer so we never see library sentinels.
	if amount <= 0 || len(strconv.Itoa(amount)) > 13 {
		fmt.Fprintf(stderr, "qris: amount must be > 0 and at most 13 digits\n")
		return 2
	}
	if fee < 0 || len(strconv.Itoa(fee)) > 13 {
		fmt.Fprintf(stderr, "qris: fee must be >= 0 and at most 13 digits\n")
		return 2
	}

	// Resolve input source: exactly one of -i, -f, stdin (piped).
	src, err := resolveInput(input, file, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "qris: %v\n", err)
		return 2
	}
	src = strings.TrimSpace(src)
	if src == "" {
		fmt.Fprintf(stderr, "qris: empty input\n")
		return 2
	}

	out, err := qris.Convert(src, qris.Options{Amount: amount, Fee: fee})
	if err != nil {
		fmt.Fprintf(stderr, "qris: %v\n", err)
		switch {
		case errors.Is(err, qris.ErrInvalidAmount), errors.Is(err, qris.ErrInvalidFee):
			return 2 // should be unreachable; CLI validates first
		default:
			return 1
		}
	}
	fmt.Fprintln(stdout, out)
	return 0
}

func resolveInput(input, file string, stdin io.Reader) (string, error) {
	sources := 0
	if input != "" {
		sources++
	}
	if file != "" {
		sources++
	}
	stdinPiped := isPiped(stdin)
	if stdinPiped {
		sources++
	}
	if sources == 0 {
		return "", errors.New("no input (use -i, -f, or pipe to stdin)")
	}
	if sources > 1 {
		return "", errors.New("multiple input sources; choose one of -i, -f, or stdin")
	}

	switch {
	case input != "":
		return input, nil
	case file != "":
		b, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("cannot read %s: %w", file, err)
		}
		return string(b), nil
	default:
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("cannot read stdin: %w", err)
		}
		return string(b), nil
	}
}

func isPiped(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return true // any non-file reader (e.g., test reader) is treated as piped
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}
```

- [ ] **Step 2: Build the binary and run manual smoke checks**

Run:
```bash
go build -o /tmp/qris ./cmd/qris
```
Expected: binary at `/tmp/qris`, no errors.

Run (no input):
```bash
/tmp/qris -a 50000; echo "exit=$?"
```
Expected: stderr `qris: no input (use -i, -f, or pipe to stdin)`, `exit=2`.

Run (help):
```bash
/tmp/qris -h
```
Expected: usage text on stdout, exit 0.

Run (version):
```bash
/tmp/qris -v
```
Expected: `qris dev` on stdout, exit 0.

Run (multiple sources error):
```bash
/tmp/qris -i "x" -f "y" -a 50000; echo "exit=$?"
```
Expected: stderr `qris: multiple input sources; choose one of -i, -f, or stdin`, `exit=2`.

Run (CRC mismatch via -i):
```bash
/tmp/qris -i "00020101021100AB" -a 50000; echo "exit=$?"
```
Expected: stderr `qris: invalid QRIS format: ...` (or CRC mismatch), `exit=1`.

End-to-end happy path is exercised by the smoke tests in Task 9.

- [ ] **Step 3: Commit**

```bash
git add cmd/qris/main.go
git commit -m "feat(cli): add qris convert command"
```

---

### Task 9: CLI smoke test

**Files:**
- Create: `cmd/qris/main_test.go`

- [ ] **Step 1: Write the smoke test**

Write `cmd/qris/main_test.go`:
```go
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
	if !strings.Contains(errBuf.String(), "no input") {
		t.Errorf("stderr missing 'no input': %s", errBuf.String())
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
```

- [ ] **Step 2: Run the smoke test**

Run:
```bash
go test ./...
```
Expected: PASS — all `TestRun_*` plus all `qris` tests.

- [ ] **Step 3: Commit**

```bash
git add cmd/qris/main_test.go
git commit -m "test(cli): add CLI smoke tests"
```

---

### Task 10: Finalize README + version tag

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace README content**

Write `README.md`:
```markdown
# go-qris-dinamis

CLI Go untuk mengubah QRIS statis menjadi dinamis. Suntik nominal transaksi, opsional service fee tetap, lalu recompute CRC16-CCITT. Zero external dependency.

Diadaptasi dari [verssache/qris-dinamis](https://github.com/verssache/qris-dinamis) (TypeScript).

## Install

```bash
go install github.com/rizkirmdhn/go-qris-dinamis/cmd/qris@latest
```

Atau build dari source:

```bash
git clone https://github.com/rizkirmdhn/go-qris-dinamis
cd go-qris-dinamis
go build -o qris ./cmd/qris
```

## CLI usage

```bash
# Input langsung
qris -i "00020101021126..." -a 50000

# Input dari file
qris -f qris.txt -a 50000

# Input dari stdin (pipe)
cat qris.txt | qris -a 50000

# Dengan fee fixed
qris -i "..." -a 50000 --fee=1000

# Pipe hasil ke file
qris -f in.txt -a 50000 > out.txt
```

### Flags

| Flag | Tipe | Wajib | Default | Deskripsi |
|---|---|---|---|---|
| `-i`, `--input` | string | salah satu dari 3 | — | QRIS string langsung |
| `-f`, `--file` | string | salah satu dari 3 | — | Path file berisi QRIS |
| (stdin) | — | salah satu dari 3 | — | Pipe QRIS via stdin |
| `-a`, `--amount` | int | ✅ | — | Nominal transaksi (rupiah, > 0, ≤ 13 digit) |
| `--fee` | int | tidak | `0` | Fee tetap (rupiah, ≥ 0, ≤ 13 digit) |
| `-h`, `--help` | — | — | — | Tampilkan usage |
| `-v`, `--version` | — | — | — | Tampilkan versi |

### Exit codes

- `0` — sukses.
- `1` — error dari library `qris` (CRC mismatch, format invalid, QRIS sudah dinamis).
- `2` — usage error (flag salah, multiple sources, no input, amount/fee invalid, file tidak terbaca).

## Library usage

```go
import "github.com/rizkirmdhn/go-qris-dinamis/qris"

out, err := qris.Convert(input, qris.Options{Amount: 50000, Fee: 1000})
```

Fungsi publik: `Parse`, `Validate`, `Convert`, `CRC16`. Sentinel errors: `ErrInvalidFormat`, `ErrCRCMismatch`, `ErrAlreadyDynamic`, `ErrInvalidAmount`, `ErrInvalidFee`.

## Development

```bash
go test ./...        # run all tests
go build ./cmd/qris  # build CLI
```

Versi binary di-inject saat build:

```bash
go build -ldflags "-X main.version=v0.1.0" -o qris ./cmd/qris
```

## License

MIT — see [LICENSE](LICENSE).
```

- [ ] **Step 2: Verify tests still pass**

Run:
```bash
go test ./...
```
Expected: PASS — all green.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: finalize README"
```

- [ ] **Step 4: Tag v0.1.0 (optional, ask user before pushing remote)**

Run:
```bash
git tag v0.1.0
git log --oneline
```
Expected: 10 commits, latest tagged `v0.1.0`. Do **not** push without explicit user permission.

---

## Done criteria

- `go test ./...` passes (all qris/* tests + cmd/qris/* smoke tests).
- `go build ./cmd/qris` produces a working binary.
- Manual smoke: `echo "<static qris>" | ./qris -a 50000` prints a valid dynamic QRIS (verifiable by `./qris -i "<output>" -a 1` returning `ErrAlreadyDynamic`).
- README documents install, CLI usage, library usage, exit codes.
- 10 atomic commits, one per task.
