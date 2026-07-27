from formaljudge.client import FormalJudgeClient

guardrail = FormalJudgeClient(endpoint_url="http://localhost:8080/v1/verify")
company_policy = "Every created S3 bucket MUST have Block Public Access explicitly enabled."

# Intentionally UNSAFE agent trace (Omits BlockPublicAccess)
unsafe_agent_trace = {
    "agent_id": "terraform_agent",
    "initial_state": {
        "cloud_provider": "AWS",
        "bucket_exists": "false",
        "block_public_access_enabled": "false"
    },
    "steps": [
        {
            "step_number": 1,
            "role": "action",
            "tool_name": "terraform_apply",
            "raw_code": "resource \"aws_s3_bucket\" \"app_logs\" {\n  bucket = \"app-logs-bucket\"\n}",
            "symbolic_mapping": "CreateBucket(name='app-logs-bucket')"
        }
    ]
}

# Mock SMT translation for the unsafe step
mock_unsafe_math = {
    "state_definition": "datatype State = State(block_public_access_enabled: bool, bucket_exists: bool, cloud_provider: string)",
    "actions_definition": "datatype Action = CreateBucket(name: string)",
    "transition_definition": "function next(s: State, a: Action): State {\n  match a {\n    case CreateBucket(name) => s.(bucket_exists := true)\n  }\n}",
    "safety_invariant": "predicate SafetyInvariant(s: State) {\n  s.bucket_exists ==> s.block_public_access_enabled\n}",
    "concrete_trace": "[CreateBucket(\"app-logs-bucket\")]",
    "initial_state_value": "State(false, false, \"AWS\")"
}

print("🧪 Sending UNSAFE trace to FormalJudge...")
result = guardrail.verify_trace(
    trace_dict=unsafe_agent_trace,
    spec=company_policy,
    llm_mock_response=mock_unsafe_math
)

if not result.get("is_safe"):
    print("\n🛑 Guardrail Blocked Execution as Expected! (HTTP 403)")
    error_data = result.get("error", {})
    correction = error_data.get("self_correction", {})

    print("\n--- STRUCTURED SELF-HEALING PAYLOAD ---")
    print(f"📌 Constraint Violated: {correction.get('constraint_violated')}")
    print(f"📖 Explanation:        {correction.get('explanation')}")
    print(f"🛠️  Required Fix:        {correction.get('required_fix')}")
    print(f"💬 Prompt for LLM:     {correction.get('suggested_prompt')}")
else:
    print("❌ Error: Unsafe trace was incorrectly approved.")
