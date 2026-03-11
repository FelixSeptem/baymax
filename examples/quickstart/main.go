package main

import (
	"context"
	"fmt"
	"os"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	openaiadapter "github.com/FelixSeptem/baymax/model/openai"
)

func main() {
	model := openaiadapter.NewClient(openaiadapter.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4.1-mini",
	})

	engine := runner.New(model)
	res, err := engine.Run(context.Background(), types.RunRequest{
		Input: "用一句话介绍 Baymax Agent Loop。",
	}, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.FinalAnswer)
}
