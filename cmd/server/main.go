package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/talife/formaljudge/pkg/compiler"
	"github.com/talife/formaljudge/pkg/models"
	"github.com/talife/formaljudge/pkg/verifier"
)

var (
	serverPubKey  ed25519.PublicKey
	serverPrivKey ed25519.PrivateKey

	policyRegistry = make(map[string]string)
)

type PolicyRequest struct {
	PolicyID     string          `json:"policy_id"`
	CompiledMath json.RawMessage `json:"compiled_math"`
}

type VerifyRequest struct {
	Spec            string          `json:"spec"`
	Trace           *models.Trace   `json:"trace"`
	LlmMockResponse json.RawMessage `json:"llm_mock_response,omitempty"`
	PolicyID        string          `json:"policy_id,omitempty"`
}

func init() {
	// Generate ephemeral Ed25519 keys for the POC on startup
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate ed25519 keys: %v", err)
	}
	serverPubKey = pub
	serverPrivKey = priv
	log.Printf("🔐 Cryptographic Module Initialized. Public Key: %s", hex.EncodeToString(serverPubKey))
}

func main() {
	// Setup the HTTP route
	http.HandleFunc("/v1/verify", verifyHandler)
	http.HandleFunc("/v1/policies", policyHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	//nolint:gosec // G706: The port environment variable is trusted and safe to log.
	log.Printf("FormalJudge Guardrail API starting on port %s...", port)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
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

	// Require a trace
	if req.Trace == nil {
		http.Error(w, "Missing 'trace' in payload", http.StatusBadRequest)
		return
	}

	// Require EITHER a dynamic Spec OR a pre-compiled PolicyID
	if req.Spec == "" && req.PolicyID == "" {
		http.Error(w, "Must provide either 'spec' or 'policy_id' in payload", http.StatusBadRequest)
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
	if req.PolicyID != "" {
		// AOT PATH: Use pre-compiled math from the registry
		registeredMath, exists := policyRegistry[req.PolicyID]
		if !exists {
			http.Error(w, fmt.Sprintf("Policy ID '%s' not found", req.PolicyID), http.StatusNotFound)
			return
		}
		mockStr = registeredMath
	} else if len(req.LlmMockResponse) > 0 {
		// FALLBACK PATH: Use the mock response provided in the request
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
	if verdict.Status == models.VerdictSafe {
		// 1. Serialize the trace (automatically includes ToolName, RawCode, and SymbolicMapping!)
		traceBytes, _ := json.Marshal(req.Trace)

		// 2. Identify which policy rule was evaluated (AOT Policy ID vs Dynamic Spec)
		policyIdentifier := req.Spec
		if policyIdentifier == "" {
			policyIdentifier = "AOT_POLICY:" + req.PolicyID
		}

		// 3. Create the exact payload string: Policy/Spec + Trace + Dafny Math
		payload := fmt.Sprintf("%s|%s|%s", policyIdentifier, string(traceBytes), verdict.DafnyOutput)

		// 4. Hash the payload with SHA-256
		hash := sha256.Sum256([]byte(payload))

		// 5. Sign the hash with our private key
		signature := ed25519.Sign(serverPrivKey, hash[:])

		// 6. Attach the hex-encoded strings to the response
		verdict.ReceiptSignature = hex.EncodeToString(signature)
		verdict.ReceiptPublicKey = hex.EncodeToString(serverPubKey)
	}

	// 4. Return the Verdict as JSON
	w.Header().Set("Content-Type", "application/json")

	switch verdict.Status {
	case models.VerdictUnsafe:
		w.WriteHeader(http.StatusForbidden)
	case models.VerdictError:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusOK)
	}

	// Check the error on the JSON encoder (fixes errcheck)
	if err := json.NewEncoder(w).Encode(verdict); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// policyHandler allows security engineers to pre-compile and register policies
func policyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.PolicyID == "" || len(req.CompiledMath) == 0 {
		http.Error(w, "Missing 'policy_id' or 'compiled_math'", http.StatusBadRequest)
		return
	}

	// Save it to our registry
	policyRegistry[req.PolicyID] = string(req.CompiledMath)
	log.Printf("🛡️ Policy Registered: %s", req.PolicyID)

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status": "success", "message": "Policy registered successfully"}`)); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
