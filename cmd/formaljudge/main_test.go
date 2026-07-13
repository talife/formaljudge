package main

import (
	"context"
	"os"
	"testing"

	"github.com/talife/formaljudge/pkg/compiler"
	"github.com/talife/formaljudge/pkg/models"
	"github.com/talife/formaljudge/pkg/verifier"
)

// This test requires Dafny to be installed on the system.
func TestEndToEndVerification(t *testing.T) {
	// 1. Setup mock LLM response for a safe trace
	safeLLMJSON := `{
		"state_definition": "datatype State = State(balance: int)",
		"actions_definition": "datatype Action = Transfer",
		"transition_definition": "function next(s: State, a: Action): State { s }",
		"safety_invariant": "predicate SafetyInvariant(s: State) { s.balance >= 0 }",
		"concrete_trace": "[]",
		"initial_state_value": "State(100)"
	}`

	dfyFile := "e2e_test.dfy"
	defer os.Remove(dfyFile)

	// 2. Compile
	comp := compiler.NewDafnyCompiler("")
	trace := &models.Trace{AgentID: "test"}

	// Pass the safeLLMJSON string directly!
	_, err := comp.Compile(context.Background(), "Spec", trace, dfyFile, safeLLMJSON)
	if err != nil {
		t.Fatalf("Compiler failed: %v", err)
	}

	// 3. Verify (Requires actual Dafny binary in PATH)
	vf := verifier.NewDafnyVerifier("dafny")
	verdict, err := vf.Verify(context.Background(), dfyFile)

	if err != nil {
		t.Fatalf("Verifier execution failed: %v", err)
	}

	if verdict.Status != models.VerdictSafe {
		t.Errorf("Expected SAFE verdict, got %s. Message: %s", verdict.Status, verdict.Message)
	}
}
