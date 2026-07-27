package compiler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/talife/formaljudge/pkg/models"
)

// RemediationEngine generates generic, domain-agnostic self-healing instructions.
type RemediationEngine struct {
	llm LLMProvider
}

// NewRemediationEngine constructs the remediation generator.
func NewRemediationEngine(provider LLMProvider) *RemediationEngine {
	return &RemediationEngine{llm: provider}
}

// Generate creates structured remediation for ANY invariant failure across any domain.
func (r *RemediationEngine) Generate(ctx context.Context, spec string, trace *models.Trace, verdict *models.Verdict) *models.SelfCorrection {
	if verdict.Status != models.VerdictUnsafe {
		return nil
	}

	errLocation := verdict.FailedInvariant
	if errLocation == "" {
		errLocation = "Safety Invariant Violation"
	}

	// 1. Generic Structural Fallback (Embeds the actual policy text so the agent knows what rule failed)
	explanationStr := fmt.Sprintf("Formal SMT verification failed. Verification error: %s.", errLocation)
	promptStr := fmt.Sprintf("GUARDRAIL REJECTION: Your proposed execution trace failed formal verification (%s). Re-plan your tool actions to satisfy all safety constraints.", errLocation)

	if spec != "" {
		explanationStr = fmt.Sprintf("Formal SMT verification failed against safety policy: \"%s\". SMT Engine error: %s.", spec, errLocation)
		promptStr = fmt.Sprintf("GUARDRAIL REJECTION: Your proposed tool execution trace violated safety policy: \"%s\". SMT Proof Error: [%s]. Re-plan your tool actions to comply with this policy.", spec, errLocation)
	}

	fallback := &models.SelfCorrection{
		ConstraintViolated: errLocation,
		Explanation:        explanationStr,
		RequiredFix:        "Update your proposed tool calls or parameters so that all state transitions satisfy the stated policy.",
		SuggestedPrompt:    promptStr,
	}

	// If no LLM provider is active or if spec is empty, return the enriched fallback
	if r.llm == nil || spec == "" {
		return fallback
	}

	// 2. Dynamic SMT Error Translation (Translates formal math error into contextual feedback)
	traceBytes, _ := json.MarshalIndent(trace, "", "  ")

	prompt := fmt.Sprintf(`You are a Formal Methods Security Expert.
An automated agent execution trace failed formal SMT verification.

SAFETY SPECIFICATION:
%s

FAILED SMT INVARIANT / DAFNY OUTPUT:
%s

AGENT EXECUTION TRACE (JSON):
%s

Task: Analyze why the formal proof failed and explain how the agent should adjust its proposed tool calls.
Output ONLY a raw JSON object matching this schema:
{
  "constraint_violated": "<the failed predicate or rule>",
  "explanation": "<short natural language explanation of the safety gap>",
  "required_fix": "<concrete adjustment required in the tool trace>",
  "suggested_prompt": "<direct instruction to inject back into the agent context window>"
}`, spec, verdict.FailedInvariant, string(traceBytes))

	respText, err := r.llm.Generate(ctx, prompt)
	if err != nil {
		return fallback
	}

	cleanJSON := cleanMarkdownFences(respText)
	var correction models.SelfCorrection
	if err := json.Unmarshal([]byte(cleanJSON), &correction); err != nil {
		return fallback
	}

	return &correction
}
