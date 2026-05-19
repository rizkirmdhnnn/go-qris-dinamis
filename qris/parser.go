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
