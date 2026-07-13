# ⚖️ FormalJudge: Agentic Security & Formal Verification Guardrail

FormalJudge is a proof-of-concept **Agentic Security Middleware** built in Go. It enforces strict, mathematically proven safety guardrails on autonomous AI agents using a **Neuro-Symbolic architecture**.

Instead of relying on LLMs to "vibe check" security, or using static analysis tools (like Rego/OPA) that miss temporal context, FormalJudge uses an LLM to abstract an agent's execution trace into a strict State Machine, and then uses an SMT Solver (**Dafny**) to mathematically prove that the agent never entered an unsafe state.

## 🚀 Why Formal Verification for AI Agents?

Standard Policy-as-Code (e.g., Rego, Checkov) is designed for static snapshots (like a Terraform plan). However, autonomous agents act sequentially across multiple tools (Terraform, AWS CLI, Bash).

FormalJudge catches what static analysis misses:
* **Transient State Violations:** Catches if an agent temporarily makes a database public to run a script, even if it makes it private again at the end.
* **Temporal Logic Constraints:** Enforces rules based on time and order (e.g., *"If the agent transfers money, it MUST log out before finishing"*).
* **Errors of Omission (Naive Agents):** Catches when an agent successfully creates a resource but fails to apply the necessary secondary security configurations (like blocking S3 public access).
* **Tool-Agnostic Intent:** Evaluates the *mathematical intent* of an action, whether the agent used Terraform, Python, or a raw CLI command.

## 🧠 Architecture (Neuro-Symbolic Guardrail)

```mermaid
graph TD
    A[Natural Language Spec<br>e.g., 'No public S3 buckets'] --> C
    B[Agent Execution Trace<br>JSON Log] --> C
    C{FormalJudge Compiler<br>Gemini 1.5 Pro} -->|Translates Intent to Math| D[Dafny File<br>State Machine & Invariants]
    D --> E{Dafny Verifier<br>SMT Solver}
    E -->|Proof Succeeds| F((SAFE))
    E -->|Math Evaluates to False| G((UNSAFE))

## 📖 Acknowledgments & References
This proof-of-concept is inspired by the research presented in:
* FormalJudge: A Neuro-Symbolic Paradigm for Agentic Oversight (Zhou et al.)
