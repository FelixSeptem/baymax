package loader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	obsTrace "github.com/FelixSeptem/baymax/observability/trace"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
	"go.opentelemetry.io/otel"
)

var skillPathPattern = regexp.MustCompile(`\(file:\s*([^\)]+)\)`)

type Loader struct {
	eventHandler types.EventHandler
	runtimeMgr   *runtimeconfig.Manager
	now          func() time.Time
}

func New(eventHandler types.EventHandler) *Loader {
	return &Loader{eventHandler: eventHandler, now: time.Now}
}

func NewWithRuntimeManager(eventHandler types.EventHandler, mgr *runtimeconfig.Manager) *Loader {
	return &Loader{eventHandler: eventHandler, runtimeMgr: mgr, now: time.Now}
}

func (l *Loader) SetRuntimeManager(mgr *runtimeconfig.Manager) {
	l.runtimeMgr = mgr
}

func (l *Loader) Discover(ctx context.Context, root string) ([]types.SkillSpec, error) {
	ctx, span := otel.Tracer("baymax/skill/loader").Start(ctx, "skill.discover")
	defer span.End()
	agentsPath := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	specs := make([]types.SkillSpec, 0)
	discoverStart := l.now()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "-") || !strings.Contains(line, "(file:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(strings.TrimPrefix(line, "-"), ":", 2)[0], "-"))
		if name == "" {
			name = normalizeName(line)
		}
		match := skillPathPattern.FindStringSubmatch(line)
		if len(match) < 2 {
			continue
		}
		skillFile := strings.TrimSpace(match[1])
		if !filepath.IsAbs(skillFile) {
			skillFile = filepath.Join(root, skillFile)
		}
		if _, err := os.Stat(skillFile); err != nil {
			l.emit(ctx, "", "skill.warning", map[string]any{"name": name, "reason": "missing skill file", "path": skillFile})
			l.recordSkill(ctx, "", name, "discover", "failed", discoverStart, types.ErrSkill, map[string]any{"reason": "missing skill file", "path": skillFile})
			continue
		}
		desc, triggers := parseSkillMeta(skillFile)
		specs = append(specs, types.SkillSpec{
			Name:        name,
			Path:        skillFile,
			Description: desc,
			Triggers:    triggers,
			Metadata: map[string]string{
				"source": "AGENTS",
			},
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	l.recordSkill(ctx, "", "", "discover", "success", discoverStart, "", map[string]any{"count": len(specs)})
	return specs, nil
}

func (l *Loader) Compile(ctx context.Context, specs []types.SkillSpec, in types.SkillInput) (types.SkillBundle, error) {
	ctx, span := otel.Tracer("baymax/skill/loader").Start(ctx, "skill.compile")
	defer span.End()
	if len(specs) == 0 {
		return types.SkillBundle{}, nil
	}

	explicit, semantic := selectSkills(specs, in.UserInput)
	selected := make([]types.SkillSpec, 0, len(specs))
	selected = append(selected, explicit...)
	for _, s := range semantic {
		if !containsSkill(selected, s.Name) {
			selected = append(selected, s)
		}
	}
	if len(selected) == 0 {
		return types.SkillBundle{}, nil
	}
	runID := in.Context["run_id"]

	fragments := make([]string, 0, len(selected)+1)
	enabledTools := make([]string, 0)
	workflowHints := []string{"Follow built-in safety constraints first."}

	for _, spec := range selected {
		stepStart := l.now()
		content, err := os.ReadFile(spec.Path)
		if err != nil {
			l.emit(ctx, runID, "skill.warning", map[string]any{"name": spec.Name, "reason": "compile read failed", "path": spec.Path})
			l.recordSkill(ctx, runID, spec.Name, "compile", "failed", stepStart, types.ErrSkill, map[string]any{"reason": "compile read failed", "path": spec.Path})
			continue
		}
		fragments = append(fragments, string(content))
		workflowHints = append(workflowHints, spec.Description)
		enabled := parseEnabledTools(string(content))
		for _, t := range enabled {
			enabledTools = append(enabledTools, t)
		}
		l.emit(ctx, runID, "skill.loaded", map[string]any{"name": spec.Name, "path": spec.Path})
		l.recordSkill(ctx, runID, spec.Name, "compile", "success", stepStart, "", map[string]any{"enabled_tools": len(enabled)})
	}

	workflowHints = resolveDirectiveConflicts(workflowHints)
	enabledTools = unique(enabledTools)

	return types.SkillBundle{
		SystemPromptFragments: fragments,
		EnabledTools:          enabledTools,
		WorkflowHints:         workflowHints,
	}, nil
}

func selectSkills(specs []types.SkillSpec, input string) (explicit []types.SkillSpec, semantic []types.SkillSpec) {
	lower := strings.ToLower(input)
	for _, s := range specs {
		nameLower := strings.ToLower(s.Name)
		if strings.Contains(lower, "$"+nameLower) || strings.Contains(lower, nameLower) {
			explicit = append(explicit, s)
			continue
		}
		score := semanticScore(lower, s)
		if score >= 0.25 {
			semantic = append(semantic, s)
		}
	}
	return explicit, semantic
}

func semanticScore(input string, s types.SkillSpec) float64 {
	if strings.TrimSpace(input) == "" {
		return 0
	}
	hay := strings.ToLower(s.Description + " " + strings.Join(s.Triggers, " "))
	if hay == "" {
		return 0
	}
	inputTokens := tokenize(input)
	hit := 0
	for _, t := range inputTokens {
		if strings.Contains(hay, t) {
			hit++
		}
	}
	if len(inputTokens) == 0 {
		return 0
	}
	return float64(hit) / float64(len(inputTokens))
}

func tokenize(in string) []string {
	f := func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_')
	}
	parts := strings.FieldsFunc(in, f)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) >= 3 {
			out = append(out, strings.ToLower(p))
		}
	}
	return out
}

func parseSkillMeta(path string) (desc string, triggers []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(t), "description:") {
			desc = strings.TrimSpace(strings.TrimPrefix(t, "description:"))
		}
		if strings.HasPrefix(strings.ToLower(t), "- trigger:") {
			triggers = append(triggers, strings.TrimSpace(strings.TrimPrefix(t, "- trigger:")))
		}
	}
	return desc, triggers
}

func parseEnabledTools(content string) []string {
	out := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(strings.ToLower(line), "- tool:") {
			continue
		}
		out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "- tool:")))
	}
	return out
}

func resolveDirectiveConflicts(hints []string) []string {
	if len(hints) == 0 {
		return hints
	}
	// Fixed precedence: system built-in > AGENTS > SKILL.
	seen := map[string]string{}
	out := make([]string, 0, len(hints))
	for _, h := range hints {
		if strings.TrimSpace(h) == "" {
			continue
		}
		key := strings.ToLower(strings.SplitN(h, ":", 2)[0])
		if prev, ok := seen[key]; ok && strings.Contains(prev, "built-in") {
			continue
		}
		seen[key] = h
		out = append(out, h)
	}
	return unique(out)
}

func unique(items []string) []string {
	set := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		if strings.TrimSpace(it) == "" {
			continue
		}
		if _, ok := set[it]; ok {
			continue
		}
		set[it] = struct{}{}
		out = append(out, it)
	}
	return out
}

func containsSkill(skills []types.SkillSpec, name string) bool {
	for _, s := range skills {
		if s.Name == name {
			return true
		}
	}
	return false
}

func normalizeName(line string) string {
	line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
	if idx := strings.Index(line, " "); idx > 0 {
		return strings.TrimSpace(line[:idx])
	}
	return strings.TrimSpace(line)
}

func (l *Loader) emit(ctx context.Context, runID string, typ string, payload map[string]any) {
	if l.eventHandler == nil {
		return
	}
	l.eventHandler.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    typ,
		RunID:   runID,
		TraceID: obsTrace.TraceIDFromContext(ctx),
		SpanID:  obsTrace.SpanIDFromContext(ctx),
		Time:    l.now(),
		Payload: payload,
	})
}

func (l *Loader) recordSkill(ctx context.Context, runID string, skillName string, action string, status string, start time.Time, errClass types.ErrorClass, payload map[string]any) {
	if l.runtimeMgr == nil {
		return
	}
	errorClass := ""
	if errClass != "" {
		errorClass = string(errClass)
	}
	l.runtimeMgr.RecordSkill(runtimediag.SkillRecord{
		Time:       l.now(),
		RunID:      runID,
		SkillName:  skillName,
		Action:     action,
		Status:     status,
		LatencyMs:  l.now().Sub(start).Milliseconds(),
		ErrorClass: errorClass,
		Payload:    payload,
	})
}

var _ types.SkillLoader = (*Loader)(nil)

func NewDefault() *Loader {
	return New(nil)
}

func MustCompile(ctx context.Context, loader types.SkillLoader, specs []types.SkillSpec, in types.SkillInput) types.SkillBundle {
	bundle, err := loader.Compile(ctx, specs, in)
	if err != nil {
		panic(fmt.Sprintf("compile skills: %v", err))
	}
	return bundle
}
