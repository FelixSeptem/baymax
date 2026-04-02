package trace

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	TraceExportStatusDisabled = "disabled"
	TraceExportStatusSuccess  = "success"
	TraceExportStatusDegraded = "degraded"
	TraceExportStatusFailed   = "failed"
)

const (
	TraceExportProtocolGRPC         = "grpc"
	TraceExportProtocolHTTPProtobuf = "http/protobuf"
)

const (
	TraceExportReasonNone                 = ""
	TraceExportReasonCollectorUnreachable = "trace.export.collector_unreachable"
	TraceExportReasonTimeout              = "trace.export.timeout"
	TraceExportReasonAuthFailed           = "trace.export.auth_failed"
	TraceExportReasonCanceled             = "trace.export.canceled"
	TraceExportReasonUnknown              = "trace.export.error"
)

const (
	TraceExportOnErrorFailFast         = "fail_fast"
	TraceExportOnErrorDegradeAndRecord = "degrade_and_record"
)

type ExportConfig struct {
	Enabled            bool
	Endpoint           string
	Protocol           string
	SchemaVersion      string
	OnError            string
	ExportTimeout      time.Duration
	ResourceAttributes map[string]string
}

type ExportPayload struct {
	SchemaVersion string
	Spans         []SemanticSpan
}

type ExportRequest struct {
	Endpoint           string
	Protocol           string
	SchemaVersion      string
	Spans              []SemanticSpan
	ResourceAttributes map[string]string
}

type ExportResult struct {
	Status        string
	ReasonCode    string
	SchemaVersion string
}

type CollectorExporter interface {
	Export(ctx context.Context, req ExportRequest) error
}

type ExportRuntime struct {
	exporter CollectorExporter
}

func NewExportRuntime(exporter CollectorExporter) *ExportRuntime {
	return &ExportRuntime{exporter: exporter}
}

func (r *ExportRuntime) Export(ctx context.Context, cfg ExportConfig, payload ExportPayload) ExportResult {
	schemaVersion := strings.ToLower(strings.TrimSpace(payload.SchemaVersion))
	if schemaVersion == "" {
		schemaVersion = strings.ToLower(strings.TrimSpace(cfg.SchemaVersion))
	}
	if schemaVersion == "" {
		schemaVersion = OTelSemconvVersionV1
	}

	if !cfg.Enabled {
		return ExportResult{
			Status:        TraceExportStatusDisabled,
			ReasonCode:    TraceExportReasonNone,
			SchemaVersion: schemaVersion,
		}
	}
	if strings.TrimSpace(cfg.Endpoint) == "" || r == nil || r.exporter == nil {
		return classifyExportErrorResult(errors.New("collector endpoint unavailable"), cfg, schemaVersion)
	}

	timeout := cfg.ExportTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	exportCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := r.exporter.Export(exportCtx, ExportRequest{
		Endpoint:           strings.TrimSpace(cfg.Endpoint),
		Protocol:           strings.ToLower(strings.TrimSpace(cfg.Protocol)),
		SchemaVersion:      schemaVersion,
		Spans:              NormalizeSemanticSpans(payload.Spans),
		ResourceAttributes: cloneStringMap(cfg.ResourceAttributes),
	})
	if err != nil {
		return classifyExportErrorResult(err, cfg, schemaVersion)
	}
	return ExportResult{
		Status:        TraceExportStatusSuccess,
		ReasonCode:    TraceExportReasonNone,
		SchemaVersion: schemaVersion,
	}
}

func classifyExportErrorResult(err error, cfg ExportConfig, schemaVersion string) ExportResult {
	reason := classifyExportError(err)
	status := TraceExportStatusDegraded
	if strings.ToLower(strings.TrimSpace(cfg.OnError)) == TraceExportOnErrorFailFast {
		status = TraceExportStatusFailed
	}
	return ExportResult{
		Status:        status,
		ReasonCode:    reason,
		SchemaVersion: schemaVersion,
	}
}

func classifyExportError(err error) string {
	if err == nil {
		return TraceExportReasonNone
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return TraceExportReasonTimeout
	case errors.Is(err, context.Canceled):
		return TraceExportReasonCanceled
	}
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "unreachable"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "dial tcp"):
		return TraceExportReasonCollectorUnreachable
	case strings.Contains(lower, "unauthorized"),
		strings.Contains(lower, "forbidden"),
		strings.Contains(lower, "401"),
		strings.Contains(lower, "403"):
		return TraceExportReasonAuthFailed
	default:
		return TraceExportReasonUnknown
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	return out
}
