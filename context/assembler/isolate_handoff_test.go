package assembler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestIngestIsolateHandoffChunksSummaryAndReferences(t *testing.T) {
	now := time.Now().UTC()
	evidence := "evidence-context"
	_, locator := referenceIdentity(evidence, "file")
	handoff := types.IsolateHandoffPayload{
		Summary: "child-summary",
		Artifacts: []types.IsolateHandoffArtifact{
			{ID: "artifact-1", Type: "markdown", Locator: "artifact://memo-1", Body: "full-body-content"},
		},
		EvidenceRefs: []types.ContextReference{
			{ID: "ref-evidence", Type: "stage2_chunk", Locator: locator, Source: "file"},
		},
		Confidence: 0.95,
		TTL:        now.Add(2 * time.Minute).UnixMilli(),
	}
	raw, err := json.Marshal(handoff)
	if err != nil {
		t.Fatalf("marshal handoff payload: %v", err)
	}
	chunks, payload, err := ingestIsolateHandoffChunks(
		[]string{string(raw), evidence},
		"file",
		now,
		runtimeconfig.RuntimeContextJITIsolateHandoffConfig{
			Enabled:       true,
			DefaultTTLMS:  300000,
			MinConfidence: 0.60,
		},
		"best_effort",
	)
	if err != nil {
		t.Fatalf("ingest isolate handoff failed: %v", err)
	}
	if payload.AcceptedTotal != 1 || payload.RejectedTotal != 0 {
		t.Fatalf("unexpected handoff ingest stats: %#v", payload)
	}
	if len(payload.Handoffs) != 1 {
		t.Fatalf("handoffs len = %d, want 1", len(payload.Handoffs))
	}
	if len(payload.Handoffs[0].Artifacts) != 1 {
		t.Fatalf("handoff artifacts len = %d, want 1", len(payload.Handoffs[0].Artifacts))
	}
	if payload.Handoffs[0].Artifacts[0].Body != "" {
		t.Fatalf("handoff artifacts body must be deferred, got %q", payload.Handoffs[0].Artifacts[0].Body)
	}
	if len(chunks) != 2 || chunks[0] != "child-summary" || chunks[1] != evidence {
		t.Fatalf("unexpected ingested chunks: %#v", chunks)
	}
}

func TestIngestIsolateHandoffValidation(t *testing.T) {
	now := time.Now().UTC()
	t.Run("fail_fast_ttl_expired", func(t *testing.T) {
		payload := types.IsolateHandoffPayload{
			Summary:    "summary",
			Artifacts:  []types.IsolateHandoffArtifact{},
			Confidence: 0.9,
			TTL:        now.Add(-1 * time.Second).UnixMilli(),
		}
		raw, _ := json.Marshal(payload)
		_, _, err := ingestIsolateHandoffChunks(
			[]string{string(raw)},
			"file",
			now,
			runtimeconfig.RuntimeContextJITIsolateHandoffConfig{
				Enabled:       true,
				DefaultTTLMS:  300000,
				MinConfidence: 0.60,
			},
			"fail_fast",
		)
		if err == nil || !strings.Contains(err.Error(), isolateHandoffRejectTTLExpired) {
			t.Fatalf("expected ttl_expired error, got %v", err)
		}
	})

	t.Run("best_effort_confidence_too_low", func(t *testing.T) {
		payload := types.IsolateHandoffPayload{
			Summary:    "summary",
			Artifacts:  []types.IsolateHandoffArtifact{},
			Confidence: 0.2,
			TTL:        now.Add(1 * time.Minute).UnixMilli(),
		}
		raw, _ := json.Marshal(payload)
		chunks, ingested, err := ingestIsolateHandoffChunks(
			[]string{string(raw)},
			"file",
			now,
			runtimeconfig.RuntimeContextJITIsolateHandoffConfig{
				Enabled:       true,
				DefaultTTLMS:  300000,
				MinConfidence: 0.60,
			},
			"best_effort",
		)
		if err != nil {
			t.Fatalf("best_effort should not fail, got %v", err)
		}
		if len(chunks) != 0 {
			t.Fatalf("rejected handoff should be dropped from summary path, chunks=%#v", chunks)
		}
		if ingested.RejectedTotal != 1 || !containsString(ingested.RejectedReasons, isolateHandoffRejectConfidenceTooLow) {
			t.Fatalf("unexpected rejected reasons: %#v", ingested.RejectedReasons)
		}
	})

	t.Run("best_effort_missing_evidence_ref", func(t *testing.T) {
		payload := types.IsolateHandoffPayload{
			Summary:   "summary",
			Artifacts: []types.IsolateHandoffArtifact{},
			EvidenceRefs: []types.ContextReference{
				{ID: "ref-missing", Type: "stage2_chunk", Locator: "stage2://file/not-found"},
			},
			Confidence: 0.9,
			TTL:        now.Add(1 * time.Minute).UnixMilli(),
		}
		raw, _ := json.Marshal(payload)
		chunks, ingested, err := ingestIsolateHandoffChunks(
			[]string{string(raw), "other"},
			"file",
			now,
			runtimeconfig.RuntimeContextJITIsolateHandoffConfig{
				Enabled:       true,
				DefaultTTLMS:  300000,
				MinConfidence: 0.60,
			},
			"best_effort",
		)
		if err != nil {
			t.Fatalf("best_effort should not fail, got %v", err)
		}
		if len(chunks) != 1 || chunks[0] != "other" {
			t.Fatalf("missing-evidence handoff should be skipped, got chunks=%#v", chunks)
		}
		if ingested.RejectedTotal != 1 || !containsString(ingested.RejectedReasons, isolateHandoffRejectEvidenceRefMissing) {
			t.Fatalf("unexpected rejected reasons: %#v", ingested.RejectedReasons)
		}
	})
}

func TestAssemblerCA2IsolateHandoffDefaultConsumption(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = runtimeconfig.ContextStage2ProviderFile
	cfg.CA2.Routing.MinInputChars = 1

	evidence := "evidence-context"
	_, locator := referenceIdentity(evidence, "file")
	body := "full-artifact-body"
	handoff := types.IsolateHandoffPayload{
		Summary: "child-summary-default-consume",
		Artifacts: []types.IsolateHandoffArtifact{
			{ID: "artifact-1", Type: "markdown", Locator: "artifact://memo-1", Body: body},
		},
		EvidenceRefs: []types.ContextReference{
			{ID: "ref-evidence", Type: "stage2_chunk", Locator: locator, Source: "file"},
		},
		Confidence: 0.95,
		TTL:        time.Now().Add(2 * time.Minute).UnixMilli(),
	}
	raw, err := json.Marshal(handoff)
	if err != nil {
		t.Fatalf("marshal handoff payload: %v", err)
	}
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"session-1","content":` + strconvQuote(string(raw)) + `}`,
		`{"session_id":"session-1","content":"` + evidence + `"}`,
	}, "\n")
	if err := os.WriteFile(stage2File, []byte(content), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File

	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.IsolateHandoff.Enabled = true
	runtimeCtx.JIT.IsolateHandoff.DefaultTTLMS = 300000
	runtimeCtx.JIT.IsolateHandoff.MinConfidence = 0.60
	runtimeCtx.JIT.ReferenceFirst.Enabled = true
	runtimeCtx.JIT.ReferenceFirst.MaxRefs = 8
	runtimeCtx.JIT.ReferenceFirst.MaxResolveTokens = 2048

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return runtimeCtx
		}),
	)
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-handoff-default-consume",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "lookup context",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-handoff-default-consume",
		Input:    "lookup context",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	if result.Stage.ContextRefDiscoverCount <= 0 || result.Stage.ContextRefResolveCount <= 0 {
		t.Fatalf("reference-first counters should be populated, stage=%#v", result.Stage)
	}

	joined := joinMessageContents(outReq.Messages)
	if !strings.Contains(joined, "stage2_isolate_handoff:") {
		t.Fatalf("missing stage2 isolate handoff payload in messages: %#v", outReq.Messages)
	}
	if strings.Contains(joined, body) {
		t.Fatalf("handoff artifact body should be deferred from model input, got messages=%#v", outReq.Messages)
	}
	if !strings.Contains(joined, "child-summary-default-consume") {
		t.Fatalf("summary should be consumed into stage2 context, got messages=%#v", outReq.Messages)
	}
}

func TestAssemblerCA2IsolateHandoffReplayIdempotent(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = runtimeconfig.ContextStage2ProviderFile
	cfg.CA2.Routing.MinInputChars = 1
	evidence := "evidence-context"
	_, locator := referenceIdentity(evidence, "file")
	handoff := types.IsolateHandoffPayload{
		Summary: "child-summary-idempotent",
		Artifacts: []types.IsolateHandoffArtifact{
			{ID: "artifact-1", Type: "markdown", Locator: "artifact://memo-1", Body: "deferred"},
		},
		EvidenceRefs: []types.ContextReference{
			{ID: "ref-evidence", Type: "stage2_chunk", Locator: locator, Source: "file"},
		},
		Confidence: 0.95,
		TTL:        time.Now().Add(2 * time.Minute).UnixMilli(),
	}
	raw, err := json.Marshal(handoff)
	if err != nil {
		t.Fatalf("marshal handoff payload: %v", err)
	}
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"session-1","content":` + strconvQuote(string(raw)) + `}`,
		`{"session_id":"session-1","content":"` + evidence + `"}`,
	}, "\n")
	if err := os.WriteFile(stage2File, []byte(content), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File

	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.IsolateHandoff.Enabled = true
	runtimeCtx.JIT.ReferenceFirst.Enabled = true
	runtimeCtx.JIT.ReferenceFirst.MaxRefs = 8
	runtimeCtx.JIT.ReferenceFirst.MaxResolveTokens = 2048

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return runtimeCtx
		}),
	)
	req := types.ContextAssembleRequest{
		RunID:         "run-handoff-replay",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "lookup context",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}
	modelReq := types.ModelRequest{
		RunID:    "run-handoff-replay",
		Input:    "lookup context",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	}
	firstReq, firstResult, err := a.Assemble(context.Background(), req, modelReq)
	if err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	secondReq, secondResult, err := a.Assemble(context.Background(), req, modelReq)
	if err != nil {
		t.Fatalf("second assemble failed: %v", err)
	}
	if firstResult.Stage.Status != secondResult.Stage.Status ||
		firstResult.Stage.ContextRefDiscoverCount != secondResult.Stage.ContextRefDiscoverCount ||
		firstResult.Stage.ContextRefResolveCount != secondResult.Stage.ContextRefResolveCount {
		t.Fatalf("handoff replay stage drift first=%#v second=%#v", firstResult.Stage, secondResult.Stage)
	}
	if countMessagePrefix(firstReq.Messages, "stage2_isolate_handoff:") != 1 ||
		countMessagePrefix(secondReq.Messages, "stage2_isolate_handoff:") != 1 {
		t.Fatalf("handoff message should remain idempotent, first=%#v second=%#v", firstReq.Messages, secondReq.Messages)
	}
	firstStage2 := firstMessageByPrefix(firstReq.Messages, "stage2_context:\n")
	secondStage2 := firstMessageByPrefix(secondReq.Messages, "stage2_context:\n")
	if firstStage2 != secondStage2 {
		t.Fatalf("stage2 context replay drift first=%q second=%q", firstStage2, secondStage2)
	}
}

func joinMessageContents(messages []types.Message) string {
	if len(messages) == 0 {
		return ""
	}
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		parts = append(parts, msg.Content)
	}
	return strings.Join(parts, "\n")
}

func countMessagePrefix(messages []types.Message, prefix string) int {
	count := 0
	for _, msg := range messages {
		if strings.HasPrefix(msg.Content, prefix) {
			count++
		}
	}
	return count
}

func firstMessageByPrefix(messages []types.Message, prefix string) string {
	for _, msg := range messages {
		if strings.HasPrefix(msg.Content, prefix) {
			return msg.Content
		}
	}
	return ""
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func strconvQuote(in string) string {
	raw, _ := json.Marshal(in)
	return string(raw)
}
