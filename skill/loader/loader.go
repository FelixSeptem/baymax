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
	"go.opentelemetry.io/otel"
)

var skillPathPattern = regexp.MustCompile(`\(file:\s*([^\)]+)\)`)

const (
	skillTriggerStrategyExplicit = "explicit"

	skillEmbeddingFallbackScorerMissing = "embedding.scorer_missing"
	skillEmbeddingFallbackTimeout       = "embedding.timeout"
	skillEmbeddingFallbackError         = "embedding.error"
	skillEmbeddingFallbackInvalidScore  = "embedding.invalid_score"
)

// SkillTriggerEmbeddingScoreRequest is normalized input for skill embedding scorer extension.
type SkillTriggerEmbeddingScoreRequest struct {
	Provider  string
	Model     string
	Input     string
	SkillName string
	SkillDesc string
	Triggers  []string
}

// SkillTriggerEmbeddingScorer scores semantic similarity for a skill candidate.
type SkillTriggerEmbeddingScorer interface {
	Score(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error)
}

// SkillTriggerEmbeddingScorerFunc adapts a function to SkillTriggerEmbeddingScorer.
type SkillTriggerEmbeddingScorerFunc func(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error)

// Score calls wrapped function.
func (f SkillTriggerEmbeddingScorerFunc) Score(ctx context.Context, req SkillTriggerEmbeddingScoreRequest) (float64, error) {
	return f(ctx, req)
}

// Loader discovers and compiles skills from repository metadata.
type Loader struct {
	eventHandler    types.EventHandler
	runtimeMgr      *runtimeconfig.Manager
	now             func() time.Time
	scorer          skillTriggerScorer
	embeddingScorer SkillTriggerEmbeddingScorer
}

// New constructs a skill loader with event handler wiring and default lexical scorer.
func New(eventHandler types.EventHandler) *Loader {
	return &Loader{
		eventHandler: eventHandler,
		now:          time.Now,
		scorer:       lexicalWeightedKeywordScorer{},
	}
}

// NewWithRuntimeManager constructs a skill loader that reads trigger scoring settings from runtime config manager.
func NewWithRuntimeManager(eventHandler types.EventHandler, mgr *runtimeconfig.Manager) *Loader {
	return &Loader{
		eventHandler: eventHandler,
		runtimeMgr:   mgr,
		now:          time.Now,
		scorer:       lexicalWeightedKeywordScorer{},
	}
}

// SetRuntimeManager updates runtime configuration source for trigger scoring and policy lookup.
func (l *Loader) SetRuntimeManager(mgr *runtimeconfig.Manager) {
	l.runtimeMgr = mgr
}

// SetEmbeddingScorer registers host-provided embedding scorer for semantic trigger scoring.
func (l *Loader) SetEmbeddingScorer(scorer SkillTriggerEmbeddingScorer) {
	l.embeddingScorer = scorer
}

// Discover parses AGENTS.md and returns discovered skill specs in deterministic name order.
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
			l.emit(ctx, "", "skill.warning", map[string]any{
				"name":        name,
				"action":      "discover",
				"status":      "warning",
				"error_class": string(types.ErrSkill),
				"reason":      "missing skill file",
				"path":        skillFile,
			})
			continue
		}
		desc, triggers, priority := parseSkillMeta(skillFile)
		specs = append(specs, types.SkillSpec{
			Name:        name,
			Path:        skillFile,
			Description: desc,
			Triggers:    triggers,
			Priority:    priority,
			Metadata: map[string]string{
				"source": "AGENTS",
			},
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	l.emit(ctx, "", "skill.discovered", map[string]any{
		"action":     "discover",
		"status":     "success",
		"count":      len(specs),
		"latency_ms": l.now().Sub(discoverStart).Milliseconds(),
	})
	return specs, nil
}

// Compile resolves explicit and semantic skill matches and builds an executable skill bundle.
func (l *Loader) Compile(ctx context.Context, specs []types.SkillSpec, in types.SkillInput) (types.SkillBundle, error) {
	ctx, span := otel.Tracer("baymax/skill/loader").Start(ctx, "skill.compile")
	defer span.End()
	if len(specs) == 0 {
		return types.SkillBundle{}, nil
	}

	explicit, semantic := l.selectSkills(ctx, specs, in.UserInput)
	selected := make([]types.SkillSpec, 0, len(specs))
	selected = append(selected, explicit...)
	semanticByName := map[string]scoredSkill{}
	for _, item := range semantic {
		semanticByName[item.spec.Name] = item
		if !containsSkill(selected, item.spec.Name) {
			selected = append(selected, item.spec)
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
		scorePayload := map[string]any{}
		if meta, ok := semanticByName[spec.Name]; ok {
			scorePayload = scorePayloadFromMeta(meta)
		} else {
			scorePayload["strategy"] = skillTriggerStrategyExplicit
			scorePayload["final_score"] = 1.0
		}
		content, err := os.ReadFile(spec.Path)
		if err != nil {
			payload := map[string]any{
				"name":        spec.Name,
				"action":      "compile",
				"status":      "warning",
				"error_class": string(types.ErrSkill),
				"reason":      "compile read failed",
				"path":        spec.Path,
				"latency_ms":  l.now().Sub(stepStart).Milliseconds(),
			}
			for k, v := range scorePayload {
				payload[k] = v
			}
			l.emit(ctx, runID, "skill.warning", payload)
			continue
		}
		fragments = append(fragments, string(content))
		workflowHints = append(workflowHints, spec.Description)
		enabled := parseEnabledTools(string(content))
		enabledTools = append(enabledTools, enabled...)
		payload := map[string]any{
			"name":          spec.Name,
			"path":          spec.Path,
			"action":        "compile",
			"status":        "success",
			"enabled_tools": len(enabled),
			"latency_ms":    l.now().Sub(stepStart).Milliseconds(),
		}
		for k, v := range scorePayload {
			payload[k] = v
		}
		l.emit(ctx, runID, "skill.loaded", payload)
	}

	workflowHints = resolveDirectiveConflicts(workflowHints)
	enabledTools = unique(enabledTools)

	return types.SkillBundle{
		SystemPromptFragments: fragments,
		EnabledTools:          enabledTools,
		WorkflowHints:         workflowHints,
	}, nil
}

func (l *Loader) selectSkills(ctx context.Context, specs []types.SkillSpec, input string) (explicit []types.SkillSpec, semantic []scoredSkill) {
	lower := strings.ToLower(input)
	scoring := l.triggerScoringConfig()
	scorer := l.scorer
	if scorer == nil {
		scorer = lexicalWeightedKeywordScorer{}
	}
	strategy := normalizedSkillTriggerStrategy(scoring.Strategy)
	candidates := make([]scoredSkill, 0, len(specs))
	for i, s := range specs {
		nameLower := strings.ToLower(s.Name)
		if strings.Contains(lower, "$"+nameLower) || strings.Contains(lower, nameLower) {
			explicit = append(explicit, s)
			continue
		}
		lexicalScore := scorer.Score(lower, s, scoring)
		finalScore := lexicalScore
		embeddingScore := 0.0
		fallbackReason := ""
		if strategy == runtimeconfig.SkillTriggerScoringStrategyLexicalPlusEmbedding {
			finalScore, embeddingScore, fallbackReason = l.scoreWithEmbedding(ctx, lower, s, scoring, lexicalScore)
		}
		if scoring.SuppressLowConfidence && finalScore < scoring.ConfidenceThreshold {
			continue
		}
		if !scoring.SuppressLowConfidence && finalScore <= 0 {
			continue
		}
		candidates = append(candidates, scoredSkill{
			spec:           s,
			score:          finalScore,
			index:          i,
			strategy:       strategy,
			lexicalScore:   lexicalScore,
			embeddingScore: embeddingScore,
			fallbackReason: fallbackReason,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.score != right.score {
			return left.score > right.score
		}
		switch scoring.TieBreak {
		case runtimeconfig.SkillTriggerScoringTieBreakFirstRegistered:
			return left.index < right.index
		default:
			if left.spec.Priority != right.spec.Priority {
				return left.spec.Priority > right.spec.Priority
			}
			if left.spec.Name != right.spec.Name {
				return left.spec.Name < right.spec.Name
			}
			return left.index < right.index
		}
	})
	return explicit, candidates
}

func tokenize(in string) []string {
	f := func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_'
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

func parseSkillMeta(path string) (desc string, triggers []string, priority int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, 0
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
		if strings.HasPrefix(strings.ToLower(t), "priority:") {
			if value, convErr := parsePriority(strings.TrimSpace(strings.TrimPrefix(t, "priority:"))); convErr == nil {
				priority = value
			}
		}
	}
	return desc, triggers, priority
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

type skillTriggerScorer interface {
	Score(input string, s types.SkillSpec, cfg runtimeconfig.SkillTriggerScoringConfig) float64
}

type lexicalWeightedKeywordScorer struct{}

func (lexicalWeightedKeywordScorer) Score(input string, s types.SkillSpec, cfg runtimeconfig.SkillTriggerScoringConfig) float64 {
	if strings.TrimSpace(input) == "" {
		return 0
	}
	hay := strings.ToLower(strings.Join([]string{s.Name, s.Description, strings.Join(s.Triggers, " ")}, " "))
	if hay == "" {
		return 0
	}
	inputTokens := tokenize(input)
	if len(inputTokens) == 0 {
		return 0
	}
	weights := cfg.KeywordWeights
	var totalWeight float64
	var hitWeight float64
	for _, token := range inputTokens {
		weight := 1.0
		if custom, ok := weights[token]; ok && custom > 0 {
			weight = custom
		}
		totalWeight += weight
		if strings.Contains(hay, token) {
			hitWeight += weight
		}
	}
	if totalWeight <= 0 {
		return 0
	}
	return hitWeight / totalWeight
}

type scoredSkill struct {
	spec           types.SkillSpec
	score          float64
	index          int
	strategy       string
	lexicalScore   float64
	embeddingScore float64
	fallbackReason string
}

func normalizedSkillTriggerStrategy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case runtimeconfig.SkillTriggerScoringStrategyLexicalPlusEmbedding:
		return runtimeconfig.SkillTriggerScoringStrategyLexicalPlusEmbedding
	default:
		return runtimeconfig.SkillTriggerScoringStrategyLexicalWeightedKeywords
	}
}

func (l *Loader) scoreWithEmbedding(
	ctx context.Context,
	input string,
	s types.SkillSpec,
	cfg runtimeconfig.SkillTriggerScoringConfig,
	lexicalScore float64,
) (finalScore float64, embeddingScore float64, fallbackReason string) {
	if !cfg.Embedding.Enabled {
		return lexicalScore, 0, skillEmbeddingFallbackScorerMissing
	}
	if l.embeddingScorer == nil {
		return lexicalScore, 0, skillEmbeddingFallbackScorerMissing
	}

	timeout := cfg.Embedding.Timeout
	if timeout <= 0 {
		timeout = runtimeconfig.DefaultConfig().Skill.TriggerScoring.Embedding.Timeout
	}
	scoreCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	score, err := l.embeddingScorer.Score(scoreCtx, SkillTriggerEmbeddingScoreRequest{
		Provider:  cfg.Embedding.Provider,
		Model:     cfg.Embedding.Model,
		Input:     input,
		SkillName: s.Name,
		SkillDesc: s.Description,
		Triggers:  append([]string(nil), s.Triggers...),
	})
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(scoreCtx.Err(), context.DeadlineExceeded) {
		return lexicalScore, 0, skillEmbeddingFallbackTimeout
	}
	if err != nil {
		return lexicalScore, 0, skillEmbeddingFallbackError
	}
	if score != score || score < 0 || score > 1 {
		return lexicalScore, 0, skillEmbeddingFallbackInvalidScore
	}
	final := cfg.Embedding.LexicalWeight*lexicalScore + cfg.Embedding.EmbeddingWeight*score
	return final, score, ""
}

func scorePayloadFromMeta(meta scoredSkill) map[string]any {
	payload := map[string]any{
		"strategy":    meta.strategy,
		"final_score": meta.score,
	}
	if meta.strategy == runtimeconfig.SkillTriggerScoringStrategyLexicalPlusEmbedding {
		payload["embedding_score"] = meta.embeddingScore
	}
	if strings.TrimSpace(meta.fallbackReason) != "" {
		payload["fallback_reason"] = strings.TrimSpace(meta.fallbackReason)
	}
	return payload
}

func parsePriority(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, errors.New("empty priority")
	}
	var value int
	_, err := fmt.Sscanf(raw, "%d", &value)
	return value, err
}

func (l *Loader) triggerScoringConfig() runtimeconfig.SkillTriggerScoringConfig {
	if l.runtimeMgr != nil {
		return l.runtimeMgr.EffectiveConfig().Skill.TriggerScoring
	}
	return runtimeconfig.DefaultConfig().Skill.TriggerScoring
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

var _ types.SkillLoader = (*Loader)(nil)

// NewDefault builds a loader with default settings and no event handler.
func NewDefault() *Loader {
	return New(nil)
}

// MustCompile compiles a skill bundle and panics on error for bootstrap-time use cases.
func MustCompile(ctx context.Context, loader types.SkillLoader, specs []types.SkillSpec, in types.SkillInput) types.SkillBundle {
	bundle, err := loader.Compile(ctx, specs, in)
	if err != nil {
		panic(fmt.Sprintf("compile skills: %v", err))
	}
	return bundle
}
