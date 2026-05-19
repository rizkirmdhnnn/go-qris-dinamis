# Design: `go-qris-dinamis` — CLI Go untuk Konversi QRIS Statis → Dinamis

- **Tanggal**: 2026-05-19
- **Status**: Draft (menunggu review user)
- **Sumber inspirasi**: https://github.com/verssache/qris-dinamis (TypeScript)
- **Repo target**: `github.com/rizkirmdhn/go-qris-dinamis`

---

## 1. Tujuan

Sediakan CLI Go satu-binary yang mengubah QRIS statis menjadi QRIS dinamis dengan menyuntikkan nominal transaksi dan (opsional) service fee tetap, lalu menghitung ulang CRC16. Package internalnya juga harus bisa dipakai sebagai library publik (`import` dari project Go lain).

**Non-goal**:
- Subcommand `parse` / `validate` terpisah (hanya tersedia via library API).
- Fee tipe `percentage` — hanya `fixed`.
- Generate gambar QR PNG.
- Web UI / decoding gambar QR.
- Mendukung children TLV (tag 26–51, 62) — tidak dibutuhkan untuk operasi konversi top-level.

---

## 2. Arsitektur

Dua lapis tegas:

1. **`cmd/qris/main.go`** — adapter CLI. Tanggung jawab: parsing flag, baca QRIS dari salah satu sumber input (flag `-i`, file `-f`, atau stdin pipe), panggil package `qris`, tulis hasil ke stdout, terjemahkan error sentinel ke pesan + exit code. Tidak boleh berisi logika TLV/CRC/QRIS.
2. **`qris/` package (publik)** — pustaka murni. Framework-agnostic, tidak melakukan I/O (semua fungsi terima/return string). Bisa diimpor sebagai library:
   ```go
   import "github.com/rizkirmdhn/go-qris-dinamis/qris"
   out, err := qris.Convert(input, qris.Options{Amount: 50000, Fee: 1000})
   ```

**Alur eksekusi**:

```
stdin / file / --input  ──▶  cmd/qris/main.go  ──▶  qris.Validate  ──▶  qris.Convert  ──▶  stdout
                                     │
                                     └─▶  exit 1 (validation) atau exit 2 (usage)
```

**Prinsip**:
- **Zero external deps** — hanya stdlib (`flag`, `os`, `io`, `errors`, `strings`, `strconv`, `fmt`, `testing`).
- `main.go` tipis; logika ada di package `qris`.
- Sentinel errors di-export untuk mempermudah konsumen library.

---

## 3. Spesifikasi CLI

### 3.1 Flag

| Flag | Tipe | Wajib | Default | Deskripsi |
|---|---|---|---|---|
| `-i`, `--input` | string | salah satu dari 3 sumber | — | QRIS string langsung |
| `-f`, `--file` | string | salah satu dari 3 sumber | — | Path file teks berisi QRIS string |
| (stdin) | — | salah satu dari 3 sumber | — | Dipakai bila stdin di-pipe (bukan TTY) dan `-i`/`-f` kosong |
| `-a`, `--amount` | int | ✅ ya | — | Nominal transaksi dalam rupiah, harus > 0, max 13 digit |
| `--fee` | int | tidak | `0` | Fee tetap (fixed) dalam rupiah; `0` = tanpa fee; tidak boleh negatif |
| `-h`, `--help` | — | — | — | Tampilkan usage |
| `-v`, `--version` | — | — | — | Tampilkan versi binary |

Catatan: stdlib `flag` tidak natively mendukung short+long flag bersamaan. Implementasi: daftarkan dua nama untuk variable yang sama (`flag.StringVar(&input, "i", "", ...)` dan `flag.StringVar(&input, "input", "", ...)`).

### 3.2 Aturan input

- Harus **tepat satu** sumber input.
  - 0 sumber → error usage.
  - >1 sumber (mis. `-i` + `-f`) → error usage.
- Stdin hanya dianggap sumber input bila **bukan TTY** (cek `os.Stdin.Stat()` → `Mode()&os.ModeCharDevice == 0`).
- Input di-trim whitespace (`strings.TrimSpace`) sebelum diteruskan ke `qris`.

### 3.3 Output

- **Sukses**: cetak QRIS dinamis ke stdout, diakhiri satu `\n`. Exit code `0`.
- **Gagal**: pesan error ke stderr dengan prefix `qris: `. Exit code:
  - `1` — error dari package `qris` saat `Validate`/`Convert` dipanggil: `ErrCRCMismatch`, `ErrInvalidFormat`, `ErrAlreadyDynamic`.
  - `2` — usage error yang dideteksi CLI sebelum memanggil library: flag salah, multiple sources, tidak ada input, `amount ≤ 0` atau >13 digit, `fee < 0` atau >13 digit, file `-f` tidak bisa dibaca.

CLI memvalidasi argumen flag (`amount`, `fee`) sebelum memanggil `qris.Convert`, jadi `ErrInvalidAmount`/`ErrInvalidFee` dari library tidak akan bocor ke user lewat CLI. Sentinel itu tetap di-export untuk konsumen library.

### 3.4 Contoh

```bash
# Input langsung
qris -i "00020101021126..." -a 50000

# Input dari file
qris -f qris.txt -a 50000

# Input dari stdin (pipe)
cat qris.txt | qris -a 50000

# Dengan service fee fixed 1000
qris -i "..." -a 50000 --fee=1000

# Pipe hasil ke file
qris -f in.txt -a 50000 > out.txt
```

Contoh error:

```
$ qris -a 50000
qris: no input (use -i, -f, or pipe to stdin)
$ echo $?
2

$ qris -i "bad" -a 50000
qris: invalid QRIS: CRC mismatch
$ echo $?
1
```

---

## 4. API Package `qris`

### 4.1 Tipe

```go
package qris

// Options untuk Convert.
type Options struct {
    Amount int // wajib, > 0
    Fee    int // opsional; 0 = tanpa fee; tidak boleh negatif
}

// Data hasil Parse. Hanya field top-level yang dibutuhkan.
type Data struct {
    Version      string // tag 00
    Method       Method // dari tag 01
    MerchantName string // tag 59
    MerchantCity string // tag 60
    Currency     string // tag 53, default "360" (IDR)
    CountryCode  string // tag 58, default "ID"
    CRC          string // tag 63
    Raw          []TLV  // semua TLV top-level urut
}

type Method string

const (
    MethodStatic  Method = "static"
    MethodDynamic Method = "dynamic"
)

type TLV struct {
    Tag    string // 2-char
    Length int
    Value  string
}
```

### 4.2 Fungsi publik

```go
// Parse mengurai string QRIS menjadi Data terstruktur (top-level saja).
func Parse(s string) (Data, error)

// Validate memeriksa struktur dasar + cocokkan CRC16-CCITT (tag 63 di akhir).
func Validate(s string) error

// Convert mengubah QRIS statis menjadi dinamis dengan amount dan fee opsional.
// Memanggil Validate dulu; error kalau QRIS sudah dinamis.
func Convert(s string, opts Options) (string, error)

// CRC16 menghitung CRC16-CCITT (poly 0x1021, init 0xFFFF) untuk s.
// Mengembalikan 4 hex char uppercase.
func CRC16(s string) string
```

### 4.3 Sentinel errors

```go
var (
    ErrInvalidFormat  = errors.New("invalid QRIS format")
    ErrCRCMismatch    = errors.New("CRC mismatch")
    ErrAlreadyDynamic = errors.New("QRIS is already dynamic")
    ErrInvalidAmount  = errors.New("amount must be > 0 and ≤ 13 digits")
    ErrInvalidFee     = errors.New("fee must be ≥ 0 and ≤ 13 digits")
)
```

CLI memetakan error ini ke pesan + exit code.

---

## 5. Algoritma

### 5.1 TLV (Tag-Length-Value)

Format EMVCo: `TT LL VVVV...` berulang.
- `TT` — 2 char ASCII (tag).
- `LL` — 2 char ASCII desimal (panjang value, 00–99).
- `VV...` — `LL` byte ASCII (value mentah).

Parser (di `qris/tlv.go`) iterates dari index 0; setiap iterasi: butuh ≥ 4 char tersisa, lalu butuh ≥ `length` char tambahan. Tidak ada parsing nested.

### 5.2 CRC16-CCITT (False)

Spec EMVCo: poly `0x1021`, init `0xFFFF`, no XOR-out, no reflect. CRC dihitung atas seluruh payload **termasuk** literal `"6304"` (tag+length dari CRC), **tidak termasuk** value CRC itu sendiri.

```go
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

Vektor referensi: `CRC16("123456789") == "29B1"`.

### 5.3 `Validate(s)`

1. `len(s) >= 8` dan `s[len(s)-8:len(s)-4] == "6304"` → kalau tidak, `ErrInvalidFormat`.
2. `payload := s[:len(s)-4]` (semua kecuali 4 hex CRC).
3. `gotCRC := strings.ToUpper(s[len(s)-4:])`.
4. Kalau `CRC16(payload) != gotCRC` → `ErrCRCMismatch`.
5. `parseTLV(s[:len(s)-8])` (payload tanpa tag 63 head+value). Gagal → `ErrInvalidFormat`.
6. Pastikan ada TLV dengan `Tag == "01"`. Tidak ada → `ErrInvalidFormat`.

### 5.4 `Convert(s, opts)` — langkah eksplisit

1. Validasi `opts`:
   - `opts.Amount <= 0` atau >13 digit → `ErrInvalidAmount`.
   - `opts.Fee < 0` atau >13 digit → `ErrInvalidFee`.
2. `Validate(s)` — propagate error.
3. `tlvs, _ := parseTLV(s[:len(s)-8])` (payload tanpa tag 63 head+value).
4. Cari TLV dengan `Tag == "01"`:
   - `Value == "12"` → `ErrAlreadyDynamic`.
   - Lainnya → set `Value = "12"` (length tetap `2`).
5. Build TLV baru:
   - tag `54`: value = `strconv.Itoa(opts.Amount)`, length = `len(value)`.
   - jika `opts.Fee > 0`:
     - tag `55`: value = `"02"`, length = `2`.
     - tag `56`: value = `strconv.Itoa(opts.Fee)`, length = `len(value)`.
6. Sisipkan TLV baru ke `tlvs` tepat **sebelum** TLV pertama dengan `Tag >= "58"` (countryCode). Bila tidak ditemukan tag ≥ "58", append di akhir.
7. Serialize `tlvs` ke string `payload`.
8. `out := payload + "6304" + CRC16(payload + "6304")`.
9. Return `out`.

### 5.5 Format amount/fee

```go
func formatNumeric(n int) string { return strconv.Itoa(n) }
```

Validasi panjang (`<= 13`) sudah dicek di langkah 1; tidak butuh truncation defensif.

### 5.6 Edge cases

| Kasus | Penanganan |
|---|---|
| Whitespace/newline di ujung input | `strings.TrimSpace` di CLI sebelum panggil `qris` |
| QRIS sudah dinamis (tag 01 = "12") | `ErrAlreadyDynamic`, exit 1 |
| `Amount <= 0` atau >13 digit | CLI: exit 2 (usage error). Library: `ErrInvalidAmount` |
| `Fee < 0` atau >13 digit | CLI: exit 2 (usage error). Library: `ErrInvalidFee` |
| CRC huruf kecil di input | normalisasi `strings.ToUpper` saat compare |
| Tag 01 tidak ditemukan | `ErrInvalidFormat` |
| Multiple input source (`-i` + `-f`) | error usage, exit 2 |
| File `-f` tidak ditemukan / unreadable | error usage, exit 2 |
| Tidak ada tag ≥ "58" sebelum 63 | tag 54/55/56 di-append di akhir payload (sebelum 63 di-rebuild) |

---

## 6. Testing strategy

Stdlib `testing` saja, tabel-driven.

### 6.1 `qris/crc16_test.go`

- `CRC16("123456789")` → `"29B1"`.
- `CRC16("")` → `"FFFF"`.
- CRC dari QRIS fixture cocok dengan tag 63 di fixture.

### 6.2 `qris/tlv_test.go`

- Parse TLV valid → urutan & isi sesuai.
- Truncated → `ErrInvalidFormat`.
- LL mismatch (length > sisa string) → error.
- Non-digit LL → error.

### 6.3 `qris/parser_test.go`

- QRIS statis fixture → `Method == MethodStatic`, `MerchantName`/`MerchantCity` benar, `CRC` non-empty.
- QRIS dinamis fixture → `Method == MethodDynamic`.

### 6.4 `qris/validator_test.go`

- QRIS valid → `nil`.
- 1 char CRC dimodifikasi → `ErrCRCMismatch`.
- Dipotong di tengah → `ErrInvalidFormat`.

### 6.5 `qris/converter_test.go` (paling kritikal)

- Convert statis + amount → output dinamis:
  - Tag `01` = `"12"`.
  - Tag `54` ada dengan value = amount.
  - `Validate(output) == nil`.
  - `Parse(output).Method == MethodDynamic`.
- Convert + amount + fee → tag `55="02"` & `56=fee` ada di posisi benar (sebelum 58).
- Convert QRIS dinamis → `ErrAlreadyDynamic`.
- Amount 0 / negatif / >13 digit → `ErrInvalidAmount`.
- Fee negatif / >13 digit → `ErrInvalidFee`.

### 6.6 `cmd/qris/main_test.go` (smoke test)

Build binary via `go build` di test setup, lalu `exec.Command`:
- `-i ... -a 50000` → exit 0, stdout berisi QRIS dinamis valid.
- `--help` → keluaran mengandung `Usage`.
- Tanpa input source → exit 2.

### 6.7 Fixture

`qris/testdata/`:
- `static_valid.txt` — QRIS statis valid yang dikenal (di-generate sekali dari versi TS atau dari sumber publik).
- `dynamic_valid.txt` — QRIS dinamis valid.

---

## 7. Struktur file final

```
go-qris-dinamis/
├── cmd/qris/
│   └── main.go
├── qris/
│   ├── tlv.go
│   ├── parser.go
│   ├── validator.go
│   ├── converter.go
│   ├── crc16.go
│   ├── errors.go
│   ├── tlv_test.go
│   ├── parser_test.go
│   ├── validator_test.go
│   ├── converter_test.go
│   ├── crc16_test.go
│   └── testdata/
│       ├── static_valid.txt
│       └── dynamic_valid.txt
├── go.mod                  # module github.com/rizkirmdhn/go-qris-dinamis, go 1.22+
├── LICENSE                 # MIT (mirror upstream)
├── .gitignore              # /qris (binary), *.test, .DS_Store
└── README.md               # install, usage CLI, contoh library, dev notes
```

---

## 8. Versioning & distribusi

- Versi di-inject saat build: `go build -ldflags "-X main.version=v0.1.0" ./cmd/qris`. Default `"dev"`.
- Distribusi via `go install`: `go install github.com/rizkirmdhn/go-qris-dinamis/cmd/qris@latest` → binary `qris` di `$GOBIN`.
- CI/release otomatis (GitHub Actions, GoReleaser) **di luar scope MVP**; bisa ditambahkan di iterasi berikutnya.

---

## 9. Open questions

Tidak ada. Semua keputusan kunci sudah dikonfirmasi user dalam sesi brainstorming.
