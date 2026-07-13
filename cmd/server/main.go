package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/talife/formaljudge/pkg/compiler"
	"github.com/talife/formaljudge/pkg/models"
	"github.com/talife/formaljudge/pkg/verifier"
)

// VerifyRequest represents the expected JSON payload from the AI Agent Orchestrator.
type VerifyRequest struct {
	Spec            string          `json:"spec"`
	Trace           *models.Trace   `json:"trace"`
	LlmMockResponse json.RawMessage `json:"llm_mock_response,omitempty"`
}

func main() {
	// Setup the HTTP route
	http.HandleFunc("/v1/verify", verifyHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("FormalJudge Guardrail API starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse the incoming request
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.Spec == "" || req.Trace == nil {
		http.Error(w, "Missing 'spec' or 'trace' in payload", http.StatusBadRequest)
		return
	}

	// Create a temporary file for the Dafny code
	tmpFile, err := os.CreateTemp("", "verification_*.dfy")
	if err != nil {
		http.Error(w, "Internal server error creating temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close() // Close it so the compiler/verifier can write/read it freely

	// 2. Compile the Trace and Spec
	mockStr := ""
	if len(req.LlmMockResponse) > 0 {
		mockStr = string(req.LlmMockResponse)
	}
	// Note: In a production environment, you might fetch the API key from a secure vault or env var.
	comp := compiler.NewDafnyCompiler(os.Getenv("GEMINI_API_KEY"))
	_, err = comp.Compile(context.Background(), req.Spec, req.Trace, tmpFile.Name(), mockStr)
	if err != nil {
		log.Printf("Compiler error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to compile specification: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Verify the Code
	vf := verifier.NewDafnyVerifier("dafny")
	verdict, err := vf.Verify(context.Background(), tmpFile.Name())
	if err != nil {
		log.Printf("Verifier runner error: %v", err)
		http.Error(w, "Verification engine failure", http.StatusInternalServerError)
		return
	}

	// 4. Return the Verdict as JSON
	w.Header().Set("Content-Type", "application/json")

	// If the status is unsafe, we can return a 403 Forbidden to clearly signal to the orchestrator to block it.
	if verdict.Status == models.VerdictUnsafe {
		w.WriteHeader(http.StatusForbidden)
	} else if verdict.Status == models.VerdictError {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(verdict)
}
