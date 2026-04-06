package assembler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/context/journal"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	tiktoken "github.com/pkoukk/tiktoken-go"
)

const semanticPrefixVersion = "context-prefix-and-journal-baseline"

func TestAssemblerStablePrefixHashWithinSessionVersion(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Messages:      []types.Message{{Role: "system", Content: "stable"}},
	}
	modelReq := types.ModelRequest{RunID: req.RunID, Messages: req.Messages}

	_, r1, err := a.Assemble(context.Background(), req, modelReq)
	if err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	_, r2, err := a.Assemble(context.Background(), req, modelReq)
	if err != nil {
		t.Fatalf("second assemble failed: %v", err)
	}
	if r1.Prefix.PrefixHash == "" || r1.Prefix.PrefixHash != r2.Prefix.PrefixHash {
		t.Fatalf("prefix hash mismatch: %q vs %q", r1.Prefix.PrefixHash, r2.Prefix.PrefixHash)
	}
}

func TestAssemblerFailFastOnPrefixDrift(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	base := types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Messages:      []types.Message{{Role: "system", Content: "stable"}},
	}

	if _, _, err := a.Assemble(context.Background(), base, types.ModelRequest{RunID: base.RunID, Messages: base.Messages}); err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	drift := base
	drift.Messages = []types.Message{{Role: "system", Content: "changed"}}
	_, result, err := a.Assemble(context.Background(), drift, types.ModelRequest{RunID: drift.RunID, Messages: drift.Messages})
	if err == nil {
		t.Fatal("expected fail-fast guard error")
	}
	if result.GuardFailure != "hash.prefix.drift" {
		t.Fatalf("guard failure = %q, want hash.prefix.drift", result.GuardFailure)
	}
}

func TestAssemblerRejectsDBBackendPlaceholder(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Storage.Backend = "db"
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
	}, types.ModelRequest{RunID: "run-1"})
	if !errors.Is(err, journal.ErrBackendNotReady) {
		t.Fatalf("err = %v, want ErrBackendNotReady", err)
	}
}

func TestAssemblerContextStage2RoutesToStage2ByKeyword(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"external-ctx"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.TriggerKeywords = []string{"lookup"}
	cfg.CA2.Routing.MinInputChars = 9999
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "please lookup details",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}
	outReq, result, err := a.Assemble(context.Background(), req, types.ModelRequest{
		RunID:    req.RunID,
		Input:    req.Input,
		Messages: req.Messages,
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	found := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "external-ctx") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stage2 context not appended: %#v", outReq.Messages)
	}
}

func TestAssemblerContextStage2AgenticCallbackRunStage2(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.RoutingMode = "agentic"
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"agentic-ctx"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 9999
	cfg.CA2.Routing.TriggerKeywords = nil

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithAgenticRouter(AgenticRouterFunc(func(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error) {
			if req.SessionID != "session-1" {
				t.Fatalf("session_id = %q, want session-1", req.SessionID)
			}
			return AgenticRoutingDecision{RunStage2: true, Reason: "agentic.force_stage2"}, nil
		})),
	)

	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-agentic-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "short",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-agentic-1",
		Input:    "short",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	if result.Stage.Stage2RouterMode != "agentic" || result.Stage.Stage2RouterDecision != "run_stage2" {
		t.Fatalf("router mode/decision mismatch: %#v", result.Stage)
	}
	if result.Stage.Stage2RouterReason != "agentic.force_stage2" {
		t.Fatalf("router reason = %q, want agentic.force_stage2", result.Stage.Stage2RouterReason)
	}
	if result.Stage.Stage2RouterError != "" {
		t.Fatalf("router error = %q, want empty", result.Stage.Stage2RouterError)
	}
	found := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "agentic-ctx") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stage2 context not appended: %#v", outReq.Messages)
	}
}

func TestAssemblerContextStage2AgenticCallbackSkipStage2(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.RoutingMode = "agentic"
	cfg.CA2.Routing.MinInputChars = 1

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithAgenticRouter(AgenticRouterFunc(func(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error) {
			return AgenticRoutingDecision{RunStage2: false, Reason: "agentic.low_value"}, nil
		})),
	)

	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-agentic-2",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "lookup please",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-agentic-2",
		Input:    "lookup please",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage1Only {
		t.Fatalf("stage status = %q, want stage1_only", result.Stage.Status)
	}
	if result.Stage.Stage2SkipReason != "routing.agentic.skip" {
		t.Fatalf("stage2 skip reason = %q, want routing.agentic.skip", result.Stage.Stage2SkipReason)
	}
	if result.Stage.Stage2RouterDecision != "skip_stage2" || result.Stage.Stage2RouterReason != "agentic.low_value" {
		t.Fatalf("router decision/reason mismatch: %#v", result.Stage)
	}
}

func TestAssemblerContextStage2AgenticCallbackFallbackClasses(t *testing.T) {
	type tc struct {
		name           string
		router         AgenticRouter
		input          string
		minInputChars  int
		timeout        time.Duration
		wantDecision   string
		wantSkipReason string
		wantErrCode    string
		wantContains   string
	}
	cases := []tc{
		{
			name:           "missing_callback",
			router:         nil,
			input:          "x",
			minInputChars:  9999,
			wantDecision:   "skip_stage2",
			wantSkipReason: "routing.threshold.not_met",
			wantErrCode:    "agentic.callback_missing",
			wantContains:   "agentic.fallback.agentic.callback_missing",
		},
		{
			name:  "timeout_callback",
			input: "lookup",
			router: AgenticRouterFunc(func(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error) {
				<-ctx.Done()
				return AgenticRoutingDecision{}, ctx.Err()
			}),
			minInputChars:  1,
			timeout:        5 * time.Millisecond,
			wantDecision:   "run_stage2",
			wantSkipReason: "",
			wantErrCode:    "agentic.callback_timeout",
			wantContains:   "agentic.fallback.agentic.callback_timeout",
		},
		{
			name:  "error_callback",
			input: "x",
			router: AgenticRouterFunc(func(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error) {
				return AgenticRoutingDecision{}, errors.New("boom")
			}),
			minInputChars:  9999,
			wantDecision:   "skip_stage2",
			wantSkipReason: "routing.threshold.not_met",
			wantErrCode:    "agentic.callback_error",
			wantContains:   "agentic.fallback.agentic.callback_error",
		},
		{
			name:  "invalid_decision",
			input: "x",
			router: AgenticRouterFunc(func(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error) {
				return AgenticRoutingDecision{RunStage2: true, Reason: ""}, nil
			}),
			minInputChars:  9999,
			wantDecision:   "skip_stage2",
			wantSkipReason: "routing.threshold.not_met",
			wantErrCode:    "agentic.invalid_decision",
			wantContains:   "agentic.fallback.agentic.invalid_decision",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			cfg := runtimeconfig.DefaultConfig().ContextAssembler
			cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
			cfg.CA2.Enabled = true
			cfg.CA2.RoutingMode = "agentic"
			cfg.CA2.Routing.MinInputChars = c.minInputChars
			cfg.CA2.Routing.TriggerKeywords = nil
			cfg.CA2.Routing.RequireSystemGuard = false
			cfg.CA2.Stage2.Provider = "file"
			stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
			if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"fallback-ctx"}`), 0o600); err != nil {
				t.Fatalf("write stage2 file: %v", err)
			}
			cfg.CA2.Stage2.FilePath = stage2File
			if c.timeout > 0 {
				cfg.CA2.Agentic.DecisionTimeout = c.timeout
			}

			var a *Assembler
			if c.router != nil {
				a = New(func() runtimeconfig.ContextAssemblerConfig { return cfg }, WithAgenticRouter(c.router))
			} else {
				a = New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
			}

			_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
				RunID:         "run-" + c.name,
				SessionID:     "session-1",
				PrefixVersion: semanticPrefixVersion,
				Input:         c.input,
				Messages:      []types.Message{{Role: "system", Content: "s"}},
			}, types.ModelRequest{
				RunID:    "run-" + c.name,
				Input:    c.input,
				Messages: []types.Message{{Role: "system", Content: "s"}},
			})
			if err != nil {
				t.Fatalf("Assemble failed: %v", err)
			}
			if result.Stage.Stage2RouterMode != "agentic" {
				t.Fatalf("router mode = %q, want agentic", result.Stage.Stage2RouterMode)
			}
			if result.Stage.Stage2RouterDecision != c.wantDecision {
				t.Fatalf("router decision = %q, want %q", result.Stage.Stage2RouterDecision, c.wantDecision)
			}
			if result.Stage.Stage2SkipReason != c.wantSkipReason {
				t.Fatalf("stage2 skip reason = %q, want %q", result.Stage.Stage2SkipReason, c.wantSkipReason)
			}
			if result.Stage.Stage2RouterError != c.wantErrCode {
				t.Fatalf("router error = %q, want %q", result.Stage.Stage2RouterError, c.wantErrCode)
			}
			if !strings.Contains(result.Stage.Stage2RouterReason, c.wantContains) {
				t.Fatalf("router reason = %q, want contains %q", result.Stage.Stage2RouterReason, c.wantContains)
			}
		})
	}
}

func TestAssemblerContextStage2Stage2BestEffort(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "rag"
	cfg.CA2.StagePolicy.Stage2 = "best_effort"
	cfg.CA2.Routing.MinInputChars = 1
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "x",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "x",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble should continue in best_effort: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusDegraded {
		t.Fatalf("stage status = %q, want degraded", result.Stage.Status)
	}
	if result.Stage.Stage2ReasonCode == "" || result.Stage.Stage2ErrorLayer == "" {
		t.Fatalf("expected stage2 layered error fields, got %#v", result.Stage)
	}
}

func TestAssemblerContextStage2Stage2FailFast(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "db"
	cfg.CA2.StagePolicy.Stage2 = "fail_fast"
	cfg.CA2.Routing.MinInputChars = 1
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "x",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "x",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err == nil {
		t.Fatal("expected fail_fast error, got nil")
	}
	if !strings.Contains(err.Error(), "endpoint is required") {
		t.Fatalf("err = %v, want endpoint required", err)
	}
}

func TestAssemblerContextStage2MemoryFallbackPolicyStageSemantics(t *testing.T) {
	type tc struct {
		name           string
		stagePolicy    string
		fallbackPolicy string
		wantErr        bool
		wantStatus     types.AssembleStageStatus
		wantSkipReason string
		wantReasonCode string
	}
	cases := []tc{
		{
			name:           "fail_fast_stage_and_fail_fast_memory_fallback_returns_error",
			stagePolicy:    "fail_fast",
			fallbackPolicy: runtimeconfig.RuntimeMemoryFallbackPolicyFailFast,
			wantErr:        true,
		},
		{
			name:           "best_effort_stage_and_fail_fast_memory_fallback_degrades",
			stagePolicy:    "best_effort",
			fallbackPolicy: runtimeconfig.RuntimeMemoryFallbackPolicyFailFast,
			wantErr:        false,
			wantStatus:     types.AssembleStageStatusDegraded,
			wantSkipReason: "stage2.fetch.failed",
		},
		{
			name:           "fail_fast_stage_and_degrade_without_memory_fallback_keeps_stage1_only",
			stagePolicy:    "fail_fast",
			fallbackPolicy: runtimeconfig.RuntimeMemoryFallbackPolicyDegradeWithoutMemory,
			wantErr:        false,
			wantStatus:     types.AssembleStageStatusStage1Only,
			wantSkipReason: "stage2.empty",
			wantReasonCode: "memory.fallback.used",
		},
		{
			name:           "best_effort_stage_and_degrade_without_memory_fallback_keeps_stage1_only",
			stagePolicy:    "best_effort",
			fallbackPolicy: runtimeconfig.RuntimeMemoryFallbackPolicyDegradeWithoutMemory,
			wantErr:        false,
			wantStatus:     types.AssembleStageStatusStage1Only,
			wantSkipReason: "stage2.empty",
			wantReasonCode: "memory.fallback.used",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			cfg := runtimeconfig.DefaultConfig().ContextAssembler
			cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
			cfg.CA2.Enabled = true
			cfg.CA2.Stage2.Provider = runtimeconfig.ContextStage2ProviderMemory
			cfg.CA2.StagePolicy.Stage2 = c.stagePolicy
			cfg.CA2.Routing.MinInputChars = 1
			cfg.CA2.Timeout.Stage2 = 80 * time.Millisecond
			cfg.CA2.Stage2.External.Endpoint = "http://127.0.0.1:1"

			memoryCfg := runtimeconfig.DefaultConfig().Runtime.Memory
			memoryCfg.Mode = runtimeconfig.RuntimeMemoryModeExternalSPI
			memoryCfg.External.Provider = "mem0"
			memoryCfg.External.Profile = "mem0"
			memoryCfg.External.ContractVersion = runtimeconfig.RuntimeMemoryContractVersionV1
			memoryCfg.Fallback.Policy = c.fallbackPolicy
			memoryCfg.Builtin.RootDir = filepath.Join(t.TempDir(), "memory-store")

			a := New(
				func() runtimeconfig.ContextAssemblerConfig { return cfg },
				WithMemoryConfigProvider(func() runtimeconfig.RuntimeMemoryConfig { return memoryCfg }),
			)
			_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
				RunID:         "run-memory-stage-policy",
				SessionID:     "session-memory-stage-policy",
				PrefixVersion: semanticPrefixVersion,
				Input:         "lookup memory",
				Messages:      []types.Message{{Role: "system", Content: "s"}},
			}, types.ModelRequest{
				RunID:    "run-memory-stage-policy",
				Input:    "lookup memory",
				Messages: []types.Message{{Role: "system", Content: "s"}},
			})

			if c.wantErr {
				if err == nil {
					t.Fatal("expected fail_fast stage2 error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Assemble should succeed, err=%v", err)
			}
			if result.Stage.Status != c.wantStatus {
				t.Fatalf("stage status = %q, want %q", result.Stage.Status, c.wantStatus)
			}
			if result.Stage.Stage2SkipReason != c.wantSkipReason {
				t.Fatalf("stage2 skip reason = %q, want %q", result.Stage.Stage2SkipReason, c.wantSkipReason)
			}
			if c.wantReasonCode != "" && result.Stage.Stage2ReasonCode != c.wantReasonCode {
				t.Fatalf("stage2 reason code = %q, want %q", result.Stage.Stage2ReasonCode, c.wantReasonCode)
			}
			if c.stagePolicy == "best_effort" && c.fallbackPolicy == runtimeconfig.RuntimeMemoryFallbackPolicyFailFast {
				if result.Stage.Stage2ReasonCode == "" {
					t.Fatalf("best_effort + fail_fast fallback should preserve stage2 reason code, got %#v", result.Stage)
				}
			}
		})
	}
}

func TestAssemblerContextStage2RecapAppended(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Routing.MinInputChars = 9999
	cfg.CA2.Routing.TriggerKeywords = nil
	cfg.CA2.TailRecap.Enabled = true
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "short",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "short",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Recap.Status != types.RecapStatusAppended && result.Recap.Status != types.RecapStatusTruncated {
		t.Fatalf("recap status = %q, want appended/truncated", result.Recap.Status)
	}
	if result.Stage.ContextRecapSource != contextRecapSourceTaskAwareV1 {
		t.Fatalf("context recap source = %q, want %q", result.Stage.ContextRecapSource, contextRecapSourceTaskAwareV1)
	}
	if !containsRecapItem(result.Recap.Tail.Decisions, "stage_status=stage1_only") {
		t.Fatalf("task-aware recap should include stage status decision, got %#v", result.Recap.Tail.Decisions)
	}
	if !containsRecapItem(result.Recap.Tail.Todo, "review_stage2_skip_reason="+result.Stage.Stage2SkipReason) {
		t.Fatalf("task-aware recap should include skip-reason todo, got %#v", result.Recap.Tail.Todo)
	}
	if recapContainsStaticTemplate(result.Recap.Tail) {
		t.Fatalf("task-aware recap should not include static template phrases, got %#v", result.Recap.Tail)
	}
	last := outReq.Messages[len(outReq.Messages)-1].Content
	if !strings.HasPrefix(last, "tail_recap:") {
		t.Fatalf("tail recap message missing: %q", last)
	}
}

func TestBuildTaskAwareTailRecapStableOrdering(t *testing.T) {
	recap, source := buildTaskAwareTailRecap(runtimeconfig.DefaultConfig().ContextAssembler.CA2, types.AssembleStage{
		Status:                        types.AssembleStageStatusStage2Used,
		Stage2RouterMode:              "agentic",
		Stage2RouterDecision:          "run_stage2",
		Stage2Provider:                "file",
		Stage2ReasonCode:              "partial_missing_refs",
		ContextRefDiscoverCount:       5,
		ContextRefResolveCount:        3,
		ContextEditGateDecision:       contextEditGateDecisionDenyGainRatio,
		ContextSwapbackRelevanceScore: 0.8254,
		ContextLifecycleTierStats: map[string]int{
			"migrate_warm_to_cold": 2,
			"hot":                  1,
			"pruned":               4,
			"warm":                 3,
			"cold":                 5,
		},
	})

	if source != contextRecapSourceTaskAwareV1 {
		t.Fatalf("recap source = %q, want %q", source, contextRecapSourceTaskAwareV1)
	}
	if !containsRecapItem(recap.Decisions, "lifecycle_tiering=hot=1,warm=3,cold=5,pruned=4,migrate_warm_to_cold=2") {
		t.Fatalf("lifecycle tiering summary must be stable-ordered, got %#v", recap.Decisions)
	}
	if !containsRecapItem(recap.Todo, "resolve_missing_refs=2") {
		t.Fatalf("todo should include unresolved refs, got %#v", recap.Todo)
	}
	if recapContainsStaticTemplate(recap) {
		t.Fatalf("task-aware recap should not include static template phrases, got %#v", recap)
	}
}

func containsRecapItem(items []string, want string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == want {
			return true
		}
	}
	return false
}

func recapContainsStaticTemplate(recap types.TailRecap) bool {
	phrases := []string{
		"review_stage2_quality",
		"evaluate_agentic_routing_todo",
	}
	all := append(append(append([]string{}, recap.Decisions...), recap.Todo...), recap.Risks...)
	for _, item := range all {
		normalized := strings.ToLower(strings.TrimSpace(item))
		for _, phrase := range phrases {
			if strings.Contains(normalized, phrase) {
				return true
			}
		}
	}
	return false
}

func TestAssemblerContextStage2Stage2ContextRedacted(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"{\"access_token\":\"secret-token\"}"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 1
	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRedactionConfigProvider(func() runtimeconfig.SecurityRedactionConfig {
			return runtimeconfig.SecurityRedactionConfig{
				Enabled:  true,
				Strategy: runtimeconfig.SecurityRedactionKeyword,
				Keywords: []string{"token"},
			}
		}),
	)
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "lookup",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "lookup",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	foundMasked := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, `"access_token":"***"`) {
			foundMasked = true
			break
		}
	}
	if !foundMasked {
		t.Fatalf("expected redacted stage2 content, got %#v", outReq.Messages)
	}
}

func TestAssemblerContextStage2Stage2DiagnosticsFields(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"session-1","content":"ctx-a"}`,
		`{"session_id":"session-1","content":"ctx-b"}`,
	}, "\n")
	if err := os.WriteFile(stage2File, []byte(content), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 1

	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "lookup",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "lookup",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Stage2HitCount != 2 {
		t.Fatalf("stage2_hit_count = %d, want 2", result.Stage.Stage2HitCount)
	}
	if result.Stage.Stage2Source != "file" {
		t.Fatalf("stage2_source = %q, want file", result.Stage.Stage2Source)
	}
	if result.Stage.Stage2Reason != "ok" {
		t.Fatalf("stage2_reason = %q, want ok", result.Stage.Stage2Reason)
	}
	if result.Stage.Stage2ReasonCode != "ok" {
		t.Fatalf("stage2_reason_code = %q, want ok", result.Stage.Stage2ReasonCode)
	}
	if result.Stage.Stage2ErrorLayer != "" {
		t.Fatalf("stage2_error_layer = %q, want empty", result.Stage.Stage2ErrorLayer)
	}
	if result.Stage.Stage2Profile != "file" {
		t.Fatalf("stage2_profile = %q, want file", result.Stage.Stage2Profile)
	}
	if result.Stage.Stage2TemplateProfile != "file" {
		t.Fatalf("stage2_template_profile = %q, want file", result.Stage.Stage2TemplateProfile)
	}
	if result.Stage.Stage2TemplateResolutionSource != runtimeconfig.Stage2TemplateResolutionExplicitOnly {
		t.Fatalf(
			"stage2_template_resolution_source = %q, want %q",
			result.Stage.Stage2TemplateResolutionSource,
			runtimeconfig.Stage2TemplateResolutionExplicitOnly,
		)
	}
	if result.Stage.Stage2HintApplied {
		t.Fatalf("stage2_hint_applied = %v, want false", result.Stage.Stage2HintApplied)
	}
	if result.Stage.Stage2HintMismatchReason != "" {
		t.Fatalf("stage2_hint_mismatch_reason = %q, want empty", result.Stage.Stage2HintMismatchReason)
	}
}

func TestAssemblerContextStage2MemoryGovernanceDiagnosticsFields(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = runtimeconfig.ContextStage2ProviderMemory
	cfg.CA2.Routing.MinInputChars = 1

	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"session-memory","content":"ctx-memory-a"}`,
		`{"session_id":"session-memory","content":"ctx-memory-b"}`,
	}, "\n")
	if err := os.WriteFile(stage2File, []byte(content), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File

	memoryCfg := runtimeconfig.DefaultConfig().Runtime.Memory
	memoryCfg.Mode = runtimeconfig.RuntimeMemoryModeBuiltinFilesystem
	memoryCfg.Builtin.RootDir = filepath.Join(t.TempDir(), "memory-store")
	memoryCfg.Fallback.Policy = runtimeconfig.RuntimeMemoryFallbackPolicyFailFast
	memoryCfg.Scope.Default = runtimeconfig.RuntimeMemoryScopeSession
	memoryCfg.Scope.Allowed = []string{
		runtimeconfig.RuntimeMemoryScopeSession,
		runtimeconfig.RuntimeMemoryScopeProject,
		runtimeconfig.RuntimeMemoryScopeGlobal,
	}
	memoryCfg.Scope.AllowOverride = true
	memoryCfg.Scope.GlobalNamespace = "global"

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithMemoryConfigProvider(func() runtimeconfig.RuntimeMemoryConfig { return memoryCfg }),
	)
	t.Cleanup(func() {
		if closer, ok := a.stage2Provider.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	})
	for i := 0; i < 2; i++ {
		_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
			RunID:         "run-memory-governance",
			SessionID:     "session-memory",
			PrefixVersion: semanticPrefixVersion,
			Input:         "ctx-memory",
			Messages:      []types.Message{{Role: "system", Content: "s"}},
		}, types.ModelRequest{
			RunID:    "run-memory-governance",
			Input:    "ctx-memory",
			Messages: []types.Message{{Role: "system", Content: "s"}},
		})
		if err != nil {
			t.Fatalf("Assemble warmup #%d failed: %v", i+1, err)
		}
	}

	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-memory-governance",
		SessionID:     "session-memory",
		PrefixVersion: semanticPrefixVersion,
		Input:         "ctx-memory",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-memory-governance",
		Input:    "ctx-memory",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	if result.Stage.Stage2Source != runtimeconfig.ContextStage2ProviderMemory {
		t.Fatalf("stage2_source = %q, want memory", result.Stage.Stage2Source)
	}
	if result.Stage.MemoryScopeSelected == "" {
		t.Fatalf("memory_scope_selected should be populated, got %#v", result.Stage)
	}
	if result.Stage.MemoryBudgetUsed <= 0 {
		t.Fatalf("memory_budget_used = %d, want > 0", result.Stage.MemoryBudgetUsed)
	}
	if result.Stage.MemoryHits <= 0 {
		t.Fatalf("memory_hits = %d, want > 0", result.Stage.MemoryHits)
	}
	if len(result.Stage.MemoryRerankStats) == 0 {
		t.Fatalf("memory_rerank_stats should be populated, got %#v", result.Stage.MemoryRerankStats)
	}
}

func TestAssemblerContextPressureEmergencyRejectsLowPriorityStage2(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"ctx"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 1
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 100
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 1, Comfort: 2, Warning: 3, Danger: 4, Emergency: 5,
	}
	cfg.CA3.Emergency.RejectLowPriority = true
	cfg.CA3.Emergency.HighPriorityTokens = []string{"urgent"}

	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("x", 500),
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    strings.Repeat("x", 500),
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Stage2SkipReason != "ca3.emergency.low_priority_rejected" {
		t.Fatalf("stage2 skip reason = %q", result.Stage.Stage2SkipReason)
	}
	if result.Stage.PressureZone == "" {
		t.Fatalf("pressure zone should be populated: %#v", result.Stage)
	}
}

func TestAssemblerContextPressureProtectedMessagesNotPruned(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 40
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 4, Comfort: 8, Warning: 12, Danger: 16, Emergency: 20,
	}
	cfg.CA3.Prune.TargetPercent = 30
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: "critical: keep this message no matter what"},
		{Role: "user", Content: strings.Repeat("filler ", 80)},
	}
	outReq, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-2",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need trim", 10),
		Messages:      msgs,
	}, types.ModelRequest{
		RunID:    "run-2",
		Input:    strings.Repeat("need trim", 10),
		Messages: msgs,
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	foundCritical := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "critical: keep this message") {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Fatalf("critical message should not be pruned: %#v", outReq.Messages)
	}
}

func TestAssemblerContextPressureSpillIdempotentAcrossRetry(t *testing.T) {
	dir := t.TempDir()
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(dir, "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 80
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 8, Comfort: 16, Warning: 24, Danger: 32, Emergency: 40,
	}
	cfg.CA3.Spill.Path = filepath.Join(dir, "spill.jsonl")
	cfg.CA3.Spill.Backend = "file"
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "run-3",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("large ", 100),
		Messages: []types.Message{
			{Role: "system", Content: "base"},
			{Role: "user", Content: strings.Repeat("payload ", 120)},
		},
	}
	modelReq := types.ModelRequest{RunID: req.RunID, Input: req.Input, Messages: req.Messages}
	if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
		t.Fatalf("second assemble failed: %v", err)
	}
	raw, err := os.ReadFile(cfg.CA3.Spill.Path)
	if err != nil {
		t.Fatalf("read spill file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	seen := map[string]struct{}{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			t.Fatalf("duplicate spill line found, expected idempotent spill writes: %s", line)
		}
		seen[line] = struct{}{}
	}
}

func TestEstimateContextTokensReturnsPositiveEstimate(t *testing.T) {
	req := types.ModelRequest{
		Model: "gpt-4.1-mini",
		Input: "你好，Baymax context assembler token test",
		Messages: []types.Message{
			{Role: "user", Content: "请帮我总结这段内容"},
		},
	}
	got := estimateContextTokens(req)
	if got <= 0 {
		t.Fatalf("estimateContextTokens should return positive estimate, got=%d", got)
	}
}

func TestEstimateContextTokensByTiktokenGracefulFallback(t *testing.T) {
	req := types.ModelRequest{
		Model: "unknown-model-id",
		Input: "fallback should still work",
	}
	tk := estimateContextTokensByTiktoken(req)
	if tk < 0 {
		t.Fatalf("unexpected negative tiktoken estimate: %d", tk)
	}
}

type failingTokenCounter struct {
	calls int
}

func (f *failingTokenCounter) CountTokens(ctx context.Context, req types.ModelRequest) (int, error) {
	f.calls++
	return 0, errors.New("counting unsupported")
}

func TestResolveContextPressureThresholdsUsesStageOverride(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA3
	cfg.Stage2.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 25, Comfort: 45, Warning: 65, Danger: 80, Emergency: 95,
	}
	cfg.Stage2.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 25000, Comfort: 45000, Warning: 65000, Danger: 80000, Emergency: 95000,
	}
	p, a := resolvePressureThresholds(cfg, "stage2")
	if p.Warning != 65 || a.Warning != 65000 {
		t.Fatalf("stage2 override not applied: percent=%+v absolute=%+v", p, a)
	}
}

func TestEvaluateContextPressureZonePrefersHigherTrigger(t *testing.T) {
	percent := runtimeconfig.ContextAssemblerCA3Thresholds{Safe: 20, Comfort: 40, Warning: 60, Danger: 75, Emergency: 90}
	absolute := runtimeconfig.ContextAssemblerCA3Thresholds{Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50}
	zone, reason, trigger := evaluatePressureZone(15, 35, percent, absolute)
	if zone != pressureZoneWarning {
		t.Fatalf("zone=%s, want warning", zone)
	}
	if reason != "absolute_token_trigger" || trigger != string(pressureZoneWarning) {
		t.Fatalf("unexpected reason/trigger: reason=%s trigger=%s", reason, trigger)
	}
}

func TestCountContextTokensSmallDeltaSkipsProviderThenRefreshes(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA3
	cfg.Tokenizer.Mode = "sdk_preferred"
	cfg.Tokenizer.SmallDeltaTokens = 32
	cfg.Tokenizer.SDKRefreshInterval = time.Second
	req := types.ModelRequest{Input: "small delta input"}
	estimate := estimateContextTokens(req)
	counterCalls := 0
	tc := tokenCounterFunc(func(ctx context.Context, m types.ModelRequest) (int, error) {
		counterCalls++
		return 77, nil
	})
	now := time.Now()
	state := &pressureRunState{
		LastTokenEstimate:  estimate,
		LastTokenSignature: pressureTokenSignature(req),
		LastSDKCountAt:     now,
	}
	a := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	a.now = func() time.Time { return now }

	first := a.countContextTokens(context.Background(), types.ContextAssembleRequest{TokenCounter: tc}, req, cfg, state)
	if first != estimate {
		t.Fatalf("small-delta path should use estimate, got=%d want=%d", first, estimate)
	}
	if counterCalls != 0 {
		t.Fatalf("token counter should be skipped before refresh interval, calls=%d", counterCalls)
	}
	now = now.Add(2 * time.Second)
	second := a.countContextTokens(context.Background(), types.ContextAssembleRequest{TokenCounter: tc}, req, cfg, state)
	if second != 77 {
		t.Fatalf("after refresh interval should call token counter, got=%d", second)
	}
	if counterCalls != 1 {
		t.Fatalf("token counter calls=%d, want 1", counterCalls)
	}
}

func TestCountContextTokensFallbackDoesNotBlockOnTokenizerFailure(t *testing.T) {
	oldModelFn := encodingForModelFn
	oldGetFn := getEncodingFn
	oldEnc := tiktokenDefaultEnc
	oldErr := tiktokenDefaultErr
	t.Cleanup(func() {
		encodingForModelFn = oldModelFn
		getEncodingFn = oldGetFn
		tiktokenDefaultEnc = oldEnc
		tiktokenDefaultErr = oldErr
		tiktokenDefaultOnce = sync.Once{}
	})
	encodingForModelFn = func(model string) (*tiktoken.Tiktoken, error) {
		return nil, errors.New("no model encoding")
	}
	getEncodingFn = func(name string) (*tiktoken.Tiktoken, error) {
		return nil, errors.New("no default encoding")
	}
	tiktokenDefaultOnce = sync.Once{}
	tiktokenDefaultEnc = nil
	tiktokenDefaultErr = nil

	cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA3
	cfg.Tokenizer.Mode = "sdk_preferred"
	cfg.Tokenizer.SmallDeltaTokens = 0
	cfg.Tokenizer.SDKRefreshInterval = time.Millisecond
	req := types.ModelRequest{Input: "fallback path should still return estimate"}
	state := &pressureRunState{}
	a := New(func() runtimeconfig.ContextAssemblerConfig { return runtimeconfig.DefaultConfig().ContextAssembler })
	failing := &failingTokenCounter{}
	got := a.countContextTokens(context.Background(), types.ContextAssembleRequest{
		TokenCounter: failing,
	}, req, cfg, state)
	if got <= 0 {
		t.Fatalf("fallback estimate should be positive, got=%d", got)
	}
	if failing.calls == 0 {
		t.Fatal("provider counter should still be attempted before fallback")
	}
}

type tokenCounterFunc func(ctx context.Context, req types.ModelRequest) (int, error)

func (f tokenCounterFunc) CountTokens(ctx context.Context, req types.ModelRequest) (int, error) {
	return f(ctx, req)
}

type modelClientFunc struct {
	generate func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error)
}

type embeddingScorerFunc func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error)

type rerankerFunc func(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error)

func (f embeddingScorerFunc) Score(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
	return f(ctx, req)
}

func (f rerankerFunc) Rerank(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error) {
	return f(ctx, req)
}

func (m modelClientFunc) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	if m.generate == nil {
		return types.ModelResponse{}, nil
	}
	return m.generate(ctx, req)
}

func (m modelClientFunc) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func TestAssemblerContextPressureSemanticCompactionUsesModelClient(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 120
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 1, Comfort: 2, Warning: 3, Danger: 4, Emergency: 5,
	}
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.SemanticTimeout = 500 * time.Millisecond
	cfg.CA2.StagePolicy.Stage1 = "fail_fast"

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary"}, nil
		},
	}
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: strings.Repeat("long semantic content ", 24)},
	}
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-semantic-success",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 18),
		Messages:      msgs,
		ModelClient:   client,
	}, types.ModelRequest{
		RunID:    "run-semantic-success",
		Input:    strings.Repeat("need compact ", 18),
		Messages: msgs,
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.CompactionMode != "semantic" {
		t.Fatalf("compaction mode = %q, want semantic", result.Stage.CompactionMode)
	}
	if result.Stage.CompactionFallback {
		t.Fatal("compaction fallback should be false")
	}
	if result.Stage.CompactionQualityScore <= 0 {
		t.Fatalf("compaction quality score = %v, want > 0", result.Stage.CompactionQualityScore)
	}
	if strings.TrimSpace(result.Stage.CompactionQualityReason) == "" {
		t.Fatal("compaction quality reason should not be empty")
	}
	foundSummary := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "semantic-summary") {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Fatalf("semantic summary not found in output messages: %#v", outReq.Messages)
	}
}

func TestAssemblerContextPressureSemanticCompactionBestEffortFallback(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 120
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.SemanticTimeout = 500 * time.Millisecond
	cfg.CA2.StagePolicy.Stage1 = "best_effort"

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{}, errors.New("semantic unavailable")
		},
	}
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: strings.Repeat("long semantic content ", 24)},
	}
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-semantic-fallback",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 18),
		Messages:      msgs,
		ModelClient:   client,
	}, types.ModelRequest{
		RunID:    "run-semantic-fallback",
		Input:    strings.Repeat("need compact ", 18),
		Messages: msgs,
	})
	if err != nil {
		t.Fatalf("Assemble should fallback in best_effort, got error: %v", err)
	}
	if !result.Stage.CompactionFallback {
		t.Fatal("compaction fallback should be true")
	}
	if result.Stage.CompactionFallbackReason != "semantic_compaction_error" {
		t.Fatalf("fallback reason = %q, want semantic_compaction_error", result.Stage.CompactionFallbackReason)
	}
	foundTruncated := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "...[squashed]") {
			foundTruncated = true
			break
		}
	}
	if !foundTruncated {
		t.Fatalf("truncate fallback not observed: %#v", outReq.Messages)
	}
}

func TestAssemblerContextPressureSemanticCompactionFailFast(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 120
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA2.StagePolicy.Stage1 = "fail_fast"

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{}, errors.New("semantic failure")
		},
	}
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: strings.Repeat("long semantic content ", 24)},
	}
	_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-semantic-fail-fast",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 18),
		Messages:      msgs,
		ModelClient:   client,
	}, types.ModelRequest{
		RunID:    "run-semantic-fail-fast",
		Input:    strings.Repeat("need compact ", 18),
		Messages: msgs,
	})
	if err == nil {
		t.Fatal("expected fail_fast semantic compaction error")
	}
}

func TestAssemblerContextPressureSemanticCompactionQualityGateBestEffortFallback(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 120
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Quality.Threshold = 0.95
	cfg.CA3.Compaction.Evidence.Keywords = []string{"mustkeep"}
	cfg.CA2.StagePolicy.Stage1 = "best_effort"

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "summary dropped keyword"}, nil
		},
	}
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: strings.Repeat("mustkeep long semantic content ", 24)},
	}
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-semantic-quality-fallback",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 18),
		Messages:      msgs,
		ModelClient:   client,
	}, types.ModelRequest{
		RunID:    "run-semantic-quality-fallback",
		Input:    strings.Repeat("need compact ", 18),
		Messages: msgs,
	})
	if err != nil {
		t.Fatalf("Assemble should fallback in best_effort, got error: %v", err)
	}
	if !result.Stage.CompactionFallback {
		t.Fatal("compaction fallback should be true")
	}
	if result.Stage.CompactionFallbackReason != "quality_below_threshold" {
		t.Fatalf("fallback reason = %q, want quality_below_threshold", result.Stage.CompactionFallbackReason)
	}
	foundTruncated := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "...[squashed]") {
			foundTruncated = true
			break
		}
	}
	if !foundTruncated {
		t.Fatalf("truncate fallback not observed: %#v", outReq.Messages)
	}
}

func TestAssemblerContextPressureSemanticCompactionHybridScoreUsesCosineWeight(t *testing.T) {
	ca3cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA3
	ca3cfg.Compaction.Embedding.Enabled = true
	ca3cfg.Compaction.Embedding.Selector = "test"
	ca3cfg.Compaction.Embedding.Provider = "openai"
	ca3cfg.Compaction.Embedding.Model = "text-embedding-3-small"
	ca3cfg.Compaction.Embedding.Timeout = 300 * time.Millisecond
	ca3cfg.Compaction.Embedding.SimilarityMetric = "cosine"
	ca3cfg.Compaction.Embedding.RuleWeight = 0.7
	ca3cfg.Compaction.Embedding.EmbeddingWeight = 0.3
	ca3cfg.Squash.MaxContentRunes = 40
	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary with compacted context"}, nil
		},
	}
	scorer := embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
		if req.Provider != "openai" {
			t.Fatalf("provider = %q, want openai", req.Provider)
		}
		return 0.8, nil
	})
	compactor := &semanticCompactor{client: client, embedding: scorer}
	result, err := compactor.compact(context.Background(), pressureCompactionRequest{
		AssembleReq: types.ContextAssembleRequest{
			Input: "compact please",
		},
		ModelReq: types.ModelRequest{
			Model: "gpt-4.1-mini",
			Messages: []types.Message{
				{Role: "user", Content: strings.Repeat("alpha beta long semantic content ", 40)},
			},
		},
		Config:      ca3cfg,
		StagePolicy: "fail_fast",
	})
	if err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if result.EmbeddingStatus != "used" {
		t.Fatalf("embedding status = %q, want used", result.EmbeddingStatus)
	}
	if result.EmbeddingProvider != "openai" {
		t.Fatalf("embedding provider = %q, want openai", result.EmbeddingProvider)
	}
	if result.EmbeddingSimilarity <= 0 {
		t.Fatalf("embedding similarity = %v, want > 0", result.EmbeddingSimilarity)
	}
	if result.EmbeddingContribution <= 0 {
		t.Fatalf("embedding contribution = %v, want > 0", result.EmbeddingContribution)
	}
	if !strings.Contains(result.QualityReason, "embedding_cosine") {
		t.Fatalf("quality reason = %q, want embedding_cosine marker", result.QualityReason)
	}
}

func TestAssemblerContextPressureSemanticCompactionEmbeddingFailureBestEffortFallback(t *testing.T) {
	ca3cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA3
	ca3cfg.Compaction.Embedding.Enabled = true
	ca3cfg.Compaction.Embedding.Selector = "test"
	ca3cfg.Compaction.Embedding.Provider = "gemini"
	ca3cfg.Compaction.Embedding.Model = "gemini-embedding-001"
	ca3cfg.Compaction.Embedding.Timeout = 300 * time.Millisecond
	ca3cfg.Squash.MaxContentRunes = 40
	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary with compacted context"}, nil
		},
	}
	scorer := embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
		return 0, errors.New("embedding service down")
	})
	compactor := &semanticCompactor{client: client, embedding: scorer}
	result, err := compactor.compact(context.Background(), pressureCompactionRequest{
		AssembleReq: types.ContextAssembleRequest{
			Input: "compact please",
		},
		ModelReq: types.ModelRequest{
			Model: "gpt-4.1-mini",
			Messages: []types.Message{
				{Role: "user", Content: strings.Repeat("alpha beta long semantic content ", 40)},
			},
		},
		Config:      ca3cfg,
		StagePolicy: "best_effort",
	})
	if err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if result.EmbeddingStatus != "fallback_rule_only" {
		t.Fatalf("embedding status = %q, want fallback_rule_only", result.EmbeddingStatus)
	}
	if result.EmbeddingFallbackReason != "embedding_score_error" {
		t.Fatalf("embedding fallback reason = %q, want embedding_score_error", result.EmbeddingFallbackReason)
	}
}

func TestAssemblerContextPressureSemanticCompactionEmbeddingFailureFailFast(t *testing.T) {
	ca3cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA3
	ca3cfg.Compaction.Embedding.Enabled = true
	ca3cfg.Compaction.Embedding.Selector = "test"
	ca3cfg.Compaction.Embedding.Provider = "anthropic"
	ca3cfg.Compaction.Embedding.Model = "claude-embedding-placeholder"
	ca3cfg.Compaction.Embedding.Timeout = 300 * time.Millisecond
	ca3cfg.Squash.MaxContentRunes = 40
	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary with compacted context"}, nil
		},
	}
	scorer := embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
		return 0, errors.New("embedding service down")
	})
	compactor := &semanticCompactor{client: client, embedding: scorer}
	_, err := compactor.compact(context.Background(), pressureCompactionRequest{
		AssembleReq: types.ContextAssembleRequest{
			Input: "compact please",
		},
		ModelReq: types.ModelRequest{
			Model: "gpt-4.1-mini",
			Messages: []types.Message{
				{Role: "user", Content: strings.Repeat("alpha beta long semantic content ", 40)},
			},
		},
		Config:      ca3cfg,
		StagePolicy: "fail_fast",
	})
	if err == nil {
		t.Fatal("expected fail_fast embedding scoring error")
	}
}

func TestAssemblerContextPressurePruneRetainsEvidenceAndReportsCount(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 80
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 8, Comfort: 16, Warning: 24, Danger: 32, Emergency: 40,
	}
	cfg.CA3.Prune.TargetPercent = 30
	cfg.CA3.Compaction.Evidence.Keywords = []string{"mustkeep"}
	cfg.CA3.Compaction.Evidence.RecentWindow = 0

	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: "mustkeep: keep this statement"},
		{Role: "user", Content: strings.Repeat("filler ", 80)},
	}
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-evidence",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need trim", 12),
		Messages:      msgs,
	}, types.ModelRequest{
		RunID:    "run-evidence",
		Input:    strings.Repeat("need trim", 12),
		Messages: msgs,
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.RetainedEvidenceCount <= 0 {
		t.Fatalf("retained evidence count = %d, want > 0", result.Stage.RetainedEvidenceCount)
	}
	foundEvidence := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "mustkeep: keep this statement") {
			foundEvidence = true
			break
		}
	}
	if !foundEvidence {
		t.Fatalf("evidence message should be retained: %#v", outReq.Messages)
	}
}

func TestBuildEmbeddingScorerAnthropicUsablePath(t *testing.T) {
	scorer, err := buildEmbeddingScorer(runtimeconfig.ContextAssemblerCA3CompactionEmbeddingConfig{
		Enabled:  true,
		Selector: "default",
		Provider: "anthropic",
		Model:    "claude-3-haiku",
		Timeout:  200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("buildEmbeddingScorer failed: %v", err)
	}
	score, err := scorer.Score(context.Background(), SemanticEmbeddingScoreRequest{
		Selector: "default",
		Provider: "anthropic",
		Model:    "claude-3-haiku",
		Source:   "alpha beta",
		Summary:  "alpha beta compact",
	})
	if err != nil {
		t.Fatalf("anthropic score failed: %v", err)
	}
	if score < 0 || score > 1 {
		t.Fatalf("anthropic score out of range: %v", score)
	}
}

func TestAssemblerContextPressureRerankerBestEffortFallback(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 120
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 1, Comfort: 2, Warning: 3, Danger: 4, Emergency: 5,
	}
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Embedding.Enabled = true
	cfg.CA3.Compaction.Embedding.Selector = "default"
	cfg.CA3.Compaction.Embedding.Provider = "openai"
	cfg.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.CA3.Compaction.Embedding.Timeout = 300 * time.Millisecond
	cfg.CA3.Compaction.Reranker.Enabled = true
	cfg.CA3.Compaction.Reranker.Timeout = 200 * time.Millisecond
	cfg.CA3.Compaction.Reranker.MaxRetries = 0
	cfg.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"openai:text-embedding-3-small": 0.5,
	}
	cfg.CA2.StagePolicy.Stage1 = "best_effort"

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary"}, nil
		},
	}
	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithSemanticEmbeddingScorer("test", embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
			return 0.8, nil
		})),
		WithSemanticReranker("openai", rerankerFunc(func(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error) {
			return SemanticRerankResult{}, errors.New("reranker unavailable")
		})),
	)
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-reranker-best-effort",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 20),
		Messages: []types.Message{
			{Role: "system", Content: "base"},
			{Role: "user", Content: strings.Repeat("long semantic content ", 30)},
		},
		ModelClient: client,
	}, types.ModelRequest{
		RunID:    "run-reranker-best-effort",
		Input:    strings.Repeat("need compact ", 20),
		Model:    "gpt-4.1-mini",
		Messages: []types.Message{{Role: "system", Content: "base"}, {Role: "user", Content: strings.Repeat("long semantic content ", 30)}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.CompactionRerankerFallbackReason != "reranker_error" {
		t.Fatalf("reranker fallback reason = %q, want reranker_error", result.Stage.CompactionRerankerFallbackReason)
	}
}

func TestAssemblerContextPressureRerankerFailFast(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 120
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 1, Comfort: 2, Warning: 3, Danger: 4, Emergency: 5,
	}
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Embedding.Enabled = true
	cfg.CA3.Compaction.Embedding.Selector = "default"
	cfg.CA3.Compaction.Embedding.Provider = "openai"
	cfg.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.CA3.Compaction.Embedding.Timeout = 300 * time.Millisecond
	cfg.CA3.Compaction.Reranker.Enabled = true
	cfg.CA3.Compaction.Reranker.Timeout = 200 * time.Millisecond
	cfg.CA3.Compaction.Reranker.MaxRetries = 0
	cfg.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"openai:text-embedding-3-small": 0.5,
	}
	cfg.CA2.StagePolicy.Stage1 = "fail_fast"
	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary"}, nil
		},
	}
	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithSemanticEmbeddingScorer("test", embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
			return 0.8, nil
		})),
		WithSemanticReranker("openai", rerankerFunc(func(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error) {
			return SemanticRerankResult{}, errors.New("reranker failed")
		})),
	)
	_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-reranker-fail-fast",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 20),
		Messages: []types.Message{
			{Role: "system", Content: "base"},
			{Role: "user", Content: strings.Repeat("long semantic content ", 30)},
		},
		ModelClient: client,
	}, types.ModelRequest{
		RunID:    "run-reranker-fail-fast",
		Input:    strings.Repeat("need compact ", 20),
		Model:    "gpt-4.1-mini",
		Messages: []types.Message{{Role: "system", Content: "base"}, {Role: "user", Content: strings.Repeat("long semantic content ", 30)}},
	})
	if err == nil {
		t.Fatal("expected fail_fast reranker error")
	}
}

func TestAssemblerContextPressureRerankerGovernanceEnforceVsDryRun(t *testing.T) {
	newCfg := func(mode string) runtimeconfig.ContextAssemblerConfig {
		cfg := runtimeconfig.DefaultConfig().ContextAssembler
		cfg.JournalPath = filepath.Join(t.TempDir(), mode+"-journal.jsonl")
		cfg.CA3.Enabled = true
		cfg.CA3.MaxContextTokens = 120
		cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
			Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
		}
		cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
			Safe: 1, Comfort: 2, Warning: 3, Danger: 4, Emergency: 5,
		}
		cfg.CA3.Compaction.Mode = "semantic"
		cfg.CA3.Compaction.Quality.Threshold = 0.2
		cfg.CA3.Compaction.Embedding.Enabled = true
		cfg.CA3.Compaction.Embedding.Selector = "default"
		cfg.CA3.Compaction.Embedding.Provider = "openai"
		cfg.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
		cfg.CA3.Compaction.Embedding.Timeout = 300 * time.Millisecond
		cfg.CA3.Compaction.Reranker.Enabled = true
		cfg.CA3.Compaction.Reranker.Timeout = 200 * time.Millisecond
		cfg.CA3.Compaction.Reranker.MaxRetries = 0
		cfg.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
			"openai:text-embedding-3-small": 0.95,
		}
		cfg.CA3.Compaction.Reranker.Governance.Mode = mode
		cfg.CA3.Compaction.Reranker.Governance.ProfileVersion = "e5-canary-v1"
		cfg.CA3.Compaction.Reranker.Governance.RolloutProviderModels = []string{"openai:text-embedding-3-small"}
		cfg.CA2.StagePolicy.Stage1 = "best_effort"
		return cfg
	}
	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary"}, nil
		},
	}
	reranker := rerankerFunc(func(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error) {
		return SemanticRerankResult{Score: 0.6}, nil
	})
	scorer := embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
		return 0.7, nil
	})
	runAssemble := func(cfg runtimeconfig.ContextAssemblerConfig, runID string) (types.ContextAssembleResult, error) {
		a := New(
			func() runtimeconfig.ContextAssemblerConfig { return cfg },
			WithSemanticEmbeddingScorer("test", scorer),
			WithSemanticReranker("openai", reranker),
		)
		_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
			RunID:         runID,
			SessionID:     "s-1",
			PrefixVersion: semanticPrefixVersion,
			Input:         strings.Repeat("need compact ", 20),
			Messages: []types.Message{
				{Role: "system", Content: "base"},
				{Role: "user", Content: strings.Repeat("long semantic content ", 30)},
			},
			ModelClient: client,
		}, types.ModelRequest{
			RunID:    runID,
			Input:    strings.Repeat("need compact ", 20),
			Model:    "gpt-4.1-mini",
			Messages: []types.Message{{Role: "system", Content: "base"}, {Role: "user", Content: strings.Repeat("long semantic content ", 30)}},
		})
		return result, err
	}
	enforceResult, enforceErr := runAssemble(newCfg(runtimeconfig.CA3RerankerGovernanceModeEnforce), "run-governance-enforce")
	if enforceErr != nil {
		t.Fatalf("enforce assemble failed: %v", enforceErr)
	}
	if !enforceResult.Stage.CompactionFallback {
		t.Fatal("enforce mode should trigger fallback on high profile threshold")
	}
	if !enforceResult.Stage.CompactionRerankerRolloutHit {
		t.Fatalf("enforce rollout hit = %v, want true", enforceResult.Stage.CompactionRerankerRolloutHit)
	}
	if enforceResult.Stage.CompactionRerankerProfileVersion != "e5-canary-v1" {
		t.Fatalf("enforce profile version = %q, want e5-canary-v1", enforceResult.Stage.CompactionRerankerProfileVersion)
	}
	if enforceResult.Stage.CompactionRerankerThresholdDrift <= 0 {
		t.Fatalf("enforce threshold drift = %v, want > 0", enforceResult.Stage.CompactionRerankerThresholdDrift)
	}

	dryRunResult, dryRunErr := runAssemble(newCfg(runtimeconfig.CA3RerankerGovernanceModeDryRun), "run-governance-dry-run")
	if dryRunErr != nil {
		t.Fatalf("dry_run assemble failed: %v", dryRunErr)
	}
	if dryRunResult.Stage.CompactionFallback {
		t.Fatal("dry_run mode should not enforce profile threshold gate")
	}
	if !dryRunResult.Stage.CompactionRerankerRolloutHit {
		t.Fatalf("dry_run rollout hit = %v, want true", dryRunResult.Stage.CompactionRerankerRolloutHit)
	}
}

func TestAssemblerContextPressureRerankerGovernanceRolloutMatchDeterministic(t *testing.T) {
	baseCfg := runtimeconfig.DefaultConfig().ContextAssembler
	baseCfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	baseCfg.CA3.Enabled = true
	baseCfg.CA3.MaxContextTokens = 120
	baseCfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	baseCfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 1, Comfort: 2, Warning: 3, Danger: 4, Emergency: 5,
	}
	baseCfg.CA3.Compaction.Mode = "semantic"
	baseCfg.CA3.Compaction.Quality.Threshold = 0.2
	baseCfg.CA3.Compaction.Embedding.Enabled = true
	baseCfg.CA3.Compaction.Embedding.Selector = "default"
	baseCfg.CA3.Compaction.Embedding.Provider = "openai"
	baseCfg.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	baseCfg.CA3.Compaction.Embedding.Timeout = 300 * time.Millisecond
	baseCfg.CA3.Compaction.Reranker.Enabled = true
	baseCfg.CA3.Compaction.Reranker.Timeout = 200 * time.Millisecond
	baseCfg.CA3.Compaction.Reranker.MaxRetries = 0
	baseCfg.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"openai:text-embedding-3-small": 0.95,
	}
	baseCfg.CA3.Compaction.Reranker.Governance.Mode = runtimeconfig.CA3RerankerGovernanceModeEnforce
	baseCfg.CA2.StagePolicy.Stage1 = "best_effort"

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary"}, nil
		},
	}
	scorer := embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
		return 0.7, nil
	})
	reranker := rerankerFunc(func(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error) {
		return SemanticRerankResult{Score: 0.6}, nil
	})
	run := func(cfg runtimeconfig.ContextAssemblerConfig, runID string) (types.ContextAssembleResult, error) {
		a := New(
			func() runtimeconfig.ContextAssemblerConfig { return cfg },
			WithSemanticEmbeddingScorer("test", scorer),
			WithSemanticReranker("openai", reranker),
		)
		_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
			RunID:         runID,
			SessionID:     "s-1",
			PrefixVersion: semanticPrefixVersion,
			Input:         strings.Repeat("need compact ", 20),
			Messages: []types.Message{
				{Role: "system", Content: "base"},
				{Role: "user", Content: strings.Repeat("long semantic content ", 30)},
			},
			ModelClient: client,
		}, types.ModelRequest{
			RunID:    runID,
			Input:    strings.Repeat("need compact ", 20),
			Model:    "gpt-4.1-mini",
			Messages: []types.Message{{Role: "system", Content: "base"}, {Role: "user", Content: strings.Repeat("long semantic content ", 30)}},
		})
		return result, err
	}

	hitCfg := baseCfg
	hitCfg.CA3.Compaction.Reranker.Governance.RolloutProviderModels = []string{"openai:text-embedding-3-small"}
	hitResult, err := run(hitCfg, "run-rollout-hit")
	if err != nil {
		t.Fatalf("rollout hit assemble failed: %v", err)
	}
	if !hitResult.Stage.CompactionFallback || !hitResult.Stage.CompactionRerankerRolloutHit {
		t.Fatalf("rollout hit should enforce threshold gate, stage=%#v", hitResult.Stage)
	}

	missCfg := baseCfg
	missCfg.CA3.Compaction.Reranker.Governance.RolloutProviderModels = []string{"gemini:text-embedding-004"}
	missResult, err := run(missCfg, "run-rollout-miss")
	if err != nil {
		t.Fatalf("rollout miss assemble failed: %v", err)
	}
	if missResult.Stage.CompactionFallback {
		t.Fatalf("rollout miss should bypass enforcement, stage=%#v", missResult.Stage)
	}
	if missResult.Stage.CompactionRerankerRolloutHit {
		t.Fatalf("rollout miss hit flag = %v, want false", missResult.Stage.CompactionRerankerRolloutHit)
	}
}

func TestAssemblerContextPressureRerankerGovernanceModeFailurePolicy(t *testing.T) {
	baseCfg := runtimeconfig.DefaultConfig().ContextAssembler
	baseCfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	baseCfg.CA3.Enabled = true
	baseCfg.CA3.MaxContextTokens = 120
	baseCfg.CA3.Compaction.Mode = "semantic"
	baseCfg.CA3.Compaction.Quality.Threshold = 0.2
	baseCfg.CA3.Compaction.Embedding.Enabled = true
	baseCfg.CA3.Compaction.Embedding.Selector = "default"
	baseCfg.CA3.Compaction.Embedding.Provider = "openai"
	baseCfg.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	baseCfg.CA3.Compaction.Embedding.Timeout = 300 * time.Millisecond
	baseCfg.CA3.Compaction.Reranker.Enabled = true
	baseCfg.CA3.Compaction.Reranker.Timeout = 200 * time.Millisecond
	baseCfg.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"openai:text-embedding-3-small": 0.8,
	}
	baseCfg.CA3.Compaction.Reranker.Governance.Mode = "invalid-mode"

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "semantic-summary"}, nil
		},
	}
	scorer := embeddingScorerFunc(func(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
		return 0.7, nil
	})
	reranker := rerankerFunc(func(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error) {
		return SemanticRerankResult{Score: 0.6}, nil
	})

	bestEffortCfg := baseCfg
	bestEffortCfg.CA2.StagePolicy.Stage1 = "best_effort"
	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return bestEffortCfg },
		WithSemanticEmbeddingScorer("test", scorer),
		WithSemanticReranker("openai", reranker),
	)
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-governance-best-effort",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 10),
		Messages: []types.Message{
			{Role: "system", Content: "base"},
			{Role: "user", Content: strings.Repeat("long semantic content ", 20)},
		},
		ModelClient: client,
	}, types.ModelRequest{
		RunID:    "run-governance-best-effort",
		Input:    strings.Repeat("need compact ", 10),
		Model:    "gpt-4.1-mini",
		Messages: []types.Message{{Role: "system", Content: "base"}, {Role: "user", Content: strings.Repeat("long semantic content ", 20)}},
	})
	if err != nil {
		t.Fatalf("best_effort should continue on governance mode error: %v", err)
	}
	if result.Stage.CompactionRerankerFallbackReason != "governance_mode_invalid" {
		t.Fatalf("governance fallback reason = %q, want governance_mode_invalid", result.Stage.CompactionRerankerFallbackReason)
	}

	failFastCfg := baseCfg
	failFastCfg.CA2.StagePolicy.Stage1 = "fail_fast"
	a = New(
		func() runtimeconfig.ContextAssemblerConfig { return failFastCfg },
		WithSemanticEmbeddingScorer("test", scorer),
		WithSemanticReranker("openai", reranker),
	)
	_, _, err = a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-governance-fail-fast",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 10),
		Messages: []types.Message{
			{Role: "system", Content: "base"},
			{Role: "user", Content: strings.Repeat("long semantic content ", 20)},
		},
		ModelClient: client,
	}, types.ModelRequest{
		RunID:    "run-governance-fail-fast",
		Input:    strings.Repeat("need compact ", 10),
		Model:    "gpt-4.1-mini",
		Messages: []types.Message{{Role: "system", Content: "base"}, {Role: "user", Content: strings.Repeat("long semantic content ", 20)}},
	})
	if err == nil {
		t.Fatal("expected fail_fast governance mode error")
	}
}
