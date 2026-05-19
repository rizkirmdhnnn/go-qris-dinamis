// Command qris converts a static QRIS string into a dynamic one.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/rizkirmdhnnn/go-qris-dinamis/qris"
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
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}

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

	// Wrap stdin in a bufio.Reader so we can peek without consuming data.
	// isPiped uses *os.File + Stat() for real files (no blocking on TTY),
	// and Peek for non-file readers (e.g., test strings.NewReader).
	br := bufio.NewReader(stdin)
	stdinPiped := isPiped(stdin, br)
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
		b, err := io.ReadAll(br)
		if err != nil {
			return "", fmt.Errorf("cannot read stdin: %w", err)
		}
		return string(b), nil
	}
}

// isPiped reports whether stdin carries actual data and should be counted as
// an input source.
//
// For *os.File (real stdin): use Stat() to check whether it is a character
// device (interactive TTY) or not (pipe/redirect).  This never blocks.
//
// For any other io.Reader (e.g. a test strings.NewReader): fall back to
// Peek(1) on the buffered wrapper.  These readers are never interactive TTYs,
// so Peek cannot block waiting for user input.
func isPiped(r io.Reader, br *bufio.Reader) bool {
	if f, ok := r.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return info.Mode()&os.ModeCharDevice == 0
	}
	// Non-file reader: peek to see if there is any data.
	_, err := br.Peek(1)
	return err == nil
}
