package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Manager struct {
	tracer oteltrace.Tracer
}

func NewManager(instrumentationName string) *Manager {
	if instrumentationName == "" {
		instrumentationName = "baymax"
	}
	return &Manager{tracer: otel.Tracer(instrumentationName)}
}

func (m *Manager) StartRun(ctx context.Context, runID string) (context.Context, oteltrace.Span) {
	return m.tracer.Start(ctx, "agent.run", oteltrace.WithAttributes(attribute.String("run.id", runID)))
}

func (m *Manager) StartStep(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
	return m.tracer.Start(ctx, name, oteltrace.WithAttributes(attrs...))
}

func TraceIDFromContext(ctx context.Context) string {
	sc := oteltrace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

func SpanIDFromContext(ctx context.Context) string {
	sc := oteltrace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.SpanID().String()
}
