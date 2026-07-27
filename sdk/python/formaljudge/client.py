import json
import urllib.request
import urllib.error
from typing import Dict, Any, Optional

class FormalJudgeClient:
    def __init__(self, endpoint_url: str = "http://localhost:8080/v1/verify"):
        self.endpoint_url = endpoint_url

    def verify_trace(
        self,
        trace_dict: Dict[str, Any],
        policy_id: Optional[str] = None,
        spec: Optional[str] = None,
        llm_mock_response: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        Sends an agent execution trace to the FormalJudge Go daemon for SMT verification.
        Supports AOT policy_id, dynamic natural language specs, and offline mock math.
        """
        if not policy_id and not spec:
            raise ValueError("Must provide either 'policy_id' (for AOT) or 'spec' (for dynamic verification).")

        payload = {
            "trace": trace_dict
        }
        if policy_id:
            payload["policy_id"] = policy_id
        if spec:
            payload["spec"] = spec
        if llm_mock_response:
            payload["llm_mock_response"] = llm_mock_response

        req = urllib.request.Request(
            self.endpoint_url,
            data=json.dumps(payload).encode('utf-8'),
            headers={'Content-Type': 'application/json'},
            method='POST'
        )

        try:
            with urllib.request.urlopen(req) as response:
                return {
                    "is_safe": True,
                    "status_code": response.getcode(),
                    "data": json.loads(response.read().decode('utf-8'))
                }
        except urllib.error.HTTPError as e:
            error_payload = {}
            try:
                error_payload = json.loads(e.read().decode('utf-8'))
            except Exception:
                error_payload = {"message": str(e)}

            return {
                "is_safe": False,
                "status_code": e.code,
                "error": error_payload
            }

class FormalJudgeResponse:
    def __init__(self, is_safe: bool, status_code: int, data: dict, error: dict = None):
        self.is_safe = is_safe
        self.status_code = status_code
        self.data = data or {}
        self.error = error or {}

    @property
    def remediation_prompt(self) -> str:
        """Returns a pre-formatted prompt string for the LLM context window upon UNSAFE verdicts."""
        if self.is_safe:
            return ""

        correction = self.error.get("self_correction", {})
        if correction:
            return correction.get("suggested_prompt", self.error.get("message", "Execution blocked by guardrail."))
        return self.error.get("message", "Execution blocked by guardrail.")
