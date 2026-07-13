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
	// 1. System Execution: Set a timeout for the SMT solver to prevent hangs
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 2. Run 'dafny verify <file>'
	cmd := exec.CommandContext(ctx, v.DafnyPath, "verify", dfyFilePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// 3. Catch critical OS errors (e.g., Dafny binary not found)
	if err != nil && stdoutStr == "" && stderrStr == "" {
		verdict := &models.Verdict{
			Status:             models.VerdictError,
			Message:            fmt.Sprintf("failed to execute dafny binary: %v", err),
			GeneratedDafnyFile: dfyFilePath,
		}
		return verdict, nil
	}

	// 4. Delegate the complex text parsing to the helper function
	verdict := v.AnalyzeOutput(stdoutStr, stderrStr)
	verdict.GeneratedDafnyFile = dfyFilePath

	return verdict, nil
}

// AnalyzeOutput parses the standard output and standard error from Dafny to determine the verdict.
func (v *DafnyVerifier) AnalyzeOutput(stdoutStr, stderrStr string) *models.Verdict {
	verdict := &models.Verdict{
		DafnyOutput: stdoutStr + "\n" + stderrStr,
	}
	combinedOutput := stdoutStr + " " + stderrStr

	// Check if Dafny even managed to compile and reach the verification summary
	if !strings.Contains(combinedOutput, "verified") {
		verdict.Status = models.VerdictError
		verdict.Message = "Formal specification compilation failed:\n" + combinedOutput
		return verdict
	}

	// Check for verification successes
	if strings.Contains(combinedOutput, "0 errors") || strings.Contains(combinedOutput, "0 verification errors") {
		verdict.Status = models.VerdictSafe
		verdict.Message = "Formal verification succeeded. All safety invariants are satisfied mathematically."
		return verdict
	}

	// If we got here, an assertion failed
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
	return verdict
}
