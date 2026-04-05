package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRealtimeRunStreamResumeParityAndIdempotency(t *testing.T) {
	mgr := newRealtimeRuntimeManagerForTest(t, "BAYMAX_A68_RUNNER_PARITY")

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				FinalAnswer: "done",
				Usage:       types.TokenUsage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
			}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type:      types.ModelEventTypeOutputTextDelta,
				TextDelta: "done",
			})
		},
	}

	runEngine := New(runModel, WithRuntimeManager(mgr))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))
	req := types.RunRequest{
		RunID:     "run-a68-parity",
		SessionID: "session-a68-parity",
		Input:     "resume",
		Realtime: &types.RealtimeRunRequest{
			Events: []types.RealtimeEventEnvelope{
				newRealtimeControlEvent("interrupt-1", "session-a68-parity", "run-a68-parity", 1, types.RealtimeEventTypeInterrupt, map[string]any{}),
				newRealtimeControlEvent("interrupt-1", "session-a68-parity", "run-a68-parity", 1, types.RealtimeEventTypeInterrupt, map[string]any{}),
				newRealtimeControlEvent("resume-1", "session-a68-parity", "run-a68-parity", 2, types.RealtimeEventTypeResume, map[string]any{
					"cursor": "session-a68-parity:run-a68-parity:1",
				}),
				newRealtimeControlEvent("resume-1", "session-a68-parity", "run-a68-parity", 2, types.RealtimeEventTypeResume, map[string]any{
					"cursor": "session-a68-parity:run-a68-parity:1",
				}),
			},
		},
	}

	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}

	runResult, runErr := runEngine.Run(context.Background(), req, runCollector)
	if runErr != nil {
		t.Fatalf("Run failed: %v", runErr)
	}
	streamResult, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)
	if streamErr != nil {
		t.Fatalf("Stream failed: %v", streamErr)
	}
	if runResult.FinalAnswer != streamResult.FinalAnswer || runResult.FinalAnswer != "done" {
		t.Fatalf("run/stream final answer mismatch run=%q stream=%q", runResult.FinalAnswer, streamResult.FinalAnswer)
	}

	runPayload := lastRunFinishedPayloadFromCollector(t, runCollector)
	streamPayload := lastRunFinishedPayloadFromCollector(t, streamCollector)
	assertRealtimePayloadCounters(t, runPayload)
	assertRealtimePayloadCounters(t, streamPayload)
	if runPayload["realtime_resume_source"] != streamPayload["realtime_resume_source"] {
		t.Fatalf("resume source mismatch run=%#v stream=%#v", runPayload["realtime_resume_source"], streamPayload["realtime_resume_source"])
	}

	assertCollectorContainsCanonicalRealtimeEnvelope(t, runCollector)
	assertCollectorContainsCanonicalRealtimeEnvelope(t, streamCollector)
}

func TestRealtimeRunStreamInvalidResumeClassificationParity(t *testing.T) {
	mgr := newRealtimeRuntimeManagerForTest(t, "BAYMAX_A68_RUNNER_INVALID_RESUME")

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "should-not-reach"}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "should-not-reach"})
		},
	}

	runEngine := New(runModel, WithRuntimeManager(mgr))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))
	req := types.RunRequest{
		RunID:     "run-a68-invalid-resume",
		SessionID: "session-a68-invalid-resume",
		Input:     "resume",
		Realtime: &types.RealtimeRunRequest{
			Events: []types.RealtimeEventEnvelope{
				newRealtimeControlEvent(
					"resume-invalid",
					"session-a68-invalid-resume",
					"run-a68-invalid-resume",
					1,
					types.RealtimeEventTypeResume,
					map[string]any{"cursor": "bad-cursor"},
				),
			},
		},
	}

	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runResult, runErr := runEngine.Run(context.Background(), req, runCollector)
	streamResult, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)

	if runErr == nil || streamErr == nil {
		t.Fatalf("expected run/stream invalid resume to fail, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runResult.Error == nil || streamResult.Error == nil {
		t.Fatalf("expected run/stream classified errors, run=%#v stream=%#v", runResult.Error, streamResult.Error)
	}
	if runResult.Error.Class != types.ErrContext || streamResult.Error.Class != types.ErrContext {
		t.Fatalf("expected ErrContext parity, run=%#v stream=%#v", runResult.Error, streamResult.Error)
	}

	runPayload := lastRunFinishedPayloadFromCollector(t, runCollector)
	streamPayload := lastRunFinishedPayloadFromCollector(t, streamCollector)
	if runPayload["reason_code"] != realtimeReasonInvalidResumeCursor ||
		streamPayload["reason_code"] != realtimeReasonInvalidResumeCursor {
		t.Fatalf("invalid resume reason code mismatch run=%#v stream=%#v", runPayload["reason_code"], streamPayload["reason_code"])
	}
	if runPayload["realtime_error_layer"] != realtimeErrorLayerSemantic ||
		streamPayload["realtime_error_layer"] != realtimeErrorLayerSemantic {
		t.Fatalf("invalid resume error layer mismatch run=%#v stream=%#v", runPayload["realtime_error_layer"], streamPayload["realtime_error_layer"])
	}
}

func TestRealtimeSequenceGapAndOrderClassification(t *testing.T) {
	engine := New(&fakeModel{})
	session := &realtimeSessionRuntime{
		engine:    engine,
		runID:     "run-a68-gap",
		sessionID: "session-a68-gap",
		cfg: runtimeconfig.RuntimeRealtimeConfig{
			Protocol: runtimeconfig.RuntimeRealtimeProtocolConfig{
				Enabled:           true,
				Version:           runtimeconfig.RuntimeRealtimeProtocolVersionV1,
				MaxBufferedEvents: 64,
			},
			InterruptResume: runtimeconfig.RuntimeRealtimeInterruptResumeConfig{
				Enabled:             true,
				ResumeCursorTTLMS:   300000,
				IdempotencyWindowMS: 120000,
			},
		},
		seenDedup: map[string]struct{}{},
	}

	err := session.ingestControlEvents(context.Background(), &eventCollector{}, 0, []types.RealtimeEventEnvelope{
		newRealtimeControlEvent("evt-1", "session-a68-gap", "run-a68-gap", 1, types.RealtimeEventTypeRequest, map[string]any{"input": "x"}),
		newRealtimeControlEvent("evt-2", "session-a68-gap", "run-a68-gap", 3, types.RealtimeEventTypeDelta, map[string]any{"delta": "y"}),
	})
	if err == nil || !strings.Contains(err.Error(), "sequence gap") {
		t.Fatalf("expected sequence gap error, got %v", err)
	}
	if session.lastErrorCode != realtimeReasonSequenceGap {
		t.Fatalf("lastErrorCode=%q, want %q", session.lastErrorCode, realtimeReasonSequenceGap)
	}

	session2 := &realtimeSessionRuntime{
		engine:    engine,
		runID:     "run-a68-order",
		sessionID: "session-a68-order",
		cfg:       session.cfg,
		seenDedup: map[string]struct{}{},
	}
	err = session2.ingestControlEvents(context.Background(), &eventCollector{}, 0, []types.RealtimeEventEnvelope{
		newRealtimeControlEvent("evt-1", "session-a68-order", "run-a68-order", 2, types.RealtimeEventTypeRequest, map[string]any{"input": "x"}),
		newRealtimeControlEvent("evt-2", "session-a68-order", "run-a68-order", 1, types.RealtimeEventTypeDelta, map[string]any{"delta": "y"}),
	})
	if err == nil || !strings.Contains(err.Error(), "out of order") {
		t.Fatalf("expected out-of-order error, got %v", err)
	}
	if session2.lastErrorCode != realtimeReasonEventOrderDrift {
		t.Fatalf("lastErrorCode=%q, want %q", session2.lastErrorCode, realtimeReasonEventOrderDrift)
	}
}

func TestRealtimeResumeBoundaryStableWithContextJITSwapBackTiering(t *testing.T) {
	dir := t.TempDir()
	spillPath := filepath.Join(dir, "spill.jsonl")
	writeRealtimeSpillFixture(t, spillPath, "run-a68-ctx-boundary")
	mgr := newRealtimeRuntimeManagerWithContextJITForTest(t, "BAYMAX_A68_CTX_BOUNDARY", spillPath)

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				FinalAnswer: "done",
				Usage:       types.TokenUsage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
			}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type:      types.ModelEventTypeOutputTextDelta,
				TextDelta: "done",
			})
		},
	}
	runEngine := New(runModel, WithRuntimeManager(mgr))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))
	req := types.RunRequest{
		RunID:     "run-a68-ctx-boundary",
		SessionID: "session-a68-ctx-boundary",
		Input:     "resume boundary status",
		Realtime: &types.RealtimeRunRequest{
			Events: []types.RealtimeEventEnvelope{
				newRealtimeControlEvent("interrupt-ctx-1", "session-a68-ctx-boundary", "run-a68-ctx-boundary", 1, types.RealtimeEventTypeInterrupt, map[string]any{}),
				newRealtimeControlEvent("interrupt-ctx-1", "session-a68-ctx-boundary", "run-a68-ctx-boundary", 1, types.RealtimeEventTypeInterrupt, map[string]any{}),
				newRealtimeControlEvent("resume-ctx-1", "session-a68-ctx-boundary", "run-a68-ctx-boundary", 2, types.RealtimeEventTypeResume, map[string]any{
					"cursor": "session-a68-ctx-boundary:run-a68-ctx-boundary:1",
				}),
				newRealtimeControlEvent("resume-ctx-1", "session-a68-ctx-boundary", "run-a68-ctx-boundary", 2, types.RealtimeEventTypeResume, map[string]any{
					"cursor": "session-a68-ctx-boundary:run-a68-ctx-boundary:1",
				}),
			},
		},
	}

	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runResult, runErr := runEngine.Run(context.Background(), req, runCollector)
	if runErr != nil {
		t.Fatalf("Run failed: %v", runErr)
	}
	streamResult, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)
	if streamErr != nil {
		t.Fatalf("Stream failed: %v", streamErr)
	}
	if runResult.FinalAnswer != "done" || streamResult.FinalAnswer != "done" {
		t.Fatalf("run/stream final answer mismatch run=%q stream=%q", runResult.FinalAnswer, streamResult.FinalAnswer)
	}

	runPayload := lastRunFinishedPayloadFromCollector(t, runCollector)
	streamPayload := lastRunFinishedPayloadFromCollector(t, streamCollector)
	assertRealtimePayloadCounters(t, runPayload)
	assertRealtimePayloadCounters(t, streamPayload)
	if runPayload["realtime_resume_source"] != streamPayload["realtime_resume_source"] {
		t.Fatalf("resume source mismatch run=%#v stream=%#v", runPayload["realtime_resume_source"], streamPayload["realtime_resume_source"])
	}
	if runPayload["reason_code"] != streamPayload["reason_code"] {
		t.Fatalf("reason_code mismatch run=%#v stream=%#v", runPayload["reason_code"], streamPayload["reason_code"])
	}
	if runPayload["realtime_error_layer"] != streamPayload["realtime_error_layer"] {
		t.Fatalf("realtime_error_layer mismatch run=%#v stream=%#v", runPayload["realtime_error_layer"], streamPayload["realtime_error_layer"])
	}
	if runPayload["context_lifecycle_tier_stats"] == nil || streamPayload["context_lifecycle_tier_stats"] == nil {
		t.Fatalf("context lifecycle tier stats should be emitted with context-jit enabled, run=%#v stream=%#v", runPayload["context_lifecycle_tier_stats"], streamPayload["context_lifecycle_tier_stats"])
	}
}

func newRealtimeRuntimeManagerForTest(t *testing.T, envPrefix string) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
runtime:
  realtime:
    protocol:
      enabled: true
      version: realtime_event_protocol.v1
      max_buffered_events: 128
    interrupt_resume:
      enabled: true
      resume_cursor_ttl_ms: 300000
      idempotency_window_ms: 120000
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: envPrefix,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func newRealtimeRuntimeManagerWithContextJITForTest(t *testing.T, envPrefix string, spillPath string) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-context-jit.yaml")
	cfg := fmt.Sprintf(`
runtime:
  realtime:
    protocol:
      enabled: true
      version: realtime_event_protocol.v1
      max_buffered_events: 128
    interrupt_resume:
      enabled: true
      resume_cursor_ttl_ms: 300000
      idempotency_window_ms: 120000
  context:
    jit:
      swap_back:
        enabled: true
        min_relevance_score: 0.20
      lifecycle_tiering:
        enabled: true
        hot_ttl_ms: 1000
        warm_ttl_ms: 2000
        cold_ttl_ms: 5000
context_assembler:
  enabled: true
  journal_path: '%s'
  ca2:
    enabled: false
  ca3:
    enabled: true
    max_context_tokens: 4096
    percent_thresholds:
      safe: 10
      comfort: 20
      warning: 30
      danger: 40
      emergency: 50
    absolute_thresholds:
      safe: 400
      comfort: 800
      warning: 1200
      danger: 1600
      emergency: 2000
    spill:
      enabled: true
      backend: file
      path: '%s'
      swap_back_limit: 8
`, filepath.ToSlash(filepath.Join(filepath.Dir(spillPath), "journal.jsonl")), filepath.ToSlash(spillPath))
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: envPrefix,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func writeRealtimeSpillFixture(t *testing.T, spillPath string, runID string) {
	t.Helper()
	rec := map[string]any{
		"run_id":        runID,
		"stage":         "stage1",
		"origin_ref":    "realtime-context-ref",
		"content":       "resume boundary evidence context",
		"evidence_tags": []string{"resume", "boundary"},
		"spilled_at":    time.Now().Add(-2500 * time.Millisecond).UTC(),
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal spill fixture: %v", err)
	}
	if err := os.WriteFile(spillPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write spill fixture: %v", err)
	}
}

func newRealtimeControlEvent(
	eventID string,
	sessionID string,
	runID string,
	seq int64,
	typ types.RealtimeEventType,
	payload map[string]any,
) types.RealtimeEventEnvelope {
	return types.RealtimeEventEnvelope{
		EventID:   eventID,
		SessionID: sessionID,
		RunID:     runID,
		Seq:       seq,
		Type:      typ,
		TS:        time.Now().UTC(),
		Payload:   payload,
	}
}

func lastRunFinishedPayloadFromCollector(t *testing.T, collector *eventCollector) map[string]any {
	t.Helper()
	last, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing non-timeline event")
	}
	if last.Type != "run.finished" {
		t.Fatalf("last event type=%q, want run.finished", last.Type)
	}
	if len(last.Payload) == 0 {
		t.Fatal("run.finished payload is empty")
	}
	return last.Payload
}

func assertRealtimePayloadCounters(t *testing.T, payload map[string]any) {
	t.Helper()
	if payload["realtime_protocol_version"] != runtimeconfig.RuntimeRealtimeProtocolVersionV1 {
		t.Fatalf("realtime_protocol_version=%#v, want %q", payload["realtime_protocol_version"], runtimeconfig.RuntimeRealtimeProtocolVersionV1)
	}
	if payload["realtime_interrupt_total"] != 1 {
		t.Fatalf("realtime_interrupt_total=%#v, want 1", payload["realtime_interrupt_total"])
	}
	if payload["realtime_resume_total"] != 1 {
		t.Fatalf("realtime_resume_total=%#v, want 1", payload["realtime_resume_total"])
	}
	if payload["realtime_idempotency_dedup_total"] != 2 {
		t.Fatalf("realtime_idempotency_dedup_total=%#v, want 2", payload["realtime_idempotency_dedup_total"])
	}
}

func assertCollectorContainsCanonicalRealtimeEnvelope(t *testing.T, collector *eventCollector) {
	t.Helper()
	collector.mu.Lock()
	defer collector.mu.Unlock()
	foundRealtime := 0
	for i := range collector.evs {
		ev := collector.evs[i]
		if ev.Type != realtimeRunnerEventType {
			continue
		}
		foundRealtime++
		raw, err := json.Marshal(ev.Payload)
		if err != nil {
			t.Fatalf("marshal realtime payload failed: %v", err)
		}
		if _, err := types.ParseRealtimeEventEnvelope(raw); err != nil {
			t.Fatalf("realtime payload is not canonical envelope: %v payload=%s", err, string(raw))
		}
	}
	if foundRealtime == 0 {
		t.Fatal("expected at least one realtime.event envelope")
	}
}
