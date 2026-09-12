package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host"
	"github.com/FelixSeptem/baymax/host/jsonl"
)

type starter struct {
	connection *host.Connection
	mode       string
	executions sync.WaitGroup
}

func (s *starter) AdmitHostRun(_ context.Context, req types.RunRequest, _ types.HostRunExecutionMode) (types.HostRunStartAdmission, error) {
	s.executions.Add(1)
	return types.HostRunStartAdmission{Status: types.HostAdmissionStatusAccepted, RunID: req.RunID, Execution: execution{connection: s.connection, request: req, mode: s.mode, done: s.executions.Done}}, nil
}

type execution struct {
	connection *host.Connection
	request    types.RunRequest
	mode       string
	done       func()
}

func (e execution) Execute(h types.EventHandler) (types.RunResult, error) {
	defer e.done()
	switch e.mode {
	case "burst":
		for i := 0; i < 256; i++ {
			h.OnEvent(context.Background(), types.Event{Version: "v1", Type: "run.progress", RunID: e.request.RunID, Time: time.Now().UTC(), Payload: map[string]any{"chunk": strings.Repeat("x", 16384)}})
		}
		_, _ = fmt.Fprintln(os.Stderr, "burst_result=returned")
		return types.RunResult{RunID: e.request.RunID}, nil
	case "serialized":
		var group sync.WaitGroup
		for i := 0; i < 32; i++ {
			group.Add(1)
			go func(sequence int) {
				defer group.Done()
				h.OnEvent(context.Background(), types.Event{Version: "v1", Type: "run.progress", RunID: e.request.RunID, Time: time.Now().UTC(), Payload: map[string]any{"sequence": sequence}})
			}(i)
		}
		group.Wait()
		return types.RunResult{RunID: e.request.RunID}, nil
	case "pending":
		_, err := e.connection.Resolve(context.Background(), types.ClarificationResolveRequest{
			RunID: e.request.RunID, SessionID: e.request.SessionID, Timeout: 10 * time.Second,
			Request: types.ClarificationRequest{RequestID: "pending-" + e.request.RunID, Questions: []string{"continue?"}},
		})
		_, _ = fmt.Fprintf(os.Stderr, "pending_result=%v\n", err)
		return types.RunResult{RunID: e.request.RunID}, err
	default:
		h.OnEvent(context.Background(), types.Event{Version: "v1", Type: "run.started", RunID: e.request.RunID, Time: time.Now().UTC(), Payload: map[string]any{"unicode": "a\u2028b\u2029c"}})
		return types.RunResult{RunID: e.request.RunID}, nil
	}
}

func (e execution) Release() {}

func main() {
	mode := flag.String("mode", "default", "conformance execution mode")
	flag.Parse()
	starter := &starter{mode: *mode}
	coordinator := host.New(nil, host.WithRunStarter(starter))
	server, err := jsonl.NewServer(coordinator, jsonl.ServerConfig{
		Input: os.Stdin, ProtocolOutput: os.Stdout, LogOutput: os.Stderr,
		DeliveryTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "helper_init=%v\n", err)
		os.Exit(1)
	}
	starter.connection = server.Connection()
	if err := server.Serve(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "helper_serve=%v\n", err)
		os.Exit(1)
	}
	starter.executions.Wait()
}
