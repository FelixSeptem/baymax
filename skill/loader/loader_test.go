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

type staticSkillScorer struct {
	byName map[string]float64
}

func (s staticSkillScorer) Score(_ string, spec types.SkillSpec, _ runtimeconfig.SkillTriggerScoringConfig) float64 {
	if s.byName == nil {
		return 0
	}
	return s.byName[spec.Name]
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

func TestCompileMixedCJKENLexicalTokenization(t *testing.T) {
	dir := t.TempDir()
	cnPath := filepath.Join(dir, "cn", "SKILL.md")
	mixPath := filepath.Join(dir, "mix", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(cnPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(mixPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cnPath, []byte("description: 数据库迁移\n- tool: local.cn"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mixPath, []byte("description: 数据库 migration\n- tool: local.mix"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs := []types.SkillSpec{
		{Name: "cn-skill", Path: cnPath, Description: "数据库迁移"},
		{Name: "mix-skill", Path: mixPath, Description: "数据库 migration"},
	}
	l := New(nil)

	cnBundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "请执行数据库迁移"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(cnBundle.SystemPromptFragments) == 0 {
		t.Fatalf("expected chinese lexical trigger hit, got %#v", cnBundle.SystemPromptFragments)
	}

	mixBundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "请做 database 迁移"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(mixBundle.SystemPromptFragments) == 0 {
		t.Fatalf("expected mixed lexical trigger hit, got %#v", mixBundle.SystemPromptFragments)
	}
}

func TestCompileTopKBudgetAndExplicitBypass(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: false
    max_semantic_candidates: 1
    budget:
      mode: fixed
      adaptive:
        min_k: 1
        max_k: 1
        min_score_margin: 0.08
    keyword_weights:
      search: 1.0
      alpha: 1.0
      beta: 1.0
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	makeSkill := func(name, tool string) string {
		path := filepath.Join(dir, name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("description: search alpha beta\n- tool: "+tool), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	specs := []types.SkillSpec{
		{Name: "explicit-skill", Path: makeSkill("explicit", "local.explicit"), Description: "search alpha beta", Priority: 1},
		{Name: "s1", Path: makeSkill("s1", "local.s1"), Description: "search alpha beta", Priority: 9},
		{Name: "s2", Path: makeSkill("s2", "local.s2"), Description: "search alpha beta", Priority: 8},
		{Name: "s3", Path: makeSkill("s3", "local.s3"), Description: "search alpha beta", Priority: 7},
	}
	col := &collector{}
	l := NewWithRuntimeManager(col, mgr)

	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "请用 explicit-skill 做 search alpha beta"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.EnabledTools) != 2 {
		t.Fatalf("enabled tools len = %d, want 2 (explicit + top1 semantic)", len(bundle.EnabledTools))
	}
	if bundle.EnabledTools[0] != "local.explicit" {
		t.Fatalf("expected explicit skill first, got %#v", bundle.EnabledTools)
	}
	if bundle.EnabledTools[1] != "local.s1" {
		t.Fatalf("expected highest-priority semantic skill kept by top-k, got %#v", bundle.EnabledTools)
	}
	loaded := 0
	for _, ev := range col.events {
		if ev.Type != "skill.loaded" {
			continue
		}
		loaded++
		if ev.Payload["tokenizer_mode"] != runtimeconfig.SkillTriggerScoringTokenizerMixedCJKEN {
			t.Fatalf("tokenizer_mode = %#v, want %q", ev.Payload["tokenizer_mode"], runtimeconfig.SkillTriggerScoringTokenizerMixedCJKEN)
		}
		if got, _ := ev.Payload["candidate_pruned_count"].(int); got != 2 {
			t.Fatalf("candidate_pruned_count = %#v, want 2", ev.Payload["candidate_pruned_count"])
		}
		if ev.Payload["budget_mode"] != runtimeconfig.SkillTriggerScoringBudgetModeFixed {
			t.Fatalf("budget_mode = %#v, want %q", ev.Payload["budget_mode"], runtimeconfig.SkillTriggerScoringBudgetModeFixed)
		}
		if ev.Payload["selected_semantic_count"] != 1 {
			t.Fatalf("selected_semantic_count = %#v, want 1", ev.Payload["selected_semantic_count"])
		}
		if ev.Payload["budget_decision_reason"] != "fixed.top_k" {
			t.Fatalf("budget_decision_reason = %#v, want fixed.top_k", ev.Payload["budget_decision_reason"])
		}
	}
	if loaded != 2 {
		t.Fatalf("loaded event count = %d, want 2", loaded)
	}
}

func TestCompileAdaptiveBudgetClearWinnerUsesMinK(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: false
    max_semantic_candidates: 5
    budget:
      mode: adaptive
      adaptive:
        min_k: 1
        max_k: 5
        min_score_margin: 0.08
    keyword_weights:
      search: 1.0
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	makeSkill := func(name, tool string) types.SkillSpec {
		path := filepath.Join(dir, name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("description: search\n- tool: "+tool), 0o644); err != nil {
			t.Fatal(err)
		}
		return types.SkillSpec{Name: name, Path: path, Description: "search"}
	}
	specs := []types.SkillSpec{
		makeSkill("s1", "local.s1"),
		makeSkill("s2", "local.s2"),
		makeSkill("s3", "local.s3"),
	}
	col := &collector{}
	l := NewWithRuntimeManager(col, mgr)
	l.scorer = staticSkillScorer{byName: map[string]float64{
		"s1": 0.95,
		"s2": 0.70,
		"s3": 0.69,
	}}

	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "search"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.EnabledTools) != 1 || bundle.EnabledTools[0] != "local.s1" {
		t.Fatalf("adaptive clear winner should keep min_k=1, got %#v", bundle.EnabledTools)
	}
	last := col.events[len(col.events)-1]
	if last.Payload["budget_mode"] != runtimeconfig.SkillTriggerScoringBudgetModeAdaptive {
		t.Fatalf("budget_mode = %#v, want %q", last.Payload["budget_mode"], runtimeconfig.SkillTriggerScoringBudgetModeAdaptive)
	}
	if last.Payload["selected_semantic_count"] != 1 {
		t.Fatalf("selected_semantic_count = %#v, want 1", last.Payload["selected_semantic_count"])
	}
	if last.Payload["candidate_pruned_count"] != 2 {
		t.Fatalf("candidate_pruned_count = %#v, want 2", last.Payload["candidate_pruned_count"])
	}
	if last.Payload["budget_decision_reason"] != "adaptive.clear_winner" {
		t.Fatalf("budget_decision_reason = %#v, want adaptive.clear_winner", last.Payload["budget_decision_reason"])
	}
}

func TestCompileAdaptiveBudgetCloseScoresExpandWithinMaxK(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: false
    max_semantic_candidates: 5
    budget:
      mode: adaptive
      adaptive:
        min_k: 1
        max_k: 3
        min_score_margin: 0.08
    keyword_weights:
      search: 1.0
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	makeSkill := func(name, tool string) types.SkillSpec {
		path := filepath.Join(dir, name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("description: search\n- tool: "+tool), 0o644); err != nil {
			t.Fatal(err)
		}
		return types.SkillSpec{Name: name, Path: path, Description: "search"}
	}
	specs := []types.SkillSpec{
		makeSkill("s1", "local.s1"),
		makeSkill("s2", "local.s2"),
		makeSkill("s3", "local.s3"),
		makeSkill("s4", "local.s4"),
	}
	col := &collector{}
	l := NewWithRuntimeManager(col, mgr)
	l.scorer = staticSkillScorer{byName: map[string]float64{
		"s1": 0.91,
		"s2": 0.88,
		"s3": 0.86,
		"s4": 0.60,
	}}

	bundle, err := l.Compile(context.Background(), specs, types.SkillInput{UserInput: "search"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(bundle.EnabledTools) != 3 {
		t.Fatalf("adaptive close scores should expand within max_k=3, got %#v", bundle.EnabledTools)
	}
	last := col.events[len(col.events)-1]
	if last.Payload["budget_mode"] != runtimeconfig.SkillTriggerScoringBudgetModeAdaptive {
		t.Fatalf("budget_mode = %#v, want %q", last.Payload["budget_mode"], runtimeconfig.SkillTriggerScoringBudgetModeAdaptive)
	}
	if last.Payload["selected_semantic_count"] != 3 {
		t.Fatalf("selected_semantic_count = %#v, want 3", last.Payload["selected_semantic_count"])
	}
	if last.Payload["candidate_pruned_count"] != 1 {
		t.Fatalf("candidate_pruned_count = %#v, want 1", last.Payload["candidate_pruned_count"])
	}
	if last.Payload["budget_decision_reason"] != "adaptive.max_k_reached" {
		t.Fatalf("budget_decision_reason = %#v, want adaptive.max_k_reached", last.Payload["budget_decision_reason"])
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

func TestCompileMultilingualBudgetRunAndStreamSemanticEquivalent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: false
    max_semantic_candidates: 2
    budget:
      mode: fixed
      adaptive:
        min_k: 1
        max_k: 2
        min_score_margin: 0.08
    keyword_weights:
      数据库: 1.5
      migrate: 1.4
      search: 1.2
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	makeSkill := func(name, tool string, p int) types.SkillSpec {
		path := filepath.Join(dir, name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("description: 数据库 migrate search\n- tool: "+tool), 0o644); err != nil {
			t.Fatal(err)
		}
		return types.SkillSpec{Name: name, Path: path, Description: "数据库 migrate search", Priority: p}
	}
	specs := []types.SkillSpec{
		makeSkill("s1", "local.s1", 9),
		makeSkill("s2", "local.s2", 8),
		makeSkill("s3", "local.s3", 7),
	}
	input := "请做数据库 migrate search"
	runCollector := &collector{}
	streamCollector := &collector{}

	runBundle, err := NewWithRuntimeManager(runCollector, mgr).Compile(context.Background(), specs, types.SkillInput{UserInput: input})
	if err != nil {
		t.Fatalf("run compile failed: %v", err)
	}
	streamBundle, err := NewWithRuntimeManager(streamCollector, mgr).Compile(context.Background(), specs, types.SkillInput{UserInput: input})
	if err != nil {
		t.Fatalf("stream compile failed: %v", err)
	}
	if len(runBundle.EnabledTools) != len(streamBundle.EnabledTools) {
		t.Fatalf("run/stream tool count mismatch: run=%#v stream=%#v", runBundle.EnabledTools, streamBundle.EnabledTools)
	}
	for i := range runBundle.EnabledTools {
		if runBundle.EnabledTools[i] != streamBundle.EnabledTools[i] {
			t.Fatalf("run/stream tool order mismatch: run=%#v stream=%#v", runBundle.EnabledTools, streamBundle.EnabledTools)
		}
	}
	assertLoadedPayload := func(t *testing.T, items []types.Event) {
		t.Helper()
		for _, ev := range items {
			if ev.Type != "skill.loaded" {
				continue
			}
			if ev.Payload["tokenizer_mode"] != runtimeconfig.SkillTriggerScoringTokenizerMixedCJKEN {
				t.Fatalf("tokenizer_mode = %#v, want %q", ev.Payload["tokenizer_mode"], runtimeconfig.SkillTriggerScoringTokenizerMixedCJKEN)
			}
			if ev.Payload["candidate_pruned_count"] != 1 {
				t.Fatalf("candidate_pruned_count = %#v, want 1", ev.Payload["candidate_pruned_count"])
			}
			if ev.Payload["budget_mode"] != runtimeconfig.SkillTriggerScoringBudgetModeFixed {
				t.Fatalf("budget_mode = %#v, want %q", ev.Payload["budget_mode"], runtimeconfig.SkillTriggerScoringBudgetModeFixed)
			}
		}
	}
	assertLoadedPayload(t, runCollector.events)
	assertLoadedPayload(t, streamCollector.events)
}

func TestCompileAdaptiveBudgetRunAndStreamSemanticEquivalent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: false
    max_semantic_candidates: 5
    budget:
      mode: adaptive
      adaptive:
        min_k: 1
        max_k: 3
        min_score_margin: 0.08
    keyword_weights:
      search: 1.0
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	makeSkill := func(name, tool string, p int) types.SkillSpec {
		path := filepath.Join(dir, name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("description: search\n- tool: "+tool), 0o644); err != nil {
			t.Fatal(err)
		}
		return types.SkillSpec{Name: name, Path: path, Description: "search", Priority: p}
	}
	specs := []types.SkillSpec{
		makeSkill("s1", "local.s1", 9),
		makeSkill("s2", "local.s2", 8),
		makeSkill("s3", "local.s3", 7),
		makeSkill("s4", "local.s4", 6),
	}
	input := "search"
	newLoader := func(c *collector) *Loader {
		l := NewWithRuntimeManager(c, mgr)
		l.scorer = staticSkillScorer{byName: map[string]float64{
			"s1": 0.91,
			"s2": 0.88,
			"s3": 0.86,
			"s4": 0.60,
		}}
		return l
	}
	runCollector := &collector{}
	streamCollector := &collector{}

	runBundle, err := newLoader(runCollector).Compile(context.Background(), specs, types.SkillInput{UserInput: input})
	if err != nil {
		t.Fatalf("run compile failed: %v", err)
	}
	streamBundle, err := newLoader(streamCollector).Compile(context.Background(), specs, types.SkillInput{UserInput: input})
	if err != nil {
		t.Fatalf("stream compile failed: %v", err)
	}
	if len(runBundle.EnabledTools) != len(streamBundle.EnabledTools) {
		t.Fatalf("run/stream tool count mismatch: run=%#v stream=%#v", runBundle.EnabledTools, streamBundle.EnabledTools)
	}
	for i := range runBundle.EnabledTools {
		if runBundle.EnabledTools[i] != streamBundle.EnabledTools[i] {
			t.Fatalf("run/stream tool order mismatch: run=%#v stream=%#v", runBundle.EnabledTools, streamBundle.EnabledTools)
		}
	}
	if len(runBundle.EnabledTools) != 3 {
		t.Fatalf("adaptive run/stream should keep 3 tools, got %#v", runBundle.EnabledTools)
	}
	assertLoadedPayload := func(t *testing.T, items []types.Event) {
		t.Helper()
		for _, ev := range items {
			if ev.Type != "skill.loaded" {
				continue
			}
			if ev.Payload["budget_mode"] != runtimeconfig.SkillTriggerScoringBudgetModeAdaptive {
				t.Fatalf("budget_mode = %#v, want %q", ev.Payload["budget_mode"], runtimeconfig.SkillTriggerScoringBudgetModeAdaptive)
			}
			if ev.Payload["selected_semantic_count"] != 3 {
				t.Fatalf("selected_semantic_count = %#v, want 3", ev.Payload["selected_semantic_count"])
			}
			if ev.Payload["candidate_pruned_count"] != 1 {
				t.Fatalf("candidate_pruned_count = %#v, want 1", ev.Payload["candidate_pruned_count"])
			}
			if ev.Payload["budget_decision_reason"] != "adaptive.max_k_reached" {
				t.Fatalf("budget_decision_reason = %#v, want adaptive.max_k_reached", ev.Payload["budget_decision_reason"])
			}
		}
	}
	assertLoadedPayload(t, runCollector.events)
	assertLoadedPayload(t, streamCollector.events)
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
	if items[0].Payload["tokenizer_mode"] != runtimeconfig.SkillTriggerScoringTokenizerMixedCJKEN {
		t.Fatalf("skill tokenizer_mode payload = %#v, want %q", items[0].Payload["tokenizer_mode"], runtimeconfig.SkillTriggerScoringTokenizerMixedCJKEN)
	}
	if items[0].Payload["candidate_pruned_count"] != 0 {
		t.Fatalf("skill candidate_pruned_count payload = %#v, want 0", items[0].Payload["candidate_pruned_count"])
	}
	if items[0].Payload["budget_mode"] != runtimeconfig.SkillTriggerScoringBudgetModeAdaptive {
		t.Fatalf("skill budget_mode payload = %#v, want %q", items[0].Payload["budget_mode"], runtimeconfig.SkillTriggerScoringBudgetModeAdaptive)
	}
	if items[0].Payload["selected_semantic_count"] != 0 {
		t.Fatalf("skill selected_semantic_count payload = %#v, want 0", items[0].Payload["selected_semantic_count"])
	}
	if items[0].Payload["score_margin_top1_top2"] != 0.0 {
		t.Fatalf("skill score_margin_top1_top2 payload = %#v, want 0", items[0].Payload["score_margin_top1_top2"])
	}
	if items[0].Payload["budget_decision_reason"] != "none.no_candidates" {
		t.Fatalf("skill budget_decision_reason payload = %#v, want none.no_candidates", items[0].Payload["budget_decision_reason"])
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
			"name":                    "missing",
			"action":                  "compile",
			"status":                  "warning",
			"error_class":             string(types.ErrSkill),
			"reason":                  "compile read failed",
			"path":                    filepath.Join(dir, "missing", "SKILL.md"),
			"strategy":                skillTriggerStrategyExplicit,
			"final_score":             float64(1),
			"tokenizer_mode":          runtimeconfig.SkillTriggerScoringTokenizerMixedCJKEN,
			"candidate_pruned_count":  0,
			"budget_mode":             runtimeconfig.SkillTriggerScoringBudgetModeAdaptive,
			"selected_semantic_count": 0,
			"score_margin_top1_top2":  float64(0),
			"budget_decision_reason":  "none.no_candidates",
		},
	}
	rec.OnEvent(context.Background(), replay)

	items = mgr.RecentSkills(10)
	if len(items) != 1 {
		t.Fatalf("replayed warning should be deduped, got %d records", len(items))
	}
}
