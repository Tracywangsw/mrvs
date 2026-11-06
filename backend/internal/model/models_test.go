package model

import "testing"

func TestIsValidStatus(t *testing.T) {
	valid := []string{StatusPlanned, StatusCheckedIn, StatusCheckedOut, StatusReported, StatusCancelled}
	for _, s := range valid {
		if !IsValidStatus(s) {
			t.Fatalf("IsValidStatus(%q) = false", s)
		}
	}
	for _, s := range []string{"", "unknown", "PLANNED", "checkin"} {
		if IsValidStatus(s) {
			t.Fatalf("IsValidStatus(%q) = true", s)
		}
	}
}

type tableNamer interface{ TableName() string }

func TestTableNames(t *testing.T) {
	tests := []struct {
		model tableNamer
		want  string
	}{
		{MR{}, "mr"},
		{Hospital{}, "hospital"},
		{HCP{}, "hcp"},
		{Product{}, "product"},
		{Visit{}, "visit"},
		{VisitMaterial{}, "visit_material"},
	}
	for _, tt := range tests {
		if got := tt.model.TableName(); got != tt.want {
			t.Fatalf("%T.TableName() = %q, want %q", tt.model, got, tt.want)
		}
	}
}
