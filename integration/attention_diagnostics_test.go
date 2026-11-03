package integration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestAttentionRecorderSnapshotAndHostEffectBoundary(t *testing.T) {
	manager, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_ATTENTION_TEST"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = manager.Close() }()
	recorder := event.NewRuntimeRecorder(manager)
	defer recorder.Close()
	for i, scenario := range []struct {
		id      string
		payload map[string]any
	}{
		{"healthy", map[string]any{}},
		{"retrying", map[string]any{"terminal_retryable": true}},
		{"unsupported", map[string]any{"protocol_capability_reason": "protocol.capability.missing_required"}},
		{"recovery", map[string]any{"recovery_conflict": true}},
		{"gate", map[string]any{"gate_timeout_count": 1}},
		{"effect", map[string]any{}},
	} {
		recorder.OnEvent(context.Background(), types.Event{Type: "run.finished", RunID: scenario.id, Time: time.Unix(int64(i+1), 0), Payload: scenario.payload})
	}
	page, err := manager.QueryRuns(diagnostics.UnifiedRunQueryRequest{})
	if err != nil || len(page.Items) != 6 {
		t.Fatalf("page=%#v err=%v", page, err)
	}
	view, err := diagnostics.ProjectAttention(diagnostics.AttentionRequest{Runs: page.Items})
	if err != nil || view.Total != 3 {
		t.Fatalf("view=%#v err=%v", view, err)
	}
	for _, item := range view.Items {
		if item.RunID == "effect" || item.RunID == "retrying" || item.RunID == "healthy" {
			t.Fatal("inferred owner fact")
		}
	}
	evidence := []diagnostics.AttentionEvidence{{RunID: "effect", Code: "effect-pending", Reference: "ledger/pending-1"}}
	view, err = diagnostics.ProjectAttention(diagnostics.AttentionRequest{Runs: page.Items, Evidence: evidence})
	if err != nil || view.Total != 4 {
		t.Fatalf("explicit effect=%#v err=%v", view, err)
	}
	after, err := manager.QueryRuns(diagnostics.UnifiedRunQueryRequest{})
	if err != nil || !reflect.DeepEqual(page, after) {
		t.Fatalf("projection modified diagnostics: %v", err)
	}
	if evidence[0].Reference != "ledger/pending-1" {
		t.Fatal("projection modified host evidence")
	}
}

func TestAttentionRealProviderAdmissionRunStreamParity(t *testing.T) {
	for _, scenario := range []struct {
		name, model, credential, outcome string
		required                         []types.ModelCapability
		want                             int
	}{
		{"missing-capability", "gpt", "ready", "blocked", []types.ModelCapability{types.ModelCapabilityToolCall}, 1},
		{"unknown-model", "unknown", "ready", "blocked", nil, 1},
		{"credential-only", "gpt", "missing", "accepted", nil, 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			views := make([]diagnostics.AttentionResult, 0, 2)
			for _, stream := range []bool{false, true} {
				path := filepath.Join(t.TempDir(), "runtime.yaml")
				cfg := "runtime:\n  provider_catalog:\n    enabled: true\n    version: v-attention\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 128\n        capabilities: [streaming]\n"
				if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
					t.Fatal(err)
				}
				mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: path, EnvPrefix: "BAYMAX_ATTENTION_PRODUCER_TEST"})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = mgr.Close() })
				recorder := event.NewRuntimeRecorder(mgr)
				t.Cleanup(recorder.Close)
				model := fakes.NewModel(nil)
				comp, err := composer.NewBuilder(model).WithRuntimeManager(mgr).WithEventHandler(recorder).Build()
				if err != nil {
					t.Fatal(err)
				}
				req := types.RunRequest{RunID: "run", Input: "test", Provider: "openai", Model: scenario.model, CredentialStatus: scenario.credential, Capabilities: types.CapabilityRequirements{Required: scenario.required}}
				if stream {
					_, err = comp.Stream(context.Background(), req, nil)
				} else {
					_, err = comp.Run(context.Background(), req, nil)
				}
				if err == nil {
					t.Fatal("expected admission rejection")
				}
				page, err := mgr.QueryRuns(diagnostics.UnifiedRunQueryRequest{})
				if err != nil || len(page.Items) != 1 {
					t.Fatalf("page=%#v err=%v", page, err)
				}
				if page.Items[0].ProviderCapabilityOutcome != scenario.outcome {
					t.Fatalf("producer outcome=%q", page.Items[0].ProviderCapabilityOutcome)
				}
				view, err := diagnostics.ProjectAttention(diagnostics.AttentionRequest{Runs: page.Items})
				if err != nil || view.Total != scenario.want {
					t.Fatalf("view=%#v err=%v", view, err)
				}
				if scenario.want > 0 && (view.Items[0].Owner != "runtime/capability" || view.Items[0].NextAction != "inspect_capability") {
					t.Fatalf("mapping=%#v", view)
				}
				views = append(views, view)
			}
			if !reflect.DeepEqual(views[0], views[1]) {
				t.Fatalf("Run/Stream drift: %#v", views)
			}
		})
	}
}
