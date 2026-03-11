package loader

import (
	"context"
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
		},
	}
	rec.OnEvent(context.Background(), replay)

	items = mgr.RecentSkills(10)
	if len(items) != 1 {
		t.Fatalf("replayed warning should be deduped, got %d records", len(items))
	}
}
