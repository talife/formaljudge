package models

// TraceStep represents a single interaction or step in the agent execution trace.
type TraceStep struct {
	StepNumber  int    `json:"step_number"`
	Role        string `json:"role"`        // e.g., "thought", "action", "observation", "system"
	Description string `json:"description"` // Natural language content of the step
}

// Trace represents the full sequence of actions taken by the agent.
type Trace struct {
	AgentID      string            `json:"agent_id"`
	InitialState map[string]string `json:"initial_state"`
	Steps        []TraceStep       `json:"steps"`
}

// VerdictType represents the result of the formal verification.
type VerdictType string

const (
	VerdictSafe   VerdictType = "SAFE"
	VerdictUnsafe VerdictType = "UNSAFE"
	VerdictError  VerdictType = "ERROR"
)

// Verdict contains the final assessment of the verification pipeline.
type Verdict struct {
	Status             VerdictType `json:"status"`
	Message            string      `json:"message"`
	FailedInvariant    string      `json:"failed_invariant,omitempty"`
	DafnyOutput        string      `json:"dafny_output,omitempty"`
	GeneratedDafnyFile string      `json:"generated_dafny_file,omitempty"`
	ReceiptSignature   string      `json:"receipt_signature,omitempty"`
	ReceiptPublicKey   string      `json:"receipt_public_key,omitempty"`
}
