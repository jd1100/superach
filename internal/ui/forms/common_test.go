package forms

import "testing"

func TestParseDollars(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"1.00", 100, false},
		{"0.99", 99, false},
		{"0", 0, false},
		{"0.5", 50, false},
		{"-5.99", -599, false},
		{"$1,234.56", 123456, false},
		{"  $1,000  ", 100000, false},
		{"+42.00", 4200, false},
		{"", 0, true},
		{"abc", 0, true},
		{"1.234", 0, true},
		{"1..0", 0, true},
		{"$", 0, true},
		{"1.0.0", 0, true},
		{"5e2", 0, true},
	}
	for _, c := range cases {
		got, err := parseDollars(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseDollars(%q): expected error, got %d", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDollars(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseDollars(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestValidateYYMMDD(t *testing.T) {
	good := []string{"260424", "250101", "991231"}
	for _, s := range good {
		if err := validateYYMMDD(s); err != nil {
			t.Errorf("validateYYMMDD(%q): unexpected error %v", s, err)
		}
	}
	bad := []string{"", "12345", "1234567", "26abcd", "261301", "260001", "260100", "260132"}
	for _, s := range bad {
		if err := validateYYMMDD(s); err == nil {
			t.Errorf("validateYYMMDD(%q): expected error", s)
		}
	}
}

func TestAllDigits(t *testing.T) {
	if !allDigits("12345") {
		t.Errorf("allDigits('12345') should be true")
	}
	if allDigits("1a3") {
		t.Errorf("allDigits('1a3') should be false")
	}
	if !allDigits("") {
		t.Errorf("allDigits('') should be true (vacuously)")
	}
}
