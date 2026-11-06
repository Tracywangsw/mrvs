package database

import (
	"testing"
)

func TestCheckConstraintDefinitions(t *testing.T) {
	if len(checkConstraints) == 0 {
		t.Fatal("check constraints must not be empty")
	}
	seen := map[string]bool{}
	for _, c := range checkConstraints {
		if c.Table == "" || c.Name == "" || c.Expr == "" {
			t.Fatalf("invalid constraint definition: %+v", c)
		}
		if seen[c.Name] {
			t.Fatalf("duplicate constraint name: %s", c.Name)
		}
		seen[c.Name] = true
	}
}

func TestSeedDataDefinitions(t *testing.T) {
	if len(seedMRs) == 0 || len(seedHospitals) == 0 || len(seedHCPs) == 0 || len(seedProducts) == 0 {
		t.Fatal("seed data must not be empty")
	}
	hospitalNames := map[string]bool{}
	for _, h := range seedHospitals {
		hospitalNames[h.Name] = true
	}
	for _, hcp := range seedHCPs {
		if !hospitalNames[hcp.HospitalName] {
			t.Fatalf("hcp %s references unknown hospital %s", hcp.Name, hcp.HospitalName)
		}
	}
}
