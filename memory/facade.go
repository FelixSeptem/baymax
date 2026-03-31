package memory

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ContractVersionMemoryV1 = "memory.v1"
)

const (
	ModeExternalSPI       = "external_spi"
	ModeBuiltinFilesystem = "builtin_filesystem"
)

const (
	FallbackPolicyFailFast             = "fail_fast"
	FallbackPolicyDegradeToBuiltin     = "degrade_to_builtin"
	FallbackPolicyDegradeWithoutMemory = "degrade_without_memory"
)

type ExternalConfig struct {
	Provider        string `json:"provider"`
	Profile         string `json:"profile"`
	ContractVersion string `json:"contract_version"`
}

type BuiltinConfig struct {
	RootDir    string                     `json:"root_dir"`
	Compaction FilesystemCompactionConfig `json:"compaction"`
}

type FallbackConfig struct {
	Policy string `json:"policy"`
}

type Config struct {
	Mode     string         `json:"mode"`
	External ExternalConfig `json:"external"`
	Builtin  BuiltinConfig  `json:"builtin"`
	Fallback FallbackConfig `json:"fallback"`
}

type ExternalEngineFactory func(cfg ExternalConfig) (Engine, error)

type Facade struct {
	mode            string
	provider        string
	profile         string
	contractVersion string
	fallbackPolicy  string

	active  Engine
	builtin Engine
}

func NewFacade(cfg Config, externalFactory ExternalEngineFactory) (*Facade, error) {
	normalized := normalizeConfig(cfg)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}
	out := &Facade{
		mode:            normalized.Mode,
		profile:         normalized.External.Profile,
		contractVersion: normalized.External.ContractVersion,
		fallbackPolicy:  normalized.Fallback.Policy,
	}
	switch normalized.Mode {
	case ModeBuiltinFilesystem:
		builtin, err := newBuiltinEngine(normalized.Builtin)
		if err != nil {
			return nil, err
		}
		out.provider = ModeBuiltinFilesystem
		out.active = builtin
		out.builtin = builtin
	case ModeExternalSPI:
		out.provider = normalized.External.Provider
		if _, err := ResolveProfile(normalized.External.Profile); err != nil {
			return nil, err
		}
		if externalFactory == nil {
			return nil, &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeProviderUnavailable,
				Layer:     LayerRuntime,
				Message:   "external memory engine factory is required in external_spi mode",
			}
		}
		external, err := externalFactory(normalized.External)
		if err != nil {
			return nil, normalizeError(OperationQuery, err)
		}
		out.active = external
		if normalized.Fallback.Policy == FallbackPolicyDegradeToBuiltin {
			builtin, err := newBuiltinEngine(normalized.Builtin)
			if err != nil {
				return nil, &Error{
					Operation: OperationQuery,
					Code:      ReasonCodeFallbackTargetMissing,
					Layer:     LayerRuntime,
					Message:   "fallback target builtin filesystem is unavailable",
					Cause:     err,
				}
			}
			out.builtin = builtin
		}
	default:
		return nil, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("unsupported memory mode %q", normalized.Mode),
		}
	}
	return out, nil
}

func (f *Facade) Query(req QueryRequest) (QueryResponse, error) {
	if f == nil || f.active == nil {
		return QueryResponse{}, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeProviderUnavailable,
			Layer:     LayerRuntime,
			Message:   "memory facade is not initialized",
		}
	}
	resp, err := f.active.Query(req)
	if err == nil {
		f.decorateQueryResponse(&resp, false, "", f.mode, f.provider)
		return resp, nil
	}
	return f.queryWithFallback(req, err)
}

func (f *Facade) Upsert(req UpsertRequest) (UpsertResponse, error) {
	if f == nil || f.active == nil {
		return UpsertResponse{}, &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeProviderUnavailable,
			Layer:     LayerRuntime,
			Message:   "memory facade is not initialized",
		}
	}
	resp, err := f.active.Upsert(req)
	if err == nil {
		f.decorateUpsertResponse(&resp, false, "", f.mode, f.provider)
		return resp, nil
	}
	return f.upsertWithFallback(req, err)
}

func (f *Facade) Delete(req DeleteRequest) (DeleteResponse, error) {
	if f == nil || f.active == nil {
		return DeleteResponse{}, &Error{
			Operation: OperationDelete,
			Code:      ReasonCodeProviderUnavailable,
			Layer:     LayerRuntime,
			Message:   "memory facade is not initialized",
		}
	}
	resp, err := f.active.Delete(req)
	if err == nil {
		f.decorateDeleteResponse(&resp, false, "", f.mode, f.provider)
		return resp, nil
	}
	return f.deleteWithFallback(req, err)
}

func (f *Facade) Close() error {
	if f == nil {
		return nil
	}
	var closeErr error
	if err := closeEngine(f.active); err != nil {
		closeErr = err
	}
	if f.builtin != nil && f.builtin != f.active {
		if err := closeEngine(f.builtin); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func (f *Facade) queryWithFallback(req QueryRequest, original error) (QueryResponse, error) {
	switch f.fallbackPolicy {
	case FallbackPolicyFailFast:
		return QueryResponse{}, normalizeError(OperationQuery, original)
	case FallbackPolicyDegradeWithoutMemory:
		resp := QueryResponse{
			OperationID: req.OperationID,
			Namespace:   strings.TrimSpace(req.Namespace),
			Records:     nil,
			Total:       0,
			ReasonCode:  ReasonCodeFallbackUsed,
		}
		f.decorateQueryResponse(&resp, true, ReasonCodeFallbackUsed, f.mode, f.provider)
		return resp, nil
	case FallbackPolicyDegradeToBuiltin:
		if f.builtin == nil {
			return QueryResponse{}, &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeFallbackTargetMissing,
				Layer:     LayerRuntime,
				Message:   "fallback target builtin filesystem is unavailable",
				Cause:     original,
			}
		}
		resp, err := f.builtin.Query(req)
		if err != nil {
			return QueryResponse{}, normalizeError(OperationQuery, err)
		}
		f.decorateQueryResponse(&resp, true, ReasonCodeFallbackUsed, ModeBuiltinFilesystem, ModeBuiltinFilesystem)
		return resp, nil
	default:
		return QueryResponse{}, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeFallbackPolicyConflict,
			Layer:     LayerRuntime,
			Message:   "memory fallback policy is unsupported",
		}
	}
}

func (f *Facade) upsertWithFallback(req UpsertRequest, original error) (UpsertResponse, error) {
	switch f.fallbackPolicy {
	case FallbackPolicyFailFast:
		return UpsertResponse{}, normalizeError(OperationUpsert, original)
	case FallbackPolicyDegradeWithoutMemory:
		resp := UpsertResponse{
			OperationID: req.OperationID,
			Namespace:   strings.TrimSpace(req.Namespace),
			Upserted:    0,
			ReasonCode:  ReasonCodeFallbackUsed,
		}
		f.decorateUpsertResponse(&resp, true, ReasonCodeFallbackUsed, f.mode, f.provider)
		return resp, nil
	case FallbackPolicyDegradeToBuiltin:
		if f.builtin == nil {
			return UpsertResponse{}, &Error{
				Operation: OperationUpsert,
				Code:      ReasonCodeFallbackTargetMissing,
				Layer:     LayerRuntime,
				Message:   "fallback target builtin filesystem is unavailable",
				Cause:     original,
			}
		}
		resp, err := f.builtin.Upsert(req)
		if err != nil {
			return UpsertResponse{}, normalizeError(OperationUpsert, err)
		}
		f.decorateUpsertResponse(&resp, true, ReasonCodeFallbackUsed, ModeBuiltinFilesystem, ModeBuiltinFilesystem)
		return resp, nil
	default:
		return UpsertResponse{}, &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeFallbackPolicyConflict,
			Layer:     LayerRuntime,
			Message:   "memory fallback policy is unsupported",
		}
	}
}

func (f *Facade) deleteWithFallback(req DeleteRequest, original error) (DeleteResponse, error) {
	switch f.fallbackPolicy {
	case FallbackPolicyFailFast:
		return DeleteResponse{}, normalizeError(OperationDelete, original)
	case FallbackPolicyDegradeWithoutMemory:
		resp := DeleteResponse{
			OperationID: req.OperationID,
			Namespace:   strings.TrimSpace(req.Namespace),
			Deleted:     0,
			ReasonCode:  ReasonCodeFallbackUsed,
		}
		f.decorateDeleteResponse(&resp, true, ReasonCodeFallbackUsed, f.mode, f.provider)
		return resp, nil
	case FallbackPolicyDegradeToBuiltin:
		if f.builtin == nil {
			return DeleteResponse{}, &Error{
				Operation: OperationDelete,
				Code:      ReasonCodeFallbackTargetMissing,
				Layer:     LayerRuntime,
				Message:   "fallback target builtin filesystem is unavailable",
				Cause:     original,
			}
		}
		resp, err := f.builtin.Delete(req)
		if err != nil {
			return DeleteResponse{}, normalizeError(OperationDelete, err)
		}
		f.decorateDeleteResponse(&resp, true, ReasonCodeFallbackUsed, ModeBuiltinFilesystem, ModeBuiltinFilesystem)
		return resp, nil
	default:
		return DeleteResponse{}, &Error{
			Operation: OperationDelete,
			Code:      ReasonCodeFallbackPolicyConflict,
			Layer:     LayerRuntime,
			Message:   "memory fallback policy is unsupported",
		}
	}
}

func (f *Facade) decorateQueryResponse(resp *QueryResponse, fallback bool, fallbackReason string, effectiveMode string, effectiveProvider string) {
	if resp == nil {
		return
	}
	if strings.TrimSpace(resp.ReasonCode) == "" {
		resp.ReasonCode = ReasonCodeOK
	}
	resp.Mode = strings.TrimSpace(effectiveMode)
	resp.Provider = strings.TrimSpace(effectiveProvider)
	resp.Profile = f.profile
	resp.ContractVersion = f.contractVersion
	resp.FallbackUsed = fallback
	resp.FallbackReasonCode = strings.TrimSpace(fallbackReason)
}

func (f *Facade) decorateUpsertResponse(resp *UpsertResponse, fallback bool, fallbackReason string, effectiveMode string, effectiveProvider string) {
	if resp == nil {
		return
	}
	if strings.TrimSpace(resp.ReasonCode) == "" {
		resp.ReasonCode = ReasonCodeOK
	}
	resp.Mode = strings.TrimSpace(effectiveMode)
	resp.Provider = strings.TrimSpace(effectiveProvider)
	resp.Profile = f.profile
	resp.ContractVersion = f.contractVersion
	resp.FallbackUsed = fallback
	resp.FallbackReasonCode = strings.TrimSpace(fallbackReason)
}

func (f *Facade) decorateDeleteResponse(resp *DeleteResponse, fallback bool, fallbackReason string, effectiveMode string, effectiveProvider string) {
	if resp == nil {
		return
	}
	if strings.TrimSpace(resp.ReasonCode) == "" {
		resp.ReasonCode = ReasonCodeOK
	}
	resp.Mode = strings.TrimSpace(effectiveMode)
	resp.Provider = strings.TrimSpace(effectiveProvider)
	resp.Profile = f.profile
	resp.ContractVersion = f.contractVersion
	resp.FallbackUsed = fallback
	resp.FallbackReasonCode = strings.TrimSpace(fallbackReason)
}

func normalizeConfig(cfg Config) Config {
	out := cfg
	out.Mode = strings.ToLower(strings.TrimSpace(out.Mode))
	if out.Mode == "" {
		out.Mode = ModeBuiltinFilesystem
	}
	out.External.Provider = strings.ToLower(strings.TrimSpace(out.External.Provider))
	out.External.Profile = strings.ToLower(strings.TrimSpace(out.External.Profile))
	if out.External.Profile == "" {
		out.External.Profile = ProfileGeneric
	}
	out.External.ContractVersion = strings.ToLower(strings.TrimSpace(out.External.ContractVersion))
	if out.External.ContractVersion == "" {
		out.External.ContractVersion = ContractVersionMemoryV1
	}
	out.Builtin.RootDir = strings.TrimSpace(out.Builtin.RootDir)
	out.Fallback.Policy = strings.ToLower(strings.TrimSpace(out.Fallback.Policy))
	if out.Fallback.Policy == "" {
		out.Fallback.Policy = FallbackPolicyFailFast
	}
	return out
}

func validateConfig(cfg Config) error {
	switch cfg.Mode {
	case ModeBuiltinFilesystem, ModeExternalSPI:
	default:
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.mode must be one of [%s,%s], got %q", ModeExternalSPI, ModeBuiltinFilesystem, cfg.Mode),
		}
	}
	switch cfg.Fallback.Policy {
	case FallbackPolicyFailFast, FallbackPolicyDegradeToBuiltin, FallbackPolicyDegradeWithoutMemory:
	default:
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeFallbackPolicyConflict,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.fallback.policy is unsupported: %q", cfg.Fallback.Policy),
		}
	}
	if cfg.External.ContractVersion != ContractVersionMemoryV1 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeContractVersionMismatch,
			Layer:     LayerSemantic,
			Message:   fmt.Sprintf("memory contract version %q is unsupported", cfg.External.ContractVersion),
		}
	}
	if cfg.Mode == ModeExternalSPI {
		if strings.TrimSpace(cfg.External.Provider) == "" {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeProviderUnavailable,
				Layer:     LayerRuntime,
				Message:   "runtime.memory.external.provider is required when mode=external_spi",
			}
		}
		if _, err := ResolveProfile(cfg.External.Profile); err != nil {
			return err
		}
	}
	if cfg.Mode == ModeBuiltinFilesystem || cfg.Fallback.Policy == FallbackPolicyDegradeToBuiltin {
		if strings.TrimSpace(cfg.Builtin.RootDir) == "" {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   "runtime.memory.builtin.root_dir is required",
			}
		}
	}
	return nil
}

func newBuiltinEngine(cfg BuiltinConfig) (Engine, error) {
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: cfg.RootDir,
		Compaction: FilesystemCompactionConfig{
			Enabled:     cfg.Compaction.Enabled,
			MinOps:      cfg.Compaction.MinOps,
			MaxWALBytes: cfg.Compaction.MaxWALBytes,
		},
	})
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func normalizeError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var memErr *Error
	if errors.As(err, &memErr) {
		if strings.TrimSpace(memErr.Operation) == "" {
			memErr.Operation = operation
		}
		return memErr
	}
	return &Error{
		Operation: operation,
		Code:      ReasonCodeStorageUnavailable,
		Layer:     LayerRuntime,
		Message:   "memory operation failed",
		Cause:     err,
	}
}

type engineCloser interface {
	Close() error
}

func closeEngine(engine Engine) error {
	if engine == nil {
		return nil
	}
	closer, ok := engine.(engineCloser)
	if !ok {
		return nil
	}
	return closer.Close()
}
