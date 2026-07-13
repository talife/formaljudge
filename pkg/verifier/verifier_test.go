package verifier

import (
	"testing"

	"github.com/talife/formaljudge/pkg/models"
)

func TestAnalyzeOutput(t *testing.T) {
	vf := NewDafnyVerifier("")

	// Table-driven test cases
	tests := []struct {
		name           string
		stdout         string
		stderr         string
		expectedStatus models.VerdictType
	}{
		{
			name:           "Safe Trace",
			stdout:         "Dafny program verifier finished with 1 verified, 0 errors",
			stderr:         "",
			expectedStatus: models.VerdictSafe,
		},
		{
			name:           "Unsafe Trace with Assertion Violation",
			stdout:         "Error: assertion might not hold\nbank_verification.dfy(22,9): Related location: assertion violation\nDafny program verifier finished with 0 verified, 1 errors",
			stderr:         "",
			expectedStatus: models.VerdictUnsafe,
		},
		{
			name:           "Syntax Error (Compilation Failure)",
			stdout:         "bank_verification.dfy(5,0): Error: a datatype must have at least one constructor",
			stderr:         "",
			expectedStatus: models.VerdictError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := vf.AnalyzeOutput(tt.stdout, tt.stderr)
			if verdict.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, verdict.Status)
			}
		})
	}
}
