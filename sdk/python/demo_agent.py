import json
from formaljudge.client import FormalJudgeClient

guardrail = FormalJudgeClient()
company_policy = "Every created S3 bucket MUST have Block Public Access explicitly enabled."

# 1. The Agent executes a single, atomic secure deployment
agent_trace = {
  "agent_id": "terraform_agent",
  "initial_state": {
    "cloud_provider": "AWS",
    "bucket_exists": "false",
    "block_public_access_enabled": "false"
  },
  "steps": [
    {"step_number": 1, "role": "action", "description": "DeploySecureBucket(name='app-logs-bucket')"}
  ]
}

# 2. Mock math representing the atomic action
mock_math = {
    "state_definition": "datatype State = State(block_public_access_enabled: bool, bucket_exists: bool, cloud_provider: string)",
    "actions_definition": "datatype Action = DeploySecureBucket(name: string)",
    "transition_definition": "function next(s: State, a: Action): State {\n  match a {\n    case DeploySecureBucket(name) => s.(bucket_exists := true, block_public_access_enabled := true)\n  }\n}",
    "safety_invariant": "predicate SafetyInvariant(s: State) {\n  s.bucket_exists ==> s.block_public_access_enabled\n}",
    "concrete_trace": "[DeploySecureBucket(\"app-logs-bucket\")]",
    "initial_state_value": "State(false, false, \"AWS\")"
}

print("Agent is attempting to execute tools...")
is_safe = guardrail.verify_trace(company_policy, agent_trace, mock_llm_response=mock_math)

if is_safe:
    print("Execution complete. Audit trail saved.")
