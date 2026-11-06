package apperr

import "testing"

func TestConstructors(t *testing.T) {
	tests := []struct {
		name    string
		err     *Error
		status  int
		code    string
		message string
	}{
		{"validation", Validation("bad"), 400, CodeValidation, "bad"},
		{"not found", NotFound("missing"), 404, CodeNotFound, "missing"},
		{"invalid state", InvalidState("state"), 409, CodeInvalidState, "state"},
		{"duplicate", DuplicateAction("dup"), 409, CodeDuplicateAction, "dup"},
		{"business", BusinessRule("rule"), 422, CodeBusinessRule, "rule"},
		{"mismatch", HCPHospitalMismatch("mismatch"), 422, CodeHCPHospitalMismatch, "mismatch"},
		{"time order", TimeOrderInvalid("time"), 422, CodeTimeOrderInvalid, "time"},
		{"time override", TimeOverrideDisabled("override"), 422, CodeTimeOverrideDisabled, "override"},
		{"material", MaterialInconsistent("material"), 422, CodeMaterialInconsistent, "material"},
		{"internal", Internal("internal"), 500, CodeInternal, "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.status || tt.err.Code != tt.code || tt.err.Message != tt.message {
				t.Fatalf("unexpected error: %+v", tt.err)
			}
		})
	}
}

func TestError(t *testing.T) {
	e := NotFound("拜访不存在")
	if got := e.Error(); got != "NOT_FOUND: 拜访不存在" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestValidationFields(t *testing.T) {
	e := ValidationFields("bad", map[string]string{"field": "required"})
	if e.Status != 400 || e.Fields["field"] != "required" {
		t.Fatalf("unexpected error: %+v", e)
	}
}
