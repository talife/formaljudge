.PHONY: build demo-bank demo-tf clean

build:
	@echo "Building formaljudge..."
	@go build -o bin/formaljudge ./cmd/formaljudge

demo-bank:
	@echo "Running Bank Transfer Demo (LLM Bypassed)..."
	@go run ./cmd/formaljudge -trace examples/bank_transfer/trace.json -spec examples/bank_transfer/spec.txt -llm-response llm_out.json -output bank_verification.dfy

demo-tf:
	@echo "Running Terraform Security Demo (LLM Bypassed)..."
	@go run ./cmd/formaljudge -trace examples/terraform_s3/trace.json -spec examples/terraform_s3/spec.txt -llm-response llm_out_tf_naive.json -output tf_naive_verification.dfy

clean:
	@rm -rf bin/
	@rm -f *.dfy

