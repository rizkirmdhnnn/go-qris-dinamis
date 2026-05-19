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
