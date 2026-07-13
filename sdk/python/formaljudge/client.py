import json
import urllib.request
import urllib.error

class FormalJudgeClient:
    def __init__(self, endpoint_url="http://localhost:8080/v1/verify"):
        self.endpoint_url = endpoint_url

    def verify_trace(self, spec: str, trace_dict: dict, mock_llm_response: dict = None) -> bool:
        """
        Sends the agent's current trace to the FormalJudge Go API.
        Returns True if SAFE, False if UNSAFE.
        """
        payload = {
            "spec": spec,
            "trace": trace_dict
        }

        if mock_llm_response:
            payload["llm_mock_response"] = mock_llm_response

        data = json.dumps(payload).encode('utf-8')
        req = urllib.request.Request(
            self.endpoint_url,
            data=data,
            headers={'Content-Type': 'application/json'},
            method='POST'
        )

        try:
            with urllib.request.urlopen(req) as response:
                # HTTP 200 OK means it's SAFE
                result = json.loads(response.read().decode('utf-8'))
                print(f"✅ FormalJudge Approved: {result.get('message')}")
                return True

        except urllib.error.HTTPError as e:
            # Our Go API returns 403 Forbidden for UNSAFE
            if e.code == 403:
                result = json.loads(e.read().decode('utf-8'))
                print(f"❌ FormalJudge Blocked Action!")
                print(f"Reason: {result.get('message')}")
                print(f"Failed Invariant: {result.get('failed_invariant', 'Unknown')}")
                return False
            else:
                print(f"⚠️ FormalJudge System Error: {e.code}")
                raise e
