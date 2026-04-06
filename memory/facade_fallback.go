package memory

import "strings"

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
