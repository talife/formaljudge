package compiler

import (
	"context"
	"os"
	"testing"

	"github.com/talife/formaljudge/pkg/models"
)

func TestCompilerWithMockLLMResponse(t *testing.T) {
	// 1. The raw mock JSON string
	mockJSON := `{
		"state_definition": "datatype State = State(balance: int)",
		"actions_definition": "datatype Action = Transfer",
		"transition_definition": "function next(s: State, a: Action): State { s }",
		"safety_invariant": "predicate SafetyInvariant(s: State) { s.balance >= 0 }",
		"concrete_trace": "[Transfer]",
		"initial_state_value": "State(100)"
	}`

	// 2. Run Compiler
	comp := NewDafnyCompiler("") // No API key needed
	outPath := "test_output.dfy"
	defer os.Remove(outPath)

	trace := &models.Trace{AgentID: "test"}

	// Pass the mockJSON string directly!
	_, err := comp.Compile(context.Background(), "Rule 1", trace, outPath, mockJSON)

	if err != nil {
		t.Fatalf("Expected successful compilation, got error: %v", err)
	}

	// 3. Verify file was created
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("Expected output file %s to be created", outPath)
	}
}
