import json
from formaljudge.client import FormalJudgeClient

guardrail = FormalJudgeClient(endpoint_url="http://localhost:8080/v1/verify")
company_policy = "Every created S3 bucket MUST have Block Public Access explicitly enabled."

agent_trace = {
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
            "raw_code": "resource \"aws_s3_bucket\" \"app_logs\" {\n  bucket = \"app-logs-bucket\"\n}\n\nresource \"aws_s3_bucket_public_access_block\" \"app_logs_security\" {\n  bucket = aws_s3_bucket.app_logs.id\n  block_public_acls       = true\n  block_public_policy     = true\n  ignore_public_acls      = true\n  restrict_public_buckets = true\n}",
            "symbolic_mapping": "DeploySecureBucket(name='app-logs-bucket')"
        }
    ]
}

mock_math = {
    "state_definition": "datatype State = State(block_public_access_enabled: bool, bucket_exists: bool, cloud_provider: string)",
    "actions_definition": "datatype Action = DeploySecureBucket(name: string)",
    "transition_definition": "function next(s: State, a: Action): State {\n  match a {\n    case DeploySecureBucket(name) => s.(bucket_exists := true, block_public_access_enabled := true)\n  }\n}",
    "safety_invariant": "predicate SafetyInvariant(s: State) {\n  s.bucket_exists ==> s.block_public_access_enabled\n}",
    "concrete_trace": "[DeploySecureBucket(\"app-logs-bucket\")]",
    "initial_state_value": "State(false, false, \"AWS\")"
}

print("Agent is attempting to execute tools...")
response = guardrail.verify_trace(
    trace_dict=agent_trace,
    spec=company_policy,
    llm_mock_response=mock_math
)

if response.get("is_safe"):
    print("✅ Execution approved by FormalJudge. Cryptographic receipt secured!")
    print(f"   Receipt Signature: {response['data']['receipt_signature'][:40]}...")
else:
    print(f"🛑 Execution blocked: {response.get('error')}")
