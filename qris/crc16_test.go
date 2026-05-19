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
