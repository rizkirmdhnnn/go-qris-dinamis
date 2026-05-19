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
