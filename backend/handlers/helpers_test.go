package handlers

import (
	"testing"
)

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"test%", `test\%`},
		{"test_", `test\_`},
		{"test\\", `test\\`},
		{"%_\\", `\%\_\\`},
	}

	for _, tt := range tests {
		got := escapeLike(tt.input)
		if got != tt.want {
			t.Errorf("escapeLike(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLikePattern(t *testing.T) {
	result := likePattern("Juma")
	if result != "%juma%" {
		t.Errorf("likePattern(Juma) = %q, want %%juma%%", result)
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0.00"},
		{1000, "1000.00"},
		{1234.5, "1234.50"},
		{1234.567, "1234.57"},
	}

	for _, tt := range tests {
		got := formatMoney(tt.input)
		if got != tt.want {
			t.Errorf("formatMoney(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		data []byte
		want string
	}{
		{[]byte("\xff\xd8\xff"), ".jpg"},
		{[]byte("\x89PNG\r\n\x1a\n"), ".png"},
		{[]byte("GIF89a"), ".gif"},
		{[]byte("GIF87a"), ".gif"},
		{[]byte("notanimage"), ""},
	}

	for _, tt := range tests {
		got := detectContentType(tt.data)
		if got != tt.want {
			t.Errorf("detectContentType(%v) = %q, want %q", tt.data, got, tt.want)
		}
	}
}

func TestSha256Hex(t *testing.T) {
	result := sha256Hex("hello")
	// SHA256 of "hello" is 64 hex chars
	if len(result) != 64 {
		t.Errorf("sha256Hex should return 64 hex chars, got %d", len(result))
	}
}

func TestVerifyDocMagicBytes(t *testing.T) {
	tests := []struct {
		data []byte
		ext  string
		want bool
	}{
		{[]byte{0x25, 0x50, 0x44, 0x46}, ".pdf", true},
		{[]byte{0x50, 0x4B, 0x03, 0x04}, ".docx", true},
		{[]byte{0x50, 0x4B, 0x03, 0x04}, ".xlsx", true},
		{[]byte("garbage"), ".pdf", false},
		{[]byte("some,text"), ".csv", true}, // CSV has no magic bytes
		{[]byte("random"), ".unknown", true}, // unknown type is allowed
	}

	for _, tt := range tests {
		got := verifyDocMagicBytes(tt.data, tt.ext)
		if got != tt.want {
			t.Errorf("verifyDocMagicBytes(%v, %q) = %v, want %v", tt.data, tt.ext, got, tt.want)
		}
	}
}
