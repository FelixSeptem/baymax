package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      string         `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      string         `json:"id"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *rpcError      `json:"error,omitempty"`
}

func main() {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	runID := "ex08-run"
	ctx := context.Background()
	dispatcher := event.NewDispatcher(
		event.NewJSONLoggerWithRuntimeManager(os.Stdout, mgr),
		event.NewRuntimeRecorder(mgr),
	)
	emit := func(t string, payload map[string]any) {
		dispatcher.Emit(ctx, types.Event{
			Version: types.EventSchemaVersionV1,
			Type:    t,
			RunID:   runID,
			Time:    time.Now(),
			Payload: payload,
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      "",
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			return
		}
		emit("agent.network.server.received", map[string]any{"method": req.Method, "id": req.ID})
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "agent.process":
			input, _ := req.Params["input"].(string)
			resp.Result = map[string]any{"output": "processed:" + input}
		default:
			resp.Error = &rpcError{Code: -32601, Message: "method not found"}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	request := rpcRequest{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "agent.process",
		Params:  map[string]any{"input": "hello-network"},
	}
	raw, _ := json.Marshal(request)
	emit("agent.network.client.sent", map[string]any{"method": request.Method, "id": request.ID})

	httpResp, err := http.Post(server.URL, "application/json", bytes.NewReader(raw))
	if err != nil {
		panic(err)
	}
	defer httpResp.Body.Close()

	var response rpcResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
		panic(err)
	}
	emit("agent.network.client.received", map[string]any{
		"id":      response.ID,
		"has_err": response.Error != nil,
	})

	if response.Error != nil {
		fmt.Printf("rpc error: code=%d message=%s\n", response.Error.Code, response.Error.Message)
		return
	}
	fmt.Printf("jsonrpc=%s id=%s result=%v\n", response.JSONRPC, response.ID, response.Result)
}
