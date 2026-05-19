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
