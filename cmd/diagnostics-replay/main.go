package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

func main() {
	var inputPath string
	flag.StringVar(&inputPath, "input", "", "path to diagnostics json file")
	flag.Parse()

	if inputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: diagnostics-replay -input <diagnostics.json>")
		os.Exit(2)
	}

	raw, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	out, err := diagnosticsreplay.ParseMinimalReplayJSON(raw)
	if err != nil {
		if vErr, ok := err.(*diagnosticsreplay.ValidationError); ok {
			fmt.Fprintf(os.Stderr, "[%s] %s\n", vErr.Code, vErr.Message)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "replay failed: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "encode output: %v\n", err)
		os.Exit(1)
	}
}
