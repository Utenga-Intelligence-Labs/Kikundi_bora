package services

import "testing"

func TestNormalizeTanzanianPhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0710123456", "+255710123456"},
		{"710123456", "+255710123456"},
		{"+255710123456", "+255710123456"},
		{"255710123456", "+255710123456"},
		{"00255710123456", "+255710123456"},
		{" 0710-123-456 ", "+255710123456"},
		{"0655123456", "+255655123456"},
	}
	for _, tc := range cases {
		got, err := NormalizeTanzanianPhone(tc.in)
		if err != nil {
			t.Errorf("NormalizeTanzanianPhone(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizeTanzanianPhone(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	invalid := []string{"", "123", "0000000000", "+255221234567", "+14445556666", "071012345", "07101234567", "abc"}
	for _, in := range invalid {
		if got, err := NormalizeTanzanianPhone(in); err == nil {
			t.Errorf("NormalizeTanzanianPhone(%q) = %q, want error", in, got)
		}
	}
}
