package loader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	obsevent "github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type collector struct {
	events []types.Event
}

func (c *collector) OnEvent(ctx context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

func TestDiscoverSkipsMissingSkillAndEmitsWarning(t *testing.T) {
	dir := t.TempDir()
	agents := `
- skill-a: test (file: ` + filepath.ToSlash(filepath.Join(dir, "missing", "SKILL.md")) + `)
`
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatal(err)
	}
	col := &collector{}
	l := New(col)

	specs, err := l.Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(specs) != 0 {
		t.Fatalf("specs len = %d, want 0", len(specs))
	}
	if len(col.events) == 0 || col.events[0].Type != "skill.warning" {
		t.Fatalf("warning event missing: %#v", col.events)
	}
}

func TestCompileExplicitTriggerWins(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "one", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("description: db task\n- tool: local.sql"), 0o644); err != nil {
		t.Fatal(err)
	}

	specs := []types.SkillSpec{{Name: "db-skill", Path: skillPath, Description: "database migration"}}
	l := New(nil)
	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "please use db-skill for this"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.SystemPromptFragments) != 1 {
		t.Fatalf("fragments len = %d, want 1", len(bundle.SystemPromptFragments))
	}
	if len(bundle.EnabledTools) != 1 || bundle.EnabledTools[0] != "local.sql" {
		t.Fatalf("enabled tools mismatch: %#v", bundle.EnabledTools)
	}
}

func TestCompilePartialFailureContinues(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(good), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte("description: valid\n- tool: local.echo"), 0o644); err != nil {
		t.Fatal(err)
	}
	col := &collector{}
	l := New(col)

	specs := []types.SkillSpec{
		{Name: "good", Path: good, Description: "valid"},
		{Name: "bad", Path: filepath.Join(dir, "bad", "SKILL.md"), Description: "bad"},
	}
	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "good bad"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.SystemPromptFragments) != 1 {
		t.Fatalf("fragments len = %d, want 1", len(bundle.SystemPromptFragments))
	}
	foundWarning := false
	for _, ev := range col.events {
		if ev.Type == "skill.warning" {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected warning event, got %#v", col.events)
	}
}

func TestCompileSemanticTieBreakUsesHighestPriority(t *testing.T) {
	dir := t.TempDir()
	highPath := filepath.Join(dir, "high", "SKILL.md")
	lowPath := filepath.Join(dir, "low", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(highPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(lowPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(highPath, []byte("description: database migration\n- tool: local.high"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lowPath, []byte("description: database migration\n- tool: local.low"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{
		{Name: "low", Path: lowPath, Description: "database migration", Priority: 1},
		{Name: "high", Path: highPath, Description: "database migration", Priority: 10},
	}

	l := New(nil)
	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "need database migration"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.SystemPromptFragments) != 2 {
		t.Fatalf("fragments len = %d, want 2", len(bundle.SystemPromptFragments))
	}
	if !strings.Contains(bundle.SystemPromptFragments[0], "local.high") {
		t.Fatalf("expected highest-priority skill first, got %q", bundle.SystemPromptFragments[0])
	}
}

func TestCompileDefaultSuppressesLowConfidenceSemanticMatch(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "one", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("description: database migration\n- tool: local.sql"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{
		{Name: "db-helper", Path: skillPath, Description: "database migration"},
	}
	l := New(nil)
	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{
		UserInput: "database alpha beta gamma delta epsilon zeta eta theta iota kappa lambda",
	})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.SystemPromptFragments) != 0 {
		t.Fatalf("expected low-confidence semantic match to be suppressed, got %#v", bundle.SystemPromptFragments)
	}
}

func TestCompileCanDisableLowConfidenceSuppressionViaRuntimeConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.9
    tie_break: highest_priority
    suppress_low_confidence: false
    keyword_weights:
      database: 1.0
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	skillPath := filepath.Join(dir, "one", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("description: database migration\n- tool: local.sql"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{{Name: "db-helper", Path: skillPath, Description: "database migration"}}
	l := NewWithRuntimeManager(nil, mgr)

	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{
		UserInput: "database alpha beta gamma delta epsilon zeta eta theta iota kappa lambda",
	})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.SystemPromptFragments) != 1 {
		t.Fatalf("expected low-confidence candidate to pass when suppression disabled, got %#v", bundle.SystemPromptFragments)
	}
}

func TestCompileLexicalPlusEmbeddingWeightedScore(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
skill:
  trigger_scoring:
    strategy: lexical_plus_embedding
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    keyword_weights:
      database: 1.0
    embedding:
      enabled: true
      provider: openai
      model: text-embedding-3-small
      timeout: 200ms
      similarity_metric: cosine
      lexical_weight: 0.6
      embedding_weight: 0.4
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	skillPath := filepath.Join(dir, "one", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("description: database tool\n- tool: local.sql"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{{Name: "db-helper", Path: skillPath, Description: "database tool"}}
	col := &collector{}
	l := NewWithRuntimeManager(col, mgr)
	l.SetEmbeddingScorer(SkillTriggerEmbeddingScorerFunc(func(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error) {
		return 0.5, nil
	}))

	_, err = l.Compile(context.Background(), specs, types.SkillInput{UserInput: "database"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	last := col.events[len(col.events)-1]
	if last.Type != "skill.loaded" {
		t.Fatalf("last event = %q, want skill.loaded", last.Type)
	}
	if last.Payload["strategy"] != runtimeconfig.SkillTriggerScoringStrategyLexicalPlusEmbedding {
		t.Fatalf("strategy = %#v, want lexical_plus_embedding", last.Payload["strategy"])
	}
	if got, _ := last.Payload["embedding_score"].(float64); got != 0.5 {
		t.Fatalf("embedding_score = %v, want 0.5", got)
	}
	if got, _ := last.Payload["final_score"].(float64); got != 0.8 {
		t.Fatalf("final_score = %v, want 0.8", got)
	}
	if _, ok := last.Payload["fallback_reason"]; ok {
		t.Fatalf("fallback_reason should be empty, payload=%#v", last.Payload)
	}
}

func TestCompileLexicalPlusEmbeddingFallbackReasons(t *testing.T) {
	newLoader := func(t *testing.T) (*Loader, []types.SkillSpec, *collector) {
		t.Helper()
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "runtime.yaml")
		cfg := `
skill:
  trigger_scoring:
    strategy: lexical_plus_embedding
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    keyword_weights:
      database: 1.0
    embedding:
      enabled: true
      provider: openai
      model: text-embedding-3-small
      timeout: 20ms
      similarity_metric: cosine
      lexical_weight: 0.7
      embedding_weight: 0.3
`
		if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
			t.Fatal(err)
		}
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })
		skillPath := filepath.Join(dir, "one", "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(skillPath, []byte("description: database tool\n- tool: local.sql"), 0o644); err != nil {
			t.Fatal(err)
		}
		specs := []types.SkillSpec{{Name: "db-helper", Path: skillPath, Description: "database tool"}}
		col := &collector{}
		return NewWithRuntimeManager(col, mgr), specs, col
	}

	t.Run("missing", func(t *testing.T) {
		l, specs, col := newLoader(t)
		_, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "database"})
		if err != nil {
			t.Fatalf("Compile failed: %v", err)
		}
		last := col.events[len(col.events)-1]
		if last.Payload["fallback_reason"] != "embedding.scorer_missing" {
			t.Fatalf("fallback_reason = %#v, want embedding.scorer_missing", last.Payload["fallback_reason"])
		}
	})

	t.Run("timeout", func(t *testing.T) {
		l, specs, col := newLoader(t)
		l.SetEmbeddingScorer(SkillTriggerEmbeddingScorerFunc(func(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		}))
		_, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "database"})
		if err != nil {
			t.Fatalf("Compile failed: %v", err)
		}
		last := col.events[len(col.events)-1]
		if last.Payload["fallback_reason"] != "embedding.timeout" {
			t.Fatalf("fallback_reason = %#v, want embedding.timeout", last.Payload["fallback_reason"])
		}
	})

	t.Run("error", func(t *testing.T) {
		l, specs, col := newLoader(t)
		l.SetEmbeddingScorer(SkillTriggerEmbeddingScorerFunc(func(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error) {
			return 0, errors.New("boom")
		}))
		_, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "database"})
		if err != nil {
			t.Fatalf("Compile failed: %v", err)
		}
		last := col.events[len(col.events)-1]
		if last.Payload["fallback_reason"] != "embedding.error" {
			t.Fatalf("fallback_reason = %#v, want embedding.error", last.Payload["fallback_reason"])
		}
	})

	t.Run("invalid_score", func(t *testing.T) {
		l, specs, col := newLoader(t)
		l.SetEmbeddingScorer(SkillTriggerEmbeddingScorerFunc(func(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error) {
			return 1.5, nil
		}))
		_, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "database"})
		if err != nil {
			t.Fatalf("Compile failed: %v", err)
		}
		last := col.events[len(col.events)-1]
		if last.Payload["fallback_reason"] != "embedding.invalid_score" {
			t.Fatalf("fallback_reason = %#v, want embedding.invalid_score", last.Payload["fallback_reason"])
		}
	})
}

func TestCompileLexicalPlusEmbeddingRunAndStreamSemanticEquivalent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
skill:
  trigger_scoring:
    strategy: lexical_plus_embedding
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    keyword_weights:
      database: 1.0
    embedding:
      enabled: true
      provider: openai
      model: text-embedding-3-small
      timeout: 200ms
      similarity_metric: cosine
      lexical_weight: 0.7
      embedding_weight: 0.3
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	skillPath := filepath.Join(dir, "one", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("description: database tool\n- tool: local.sql"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{{Name: "db-helper", Path: skillPath, Description: "database tool", Priority: 3}}
	newLoader := func() *Loader {
		l := NewWithRuntimeManager(nil, mgr)
		l.SetEmbeddingScorer(SkillTriggerEmbeddingScorerFunc(func(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error) {
			return 0.6, nil
		}))
		return l
	}
	runBundle, err := newLoader().Compile(context.Background(), specs, types.SkillInput{UserInput: "database"})
	if err != nil {
		t.Fatalf("run compile failed: %v", err)
	}
	streamBundle, err := newLoader().Compile(context.Background(), specs, types.SkillInput{UserInput: "database"})
	if err != nil {
		t.Fatalf("stream compile failed: %v", err)
	}
	if len(runBundle.SystemPromptFragments) != len(streamBundle.SystemPromptFragments) {
		t.Fatalf("run/stream bundle size mismatch: run=%d stream=%d", len(runBundle.SystemPromptFragments), len(streamBundle.SystemPromptFragments))
	}
	if len(runBundle.EnabledTools) != len(streamBundle.EnabledTools) || runBundle.EnabledTools[0] != streamBundle.EnabledTools[0] {
		t.Fatalf("run/stream tool selection mismatch: run=%#v stream=%#v", runBundle.EnabledTools, streamBundle.EnabledTools)
	}
}

func TestCompileTieBreakDeterministicAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a", "SKILL.md")
	bPath := filepath.Join(dir, "b", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(aPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(bPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aPath, []byte("description: api search\n- tool: local.a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte("description: api search\n- tool: local.b"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{
		{Name: "b-skill", Path: bPath, Description: "api search", Priority: 5},
		{Name: "a-skill", Path: aPath, Description: "api search", Priority: 5},
	}
	l := New(nil)
	first := ""
	for i := 0; i < 10; i++ {
		bundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "api search task"})
		if err != nil {
			t.Fatalf("Compile failed: %v", err)
		}
		if len(bundle.SystemPromptFragments) == 0 {
			t.Fatal("expected semantic match")
		}
		current := bundle.SystemPromptFragments[0]
		if i == 0 {
			first = current
			continue
		}
		if current != first {
			t.Fatalf("non-deterministic tie-break: first=%q current=%q", first, current)
		}
	}
}

func TestConflictResolutionPrecedence(t *testing.T) {
	in := []string{
		"Follow built-in safety constraints first.",
		"mode: from-agents",
		"mode: from-skill",
	}
	out := resolveDirectiveConflicts(in)
	if len(out) < 2 {
		t.Fatalf("unexpected output: %#v", out)
	}
	if out[0] != "Follow built-in safety constraints first." {
		t.Fatalf("built-in hint should be kept first: %#v", out)
	}
}

func TestSkillDiagnosticsWithRuntimeManager(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
diagnostics:
  max_skill_records: 10
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	skillPath := filepath.Join(dir, "one", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("description: db task\n- tool: local.sql"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{{Name: "db-skill", Path: skillPath, Description: "database migration"}}

	rec := obsevent.NewRuntimeRecorder(mgr)
	l := NewWithRuntimeManager(rec, mgr)
	_, err = l.Compile(context.Background(), specs, types.SkillInput{UserInput: "db-skill", Context: map[string]string{"run_id": "run-1"}})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	items := mgr.RecentSkills(1)
	if len(items) != 1 {
		t.Fatalf("skill diagnostics len = %d, want 1", len(items))
	}
	if items[0].SkillName != "db-skill" || items[0].Status != "success" || items[0].Action != "compile" {
		t.Fatalf("unexpected skill diag: %#v", items[0])
	}
	if items[0].Payload["strategy"] != skillTriggerStrategyExplicit {
		t.Fatalf("skill strategy payload = %#v, want explicit", items[0].Payload["strategy"])
	}
	if items[0].Payload["final_score"] != float64(1) {
		t.Fatalf("skill final_score payload = %#v, want 1", items[0].Payload["final_score"])
	}
}

func TestSkillDiagnosticsWarningAndReplayDedup(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
diagnostics:
  max_skill_records: 10
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := obsevent.NewRuntimeRecorder(mgr)
	l := New(rec)

	specs := []types.SkillSpec{
		{Name: "missing", Path: filepath.Join(dir, "missing", "SKILL.md"), Description: "bad"},
	}
	_, err = l.Compile(context.Background(), specs, types.SkillInput{UserInput: "missing", Context: map[string]string{"run_id": "run-1"}})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	items := mgr.RecentSkills(10)
	if len(items) != 1 {
		t.Fatalf("skill diagnostics len = %d, want 1", len(items))
	}
	if items[0].Status != "warning" || items[0].Action != "compile" {
		t.Fatalf("unexpected warning diag: %#v", items[0])
	}

	replay := types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.warning",
		RunID:   "run-1",
		Payload: map[string]any{
			"name":        "missing",
			"action":      "compile",
			"status":      "warning",
			"error_class": string(types.ErrSkill),
			"reason":      "compile read failed",
			"path":        filepath.Join(dir, "missing", "SKILL.md"),
			"strategy":    skillTriggerStrategyExplicit,
			"final_score": float64(1),
		},
	}
	rec.OnEvent(context.Background(), replay)

	items = mgr.RecentSkills(10)
	if len(items) != 1 {
		t.Fatalf("replayed warning should be deduped, got %d records", len(items))
	}
}
