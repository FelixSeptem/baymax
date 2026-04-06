package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FelixSeptem/baymax/context/assembler"
)

func main() {
	var (
		inputPath  string
		outputPath string
	)
	flag.StringVar(&inputPath, "input", "", "path to tuning input json")
	flag.StringVar(&outputPath, "output", "", "path to markdown report output")
	flag.Parse()
	if inputPath == "" || outputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: context-threshold-governance-tuning -input <input.json> -output <report.md>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}
	var req assembler.ThresholdTuningRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		fmt.Fprintf(os.Stderr, "decode input: %v\n", err)
		os.Exit(1)
	}
	report, err := assembler.RunThresholdTuning(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuning failed: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, []byte(assembler.RenderThresholdTuningMarkdown(report)), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}
