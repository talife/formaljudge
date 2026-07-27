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
	"strings"
	"time"

	"github.com/talife/formaljudge/pkg/compiler"
	"github.com/talife/formaljudge/pkg/models"
	"github.com/talife/formaljudge/pkg/verifier"
)

var (
	serverPubKey            ed25519.PublicKey
	serverPrivKey           ed25519.PrivateKey
	policyRegistry          = make(map[string]string)
	globalCompiler          *compiler.Compiler
	globalRemediationEngine *compiler.RemediationEngine
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
	log.Printf("🔒 Cryptographic Module Initialized. Public Key: %s", hex.EncodeToString(serverPubKey))
}

func main() {
	// 1. Initialize the Model-Agnostic Compiler ONCE at startup
	cfg := compiler.LoadConfig()
	provider, err := compiler.NewProvider(cfg)
	if err != nil {
		log.Printf("  Warning: LLM provider initialization failed: %v (Mock/AOT paths will still work)", err)
	}

	globalCompiler = compiler.New(provider)
	globalRemediationEngine = compiler.NewRemediationEngine(provider) // <-- ADD THIS LINE

	log.Printf("  FormalJudge Engine Initialized | Provider: [%s] | Model: [%s]", strings.ToUpper(cfg.Provider), cfg.Model)

	// 2. Setup HTTP routes
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	if req.Trace == nil {
		http.Error(w, "Missing 'trace' in payload", http.StatusBadRequest)
		return
	}

	if req.Spec == "" && req.PolicyID == "" {
		http.Error(w, "Must provide either 'spec' or 'policy_id' in payload", http.StatusBadRequest)
		return
	}

	tmpFile, err := os.CreateTemp("", "verification_*.dfy")
	if err != nil {
		http.Error(w, "Internal server error creating temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	mockStr := ""
	if req.PolicyID != "" {
		registeredMath, exists := policyRegistry[req.PolicyID]
		if !exists {
			http.Error(w, fmt.Sprintf("Policy ID '%s' not found", req.PolicyID), http.StatusNotFound)
			return
		}
		mockStr = registeredMath
	} else if len(req.LlmMockResponse) > 0 {
		mockStr = string(req.LlmMockResponse)
	}

	// Use our initialized globalCompiler!
	_, err = globalCompiler.Compile(context.Background(), req.Spec, req.Trace, tmpFile.Name(), mockStr)
	if err != nil {
		log.Printf("Compiler error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to compile specification: %v", err), http.StatusInternalServerError)
		return
	}

	vf := verifier.NewDafnyVerifier("dafny")
	verdict, err := vf.Verify(context.Background(), tmpFile.Name())
	if err != nil {
		log.Printf("Verifier runner error: %v", err)
		http.Error(w, "Verification engine failure", http.StatusInternalServerError)
		return
	}

	if verdict.Status == models.VerdictSafe {
		traceBytes, _ := json.Marshal(req.Trace)
		policyIdentifier := req.Spec
		if policyIdentifier == "" {
			policyIdentifier = "AOT_POLICY:" + req.PolicyID
		}

		payload := fmt.Sprintf("%s|%s|%s", policyIdentifier, string(traceBytes), verdict.DafnyOutput)
		hash := sha256.Sum256([]byte(payload))
		signature := ed25519.Sign(serverPrivKey, hash[:])

		verdict.ReceiptSignature = hex.EncodeToString(signature)
		verdict.ReceiptPublicKey = hex.EncodeToString(serverPubKey)
	}

	w.Header().Set("Content-Type", "application/json")
	switch verdict.Status {
	case models.VerdictUnsafe:
		verdict.SelfCorrection = globalRemediationEngine.Generate(
			r.Context(),
			req.Spec,
			req.Trace,
			verdict,
		)
		w.WriteHeader(http.StatusForbidden)
	case models.VerdictError:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(verdict); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

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

	policyRegistry[req.PolicyID] = string(req.CompiledMath)
	log.Printf("  Policy Registered: %s", req.PolicyID)

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status": "success", "message": "Policy registered successfully"}`)); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
