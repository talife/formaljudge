package verifier

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/talife/formaljudge/pkg/models"
)

// DafnyVerifier runs the Dafny compilation/verification command.
type DafnyVerifier struct {
	DafnyPath string // Path to the dafny binary, defaults to "dafny" in PATH
}

// NewDafnyVerifier initializes the verifier.
func NewDafnyVerifier(dafnyPath string) *DafnyVerifier {
	if dafnyPath == "" {
		dafnyPath = "dafny"
	}
	return &DafnyVerifier{
		DafnyPath: dafnyPath,
	}
}

// Verify runs the formal verification on the generated .dfy file.
func (v *DafnyVerifier) Verify(ctx context.Context, dfyFilePath string) (*models.Verdict, error) {
	// We set a timeout for the SMT solver to prevent hangs
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Modern Dafny uses 'verify', older versions can just compile without generating code
	// We'll run 'dafny verify <file>' which is standard for Dafny 4+
	cmd := exec.CommandContext(ctx, v.DafnyPath, "verify", dfyFilePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	verdict := &models.Verdict{
		GeneratedDafnyFile: dfyFilePath,
		DafnyOutput:        stdoutStr + "\n" + stderrStr,
	}

	// If the binary is missing or doesn't execute at all
	if err != nil && stdoutStr == "" && stderrStr == "" {
		verdict.Status = models.VerdictError
		verdict.Message = fmt.Sprintf("failed to execute dafny binary: %v", err)
		return verdict, nil
	}

	// Analyze outputs
	// Dafny outputs typically look like:
	// "Dafny program verifier finished with 0 verified, 0 errors" (or "1 verified, 0 errors")
	// If there are errors, it states: "finished with X verified, Y errors" or shows assertion violations.
	combinedOutput := stdoutStr + " " + stderrStr

	// Check if Dafny even managed to compile and reach the verification summary
	if !strings.Contains(combinedOutput, "verified") {
		// If it doesn't contain "verified", it means compilation/syntax failed
		verdict.Status = models.VerdictError
		verdict.Message = "Formal specification compilation failed:\n" + combinedOutput
		return verdict, nil
	}

	// Check for verification errors (SMT solver failures)
	// We search for patterns like "0 errors", "0 verification errors" or "verified, 0 errors"
	if strings.Contains(combinedOutput, "0 errors") || strings.Contains(combinedOutput, "0 verification errors") {
		verdict.Status = models.VerdictSafe
		verdict.Message = "Formal verification succeeded. All safety invariants are satisfied mathematically."
		return verdict, nil
	}

	// If we got here and there are errors/assertions that failed:
	verdict.Status = models.VerdictUnsafe
	verdict.Message = "Safety invariant violation detected. The agent trace is unsafe."

	// Try to extract failed invariant info
	lines := strings.Split(combinedOutput, "\n")
	for _, line := range lines {
		if strings.Contains(line, "assertion violation") || strings.Contains(line, "Could not prove") {
			verdict.FailedInvariant = strings.TrimSpace(line)
			break
		}
	}

	return verdict, nil
}
