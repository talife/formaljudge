package compiler

import (
	"context"
	"os"
	"testing"

	"github.com/talife/formaljudge/pkg/models"
)

func TestCompilerWithMockLLMResponse(t *testing.T) {
	// 1. Create a dummy JSON file to act as the LLM response
	mockJSON := `{
		"state_definition": "datatype State = State(balance: int)",
		"actions_definition": "datatype Action = Transfer",
		"transition_definition": "function next(s: State, a: Action): State { s }",
		"safety_invariant": "predicate SafetyInvariant(s: State) { s.balance >= 0 }",
		"concrete_trace": "[Transfer]",
		"initial_state_value": "State(100)"
	}`
	tmpFile, _ := os.CreateTemp("", "mock_llm_*.json")
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte(mockJSON)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	// 2. Run Compiler
	comp := NewDafnyCompiler("") // No API key needed
	outPath := "test_output.dfy"
	defer os.Remove(outPath)

	trace := &models.Trace{AgentID: "test"}
	_, err := comp.Compile(context.Background(), "Rule 1", trace, outPath, tmpFile.Name())

	if err != nil {
		t.Fatalf("Expected successful compilation, got error: %v", err)
	}

	// 3. Verify file was created
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("Expected output file %s to be created", outPath)
	}
}
