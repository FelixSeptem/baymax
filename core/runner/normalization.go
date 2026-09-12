package runner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

func normalizeLifecycleHooks(hooks []types.AgentLifecycleHook) []types.AgentLifecycleHook {
	if len(hooks) == 0 {
		return nil
	}
	out := make([]types.AgentLifecycleHook, 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			out = append(out, hook)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeToolMiddlewares(middlewares []types.ToolMiddleware) []types.ToolMiddleware {
	if len(middlewares) == 0 {
		return nil
	}
	out := make([]types.ToolMiddleware, 0, len(middlewares))
	for _, middleware := range middlewares {
		if middleware != nil {
			out = append(out, middleware)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeToolName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeActionGateDecision(in types.ActionGateDecision) types.ActionGateDecision {
	switch strings.ToLower(strings.TrimSpace(string(in))) {
	case string(types.ActionGateDecisionAllow):
		return types.ActionGateDecisionAllow
	case string(types.ActionGateDecisionDeny):
		return types.ActionGateDecisionDeny
	case string(types.ActionGateDecisionRequireConfirm):
		return types.ActionGateDecisionRequireConfirm
	default:
		return types.ActionGateDecisionDeny
	}
}

func marshalToolArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%v", args)
	}
	return string(raw)
}
