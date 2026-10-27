package types

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxProjectionReferenceBytes = 1024

var ErrProjectionRecoveryRequired = errors.New("source projection recovery required")

// SourceCommitInput supplies a source fact for explicit owner persistence.
// Event payloads are intentionally omitted by the owner before retention.
type SourceCommitInput struct {
	Event            NativeExecutionEvent `json:"event"`
	TerminalEvidence string               `json:"terminal_evidence,omitempty"`
}

// SourceCommitRecord is a retained source fact. Its cursor is source-owned and
// cannot authorize execution recovery or an external business commit.
type SourceCommitRecord struct {
	Event            NativeExecutionEvent `json:"event"`
	Digest           string               `json:"digest"`
	Cursor           OpaqueCursor         `json:"cursor"`
	TerminalEvidence string               `json:"terminal_evidence,omitempty"`
}

// ProjectionRecoveryResult reports source-evidence reconciliation. Duplicate
// is explicit so callers can distinguish an idempotent replay from a newly
// persisted ledger record without treating either outcome as execution.
type ProjectionRecoveryResult struct {
	Record    SourceCommitRecord `json:"record"`
	Duplicate bool               `json:"duplicate"`
}

// ProjectionAcknowledgement records only that a consumer has durably accepted
// an opaque source fact. Consumers remain responsible for business-write
// idempotency in their own storage boundary.
type ProjectionAcknowledgement struct {
	Source     SourceRef    `json:"source"`
	EventID    string       `json:"event_id"`
	Digest     string       `json:"digest"`
	Reference  string       `json:"reference"`
	NextCursor OpaqueCursor `json:"next_cursor"`
}

func (a ProjectionAcknowledgement) Validate() error {
	if err := a.Source.Validate(); err != nil {
		return err
	}
	for _, value := range []string{a.EventID, a.Digest, a.Reference} {
		if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || len(value) > MaxProjectionReferenceBytes {
			return ErrHistoryInvalid
		}
	}
	return a.NextCursor.Validate()
}

type ProjectionAcknowledgementResult struct {
	Source     SourceRef    `json:"source"`
	EventID    string       `json:"event_id"`
	Digest     string       `json:"digest"`
	Reference  string       `json:"reference"`
	NextCursor OpaqueCursor `json:"next_cursor"`
	Duplicate  bool         `json:"duplicate"`
}

type SourceCommitPage struct {
	Records        []SourceCommitRecord `json:"records"`
	NextCursor     *OpaqueCursor        `json:"next_cursor,omitempty"`
	RetentionFloor *OpaqueCursor        `json:"retention_floor,omitempty"`
	Complete       bool                 `json:"complete"`
}

// SourceCommitRecoveryReader remains a read-only source evidence surface.
// Implementations MUST NOT execute, retry, resume, acknowledge or mutate an
// underlying execution owner while serving a read.
type SourceCommitRecoveryReader interface {
	ReadCommittedAfter(context.Context, *OpaqueCursor, int) (SourceCommitPage, error)
}
