package observability

import (
	"context"

	"github.com/FelixSeptem/baymax/core/types"
	mcpdiag "github.com/FelixSeptem/baymax/mcp/diag"
	obsTrace "github.com/FelixSeptem/baymax/observability/trace"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func EmitEvent(ctx context.Context, h types.EventHandler, ev types.Event) {
	if h == nil {
		return
	}
	if ev.Version == "" {
		ev.Version = types.EventSchemaVersionV1
	}
	ev.TraceID = obsTrace.TraceIDFromContext(ctx)
	ev.SpanID = obsTrace.SpanIDFromContext(ctx)
	h.OnEvent(ctx, ev)
}

func RecordCall(diagStore *mcpdiag.Store, runtimeMgr *runtimeconfig.Manager, runID string, rec mcpdiag.CallRecord) {
	if diagStore != nil {
		diagStore.Add(rec)
	}
	if runtimeMgr != nil {
		runtimeMgr.RecordCall(runtimediag.CallRecord{
			Time:           rec.Time,
			Component:      "mcp",
			Transport:      rec.Transport,
			Profile:        rec.Profile,
			RunID:          runID,
			CallID:         rec.CallID,
			Name:           rec.Tool,
			Action:         rec.Action,
			LatencyMs:      rec.LatencyMs,
			RetryCount:     rec.RetryCount,
			ReconnectCount: rec.ReconnectCount,
			ErrorClass:     rec.ErrorClass,
		})
	}
}

func RecentCalls(diagStore *mcpdiag.Store, runtimeMgr *runtimeconfig.Manager, n int) []mcpdiag.CallRecord {
	if runtimeMgr != nil {
		items := runtimeMgr.RecentCalls(n)
		out := make([]mcpdiag.CallRecord, 0, len(items))
		for _, rec := range items {
			out = append(out, mcpdiag.CallRecord{
				Time:           rec.Time,
				Transport:      rec.Transport,
				Profile:        rec.Profile,
				RunID:          rec.RunID,
				CallID:         rec.CallID,
				Tool:           rec.Name,
				Action:         rec.Action,
				LatencyMs:      rec.LatencyMs,
				RetryCount:     rec.RetryCount,
				ReconnectCount: rec.ReconnectCount,
				ErrorClass:     rec.ErrorClass,
			})
		}
		return out
	}
	if diagStore == nil {
		return nil
	}
	return diagStore.Recent(n)
}
