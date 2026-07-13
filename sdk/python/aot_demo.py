import json
import urllib.request

BASE_URL = "http://localhost:8080"

# ==========================================
# PHASE 1: SECURITY ENGINEER REGISTERS POLICY
# ==========================================
print("🛡️ [Phase 1] Security Team is registering the S3 Policy...")

policy_payload = {
    "policy_id": "aws-s3-public-block",
    "compiled_math": {
        "state_definition": "datatype State = State(block_public_access_enabled: bool, bucket_exists: bool, cloud_provider: string)",
        "actions_definition": "datatype Action = DeploySecureBucket(name: string)",
        "transition_definition": "function next(s: State, a: Action): State {\n  match a {\n    case DeploySecureBucket(name) => s.(bucket_exists := true, block_public_access_enabled := true)\n  }\n}",
        "safety_invariant": "predicate SafetyInvariant(s: State) {\n  s.bucket_exists ==> s.block_public_access_enabled\n}",
        "concrete_trace": "[DeploySecureBucket(\"app-logs-bucket\")]",
        "initial_state_value": "State(false, false, \"AWS\")"
    }
}

req1 = urllib.request.Request(f"{BASE_URL}/v1/policies", data=json.dumps(policy_payload).encode('utf-8'), headers={'Content-Type': 'application/json'}, method='POST')
with urllib.request.urlopen(req1) as response:
    print(f"   Success: {json.loads(response.read().decode('utf-8'))['message']}\n")


# ==========================================
# PHASE 2: AI AGENT REQUESTS VERIFICATION
# ==========================================
print("🤖 [Phase 2] Agent evaluating trace against Policy 'aws-s3-public-block'...")

# Notice we DO NOT send the heavy compiled_math or spec anymore!
verify_payload = {
    "policy_id": "aws-s3-public-block",
    "trace": {
        "agent_id": "terraform_agent",
        "initial_state": {"cloud_provider": "AWS", "bucket_exists": "false", "block_public_access_enabled": "false"},
        "steps": [
            {"step_number": 1, "role": "action", "description": "DeploySecureBucket(name='app-logs-bucket')"}
        ]
    }
}

req2 = urllib.request.Request(f"{BASE_URL}/v1/verify", data=json.dumps(verify_payload).encode('utf-8'), headers={'Content-Type': 'application/json'}, method='POST')
with urllib.request.urlopen(req2) as response:
    result = json.loads(response.read().decode('utf-8'))
    print(f"✅ Fast AOT Verification Approved!")
    print(f"   Receipt Signature: {result['receipt_signature'][:40]}...")
