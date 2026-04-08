package event

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	RuntimeExportStatusDisabled = "disabled"
	RuntimeExportStatusSuccess  = "success"
	RuntimeExportStatusDegraded = "degraded"
	RuntimeExportStatusFailed   = "failed"
)

const (
	RuntimeExportReasonQueueOverflow = "observability.export.queue_overflow"
	RuntimeExportReasonUnknown       = "observability.export.error"
)

type RuntimeExportSnapshot struct {
	Enabled        bool   `json:"enabled"`
	Profile        string `json:"profile"`
	Status         string `json:"status"`
	LastReasonCode string `json:"last_reason_code,omitempty"`
	OnError        string `json:"on_error"`
	QueueCapacity  int    `json:"queue_capacity"`
	QueueDepthPeak int    `json:"queue_depth_peak,omitempty"`
	ErrorTotal     int    `json:"error_total,omitempty"`
	DropTotal      int    `json:"drop_total,omitempty"`
}

type RuntimeExportError struct {
	Code    string
	Message string
	Err     error
}

func (e *RuntimeExportError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Err != nil {
		msg = strings.TrimSpace(e.Err.Error())
	}
	code := strings.TrimSpace(e.Code)
	switch {
	case code != "" && msg != "":
		return code + ": " + msg
	case code != "":
		return code
	default:
		return msg
	}
}

func (e *RuntimeExportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type RuntimeExporter interface {
	ExportEvents(ctx context.Context, events []types.Event) error
	Flush(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type RuntimeExporterFactory func(cfg runtimeconfig.RuntimeObservabilityExportConfig) (RuntimeExporter, error)

type RuntimeExporterResolver interface {
	Resolve(cfg runtimeconfig.RuntimeObservabilityExportConfig) (RuntimeExporter, error)
}

type defaultRuntimeExporterResolver struct {
	customFactories map[string]RuntimeExporterFactory
}

func newDefaultRuntimeExporterResolver(customFactories map[string]RuntimeExporterFactory) RuntimeExporterResolver {
	out := map[string]RuntimeExporterFactory{}
	for k, v := range customFactories {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "" || v == nil {
			continue
		}
		out[key] = v
	}
	return &defaultRuntimeExporterResolver{customFactories: out}
}

func (r *defaultRuntimeExporterResolver) Resolve(cfg runtimeconfig.RuntimeObservabilityExportConfig) (RuntimeExporter, error) {
	profile := strings.ToLower(strings.TrimSpace(cfg.Profile))
	switch profile {
	case "", runtimeconfig.RuntimeObservabilityExportProfileNone:
		return &noopRuntimeExporter{}, nil
	case runtimeconfig.RuntimeObservabilityExportProfileOTLP,
		runtimeconfig.RuntimeObservabilityExportProfileLangfuse,
		runtimeconfig.RuntimeObservabilityExportProfileCustom:
		if factory, ok := r.customFactories[profile]; ok {
			return factory(cfg)
		}
		return &endpointRuntimeExporter{
			profile:  profile,
			endpoint: strings.TrimSpace(cfg.Endpoint),
		}, nil
	default:
		return nil, &RuntimeExportError{
			Code:    runtimeconfig.ReadinessCodeObservabilityExportProfileInvalid,
			Message: fmt.Sprintf("unsupported observability export profile %q", cfg.Profile),
		}
	}
}

func canonicalizeRuntimeExportError(err error, profile string) RuntimeExportError {
	if err == nil {
		return RuntimeExportError{}
	}
	var typed *RuntimeExportError
	if errors.As(err, &typed) && typed != nil {
		code := normalizeRuntimeExportReasonCode(typed.Code)
		if code == "" {
			code = RuntimeExportReasonUnknown
		}
		return RuntimeExportError{
			Code:    code,
			Message: strings.TrimSpace(typed.Error()),
			Err:     typed.Unwrap(),
		}
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "queue_overflow"), strings.Contains(msg, "queue overflow"):
		return RuntimeExportError{Code: RuntimeExportReasonQueueOverflow, Message: strings.TrimSpace(err.Error()), Err: err}
	case strings.Contains(msg, "auth"), strings.Contains(msg, "401"), strings.Contains(msg, "403"):
		return RuntimeExportError{Code: runtimeconfig.ReadinessCodeObservabilityExportAuthInvalid, Message: strings.TrimSpace(err.Error()), Err: err}
	case strings.Contains(msg, "unavailable"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "connection"),
		strings.Contains(msg, "refused"),
		strings.Contains(msg, "dial tcp"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "sink_unavailable"),
		strings.Contains(msg, "127.0.0.1:9"),
		strings.Contains(msg, "localhost:9"),
		strings.Contains(msg, "[::1]:9"):
		return RuntimeExportError{Code: runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable, Message: strings.TrimSpace(err.Error()), Err: err}
	default:
		_ = profile
		return RuntimeExportError{Code: RuntimeExportReasonUnknown, Message: strings.TrimSpace(err.Error()), Err: err}
	}
}

func normalizeRuntimeExportReasonCode(code string) string {
	normalized := strings.TrimSpace(strings.ToLower(code))
	switch normalized {
	case runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable,
		runtimeconfig.ReadinessCodeObservabilityExportAuthInvalid,
		runtimeconfig.ReadinessCodeObservabilityExportProfileInvalid,
		RuntimeExportReasonQueueOverflow,
		RuntimeExportReasonUnknown:
		return normalized
	default:
		return ""
	}
}

type endpointRuntimeExporter struct {
	profile  string
	endpoint string
}

func (e *endpointRuntimeExporter) ExportEvents(_ context.Context, _ []types.Event) error {
	raw := strings.ToLower(strings.TrimSpace(e.endpoint))
	if raw == "" {
		return &RuntimeExportError{
			Code:    runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable,
			Message: "empty observability export endpoint",
		}
	}
	if strings.Contains(raw, "sink_unavailable") ||
		strings.Contains(raw, "127.0.0.1:9") ||
		strings.Contains(raw, "localhost:9") ||
		strings.Contains(raw, "[::1]:9") {
		return &RuntimeExportError{
			Code:    runtimeconfig.ReadinessCodeObservabilityExportSinkUnavailable,
			Message: "observability export sink is unavailable",
		}
	}
	if strings.TrimSpace(e.profile) == runtimeconfig.RuntimeObservabilityExportProfileLangfuse &&
		strings.Contains(raw, "auth_invalid") {
		return &RuntimeExportError{
			Code:    runtimeconfig.ReadinessCodeObservabilityExportAuthInvalid,
			Message: "observability export auth is invalid",
		}
	}
	return nil
}

func (e *endpointRuntimeExporter) Flush(_ context.Context) error {
	return nil
}

func (e *endpointRuntimeExporter) Shutdown(_ context.Context) error {
	return nil
}

type noopRuntimeExporter struct{}

func (e *noopRuntimeExporter) ExportEvents(_ context.Context, _ []types.Event) error {
	return nil
}

func (e *noopRuntimeExporter) Flush(_ context.Context) error {
	return nil
}

func (e *noopRuntimeExporter) Shutdown(_ context.Context) error {
	return nil
}

type runtimeExporterRuntime struct {
	mu       sync.Mutex
	resolver RuntimeExporterResolver

	signature string
	onError   string
	active    bool

	exporter RuntimeExporter
	queue    chan types.Event
	cancel   context.CancelFunc

	snapshot RuntimeExportSnapshot
}

func newRuntimeExporterRuntime(resolver RuntimeExporterResolver) *runtimeExporterRuntime {
	if resolver == nil {
		resolver = newDefaultRuntimeExporterResolver(nil)
	}
	return &runtimeExporterRuntime{resolver: resolver}
}

func (r *runtimeExporterRuntime) HandleEvent(cfg runtimeconfig.RuntimeObservabilityExportConfig, ev types.Event) {
	if r == nil {
		return
	}
	cancel := r.configureIfNeeded(cfg)
	if cancel != nil {
		cancel()
	}
	r.enqueue(ev)
}

func (r *runtimeExporterRuntime) Snapshot() RuntimeExportSnapshot {
	if r == nil {
		return RuntimeExportSnapshot{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot
}

func (r *runtimeExporterRuntime) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	r.active = false
	r.exporter = nil
	r.queue = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (r *runtimeExporterRuntime) configureIfNeeded(cfg runtimeconfig.RuntimeObservabilityExportConfig) context.CancelFunc {
	profile := strings.ToLower(strings.TrimSpace(cfg.Profile))
	if profile == "" {
		profile = runtimeconfig.RuntimeObservabilityExportProfileNone
	}
	onError := strings.ToLower(strings.TrimSpace(cfg.OnError))
	if onError == "" {
		onError = runtimeconfig.RuntimeObservabilityExportOnErrorDegradeAndRecord
	}
	queueCapacity := cfg.QueueCapacity
	if queueCapacity <= 0 {
		queueCapacity = runtimeconfig.DefaultConfig().Runtime.Observability.Export.QueueCapacity
	}
	maxBatchSize := cfg.MaxBatchSize
	if maxBatchSize <= 0 {
		maxBatchSize = runtimeconfig.DefaultConfig().Runtime.Observability.Export.MaxBatchSize
	}
	maxFlushLatency := cfg.MaxFlushLatency
	if maxFlushLatency <= 0 {
		maxFlushLatency = runtimeconfig.DefaultConfig().Runtime.Observability.Export.MaxFlushLatency
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	signature := fmt.Sprintf(
		"%t|%s|%s|%d|%d|%d|%s",
		cfg.Enabled,
		profile,
		endpoint,
		queueCapacity,
		maxBatchSize,
		maxFlushLatency,
		onError,
	)

	r.mu.Lock()
	defer r.mu.Unlock()
	if signature == r.signature {
		return nil
	}

	prevCancel := r.cancel
	r.signature = signature
	r.onError = onError
	r.snapshot = RuntimeExportSnapshot{
		Enabled:       cfg.Enabled,
		Profile:       profile,
		Status:        RuntimeExportStatusDisabled,
		OnError:       onError,
		QueueCapacity: queueCapacity,
	}
	r.active = false
	r.exporter = nil
	r.queue = nil
	r.cancel = nil

	if !cfg.Enabled || profile == runtimeconfig.RuntimeObservabilityExportProfileNone {
		return prevCancel
	}

	exporter, err := r.resolver.Resolve(runtimeconfig.RuntimeObservabilityExportConfig{
		Enabled:         cfg.Enabled,
		Profile:         profile,
		Endpoint:        endpoint,
		QueueCapacity:   queueCapacity,
		MaxBatchSize:    maxBatchSize,
		MaxFlushLatency: maxFlushLatency,
		OnError:         onError,
	})
	if err != nil {
		r.recordExportErrorLocked(canonicalizeRuntimeExportError(err, profile))
		return prevCancel
	}

	ctx, cancel := context.WithCancel(context.Background())
	q := make(chan types.Event, queueCapacity)
	r.exporter = exporter
	r.queue = q
	r.cancel = cancel
	r.active = true
	r.snapshot.Status = RuntimeExportStatusSuccess
	go r.worker(ctx, exporter, q, profile, maxBatchSize, maxFlushLatency)
	return prevCancel
}

func (r *runtimeExporterRuntime) enqueue(ev types.Event) {
	if r == nil {
		return
	}
	var cancel context.CancelFunc
	r.mu.Lock()
	if !r.active || r.queue == nil {
		r.mu.Unlock()
		return
	}
	select {
	case r.queue <- ev:
		if depth := len(r.queue); depth > r.snapshot.QueueDepthPeak {
			r.snapshot.QueueDepthPeak = depth
		}
		r.mu.Unlock()
		return
	default:
		r.snapshot.DropTotal++
		r.snapshot.LastReasonCode = RuntimeExportReasonQueueOverflow
		if r.onError == runtimeconfig.RuntimeObservabilityExportOnErrorFailFast {
			r.snapshot.ErrorTotal++
			r.snapshot.Status = RuntimeExportStatusFailed
			r.active = false
			cancel = r.cancel
			r.cancel = nil
			r.exporter = nil
			r.queue = nil
		} else if r.snapshot.Status != RuntimeExportStatusFailed {
			r.snapshot.Status = RuntimeExportStatusDegraded
		}
	}
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (r *runtimeExporterRuntime) worker(
	ctx context.Context,
	exporter RuntimeExporter,
	queue <-chan types.Event,
	profile string,
	maxBatchSize int,
	maxFlushLatency time.Duration,
) {
	if maxBatchSize <= 0 {
		maxBatchSize = runtimeconfig.DefaultConfig().Runtime.Observability.Export.MaxBatchSize
	}
	if maxFlushLatency <= 0 {
		maxFlushLatency = runtimeconfig.DefaultConfig().Runtime.Observability.Export.MaxFlushLatency
	}

	defer func() {
		flushCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		_ = exporter.Flush(flushCtx)
		cancel()
		_ = exporter.Shutdown(context.Background())
	}()

	stopTimer := func(timer *time.Timer, active *bool) {
		if !*active {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		*active = false
	}

	batch := make([]types.Event, 0, maxBatchSize)
	timer := time.NewTimer(maxFlushLatency)
	timerActive := false
	stopTimer(timer, &timerActive)
	defer stopTimer(timer, &timerActive)

	flushBatch := func(exportCtx context.Context) bool {
		if len(batch) == 0 {
			return false
		}
		exportBatch := append([]types.Event(nil), batch...)
		batch = batch[:0]
		err := exporter.ExportEvents(exportCtx, exportBatch)
		if err == nil {
			return false
		}
		canonical := canonicalizeRuntimeExportError(err, profile)
		var cancel context.CancelFunc
		r.mu.Lock()
		if strings.TrimSpace(canonical.Code) == "" {
			canonical.Code = RuntimeExportReasonUnknown
		}
		r.recordExportErrorLocked(canonical)
		if r.onError == runtimeconfig.RuntimeObservabilityExportOnErrorFailFast {
			r.active = false
			cancel = r.cancel
			r.cancel = nil
			r.exporter = nil
			r.queue = nil
		}
		r.mu.Unlock()
		if cancel != nil {
			cancel()
			return true
		}
		return false
	}

	drainQueueAndFlush := func() bool {
		for {
			select {
			case ev, ok := <-queue:
				if !ok {
					flushCtx, cancel := context.WithTimeout(context.Background(), maxFlushLatency)
					shouldStop := flushBatch(flushCtx)
					cancel()
					return shouldStop
				}
				batch = append(batch, ev)
				if len(batch) >= maxBatchSize {
					flushCtx, cancel := context.WithTimeout(context.Background(), maxFlushLatency)
					shouldStop := flushBatch(flushCtx)
					cancel()
					if shouldStop {
						return true
					}
				}
			default:
				flushCtx, cancel := context.WithTimeout(context.Background(), maxFlushLatency)
				shouldStop := flushBatch(flushCtx)
				cancel()
				return shouldStop
			}
		}
	}

	for {
		timerCh := (<-chan time.Time)(nil)
		if timerActive {
			timerCh = timer.C
		}
		select {
		case <-ctx.Done():
			stopTimer(timer, &timerActive)
			_ = drainQueueAndFlush()
			return
		case <-timerCh:
			timerActive = false
			if flushBatch(ctx) {
				return
			}
		case ev, ok := <-queue:
			if !ok {
				stopTimer(timer, &timerActive)
				_ = drainQueueAndFlush()
				return
			}
			batch = append(batch, ev)
			if len(batch) == 1 {
				timer.Reset(maxFlushLatency)
				timerActive = true
			}
			if len(batch) >= maxBatchSize {
				stopTimer(timer, &timerActive)
				if flushBatch(ctx) {
					return
				}
			}
		}
	}
}

func (r *runtimeExporterRuntime) recordExportErrorLocked(err RuntimeExportError) {
	r.snapshot.ErrorTotal++
	r.snapshot.LastReasonCode = strings.TrimSpace(err.Code)
	if r.onError == runtimeconfig.RuntimeObservabilityExportOnErrorFailFast {
		r.snapshot.Status = RuntimeExportStatusFailed
		return
	}
	if r.snapshot.Status != RuntimeExportStatusFailed {
		r.snapshot.Status = RuntimeExportStatusDegraded
	}
}
