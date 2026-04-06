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

func TestContextStage2ExternalRetrieverHTTPIntegration(t *testing.T) {
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
	if runs[0].Stage2ReasonCode != "ok" {
		t.Fatalf("stage2_reason_code = %q, want ok", runs[0].Stage2ReasonCode)
	}
	if runs[0].Stage2ErrorLayer != "" {
		t.Fatalf("stage2_error_layer = %q, want empty", runs[0].Stage2ErrorLayer)
	}
	if runs[0].Stage2Profile != runtimeconfig.ContextStage2ExternalProfileHTTPGeneric {
		t.Fatalf("stage2_profile = %q, want http_generic", runs[0].Stage2Profile)
	}
	if runs[0].Stage2TemplateProfile != runtimeconfig.ContextStage2ExternalProfileHTTPGeneric {
		t.Fatalf("stage2_template_profile = %q, want http_generic", runs[0].Stage2TemplateProfile)
	}
	if runs[0].Stage2TemplateResolutionSource == "" {
		t.Fatalf("stage2_template_resolution_source should not be empty: %#v", runs[0])
	}
	if runs[0].Stage2HintApplied {
		t.Fatalf("stage2_hint_applied = %v, want false", runs[0].Stage2HintApplied)
	}
}

func TestContextStage2ExternalRetrieverRunStreamSemanticEquivalenceHintApplied(t *testing.T) {
	retriever := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"chunks": []string{"external-ctx-1"},
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
        profile: ragflow_like
        endpoint: ` + retriever.URL + `
        hints:
          enabled: true
          capabilities: [metadata_filter]
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
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"}}, nil)
	rec := obsevent.NewRuntimeRecorder(mgr)
	eng := runner.New(model, runner.WithRuntimeManager(mgr))

	if _, err := eng.Run(context.Background(), types.RunRequest{Input: "need retrieval", SessionID: "s-run"}, rec); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if _, err := eng.Stream(context.Background(), types.RunRequest{Input: "need retrieval", SessionID: "s-stream"}, rec); err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	runs := mgr.RecentRuns(2)
	if len(runs) != 2 {
		t.Fatalf("run diagnostics len = %d, want 2", len(runs))
	}
	for i := range runs {
		if !runs[i].Stage2HintApplied {
			t.Fatalf("run[%d].stage2_hint_applied = %v, want true", i, runs[i].Stage2HintApplied)
		}
		if runs[i].Stage2HintMismatchReason != "" {
			t.Fatalf("run[%d].stage2_hint_mismatch_reason = %q, want empty", i, runs[i].Stage2HintMismatchReason)
		}
		if runs[i].Stage2TemplateProfile != runtimeconfig.ContextStage2ExternalProfileRAGFlowLike {
			t.Fatalf("run[%d].stage2_template_profile = %q, want ragflow_like", i, runs[i].Stage2TemplateProfile)
		}
	}
	if runs[0].Stage2TemplateResolutionSource != runs[1].Stage2TemplateResolutionSource {
		t.Fatalf(
			"run/stream template resolution source mismatch: run=%q stream=%q",
			runs[0].Stage2TemplateResolutionSource,
			runs[1].Stage2TemplateResolutionSource,
		)
	}
}

func TestContextStage2ExternalRetrieverRunStreamSemanticEquivalenceHintMismatch(t *testing.T) {
	retriever := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"chunks": []string{"external-ctx-1"},
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
        profile: ragflow_like
        endpoint: ` + retriever.URL + `
        hints:
          enabled: true
          capabilities: [dsl_query]
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
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"}}, nil)
	rec := obsevent.NewRuntimeRecorder(mgr)
	eng := runner.New(model, runner.WithRuntimeManager(mgr))

	if _, err := eng.Run(context.Background(), types.RunRequest{Input: "need retrieval", SessionID: "s-run"}, rec); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if _, err := eng.Stream(context.Background(), types.RunRequest{Input: "need retrieval", SessionID: "s-stream"}, rec); err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	runs := mgr.RecentRuns(2)
	if len(runs) != 2 {
		t.Fatalf("run diagnostics len = %d, want 2", len(runs))
	}
	for i := range runs {
		if runs[i].Stage2HintApplied {
			t.Fatalf("run[%d].stage2_hint_applied = %v, want false", i, runs[i].Stage2HintApplied)
		}
		if runs[i].Stage2HintMismatchReason != "hint.unsupported" {
			t.Fatalf("run[%d].stage2_hint_mismatch_reason = %q, want hint.unsupported", i, runs[i].Stage2HintMismatchReason)
		}
	}
	if runs[0].Stage2HintMismatchReason != runs[1].Stage2HintMismatchReason {
		t.Fatalf(
			"run/stream hint mismatch reason mismatch: run=%q stream=%q",
			runs[0].Stage2HintMismatchReason,
			runs[1].Stage2HintMismatchReason,
		)
	}
}
