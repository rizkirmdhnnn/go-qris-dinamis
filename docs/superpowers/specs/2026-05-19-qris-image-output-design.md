# QRIS Image Output — Design Spec

**Date:** 2026-05-19
**Status:** Approved (brainstorm), ready for implementation plan
**Author:** Achmad Rizki Ramadhan (with Claude)

## Context

`go-qris-dinamis` is a zero-dependency Go CLI that converts a static QRIS string into a dynamic one with an injected amount and optional fee. Today the CLI's only output is the dynamic QRIS string on stdout.

The motivating use case is that this tool will be invoked by an AI agent (CLI subprocess pattern, e.g. Claude Code-style Bash + Read). The agent needs to be able to obtain a visual representation of the resulting QRIS — i.e. the QR code rendered as an image — so it can be displayed back to the end user or otherwise consumed as multimodal content.

## Goal

Add the ability to emit a PNG QR code rendering of the dynamic QRIS, in a form that is convenient for an agent that exec's the CLI as a subprocess.

## Non-Goals

- Outputting base64 directly to stdout. (Not needed for CLI subprocess use; revisit if/when this tool is wrapped as an MCP server.)
- Other image formats (SVG, JPEG). PNG only for v1.
- Configurable QR size or error-correction level via CLI flags. Hardcoded defaults in v1.
- A separate `--quiet` flag to suppress the QRIS string on stdout when `--qr` is set. Stdout behavior is unchanged.
- Customizing colors, logos, styled QR codes.

## Decisions

### D1. Format: PNG file via `--qr <path>` flag

The CLI gains exactly one new flag, `--qr`, which takes a filesystem path. When set, the binary writes a PNG QR code of the dynamic QRIS to that path. Stdout still prints the dynamic QRIS string.

Rationale: the agent flow is `Bash qris ... --qr /tmp/q.png` followed by `Read /tmp/q.png`, which lets the agent's multimodal Read tool ingest the image natively. Base64 to stdout would either pollute the existing stdout contract or require a second mutually-exclusive flag; both were judged YAGNI for the current subprocess use case.

### D2. Dependency on `github.com/skip2/go-qrcode`

The module gives up its "zero external dependency" stance and pulls in `github.com/skip2/go-qrcode` (MIT, widely used, simple API, direct PNG output) for QR encoding.

The rendering helper lives inside the `qris/` package (not gated to `cmd/qris/`), so consumers who import the library as Go code can also render QR codes. This is intentional — there was no useful reason to keep the library pure once the dependency was accepted at the module level.

### D3. Library API: `qris.RenderPNG`

New public function in `qris/qrcode.go`:

```go
// RenderPNG renders an arbitrary QRIS string as a PNG QR code of the given
// pixel size (width = height = size). It does not parse or validate the input;
// callers pass in whatever string they want encoded (typically the output of
// Convert).
//
// Returns the PNG-encoded bytes, suitable for writing to a file or io.Writer.
func RenderPNG(qrisString string, size int) ([]byte, error)
```

- Error-correction level is fixed at `qrcode.Medium` inside the body. Not exposed in the signature to keep the surface minimal.
- No QRIS parse/validate inside this function. It is a pure encoder.
- Errors from `skip2/go-qrcode` are returned wrapped with context (e.g. `fmt.Errorf("qris: render PNG: %w", err)`).
- No new sentinel error types are introduced.

### D4. CLI flag surface

| Flag    | Type   | Required | Default | Description                                                                 |
|---------|--------|----------|---------|-----------------------------------------------------------------------------|
| `--qr`  | string | no       | `""`    | If set, write a PNG QR code of the dynamic QRIS to this path. Overwrites if the file exists. |

Hardcoded inside `cmd/qris/main.go` when calling `RenderPNG`:

- Size: **512** pixels (square).
- Error-correction level: `qrcode.Medium` (via `RenderPNG`).

There is no `-q` short flag. There is no `--qr-size` flag in v1; if needed later it is an easy additive change.

### D5. Execution order and stdout/file semantics

In `run(...)` inside `cmd/qris/main.go`, after the existing convert step:

1. `out, err := qris.Convert(...)`. On error, behavior is unchanged (exit 1).
2. `fmt.Fprintln(stdout, out)` — print the dynamic QRIS to stdout, unchanged from today.
3. If `--qr` is set:
   a. `png, err := qris.RenderPNG(out, 512)`. On error → exit code 1, stderr `qris: cannot render QR: <err>`.
   b. `os.WriteFile(qrPath, png, 0o644)`. On error → exit code 2, stderr `qris: cannot write QR file: <err>`.
4. Exit 0.

Stdout is *not* rolled back if step 3 fails. The agent still gets the QRIS string, plus a non-zero exit code and a clear stderr message indicating the QR file step failed. This is judged more useful than failing atomically.

Files are overwritten if they exist. The CLI does not create intermediate parent directories — `WriteFile` will surface an IO error for a missing directory, which becomes the exit-2 path.

### D6. Exit codes

Existing exit codes are preserved. New conditions:

| Condition                                     | Exit | Stderr                                  |
|-----------------------------------------------|------|-----------------------------------------|
| `RenderPNG` encoder error                     | 1    | `qris: cannot render QR: <err>`         |
| File write error (bad path, permission, etc.) | 2    | `qris: cannot write QR file: <err>`     |
| Sukses (with or without `--qr`)               | 0    | —                                       |

### D7. Spec storage and .gitignore exception

The repo's `.gitignore` previously ignored `/docs/` entirely (commit `7aa3d51`). To keep this spec tracked without re-tracking the rest of `/docs/`, the `.gitignore` gains a single whitelist line:

```
# Docs
/docs/
!/docs/superpowers/
```

This scope is intentionally narrow — only `docs/superpowers/` is unignored; anything else added under `/docs/` later stays ignored.

## Architecture

```
qris/
├── qris/
│   ├── converter.go       (unchanged)
│   ├── crc16.go           (unchanged)
│   ├── errors.go          (unchanged)
│   ├── parser.go          (unchanged)
│   ├── tlv.go             (unchanged)
│   ├── validator.go       (unchanged)
│   ├── qrcode.go          (NEW: RenderPNG)
│   └── qrcode_test.go     (NEW)
├── cmd/qris/
│   ├── main.go            (MODIFIED: --qr flag, usage update, render+write step)
│   └── main_test.go       (MODIFIED: add --qr tests)
├── go.mod                 (MODIFIED: + github.com/skip2/go-qrcode)
├── go.sum                 (NEW or MODIFIED)
├── .gitignore             (MODIFIED: + !/docs/superpowers/)
├── README.md              (MODIFIED: document --qr flag and RenderPNG)
└── docs/superpowers/specs/
    └── 2026-05-19-qris-image-output-design.md  (this file)
```

Dependency direction is one-way: `cmd/qris/` → `qris/` → `skip2/go-qrcode`. The `cmd/qris/` package does not import the QR encoder directly; it only knows about `qris.RenderPNG`.

## Testing

### Unit tests in `qris/qrcode_test.go`

- `TestRenderPNG_ValidQRIS`: call `RenderPNG("00020101021126...", 512)`, assert:
  - `err == nil`
  - returned bytes start with the PNG signature `\x89PNG\r\n\x1a\n`
  - `len(bytes) > 100` (sanity floor)
- `TestRenderPNG_EmptyInput`: call `RenderPNG("", 256)`. Document the actual behavior of `skip2/go-qrcode` for empty input during implementation (it either produces an empty-payload QR or returns an error). The test asserts whichever is real — no assumption is baked into the spec.

No pixel-level / visual diff testing. The PNG signature plus a non-trivial byte count is the contract.

### Integration tests in `cmd/qris/main_test.go`

Follow the existing test style (table-driven `run(...)` invocations).

- `TestRun_WithQRFlag`: use `t.TempDir()` to get a path, run `run([]string{"-i", validQRIS, "-a", "50000", "--qr", path}, ...)`. Assert:
  - exit code `0`
  - stdout contains the dynamic QRIS (existing assertion pattern)
  - file at `path` exists and starts with PNG signature
- `TestRun_QRFlag_BadPath`: pass a path under a non-existent directory (e.g. `t.TempDir() + "/does-not-exist/q.png"`). Assert:
  - exit code `2`
  - stderr contains `cannot write QR file`
  - stdout still contains the dynamic QRIS (step 2 ran before step 3 failed)

No mocking — tests hit the real encoder and real filesystem via `t.TempDir()`.

## Open Questions

None. All decisions above were made explicitly during brainstorming.

## Out of Scope (Future)

- `--qr-base64` flag for stdout base64 output (useful when wrapping as MCP server).
- `--qr-size <px>` for custom dimensions.
- `--qr-ec <L|M|Q|H>` for error-correction level.
- SVG / vector output.
- Wrapping the tool as an MCP server with image content blocks.
