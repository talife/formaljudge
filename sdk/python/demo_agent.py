import json
from formaljudge.client import FormalJudgeClient

# 1. Initialize your Guardrail
guardrail = FormalJudgeClient()

company_policy = "Every created S3 bucket MUST have Block Public Access explicitly enabled."

# 2. Simulate the Agent's state (it decided to create a bucket but forgot public access)
agent_trace = {
  "agent_id": "terraform_agent",
  "initial_state": {
    "cloud_provider": "AWS",
    "bucket_exists": "false",
    "block_public_access_enabled": "false"
  },
  "steps": [
    {"step_number": 1, "role": "action", "description": "Executed: CreateBucket(name='app-logs-bucket')"}
  ]
}

# (For local testing without an API key, load the mock we created earlier)
with open("../../examples/terraform_s3/llm_out_tf_naive.json", "r") as f:
    mock_math = json.load(f)

# 3. Intercept the execution!
print("Agent is attempting to execute tools...")
is_safe = guardrail.verify_trace(company_policy, agent_trace, mock_llm_response=mock_math)

if is_safe:
    print("Executing Tool...")
    # Actually run the terraform command
else:
    print("Execution halted. Returning error to LLM so it can fix its mistake.")
    # Return an error to the agent, prompting it to try again and turn on block_public_access!
