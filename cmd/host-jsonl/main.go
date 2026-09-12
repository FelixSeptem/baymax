// Command host-jsonl serves the embedded host protocol over strict JSONL on
// stdin/stdout. It intentionally has no model, network listener, or business
// state owner; applications compose those dependencies around host.Coordinator.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/FelixSeptem/baymax/host"
	"github.com/FelixSeptem/baymax/host/jsonl"
)

const (
	exitOK                 = 0
	exitMalformedInput     = 2
	exitUnsupportedVersion = 3
	exitNegotiation        = 4
	exitOutputFailure      = 5
	exitCanceled           = 6
	exitProtocolFailure    = 7
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	server, err := jsonl.NewServer(host.New(nil), jsonl.DefaultServerConfig())
	if err == nil {
		err = server.Serve(ctx)
	}
	if err == nil {
		return exitOK
	}
	return exitCode(jsonl.ClassifyExit(err))
}

// exitCode is deliberately exhaustive so adding a classification cannot
// silently change process behavior. Unknown failures use the protocol code.
func exitCode(classification jsonl.ExitClassification) int {
	switch classification {
	case jsonl.ExitSuccess:
		return exitOK
	case jsonl.ExitMalformedInput, jsonl.ExitFrameTooLarge:
		return exitMalformedInput
	case jsonl.ExitUnsupportedVersion:
		return exitUnsupportedVersion
	case jsonl.ExitNegotiation:
		return exitNegotiation
	case jsonl.ExitOutputFailure:
		return exitOutputFailure
	case jsonl.ExitParentCanceled:
		return exitCanceled
	case jsonl.ExitProtocol:
		return exitProtocolFailure
	default:
		return exitProtocolFailure
	}
}
