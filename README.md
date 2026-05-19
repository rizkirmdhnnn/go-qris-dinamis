# go-qris-dinamis

CLI Go untuk mengubah QRIS statis menjadi dinamis. Suntik nominal transaksi, opsional service fee tetap, lalu recompute CRC16-CCITT. Satu dependensi eksternal: `github.com/skip2/go-qrcode` untuk render QR PNG.

Diadaptasi dari [verssache/qris-dinamis](https://github.com/verssache/qris-dinamis) (TypeScript).

## Install

```bash
go install github.com/rizkirmdhnnn/go-qris-dinamis/cmd/qris@latest
```

Atau build dari source:

```bash
git clone https://github.com/rizkirmdhnnn/go-qris-dinamis
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

# Sekaligus tulis gambar QR code ke PNG
qris -i "..." -a 50000 --qr qr.png
```

### Flags

| Flag | Tipe | Wajib | Default | Deskripsi |
|---|---|---|---|---|
| `-i`, `--input` | string | salah satu dari 3 | — | QRIS string langsung |
| `-f`, `--file` | string | salah satu dari 3 | — | Path file berisi QRIS |
| (stdin) | — | salah satu dari 3 | — | Pipe QRIS via stdin |
| `-a`, `--amount` | int | ✅ | — | Nominal transaksi (rupiah, > 0, ≤ 13 digit) |
| `--fee` | int | tidak | `0` | Fee tetap (rupiah, ≥ 0, ≤ 13 digit) |
| `--qr` | string | tidak | — | Path file PNG; jika di-set, tulis QR code dari QRIS dinamis ke path tsb |
| `-h`, `--help` | — | — | — | Tampilkan usage |
| `-v`, `--version` | — | — | — | Tampilkan versi |

### Exit codes

- `0` — sukses.
- `1` — error dari library `qris` (CRC mismatch, format invalid, QRIS sudah dinamis, atau gagal render QR).
- `2` — usage error (flag salah, multiple sources, no input, amount/fee invalid, file tidak terbaca, atau gagal tulis file QR PNG).

## Library usage

```go
import "github.com/rizkirmdhnnn/go-qris-dinamis/qris"

out, err := qris.Convert(input, qris.Options{Amount: 50000, Fee: 1000})
```

Fungsi publik: `Parse`, `Validate`, `Convert`, `CRC16`, `RenderPNG`. Sentinel errors: `ErrInvalidFormat`, `ErrCRCMismatch`, `ErrAlreadyDynamic`, `ErrInvalidAmount`, `ErrInvalidFee`.

Untuk merender QRIS jadi gambar QR code PNG (mis. setelah `Convert`), pakai `qris.RenderPNG(qrisString, size)` — mengembalikan byte PNG siap ditulis ke file atau `io.Writer`. Error-correction level dipatok ke Medium.

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
