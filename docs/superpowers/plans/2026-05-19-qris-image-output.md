# QRIS Image Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `--qr <path>` CLI flag that writes a PNG QR code of the dynamic QRIS to disk, and expose `qris.RenderPNG` in the library.

**Architecture:** A new public function `RenderPNG(qrisString string, size int) ([]byte, error)` lives in `qris/qrcode.go` and wraps `github.com/skip2/go-qrcode`. The CLI in `cmd/qris/main.go` gains a single optional `--qr` flag; when set, after printing the dynamic QRIS to stdout it calls `RenderPNG(out, 512)` and writes the bytes via `os.WriteFile`. Stdout behavior is unchanged when the flag is absent. Encoder errors exit `1`; file-write errors exit `2`.

**Tech Stack:** Go (stdlib `flag`, `os`, `bytes`), `github.com/skip2/go-qrcode` for QR encoding, standard `testing` package with `t.TempDir()` for filesystem isolation.

**Reference spec:** `docs/superpowers/specs/2026-05-19-qris-image-output-design.md`

**Project conventions to follow:**
- Tabs for indentation (Go default).
- Test naming: `TestRun_<Behavior>` in `cmd/qris/`, `TestRenderPNG_<Behavior>` in `qris/`.
- CLI tests use `bytes.Buffer` for stdout/stderr and `strings.NewReader` for stdin; call `run(args, stdin, &out, &errBuf)` directly.
- CLI tests cannot import `validStaticQRIS()` from `qris/` (package-private); a local `validStatic()` helper already exists in `cmd/qris/main_test.go` — reuse it.
- Sentinel errors are checked via `errors.Is`; this plan does not introduce new sentinels.

**File map:**
- Create: `qris/qrcode.go` — one function: `RenderPNG`.
- Create: `qris/qrcode_test.go` — unit tests for `RenderPNG`.
- Modify: `cmd/qris/main.go` — add `--qr` flag, usage doc, render+write step.
- Modify: `cmd/qris/main_test.go` — add `TestRun_WithQRFlag` and `TestRun_QRFlag_BadPath`.
- Modify: `go.mod` and `go.sum` — add `github.com/skip2/go-qrcode` dependency.
- Modify: `README.md` — document the flag and the library function.

---

### Task 1: Add the `skip2/go-qrcode` dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum` (created on first `go mod tidy`)

- [ ] **Step 1: Add the dependency**

Run from repo root:

```bash
go get github.com/skip2/go-qrcode@latest
```

Expected: a line `require github.com/skip2/go-qrcode vX.Y.Z` appears in `go.mod`; `go.sum` is created/updated.

- [ ] **Step 2: Verify the module still builds**

```bash
go build ./...
```

Expected: no output, exit 0. (Nothing imports the new dep yet, so a `// indirect` marker may appear next to it in `go.mod` — that is fine and will go away once `qris/qrcode.go` imports it in Task 2.)

- [ ] **Step 3: Run existing tests to confirm no regression**

```bash
go test ./...
```

Expected: all existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add github.com/skip2/go-qrcode for QR rendering"
```

---

### Task 2: Library — `RenderPNG` happy path (TDD)

**Files:**
- Create: `qris/qrcode.go`
- Create: `qris/qrcode_test.go`

- [ ] **Step 1: Write the failing test**

Create `qris/qrcode_test.go` with the following contents:

```go
package qris

import (
	"bytes"
	"testing"
)

// pngSignature is the 8-byte magic header every PNG file starts with.
var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

func TestRenderPNG_ValidQRIS(t *testing.T) {
	in := validStaticQRIS()

	got, err := RenderPNG(in, 512)
	if err != nil {
		t.Fatalf("RenderPNG returned error: %v", err)
	}
	if !bytes.HasPrefix(got, pngSignature) {
		t.Errorf("output does not start with PNG signature; first 8 bytes = % x", got[:min(8, len(got))])
	}
	if len(got) < 100 {
		t.Errorf("PNG suspiciously small: %d bytes", len(got))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

(`validStaticQRIS()` already exists in `qris/helpers_test.go` and is reusable from any `_test.go` file in the same package.)

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./qris/ -run TestRenderPNG_ValidQRIS -v
```

Expected: build failure with `undefined: RenderPNG`.

- [ ] **Step 3: Write the minimal implementation**

Create `qris/qrcode.go`:

```go
package qris

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

// RenderPNG renders an arbitrary QRIS string as a PNG QR code of the given
// pixel size (width = height = size). It does not parse or validate the input;
// callers pass in whatever string they want encoded (typically the output of
// Convert). Error-correction level is fixed at Medium.
//
// Returns the PNG-encoded bytes, suitable for writing to a file or io.Writer.
func RenderPNG(qrisString string, size int) ([]byte, error) {
	png, err := qrcode.Encode(qrisString, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("qris: render PNG: %w", err)
	}
	return png, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./qris/ -run TestRenderPNG_ValidQRIS -v
```

Expected: `--- PASS: TestRenderPNG_ValidQRIS`.

- [ ] **Step 5: Run the full library test suite to confirm no regression**

```bash
go test ./qris/
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add qris/qrcode.go qris/qrcode_test.go
git commit -m "feat(qris): add RenderPNG for PNG QR code rendering"
```

---

### Task 3: Library — `RenderPNG` empty-input behavior

The spec leaves the assertion for empty input open because `skip2/go-qrcode`'s behavior on `""` is not assumed. Probe it, then lock in the test.

**Files:**
- Modify: `qris/qrcode_test.go`

- [ ] **Step 1: Probe the actual behavior**

Write a one-off probe program to a temp file and run it from the repo root:

```bash
cat > /tmp/qris_probe.go <<'EOF'
package main

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

func main() {
	b, err := qrcode.Encode("", qrcode.Medium, 256)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Printf("OK: %d bytes, first 8 = % x\n", len(b), b[:8])
}
EOF
go run /tmp/qris_probe.go
rm /tmp/qris_probe.go
```

Note the result. There are exactly two outcomes:

- **(A) `ERROR: <message>`** — encoder rejects empty input. Use the assertion in Step 2a.
- **(B) `OK: <N> bytes, first 8 = 89 50 4e 47 0d 0a 1a 0a`** — encoder succeeds. Use the assertion in Step 2b.

- [ ] **Step 2a (if probe returned an error): Write the empty-input test for the error path**

Append to `qris/qrcode_test.go`:

```go
func TestRenderPNG_EmptyInput(t *testing.T) {
	_, err := RenderPNG("", 256)
	if err == nil {
		t.Fatal("RenderPNG(\"\", 256) returned nil error, want error")
	}
}
```

- [ ] **Step 2b (if probe succeeded): Write the empty-input test for the success path**

Append to `qris/qrcode_test.go`:

```go
func TestRenderPNG_EmptyInput(t *testing.T) {
	got, err := RenderPNG("", 256)
	if err != nil {
		t.Fatalf("RenderPNG(\"\", 256) returned error: %v", err)
	}
	if !bytes.HasPrefix(got, pngSignature) {
		t.Errorf("output does not start with PNG signature; first 8 bytes = % x", got[:min(8, len(got))])
	}
}
```

Pick exactly one of 2a or 2b based on the probe result. Do not include both.

- [ ] **Step 3: Run the test to verify it passes**

```bash
go test ./qris/ -run TestRenderPNG_EmptyInput -v
```

Expected: `--- PASS: TestRenderPNG_EmptyInput`.

- [ ] **Step 4: Commit**

```bash
git add qris/qrcode_test.go
git commit -m "test(qris): cover RenderPNG empty-input behavior"
```

---

### Task 4: CLI — `--qr` flag happy path (TDD)

**Files:**
- Modify: `cmd/qris/main_test.go`
- Modify: `cmd/qris/main.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/qris/main_test.go`:

```go
func TestRun_WithQRFlag(t *testing.T) {
	dir := t.TempDir()
	qrPath := dir + "/q.png"

	var out, errBuf bytes.Buffer
	code := run(
		[]string{"-i", validStatic(), "-a", "50000", "--qr", qrPath},
		strings.NewReader(""),
		&out, &errBuf,
	)
	if code != 0 {
		t.Fatalf("exit=%d, want 0; stderr=%s", code, errBuf.String())
	}

	// Stdout still carries the dynamic QRIS string.
	if err := qris.Validate(strings.TrimSpace(out.String())); err != nil {
		t.Fatalf("stdout invalid QRIS: %v", err)
	}

	// File at qrPath exists and starts with PNG signature.
	data, err := os.ReadFile(qrPath)
	if err != nil {
		t.Fatalf("reading QR file: %v", err)
	}
	pngSig := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(data, pngSig) {
		t.Errorf("QR file does not start with PNG signature; first bytes = % x", data[:8])
	}
}
```

Also add `"os"` to the `import` block of `cmd/qris/main_test.go` if it is not already imported. (After this edit the imports become `"bytes"`, `"os"`, `"strings"`, `"testing"`, plus the existing `"github.com/rizkirmdhnnn/go-qris-dinamis/qris"`.)

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./cmd/qris/ -run TestRun_WithQRFlag -v
```

Expected: FAIL — the `--qr` flag is unknown, so `flag.Parse` returns an error and `run` exits with code `2`.

- [ ] **Step 3: Add the `--qr` flag declaration in `main.go`**

In `cmd/qris/main.go`, inside `run(...)`, locate the block where the other flags are declared (the `fs.StringVar`, `fs.IntVar`, etc. calls just after `fs := flag.NewFlagSet(...)`). Add the new variable declaration alongside the existing locals, and register the flag:

```go
var (
	input   string
	file    string
	amount  int
	fee     int
	qrPath  string
	showVer bool
)
```

Then in the flag-registration block, after the existing `fs.IntVar(&fee, "fee", 0, "")` line, add:

```go
fs.StringVar(&qrPath, "qr", "", "")
```

- [ ] **Step 4: Add the render-and-write step**

In `cmd/qris/main.go`, find the existing happy-path block (currently the last few lines of `run`):

```go
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
```

Replace the final `fmt.Fprintln(stdout, out); return 0` with the rendering branch:

```go
	fmt.Fprintln(stdout, out)

	if qrPath != "" {
		png, err := qris.RenderPNG(out, 512)
		if err != nil {
			fmt.Fprintf(stderr, "qris: cannot render QR: %v\n", err)
			return 1
		}
		if err := os.WriteFile(qrPath, png, 0o644); err != nil {
			fmt.Fprintf(stderr, "qris: cannot write QR file: %v\n", err)
			return 2
		}
	}

	return 0
}
```

`os` is already imported in `main.go`, so no import change is required.

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./cmd/qris/ -run TestRun_WithQRFlag -v
```

Expected: `--- PASS: TestRun_WithQRFlag`.

- [ ] **Step 6: Run the full CLI test suite to confirm no regression**

```bash
go test ./cmd/qris/
```

Expected: all tests pass, including the existing `TestRun_*` suite.

- [ ] **Step 7: Commit**

```bash
git add cmd/qris/main.go cmd/qris/main_test.go
git commit -m "feat(cli): add --qr flag to write PNG QR code"
```

---

### Task 5: CLI — `--qr` bad-path error (TDD)

Verify that an unwritable path produces exit code `2` and a clear stderr message, and that stdout is unaffected (the QRIS string has already been printed by the time the write fails).

**Files:**
- Modify: `cmd/qris/main_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/qris/main_test.go`:

```go
func TestRun_QRFlag_BadPath(t *testing.T) {
	// Path under a non-existent directory inside the temp dir — os.WriteFile
	// will fail with "no such file or directory".
	dir := t.TempDir()
	qrPath := dir + "/does-not-exist/q.png"

	var out, errBuf bytes.Buffer
	code := run(
		[]string{"-i", validStatic(), "-a", "50000", "--qr", qrPath},
		strings.NewReader(""),
		&out, &errBuf,
	)
	if code != 2 {
		t.Fatalf("exit=%d, want 2; stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "cannot write QR file") {
		t.Errorf("stderr missing 'cannot write QR file': %s", errBuf.String())
	}
	// Stdout already received the dynamic QRIS string before the write failed.
	if err := qris.Validate(strings.TrimSpace(out.String())); err != nil {
		t.Errorf("stdout invalid QRIS (should still be printed): %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it passes immediately**

```bash
go test ./cmd/qris/ -run TestRun_QRFlag_BadPath -v
```

Expected: PASS. This test should pass on first run because Task 4 already implemented the error path. (If it fails, fix Task 4's error handling before continuing — do not change the test to match a broken implementation.)

- [ ] **Step 3: Run the full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/qris/main_test.go
git commit -m "test(cli): cover --qr bad-path error case"
```

---

### Task 6: CLI — update the usage string

The `usage` constant in `main.go` documents every flag; add `--qr` so `qris -h` and the no-args banner stay accurate.

**Files:**
- Modify: `cmd/qris/main.go`

- [ ] **Step 1: Update the usage constant**

Open `cmd/qris/main.go`. The current `usage` value is:

```go
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
```

Replace it with:

```go
const usage = `Usage: qris [flags]

Convert a static QRIS string into a dynamic one with an injected amount and
optional fixed fee. Input may come from -i, -f, or stdin (piped). When --qr
is set, a PNG QR code of the dynamic QRIS is also written to the given path
(stdout still prints the QRIS string).

Flags:
  -i, --input    string  QRIS source string
  -f, --file     string  path to a file containing the QRIS string
  -a, --amount   int     transaction amount in rupiah (required, > 0)
      --fee      int     optional fixed service fee in rupiah (default 0)
      --qr       string  path to write a PNG QR code of the dynamic QRIS
  -h, --help             show this message
  -v, --version          print version
`
```

- [ ] **Step 2: Run the help test to confirm it still passes**

```bash
go test ./cmd/qris/ -run TestRun_HelpExitsZero -v
```

Expected: PASS. The existing test only checks that the output contains `Usage:`, so updating the body does not break it.

- [ ] **Step 3: Run the full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/qris/main.go
git commit -m "docs(cli): document --qr flag in usage string"
```

---

### Task 7: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add a `--qr` example to the CLI usage section**

In `README.md`, find the existing fenced block of CLI examples (under `## CLI usage`). It currently ends with:

```bash
# Pipe hasil ke file
qris -f in.txt -a 50000 > out.txt
```

Append a new example after that line, inside the same fenced block:

```bash
# Sekaligus tulis gambar QR code ke PNG
qris -i "..." -a 50000 --qr qr.png
```

- [ ] **Step 2: Add the `--qr` row to the flags table**

In the flags table, insert one new row directly above the `-h, --help` row:

```
| `--qr` | string | tidak | — | Path file PNG; jika di-set, tulis QR code dari QRIS dinamis ke path tsb |
```

The resulting table fragment should look like:

```
|      `--fee` | int | tidak | `0` | Fee tetap (rupiah, ≥ 0, ≤ 13 digit) |
| `--qr` | string | tidak | — | Path file PNG; jika di-set, tulis QR code dari QRIS dinamis ke path tsb |
| `-h`, `--help` | — | — | — | Tampilkan usage |
```

- [ ] **Step 3: Document `RenderPNG` in the library section**

In the `## Library usage` section, after the existing line:

```
Fungsi publik: `Parse`, `Validate`, `Convert`, `CRC16`. Sentinel errors: `ErrInvalidFormat`, `ErrCRCMismatch`, `ErrAlreadyDynamic`, `ErrInvalidAmount`, `ErrInvalidFee`.
```

Add one short paragraph beneath it:

```markdown
Untuk merender QRIS jadi gambar QR code PNG (mis. setelah `Convert`), pakai `qris.RenderPNG(qrisString, size)` — mengembalikan byte PNG siap ditulis ke file atau `io.Writer`. Error-correction level dipatok ke Medium.
```

And update the function list line so it now reads:

```markdown
Fungsi publik: `Parse`, `Validate`, `Convert`, `CRC16`, `RenderPNG`. Sentinel errors: `ErrInvalidFormat`, `ErrCRCMismatch`, `ErrAlreadyDynamic`, `ErrInvalidAmount`, `ErrInvalidFee`.
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs(readme): document --qr flag and RenderPNG"
```

---

## Verification

After all tasks are complete, run the full pipeline once more to be sure:

- [ ] `go build ./...` succeeds.
- [ ] `go test ./...` — all tests pass (existing CLI + library tests, plus `TestRenderPNG_ValidQRIS`, `TestRenderPNG_EmptyInput`, `TestRun_WithQRFlag`, `TestRun_QRFlag_BadPath`).
- [ ] Manual smoke (optional, for the implementer's confidence):

```bash
# Build a one-off binary, generate a dynamic QRIS plus its PNG.
go build -o /tmp/qris ./cmd/qris
echo "00020101021126570011ID.DANA.WWW..." | /tmp/qris -a 50000 --qr /tmp/q.png
file /tmp/q.png   # should report: PNG image data, 512 x 512, 8-bit grayscale/colormap, ...
```

If `file` reports a valid PNG with the expected dimensions and stdout contained a dynamic QRIS string, the feature is working end-to-end.

The agent use case (CLI subprocess + `Read` on the PNG) needs no extra wiring — it falls out automatically from this implementation.
