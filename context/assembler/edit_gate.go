package assembler

import (
	"strings"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	contextEditGateDecisionAllow           = "allow.threshold_met"
	contextEditGateDecisionDenyNoSavings   = "deny.no_savings"
	contextEditGateDecisionDenySavedTokens = "deny.saved_tokens_below_threshold"
	contextEditGateDecisionDenyGainRatio   = "deny.gain_ratio_below_threshold"
	contextEditGateDecisionDenyConfig      = "deny.config_conflict"
	contextEditGateDecisionBypassNoop      = "bypass.noop"
	contextEditGateDecisionBypassDisabled  = "bypass.disabled"
)

type contextEditGateResult struct {
	Chunks               []string
	EstimatedSavedTokens int
	GainRatio            float64
	Decision             string
}

func applyContextEditGate(chunks []string, cfg runtimeconfig.RuntimeContextJITEditGateConfig) contextEditGateResult {
	out := contextEditGateResult{
		Chunks: append([]string(nil), chunks...),
	}
	if len(chunks) == 0 {
		out.Decision = contextEditGateDecisionBypassNoop
		return out
	}
	if !cfg.Enabled {
		out.Decision = contextEditGateDecisionBypassDisabled
		return out
	}
	if cfg.ClearAtLeastTokens <= 0 || cfg.MinGainRatio <= 0 {
		out.Decision = contextEditGateDecisionDenyConfig
		return out
	}

	edited, saved := dedupeChunksForEditGate(chunks)
	out.EstimatedSavedTokens = saved
	if saved <= 0 {
		out.Decision = contextEditGateDecisionDenyNoSavings
		return out
	}
	totalTokens := 0
	for _, chunk := range chunks {
		totalTokens += estimateReferenceTokens(chunk)
	}
	if totalTokens <= 0 {
		out.Decision = contextEditGateDecisionDenyNoSavings
		return out
	}
	out.GainRatio = float64(saved) / float64(totalTokens)
	if saved < cfg.ClearAtLeastTokens {
		out.Decision = contextEditGateDecisionDenySavedTokens
		return out
	}
	if out.GainRatio < cfg.MinGainRatio {
		out.Decision = contextEditGateDecisionDenyGainRatio
		return out
	}
	out.Decision = contextEditGateDecisionAllow
	out.Chunks = edited
	return out
}

func dedupeChunksForEditGate(chunks []string) ([]string, int) {
	if len(chunks) == 0 {
		return nil, 0
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(chunks))
	savedTokens := 0
	for _, chunk := range chunks {
		key := strings.TrimSpace(chunk)
		if _, ok := seen[key]; ok {
			savedTokens += estimateReferenceTokens(chunk)
			continue
		}
		seen[key] = struct{}{}
		out = append(out, chunk)
	}
	return out, savedTokens
}
