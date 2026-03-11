package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
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
