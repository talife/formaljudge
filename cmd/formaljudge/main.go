package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/talife/formaljudge/pkg/compiler"
	"github.com/talife/formaljudge/pkg/models"
	"github.com/talife/formaljudge/pkg/verifier"
)

func main() {
	tracePath := flag.String("trace", "", "Path to the agent execution trace JSON file")
	specPath := flag.String("spec", "", "Path to the natural language safety specification file")
	outputPath := flag.String("output", "verification.dfy", "Path to save the generated Dafny file")
	dafnyBin := flag.String("dafny", "dafny", "Path to the Dafny executable")
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	llmResponse := flag.String("llm-response", "", "Path to a JSON file containing the LLM's response (bypasses API)")

	flag.Parse()

	if *tracePath == "" || *specPath == "" {
		fmt.Println("Error: both -trace and -spec arguments are required.")
		flag.Usage()
		os.Exit(1)
	}

	// 1. Read the trace file
	traceData, err := os.ReadFile(*tracePath)
	if err != nil {
		fail(fmt.Sprintf("Failed to read trace file: %v", err), *jsonOutput)
	}

	var trace models.Trace
	if err := json.Unmarshal(traceData, &trace); err != nil {
		fail(fmt.Sprintf("Failed to parse trace JSON: %v", err), *jsonOutput)
	}

	// 2. Read the specification file
	specData, err := os.ReadFile(*specPath)
	if err != nil {
		fail(fmt.Sprintf("Failed to read safety specification file: %v", err), *jsonOutput)
	}

	// 3. Compile NL Spec and Trace into Dafny code
	fmt.Println("[*] Compiling specification and abstracting trace...")
	comp := compiler.NewDafnyCompiler(os.Getenv("GEMINI_API_KEY"))

	dfyFile, err := comp.Compile(context.Background(), string(specData), &trace, *outputPath, *llmResponse)
	if err != nil {
		if err.Error() == "PROMPT_PRINTED" {
			fmt.Println("\n[!] Please copy the prompt above, provide it to your LLM, save the JSON response to a file (e.g., llm_out.json), and rerun with '-llm-response llm_out.json'")
			os.Exit(0)
		}
		fail(fmt.Sprintf("Failed to compile Dafny specification: %v", err), *jsonOutput)
	}
	fmt.Printf("[+] Dafny specification file generated at: %s\n", dfyFile)

	// 4. Run the Dafny Verifier
	fmt.Println("[*] Executing formal verification with Dafny...")
	vf := verifier.NewDafnyVerifier(*dafnyBin)
	verdict, err := vf.Verify(context.Background(), dfyFile)
	if err != nil {
		fail(fmt.Sprintf("Verification runner error: %v", err), *jsonOutput)
	}

	// 5. Output results
	if *jsonOutput {
		jsonBytes, _ := json.MarshalIndent(verdict, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("\n==================== VERDICT ====================")
		fmt.Printf("STATUS: %s\n", verdict.Status)
		fmt.Printf("MESSAGE: %s\n", verdict.Message)
		if verdict.FailedInvariant != "" {
			fmt.Printf("FAILED INVARIANT: %s\n", verdict.FailedInvariant)
		}
		fmt.Println("=================================================")
	}

	if verdict.Status == models.VerdictUnsafe {
		os.Exit(2)
	} else if verdict.Status == models.VerdictError {
		os.Exit(3)
	}
}

func fail(msg string, jsonOut bool) {
	if jsonOut {
		result := models.Verdict{
			Status:  models.VerdictError,
			Message: msg,
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "FATAL ERROR: %s\n", msg)
	}
	os.Exit(1)
}
