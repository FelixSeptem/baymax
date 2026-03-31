package memory

import (
	"fmt"
	"strings"
	"time"
)

const (
	OperationQuery  = "query"
	OperationUpsert = "upsert"
	OperationDelete = "delete"
)

const (
	ReasonCodeOK                      = "memory.ok"
	ReasonCodeInvalidRequest          = "memory.invalid_request"
	ReasonCodeNotFound                = "memory.not_found"
	ReasonCodeStorageUnavailable      = "memory.storage_unavailable"
	ReasonCodeProviderUnavailable     = "memory.provider_unavailable"
	ReasonCodeProviderUnsupported     = "memory.provider_not_supported"
	ReasonCodeProfileUnknown          = "memory.profile_unknown"
	ReasonCodeContractVersionMismatch = "memory.contract_version_mismatch"
	ReasonCodeUnsupportedOperation    = "memory.unsupported_operation"
	ReasonCodeFallbackUsed            = "memory.fallback.used"
	ReasonCodeFallbackTargetMissing   = "memory.fallback.target_unavailable"
	ReasonCodeFallbackPolicyConflict  = "memory.fallback.policy_conflict"
)

const (
	LayerRuntime   = "runtime"
	LayerStorage   = "storage"
	LayerTransport = "transport"
	LayerSemantic  = "semantic"
)

type Error struct {
	Operation string         `json:"operation"`
	Code      string         `json:"code"`
	Layer     string         `json:"layer"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable,omitempty"`
	Raw       map[string]any `json:"raw,omitempty"`
	Cause     error          `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Cause != nil {
		msg = e.Cause.Error()
	}
	if msg == "" {
		msg = "memory operation failed"
	}
	if strings.TrimSpace(e.Code) == "" {
		return msg
	}
	return fmt.Sprintf("%s: %s", strings.TrimSpace(e.Code), msg)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type Record struct {
	ID        string            `json:"id"`
	Namespace string            `json:"namespace"`
	SessionID string            `json:"session_id,omitempty"`
	RunID     string            `json:"run_id,omitempty"`
	Content   string            `json:"content,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty"`
}

type QueryRequest struct {
	OperationID string            `json:"operation_id,omitempty"`
	Namespace   string            `json:"namespace"`
	SessionID   string            `json:"session_id,omitempty"`
	RunID       string            `json:"run_id,omitempty"`
	Query       string            `json:"query,omitempty"`
	IDs         []string          `json:"ids,omitempty"`
	MaxItems    int               `json:"max_items,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type QueryResponse struct {
	OperationID        string         `json:"operation_id,omitempty"`
	Namespace          string         `json:"namespace,omitempty"`
	Records            []Record       `json:"records,omitempty"`
	Total              int            `json:"total,omitempty"`
	ReasonCode         string         `json:"reason_code,omitempty"`
	LatencyMs          int64          `json:"latency_ms,omitempty"`
	Mode               string         `json:"mode,omitempty"`
	Provider           string         `json:"provider,omitempty"`
	Profile            string         `json:"profile,omitempty"`
	ContractVersion    string         `json:"contract_version,omitempty"`
	FallbackUsed       bool           `json:"fallback_used,omitempty"`
	FallbackReasonCode string         `json:"fallback_reason_code,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type UpsertRequest struct {
	OperationID string   `json:"operation_id,omitempty"`
	Namespace   string   `json:"namespace"`
	Records     []Record `json:"records"`
}

type UpsertResponse struct {
	OperationID        string `json:"operation_id,omitempty"`
	Namespace          string `json:"namespace,omitempty"`
	Upserted           int    `json:"upserted,omitempty"`
	ReasonCode         string `json:"reason_code,omitempty"`
	LatencyMs          int64  `json:"latency_ms,omitempty"`
	Mode               string `json:"mode,omitempty"`
	Provider           string `json:"provider,omitempty"`
	Profile            string `json:"profile,omitempty"`
	ContractVersion    string `json:"contract_version,omitempty"`
	FallbackUsed       bool   `json:"fallback_used,omitempty"`
	FallbackReasonCode string `json:"fallback_reason_code,omitempty"`
}

type DeleteRequest struct {
	OperationID string   `json:"operation_id,omitempty"`
	Namespace   string   `json:"namespace"`
	IDs         []string `json:"ids"`
}

type DeleteResponse struct {
	OperationID        string `json:"operation_id,omitempty"`
	Namespace          string `json:"namespace,omitempty"`
	Deleted            int    `json:"deleted,omitempty"`
	ReasonCode         string `json:"reason_code,omitempty"`
	LatencyMs          int64  `json:"latency_ms,omitempty"`
	Mode               string `json:"mode,omitempty"`
	Provider           string `json:"provider,omitempty"`
	Profile            string `json:"profile,omitempty"`
	ContractVersion    string `json:"contract_version,omitempty"`
	FallbackUsed       bool   `json:"fallback_used,omitempty"`
	FallbackReasonCode string `json:"fallback_reason_code,omitempty"`
}

type Engine interface {
	Query(req QueryRequest) (QueryResponse, error)
	Upsert(req UpsertRequest) (UpsertResponse, error)
	Delete(req DeleteRequest) (DeleteResponse, error)
}
