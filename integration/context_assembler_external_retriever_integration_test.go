package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	obsevent "github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestCA2ExternalRetrieverHTTPIntegration(t *testing.T) {
	retriever := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["query"] != "need retrieval" {
			t.Fatalf("query = %#v, want need retrieval", payload["query"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"chunks": []string{"external-ctx-1", "external-ctx-2"},
			"source": "http",
			"reason": "ok",
		})
	}))
	defer retriever.Close()

	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
context_assembler:
  enabled: true
  ca2:
    enabled: true
    routing_mode: rules
    stage_policy:
      stage1: fail_fast
      stage2: fail_fast
    stage2:
      provider: http
      external:
        endpoint: ` + retriever.URL + `
    routing:
      min_input_chars: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	rec := obsevent.NewRuntimeRecorder(mgr)
	eng := runner.New(model, runner.WithRuntimeManager(mgr))
	res, err := eng.Run(context.Background(), types.RunRequest{
		Input:     "need retrieval",
		SessionID: "s-1",
	}, rec)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.FinalAnswer != "ok" {
		t.Fatalf("final answer = %q, want ok", res.FinalAnswer)
	}

	lastReq := model.LastRequest()
	foundStage2 := false
	for _, msg := range lastReq.Messages {
		if strings.Contains(msg.Content, "external-ctx-1") {
			foundStage2 = true
			break
		}
	}
	if !foundStage2 {
		t.Fatalf("stage2 context not merged into model request: %#v", lastReq.Messages)
	}

	runs := mgr.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("run diagnostics len = %d, want 1", len(runs))
	}
	if runs[0].Stage2HitCount != 2 {
		t.Fatalf("stage2_hit_count = %d, want 2", runs[0].Stage2HitCount)
	}
	if runs[0].Stage2Source != "http" {
		t.Fatalf("stage2_source = %q, want http", runs[0].Stage2Source)
	}
	if runs[0].Stage2Reason != "ok" {
		t.Fatalf("stage2_reason = %q, want ok", runs[0].Stage2Reason)
	}
}
