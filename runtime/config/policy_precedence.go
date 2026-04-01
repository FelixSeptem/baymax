package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const (
	RuntimePolicyPrecedenceVersionPolicyStackV1 = "policy_stack.v1"

	RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder = "lexical_code_then_source_order"

	RuntimePolicyDecisionAllow = "allow"
	RuntimePolicyDecisionDeny  = "deny"
)

const (
	RuntimePolicyStageActionGate         = "action_gate"
	RuntimePolicyStageSecurityS2         = "security_s2"
	RuntimePolicyStageSandboxAction      = "sandbox_action"
	RuntimePolicyStageSandboxEgress      = "sandbox_egress"
	RuntimePolicyStageAdapterAllowlist   = "adapter_allowlist"
	RuntimePolicyStageReadinessAdmission = "readiness_admission"
)

var runtimePolicyCanonicalStageOrder = []string{
	RuntimePolicyStageActionGate,
	RuntimePolicyStageSecurityS2,
	RuntimePolicyStageSandboxAction,
	RuntimePolicyStageSandboxEgress,
	RuntimePolicyStageAdapterAllowlist,
	RuntimePolicyStageReadinessAdmission,
}

type RuntimePolicyConfig struct {
	Precedence     RuntimePolicyPrecedenceConfig     `json:"precedence"`
	TieBreaker     RuntimePolicyTieBreakerConfig     `json:"tie_breaker"`
	Explainability RuntimePolicyExplainabilityConfig `json:"explainability"`
}

type RuntimePolicyPrecedenceConfig struct {
	Version string         `json:"version"`
	Matrix  map[string]int `json:"matrix"`
}

type RuntimePolicyTieBreakerConfig struct {
	Mode        string   `json:"mode"`
	SourceOrder []string `json:"source_order"`
}

type RuntimePolicyExplainabilityConfig struct {
	Enabled bool `json:"enabled"`
}

type RuntimePolicyCandidate struct {
	Stage    string `json:"stage"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Decision string `json:"decision,omitempty"`
}

type RuntimePolicyDecisionResult struct {
	Version            string                   `json:"version"`
	WinnerStage        string                   `json:"winner_stage,omitempty"`
	DenySource         string                   `json:"deny_source,omitempty"`
	TieBreakReason     string                   `json:"tie_break_reason,omitempty"`
	PolicyDecisionPath []RuntimePolicyCandidate `json:"policy_decision_path,omitempty"`
	winnerCandidate    RuntimePolicyCandidate
}

func RuntimePolicyCanonicalStages() []string {
	return append([]string(nil), runtimePolicyCanonicalStageOrder...)
}

func DefaultRuntimePolicyPrecedenceMatrix() map[string]int {
	out := make(map[string]int, len(runtimePolicyCanonicalStageOrder))
	for idx, stage := range runtimePolicyCanonicalStageOrder {
		out[stage] = idx + 1
	}
	return out
}

func EvaluateRuntimePolicyDecision(cfg RuntimePolicyConfig, candidates []RuntimePolicyCandidate) (RuntimePolicyDecisionResult, error) {
	normalized := normalizeRuntimePolicyConfig(cfg)
	if err := ValidateRuntimePolicyConfig(normalized); err != nil {
		return RuntimePolicyDecisionResult{}, err
	}
	normalizedCandidates := normalizeRuntimePolicyCandidates(candidates)
	if len(normalizedCandidates) == 0 {
		return RuntimePolicyDecisionResult{
			Version: strings.TrimSpace(normalized.Precedence.Version),
		}, nil
	}
	if err := validateRuntimePolicyCandidates(normalized, normalizedCandidates); err != nil {
		return RuntimePolicyDecisionResult{}, err
	}

	orderedPath := sortRuntimePolicyCandidates(normalizedCandidates, normalized)
	blocking := make([]RuntimePolicyCandidate, 0, len(orderedPath))
	for i := range orderedPath {
		if orderedPath[i].Decision == RuntimePolicyDecisionDeny {
			blocking = append(blocking, orderedPath[i])
		}
	}
	scope := orderedPath
	if len(blocking) > 0 {
		scope = blocking
	}
	minRank := 0
	for i := range scope {
		rank := normalized.Precedence.Matrix[scope[i].Stage]
		if i == 0 || rank < minRank {
			minRank = rank
		}
	}
	top := make([]RuntimePolicyCandidate, 0, len(scope))
	for i := range scope {
		if normalized.Precedence.Matrix[scope[i].Stage] == minRank {
			top = append(top, scope[i])
		}
	}
	top = sortRuntimePolicyCandidates(top, normalized)
	winner := top[0]

	result := RuntimePolicyDecisionResult{
		Version:         strings.TrimSpace(normalized.Precedence.Version),
		WinnerStage:     winner.Stage,
		winnerCandidate: winner,
	}
	if normalized.Explainability.Enabled {
		result.PolicyDecisionPath = orderedPath
		if len(top) > 1 {
			result.TieBreakReason = RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder
		}
	}
	if winner.Decision == RuntimePolicyDecisionDeny {
		result.DenySource = winner.Source
		if result.DenySource == "" {
			result.DenySource = winner.Stage
		}
	}
	return result, nil
}

func ValidateRuntimePolicyConfig(cfg RuntimePolicyConfig) error {
	normalized := normalizeRuntimePolicyConfig(cfg)
	if normalized.Precedence.Version != RuntimePolicyPrecedenceVersionPolicyStackV1 {
		return fmt.Errorf(
			"runtime.policy.precedence.version must be %q, got %q",
			RuntimePolicyPrecedenceVersionPolicyStackV1,
			cfg.Precedence.Version,
		)
	}

	knownStages := make(map[string]struct{}, len(runtimePolicyCanonicalStageOrder))
	for i := range runtimePolicyCanonicalStageOrder {
		knownStages[runtimePolicyCanonicalStageOrder[i]] = struct{}{}
	}
	rankOwner := map[int]string{}
	for stage, rank := range normalized.Precedence.Matrix {
		if _, ok := knownStages[stage]; !ok {
			return fmt.Errorf("runtime.policy.precedence.matrix.%s uses unsupported stage", strings.TrimSpace(stage))
		}
		if rank <= 0 {
			return fmt.Errorf("runtime.policy.precedence.matrix.%s must be > 0, got %d", stage, rank)
		}
		if existing, ok := rankOwner[rank]; ok && existing != stage {
			return fmt.Errorf(
				"runtime.policy.precedence.matrix conflict: stages %q and %q share rank=%d",
				existing,
				stage,
				rank,
			)
		}
		rankOwner[rank] = stage
	}
	for i := range runtimePolicyCanonicalStageOrder {
		stage := runtimePolicyCanonicalStageOrder[i]
		if _, ok := normalized.Precedence.Matrix[stage]; !ok {
			return fmt.Errorf("runtime.policy.precedence.matrix.%s is required", stage)
		}
	}
	if len(rankOwner) != len(runtimePolicyCanonicalStageOrder) {
		return fmt.Errorf(
			"runtime.policy.precedence.matrix must map each canonical stage to a unique rank, got %d unique ranks",
			len(rankOwner),
		)
	}

	if normalized.TieBreaker.Mode != RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder {
		return fmt.Errorf(
			"runtime.policy.tie_breaker.mode must be %q, got %q",
			RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder,
			cfg.TieBreaker.Mode,
		)
	}
	if len(normalized.TieBreaker.SourceOrder) == 0 {
		return fmt.Errorf("runtime.policy.tie_breaker.source_order must not be empty")
	}
	seenSource := map[string]struct{}{}
	for i := range normalized.TieBreaker.SourceOrder {
		stage := normalized.TieBreaker.SourceOrder[i]
		if _, ok := knownStages[stage]; !ok {
			return fmt.Errorf(
				"runtime.policy.tie_breaker.source_order[%d] uses unsupported stage %q",
				i,
				cfg.TieBreaker.SourceOrder[i],
			)
		}
		if _, ok := seenSource[stage]; ok {
			return fmt.Errorf("runtime.policy.tie_breaker.source_order contains duplicate stage %q", stage)
		}
		seenSource[stage] = struct{}{}
	}
	if len(seenSource) != len(runtimePolicyCanonicalStageOrder) {
		return fmt.Errorf(
			"runtime.policy.tie_breaker.source_order must include all canonical stages, got %d",
			len(seenSource),
		)
	}
	return nil
}

func buildRuntimePolicyConfig(v *viper.Viper) (RuntimePolicyConfig, error) {
	base := DefaultConfig().Runtime.Policy
	cfg := RuntimePolicyConfig{
		Precedence: RuntimePolicyPrecedenceConfig{
			Version: strings.ToLower(strings.TrimSpace(v.GetString("runtime.policy.precedence.version"))),
			Matrix:  map[string]int{},
		},
		TieBreaker: RuntimePolicyTieBreakerConfig{
			Mode: strings.ToLower(strings.TrimSpace(v.GetString("runtime.policy.tie_breaker.mode"))),
		},
		Explainability: RuntimePolicyExplainabilityConfig{},
	}
	if strings.TrimSpace(cfg.Precedence.Version) == "" {
		cfg.Precedence.Version = base.Precedence.Version
	}
	for i := range runtimePolicyCanonicalStageOrder {
		stage := runtimePolicyCanonicalStageOrder[i]
		key := "runtime.policy.precedence.matrix." + stage
		rank, err := strictIntConfigValue(v, key)
		if err != nil {
			return RuntimePolicyConfig{}, err
		}
		cfg.Precedence.Matrix[stage] = rank
	}
	rawMatrix := v.GetStringMap("runtime.policy.precedence.matrix")
	for rawStage, rawValue := range rawMatrix {
		stage := strings.ToLower(strings.TrimSpace(rawStage))
		if stage == "" {
			return RuntimePolicyConfig{}, fmt.Errorf("runtime.policy.precedence.matrix contains empty stage name")
		}
		if _, ok := cfg.Precedence.Matrix[stage]; ok {
			continue
		}
		rank, err := strictIntAnyConfigValue(rawValue, "runtime.policy.precedence.matrix."+rawStage)
		if err != nil {
			return RuntimePolicyConfig{}, err
		}
		cfg.Precedence.Matrix[stage] = rank
	}

	sourceOrder := normalizeRuntimePolicyStageList(v.GetStringSlice("runtime.policy.tie_breaker.source_order"))
	if len(sourceOrder) == 0 {
		sourceOrder = append([]string(nil), base.TieBreaker.SourceOrder...)
	}
	cfg.TieBreaker.SourceOrder = sourceOrder

	explainabilityEnabled, err := strictBoolConfigValue(v, "runtime.policy.explainability.enabled")
	if err != nil {
		return RuntimePolicyConfig{}, err
	}
	cfg.Explainability.Enabled = explainabilityEnabled
	return normalizeRuntimePolicyConfig(cfg), nil
}

func normalizeRuntimePolicyConfig(in RuntimePolicyConfig) RuntimePolicyConfig {
	base := DefaultConfig().Runtime.Policy
	out := in
	out.Precedence.Version = strings.ToLower(strings.TrimSpace(out.Precedence.Version))
	if out.Precedence.Version == "" {
		out.Precedence.Version = strings.ToLower(strings.TrimSpace(base.Precedence.Version))
	}
	if len(out.Precedence.Matrix) == 0 {
		out.Precedence.Matrix = DefaultRuntimePolicyPrecedenceMatrix()
	} else {
		matrix := make(map[string]int, len(out.Precedence.Matrix))
		for rawStage, rank := range out.Precedence.Matrix {
			stage := strings.ToLower(strings.TrimSpace(rawStage))
			if stage == "" {
				continue
			}
			matrix[stage] = rank
		}
		out.Precedence.Matrix = matrix
	}
	out.TieBreaker.Mode = strings.ToLower(strings.TrimSpace(out.TieBreaker.Mode))
	if out.TieBreaker.Mode == "" {
		out.TieBreaker.Mode = strings.ToLower(strings.TrimSpace(base.TieBreaker.Mode))
	}
	out.TieBreaker.SourceOrder = normalizeRuntimePolicyStageList(out.TieBreaker.SourceOrder)
	if len(out.TieBreaker.SourceOrder) == 0 {
		out.TieBreaker.SourceOrder = append([]string(nil), base.TieBreaker.SourceOrder...)
	}
	return out
}

func normalizeRuntimePolicyCandidates(in []RuntimePolicyCandidate) []RuntimePolicyCandidate {
	if len(in) == 0 {
		return nil
	}
	out := make([]RuntimePolicyCandidate, 0, len(in))
	for i := range in {
		stage := strings.ToLower(strings.TrimSpace(in[i].Stage))
		if stage == "" {
			continue
		}
		decision := strings.ToLower(strings.TrimSpace(in[i].Decision))
		switch decision {
		case RuntimePolicyDecisionDeny:
		case RuntimePolicyDecisionAllow:
		default:
			decision = RuntimePolicyDecisionAllow
		}
		out = append(out, RuntimePolicyCandidate{
			Stage:    stage,
			Code:     strings.TrimSpace(in[i].Code),
			Source:   strings.ToLower(strings.TrimSpace(in[i].Source)),
			Decision: decision,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func validateRuntimePolicyCandidates(cfg RuntimePolicyConfig, candidates []RuntimePolicyCandidate) error {
	for i := range candidates {
		stage := strings.ToLower(strings.TrimSpace(candidates[i].Stage))
		if stage == "" {
			return fmt.Errorf("runtime.policy.candidates[%d].stage is required", i)
		}
		if _, ok := cfg.Precedence.Matrix[stage]; !ok {
			return fmt.Errorf("runtime.policy.candidates[%d].stage uses unsupported stage %q", i, candidates[i].Stage)
		}
	}
	return nil
}

func sortRuntimePolicyCandidates(in []RuntimePolicyCandidate, cfg RuntimePolicyConfig) []RuntimePolicyCandidate {
	if len(in) == 0 {
		return nil
	}
	out := append([]RuntimePolicyCandidate(nil), in...)
	sourceRank := make(map[string]int, len(cfg.TieBreaker.SourceOrder))
	for idx := range cfg.TieBreaker.SourceOrder {
		sourceRank[cfg.TieBreaker.SourceOrder[idx]] = idx
	}
	sort.SliceStable(out, func(i, j int) bool {
		li := cfg.Precedence.Matrix[out[i].Stage]
		lj := cfg.Precedence.Matrix[out[j].Stage]
		if li != lj {
			return li < lj
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		ri := resolveRuntimePolicySourceRank(sourceRank, out[i].Source)
		rj := resolveRuntimePolicySourceRank(sourceRank, out[j].Source)
		if ri != rj {
			return ri < rj
		}
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Decision != out[j].Decision {
			return out[i].Decision < out[j].Decision
		}
		return out[i].Stage < out[j].Stage
	})
	return out
}

func resolveRuntimePolicySourceRank(sourceRank map[string]int, source string) int {
	if len(sourceRank) == 0 {
		return 1 << 30
	}
	normalized := normalizeRuntimePolicySourceToken(source)
	if normalized == "" {
		return 1 << 30
	}
	if rank, ok := sourceRank[normalized]; ok {
		return rank
	}
	alias := strings.ReplaceAll(normalized, ".", "_")
	alias = strings.ReplaceAll(alias, ":", "_")
	alias = strings.ReplaceAll(alias, "/", "_")
	alias = strings.ReplaceAll(alias, "-", "_")
	if rank, ok := sourceRank[alias]; ok {
		return rank
	}
	parts := strings.Split(normalized, ".")
	if len(parts) > 0 {
		if rank, ok := sourceRank[parts[0]]; ok {
			return rank
		}
	}
	if len(parts) > 1 {
		combined := strings.TrimSpace(parts[0]) + "_" + strings.TrimSpace(parts[1])
		if rank, ok := sourceRank[combined]; ok {
			return rank
		}
	}
	return 1 << 30
}

func normalizeRuntimePolicySourceToken(in string) string {
	return strings.ToLower(strings.TrimSpace(in))
}

func normalizeRuntimePolicyStageList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for i := range in {
		chunks := strings.Split(in[i], ",")
		for j := range chunks {
			item := strings.ToLower(strings.TrimSpace(chunks[j]))
			if item == "" {
				continue
			}
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func strictIntConfigValue(v *viper.Viper, key string) (int, error) {
	raw := v.Get(key)
	if raw == nil {
		return v.GetInt(key), nil
	}
	return strictIntAnyConfigValue(raw, key)
}

func strictIntAnyConfigValue(raw any, key string) (int, error) {
	switch value := raw.(type) {
	case int:
		return value, nil
	case int8:
		return int(value), nil
	case int16:
		return int(value), nil
	case int32:
		return int(value), nil
	case int64:
		return int(value), nil
	case uint:
		return int(value), nil
	case uint8:
		return int(value), nil
	case uint16:
		return int(value), nil
	case uint32:
		return int(value), nil
	case uint64:
		return int(value), nil
	case float64:
		parsed := int(value)
		if float64(parsed) != value {
			return 0, fmt.Errorf("%s must be an integer, got %v", key, raw)
		}
		return parsed, nil
	case float32:
		parsed := int(value)
		if float32(parsed) != value {
			return 0, fmt.Errorf("%s must be an integer, got %v", key, raw)
		}
		return parsed, nil
	case string:
		trimmed := strings.TrimSpace(value)
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer, got %q", key, value)
		}
		return parsed, nil
	default:
		trimmed := strings.TrimSpace(fmt.Sprint(raw))
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer, got %v", key, raw)
		}
		return parsed, nil
	}
}
