curl -i -X POST http://localhost:8080/v1/verify \
  -H "Content-Type: application/json" \
  -d '{
    "spec": "Provisioning permitted strictly in private subnets. Public IP assignment strictly prohibited.",
    "trace": {
      "agent_id": "aws_ec2_agent",
      "initial_state": { "has_public_ip": "false", "is_private_subnet": "true" },
      "steps": [
        {
          "step_number": 1,
          "role": "action",
          "raw_code": "aws ec2 run-instances --subnet-id subnet-private-01a --associate-public-ip-address"
        }
      ]
    },
    "llm_mock_response": {
      "state_definition": "datatype State = State(is_private_subnet: bool, has_public_ip: bool)",
      "actions_definition": "datatype Action = RunInstance(is_private: bool, public_ip: bool)",
      "transition_definition": "function next(s: State, a: Action): State { match a { case RunInstance(p, pub) => State(p, pub) } }",
      "safety_invariant": "predicate SafetyInvariant(s: State) { s.is_private_subnet && !s.has_public_ip }",
      "concrete_trace": "[RunInstance(true, true)]",
      "initial_state_value": "State(true, false)"
    }
  }'
