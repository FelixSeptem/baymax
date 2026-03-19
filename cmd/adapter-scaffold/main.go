package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/FelixSeptem/baymax/adapter/scaffold"
)

func main() {
	var (
		scaffoldType string
		name         string
		output       string
		force        bool
	)
	flag.StringVar(&scaffoldType, "type", "", "adapter scaffold type: mcp|model|tool")
	flag.StringVar(&name, "name", "", "adapter scaffold name (^[a-z][a-z0-9-]*$)")
	flag.StringVar(&output, "output", "", "output directory, default examples/adapters/<type>-<name>")
	flag.BoolVar(&force, "force", false, "overwrite existing planned files")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: adapter-scaffold -type <mcp|model|tool> -name <adapter-name> [-output <dir>] [-force]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "[adapter-scaffold][invalid-args] positional arguments are not supported")
		flag.Usage()
		os.Exit(2)
	}

	plan, err := scaffold.Generate(scaffold.Options{
		Type:    scaffoldType,
		Name:    name,
		Output:  output,
		Force:   force,
		BaseDir: ".",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		if scaffold.IsValidationError(err) {
			flag.Usage()
			os.Exit(2)
		}
		os.Exit(1)
	}

	fmt.Printf("[adapter-scaffold] generated %d files at %s\n", len(plan.Files), plan.OutputDir)
}
