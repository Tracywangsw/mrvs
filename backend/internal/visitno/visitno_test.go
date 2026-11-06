package visitno

import (
	"errors"
	"testing"
	"time"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestGenerate(t *testing.T) {
	now := time.Date(2026, 11, 5, 4, 0, 0, 0, time.UTC)
	no, err := Generate(now, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Valid(no) {
		t.Fatalf("generated visit_no %q is invalid", no)
	}
	if no[:9] != "V20261105" {
		t.Fatalf("date prefix = %q", no[:9])
	}
}

func TestGenerateUnique(t *testing.T) {
	now := time.Now()
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		no, err := Generate(now, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seen[no] {
			t.Fatalf("duplicate visit_no: %s", no)
		}
		seen[no] = true
	}
}

func TestGenerateError(t *testing.T) {
	if _, err := Generate(time.Now(), errReader{}); err == nil {
		t.Fatal("expected error from random source")
	}
}

func TestValid(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"V20261105-482913", true},
		{"V20261105-000000", true},
		{"X20261105-482913", false},
		{"V20261105482913", false},
		{"V20261105-48291", false},
		{"V20261105-4829133", false},
		{"V2026110A-482913", false},
		{"V20261105-48291A", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := Valid(tt.input); got != tt.want {
			t.Fatalf("Valid(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
