package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/FelixSeptem/baymax/tool/contributioncheck"
)

type pullRequestEvent struct {
	PullRequest struct {
		Body string `json:"body"`
	} `json:"pull_request"`
}

func main() {
	var (
		eventPath string
		bodyFile  string
	)
	flag.StringVar(&eventPath, "event", "", "path to GitHub pull_request event payload json")
	flag.StringVar(&bodyFile, "body-file", "", "path to plain text pull request body file")
	flag.Parse()

	body, err := loadBody(eventPath, bodyFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	violations := contributioncheck.ValidatePullRequestBody(body)
	if len(violations) == 0 {
		fmt.Println("contribution template check passed")
		return
	}

	for _, v := range violations {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", v.Code, v.Message)
	}
	os.Exit(1)
}

func loadBody(eventPath, bodyFile string) (string, error) {
	if strings.TrimSpace(bodyFile) != "" {
		raw, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", fmt.Errorf("read body file: %w", err)
		}
		return string(raw), nil
	}

	if strings.TrimSpace(eventPath) == "" {
		return "", fmt.Errorf("usage: contribution-template-check -event <event.json> OR -body-file <body.txt>")
	}

	raw, err := os.ReadFile(eventPath)
	if err != nil {
		return "", fmt.Errorf("read event payload: %w", err)
	}
	var evt pullRequestEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return "", fmt.Errorf("decode event payload: %w", err)
	}
	if strings.TrimSpace(evt.PullRequest.Body) == "" {
		return "", fmt.Errorf("event payload missing pull_request.body")
	}
	return evt.PullRequest.Body, nil
}
