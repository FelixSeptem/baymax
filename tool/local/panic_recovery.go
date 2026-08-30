package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

func invokeWithPanicRecovery(ctx context.Context, invoker types.ToolInvokeFunc, call types.ToolCall) (result types.ToolResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("tool panic: %v", recovered)
		}
	}()
	return invoker(ctx, call)
}

func toolFailureDetails(err error, retryCount int) map[string]any {
	details := map[string]any{"retry_count": retryCount, "dispatch_phase": string(types.ActionPhaseTool)}
	if err != nil && strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "tool panic:") {
		details["panic_recovered"] = true
	}
	return details
}
