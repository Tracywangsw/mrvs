package timeutil

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	at := time.Date(2026, 11, 5, 12, 30, 0, 0, time.FixedZone("CST", 8*3600))
	if got := Format(at); got != "2026-11-05T04:30:00Z" {
		t.Fatalf("Format() = %s", got)
	}
}

func TestFormatPtr(t *testing.T) {
	if got := FormatPtr(nil); got != nil {
		t.Fatalf("FormatPtr(nil) = %v", got)
	}
	at := time.Date(2026, 11, 5, 4, 30, 0, 0, time.UTC)
	got := FormatPtr(&at)
	if got == nil || *got != "2026-11-05T04:30:00Z" {
		t.Fatalf("FormatPtr() = %v", got)
	}
}

func TestParseRFC3339(t *testing.T) {
	got, err := ParseRFC3339("2026-11-05T12:30:00+08:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Location() != time.UTC || got.Hour() != 4 {
		t.Fatalf("ParseRFC3339() = %v", got)
	}
	if _, err := ParseRFC3339("2026-11-05"); err == nil {
		t.Fatal("expected error for date-only input")
	}
}

func TestMonthRange(t *testing.T) {
	start, end, err := MonthRange("2026-11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC); !end.Equal(want) {
		t.Fatalf("end = %v, want %v", end, want)
	}

	for _, invalid := range []string{"2026-13", "202611", "2026-1", "", "2026-00"} {
		if _, _, err := MonthRange(invalid); err == nil {
			t.Fatalf("MonthRange(%q) should fail", invalid)
		}
	}
}

func TestDateRange(t *testing.T) {
	start, end, err := DateRange("2026-11-01", "2026-11-05")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, 11, 5, 16, 0, 0, 0, time.UTC); !end.Equal(want) {
		t.Fatalf("end = %v, want %v", end, want)
	}

	if _, _, err := DateRange("2026-11-05", "2026-11-01"); err == nil {
		t.Fatal("expected error when date_to before date_from")
	}
	if _, _, err := DateRange("2026/11/01", "2026-11-05"); err == nil {
		t.Fatal("expected error for invalid date format")
	}
}

func TestParseDateStartAndEnd(t *testing.T) {
	start, err := ParseDateStart("2026-11-05")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := time.Date(2026, 11, 4, 16, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
	end, err := ParseDateEnd("2026-11-05")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := time.Date(2026, 11, 5, 16, 0, 0, 0, time.UTC); !end.Equal(want) {
		t.Fatalf("end = %v, want %v", end, want)
	}
}

func TestShanghaiLocation(t *testing.T) {
	loc := ShanghaiLocation()
	_, offset := time.Now().In(loc).Zone()
	if offset != 8*3600 {
		t.Fatalf("offset = %d, want %d", offset, 8*3600)
	}
}
