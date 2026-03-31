package provider

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	memoryspi "github.com/FelixSeptem/baymax/memory"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type memoryProvider struct {
	facade *memoryspi.Facade
	legacy Provider
}

func newMemoryProvider(cfg Config) (Provider, error) {
	memoryCfg := cfg.Memory
	if strings.TrimSpace(memoryCfg.Mode) == "" {
		memoryCfg = runtimeconfig.DefaultConfig().Runtime.Memory
	}
	if strings.TrimSpace(memoryCfg.Builtin.RootDir) == "" {
		base := runtimeconfig.DefaultConfig().Runtime.Memory.Builtin.RootDir
		if strings.TrimSpace(cfg.FilePath) != "" {
			base = filepath.Join(filepath.Dir(strings.TrimSpace(cfg.FilePath)), "memory-store")
		}
		memoryCfg.Builtin.RootDir = base
	}
	facadeCfg := memoryspi.Config{
		Mode: memoryCfg.Mode,
		External: memoryspi.ExternalConfig{
			Provider:        memoryCfg.External.Provider,
			Profile:         memoryCfg.External.Profile,
			ContractVersion: memoryCfg.External.ContractVersion,
		},
		Builtin: memoryspi.BuiltinConfig{
			RootDir: memoryCfg.Builtin.RootDir,
			Compaction: memoryspi.FilesystemCompactionConfig{
				Enabled:     memoryCfg.Builtin.Compaction.Enabled,
				MinOps:      memoryCfg.Builtin.Compaction.MinOps,
				MaxWALBytes: memoryCfg.Builtin.Compaction.MaxWALBytes,
			},
		},
		Fallback: memoryspi.FallbackConfig{
			Policy: memoryCfg.Fallback.Policy,
		},
	}
	externalFactory := func(ext memoryspi.ExternalConfig) (memoryspi.Engine, error) {
		if strings.TrimSpace(cfg.External.Endpoint) == "" {
			return nil, &memoryspi.Error{
				Operation: memoryspi.OperationQuery,
				Code:      memoryspi.ReasonCodeProviderUnavailable,
				Layer:     memoryspi.LayerRuntime,
				Message:   "context stage2 external endpoint is required for runtime.memory.mode=external_spi",
			}
		}
		return &httpMemoryEngine{
			p: &httpProvider{
				name:   strings.TrimSpace(ext.Provider),
				cfg:    cfg.External,
				client: defaultHTTPClient(),
			},
		}, nil
	}
	facade, err := memoryspi.NewFacade(facadeCfg, externalFactory)
	if err != nil {
		return nil, fmt.Errorf("init context stage2 memory facade: %w", err)
	}
	out := &memoryProvider{facade: facade}
	if strings.TrimSpace(cfg.FilePath) != "" {
		out.legacy = &fileProvider{path: strings.TrimSpace(cfg.FilePath)}
	}
	return out, nil
}

func (m *memoryProvider) Name() string {
	return runtimeconfig.ContextStage2ProviderMemory
}

func (m *memoryProvider) Fetch(ctx context.Context, req Request) (Response, error) {
	if m == nil || m.facade == nil {
		return Response{}, ErrProviderNotReady
	}
	if err := ctx.Err(); err != nil {
		return Response{}, classifyTransportError(err)
	}
	namespace := stage2Namespace(req)
	opResp, err := m.facade.Query(memoryspi.QueryRequest{
		OperationID: strings.TrimSpace(req.RunID),
		Namespace:   namespace,
		SessionID:   strings.TrimSpace(req.SessionID),
		RunID:       strings.TrimSpace(req.RunID),
		Query:       strings.TrimSpace(req.Input),
		MaxItems:    req.MaxItems,
	})
	if err != nil {
		return Response{}, mapMemoryError(err)
	}
	chunks := recordsToChunks(opResp.Records)
	if len(chunks) == 0 && m.legacy != nil {
		legacyResp, legacyErr := m.legacy.Fetch(ctx, req)
		if legacyErr == nil && len(legacyResp.Chunks) > 0 {
			m.backfillLegacy(req, namespace, legacyResp.Chunks)
			return legacyResp, nil
		}
	}
	return Response{
		Chunks: chunks,
		Meta: map[string]any{
			"source":                      "memory",
			"matched":                     len(chunks),
			"reason":                      memoryReasonToStage2Reason(opResp.ReasonCode),
			"reason_code":                 strings.TrimSpace(opResp.ReasonCode),
			"error_layer":                 "",
			"profile":                     strings.TrimSpace(opResp.Profile),
			"template_profile":            strings.TrimSpace(opResp.Profile),
			"template_resolution_source":  runtimeconfig.Stage2TemplateResolutionExplicitOnly,
			"hint_applied":                false,
			"hint_mismatch_reason":        "",
			"memory_mode":                 strings.TrimSpace(opResp.Mode),
			"memory_provider":             strings.TrimSpace(opResp.Provider),
			"memory_contract_version":     strings.TrimSpace(opResp.ContractVersion),
			"memory_fallback_used":        opResp.FallbackUsed,
			"memory_fallback_reason_code": strings.TrimSpace(opResp.FallbackReasonCode),
		},
	}, nil
}

func (m *memoryProvider) backfillLegacy(req Request, namespace string, chunks []string) {
	if m == nil || m.facade == nil || len(chunks) == 0 {
		return
	}
	records := make([]memoryspi.Record, 0, len(chunks))
	sessionID := strings.TrimSpace(req.SessionID)
	runID := strings.TrimSpace(req.RunID)
	ts := time.Now().UTC().UnixNano()
	for i, chunk := range chunks {
		content := strings.TrimSpace(chunk)
		if content == "" {
			continue
		}
		records = append(records, memoryspi.Record{
			ID:        fmt.Sprintf("legacy-%d-%d", ts, i),
			Namespace: namespace,
			SessionID: sessionID,
			RunID:     runID,
			Content:   content,
		})
	}
	if len(records) == 0 {
		return
	}
	_, _ = m.facade.Upsert(memoryspi.UpsertRequest{
		OperationID: runID,
		Namespace:   namespace,
		Records:     records,
	})
}

func stage2Namespace(req Request) string {
	if sessionID := strings.TrimSpace(req.SessionID); sessionID != "" {
		return "session:" + sessionID
	}
	if runID := strings.TrimSpace(req.RunID); runID != "" {
		return "run:" + runID
	}
	return "default"
}

func recordsToChunks(records []memoryspi.Record) []string {
	if len(records) == 0 {
		return nil
	}
	out := make([]string, 0, len(records))
	for _, record := range records {
		content := strings.TrimSpace(record.Content)
		if content == "" {
			continue
		}
		out = append(out, content)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func memoryReasonToStage2Reason(reasonCode string) string {
	switch strings.ToLower(strings.TrimSpace(reasonCode)) {
	case memoryspi.ReasonCodeOK:
		return "ok"
	case memoryspi.ReasonCodeFallbackUsed:
		return "fallback"
	case memoryspi.ReasonCodeNotFound:
		return "empty"
	default:
		return "fetch_error"
	}
}

func mapMemoryError(err error) error {
	if err == nil {
		return nil
	}
	var memErr *memoryspi.Error
	if !errors.As(err, &memErr) {
		return &FetchError{
			Layer:   ErrorLayerProtocol,
			Code:    "memory_error",
			Message: "context stage2 memory provider failed",
			Cause:   err,
		}
	}
	layer := ErrorLayerProtocol
	switch strings.ToLower(strings.TrimSpace(memErr.Layer)) {
	case memoryspi.LayerTransport:
		layer = ErrorLayerTransport
	case memoryspi.LayerSemantic:
		layer = ErrorLayerSemantic
	}
	code := strings.TrimSpace(memErr.Code)
	if code == "" {
		code = "memory_error"
	}
	msg := strings.TrimSpace(memErr.Message)
	if msg == "" {
		msg = "context stage2 memory provider failed"
	}
	return &FetchError{
		Layer:   layer,
		Code:    code,
		Message: msg,
		Cause:   err,
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
