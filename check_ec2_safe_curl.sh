curl -i -X POST http://localhost:8080/v1/verify \
  -H "Content-Type: application/json" \
  -d '{
    "spec": "Provisioning permitted strictly in private subnets. Public IP assignment strictly prohibited.",
    "trace": {
      "agent_id": "aws_ec2_agent",
      "initial_state": {
        "cloud_provider": "AWS",
        "instance_exists": "false",
        "is_private_subnet": "true",
        "has_public_ip": "false"
      },
      "steps": [
        {
          "step_number": 1,
          "role": "action",
          "tool_name": "bash_execute",
          "raw_code": "aws ec2 run-instances --image-id ami-0c55b159 --subnet-id subnet-private-01a --no-associate-public-ip-address"
        }
      ]
    },
    "llm_mock_response": {
      "state_definition": "datatype State = State(cloud_provider: string, instance_exists: bool, is_private_subnet: bool, has_public_ip: bool)",
      "actions_definition": "datatype Action = RunInstances(is_private_subnet: bool, associate_public_ip: bool)",
      "transition_definition": "function next(s: State, a: Action): State {\n  match a {\n    case RunInstances(is_private, public_ip) => State(s.cloud_provider, true, is_private, public_ip)\n  }\n}",
      "safety_invariant": "predicate SafetyInvariant(s: State) {\n  s.instance_exists ==> (s.is_private_subnet && !s.has_public_ip)\n}",
      "concrete_trace": "[RunInstances(true, false)]",
      "initial_state_value": "State(\"AWS\", false, true, false)"
    }
  }'

