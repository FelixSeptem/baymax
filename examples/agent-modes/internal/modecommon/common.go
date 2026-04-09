package modecommon

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	VariantMinimal    = "minimal"
	VariantProduction = "production-ish"

	defaultRuntimePath = "core/runner,tool/local,runtime/config"
)

func ComposeRuntimePath(modeDomains []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 3+len(modeDomains))
	for _, item := range append(strings.Split(defaultRuntimePath, ","), modeDomains...) {
		entry := strings.TrimSpace(item)
		if entry == "" {
			continue
		}
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		out = append(out, entry)
	}
	return out
}

func RuntimePathStatus(calls []types.ToolCallSummary, expectedCount int) string {
	if len(calls) != expectedCount || expectedCount == 0 {
		return "failed"
	}
	for _, call := range calls {
		if call.Error != nil {
			return "failed"
		}
	}
	return "ok"
}

func MarkerToken(in string) string {
	token := strings.ToLower(strings.TrimSpace(in))
	replacer := strings.NewReplacer("-", "_", ".", "_", "/", "_", " ", "_", ":", "_", ",", "_")
	token = replacer.Replace(token)
	for strings.Contains(token, "__") {
		token = strings.ReplaceAll(token, "__", "_")
	}
	return strings.Trim(token, "_")
}

func SemanticScore(parts ...string) int {
	h := fnv.New32a()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte("|"))
	}
	return int(h.Sum32()%900) + 100
}

func AsInt(value any) (int, bool) {
	switch tv := value.(type) {
	case int:
		return tv, true
	case int32:
		return int(tv), true
	case int64:
		return int(tv), true
	case float64:
		return int(tv), true
	default:
		return 0, false
	}
}

func ComputeSignature(finalAnswer string, calls []types.ToolCallSummary) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(finalAnswer))
	for _, call := range calls {
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(call.Name))
		if call.Error != nil {
			_, _ = h.Write([]byte(call.Error.Class))
			_, _ = h.Write([]byte(call.Error.Message))
		}
	}
	return h.Sum64()
}

func EnsureVariant(variant string) error {
	if variant == VariantMinimal || variant == VariantProduction {
		return nil
	}
	return fmt.Errorf("unsupported variant: %s", variant)
}
