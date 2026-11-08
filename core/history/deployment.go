package history

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/sourcecoordination"
	"github.com/FelixSeptem/baymax/core/types"
)

// OwnerStoreOptions is the explicit deployment input for a source owner's
// retained history. No store is created implicitly when a path or coordinator
// is omitted.
type OwnerStoreOptions struct {
	Source               types.SourceRef
	JournalPath          string
	ProjectionLedgerPath string
	Coordinator          sourcecoordination.Coordinator
	WriterID             string
	LeaseTTL             time.Duration
	Retention            int
	ProjectionRetention  int
	DeploymentRequest    types.DeploymentRequest
}

// OwnerStore binds source history, optional projection recovery, and its
// deployment evidence. It is the public seam owners should inject instead of
// assembling independent stores at each call site.
type OwnerStore struct {
	source      types.SourceRef
	journal     *Journal
	ledger      *ProjectionLedger
	deployment  types.DeploymentScope
	request     types.DeploymentRequest
	coordinator sourcecoordination.Coordinator
}

// OpenOwnerStore opens an explicit source-owned deployment. A durable binding
// requires either a journal path or a coordinator; a memory-only journal is
// intentionally not promoted through this factory.
func OpenOwnerStore(options OwnerStoreOptions) (*OwnerStore, error) {
	if err := options.Source.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(options.JournalPath) == "" && options.Coordinator == nil {
		return nil, types.ErrHistoryUnsupported
	}
	if options.Coordinator != nil && strings.TrimSpace(options.WriterID) == "" {
		return nil, types.ErrOwnerUnavailable
	}

	var ledger *ProjectionLedger
	var err error
	if strings.TrimSpace(options.ProjectionLedgerPath) != "" {
		ledger, err = OpenProjectionLedger(filepath.Clean(options.ProjectionLedgerPath), options.Source, projectionOptions(options.ProjectionRetention)...)
		if err != nil {
			return nil, err
		}
	}

	journalOptions := make([]Option, 0, 3)
	if options.Retention > 0 {
		journalOptions = append(journalOptions, WithRetention(options.Retention))
	}
	if ledger != nil {
		journalOptions = append(journalOptions, WithProjectionLedger(ledger))
	}
	if options.Coordinator != nil {
		journalOptions = append(journalOptions, WithCoordinator(options.Coordinator, options.WriterID, options.LeaseTTL))
	}

	var journal *Journal
	if strings.TrimSpace(options.JournalPath) != "" {
		journal, err = OpenFile(filepath.Clean(options.JournalPath), options.Source, journalOptions...)
	} else {
		journal, err = New(options.Source, journalOptions...)
	}
	if err != nil {
		return nil, err
	}
	caps, err := journal.Capabilities(context.Background(), options.Source)
	if err != nil {
		return nil, err
	}
	if !caps.Durable {
		return nil, types.ErrHistoryUnsupported
	}
	if err := caps.Deployment.Check(options.DeploymentRequest); err != nil {
		return nil, err
	}
	return &OwnerStore{source: options.Source, journal: journal, ledger: ledger, deployment: caps.Deployment, request: options.DeploymentRequest, coordinator: options.Coordinator}, nil
}

func projectionOptions(retention int) []ProjectionOption {
	if retention > 0 {
		return []ProjectionOption{WithProjectionRetention(retention)}
	}
	return nil
}

// Source returns the immutable owner source identity.
func (s *OwnerStore) Source() types.SourceRef {
	if s == nil {
		return types.SourceRef{}
	}
	return s.source
}

// DeploymentScope returns detached capability evidence for this binding.
func (s *OwnerStore) DeploymentScope() types.DeploymentScope {
	if s == nil {
		return types.DefaultDeploymentScope("source-owner", "owner store is unavailable")
	}
	return s.deployment
}

// Capabilities revalidates the bound coordinator before returning history
// evidence. It never creates a replacement store or starts execution.
func (s *OwnerStore) Capabilities(ctx context.Context) (types.HistoryCapabilities, error) {
	if s == nil || s.journal == nil {
		return types.HistoryCapabilities{}, types.ErrSourceUnavailable
	}
	if s.coordinator != nil {
		caps, err := s.journal.Capabilities(ctx, s.source)
		if err != nil {
			return types.HistoryCapabilities{}, err
		}
		if err := caps.Deployment.Check(s.request); err != nil {
			return types.HistoryCapabilities{}, err
		}
		return caps, nil
	}
	return s.journal.Capabilities(ctx, s.source)
}

// Append publishes a source fact through the bound journal. If the fact
// carries a committed identity, identical retained evidence converges as a
// duplicate and conflicting evidence fails closed.
func (s *OwnerStore) Append(ctx context.Context, event types.NativeExecutionEvent) (types.NativeHistoryIdentity, error) {
	if s == nil || s.journal == nil {
		return types.NativeHistoryIdentity{}, types.ErrSourceUnavailable
	}
	if _, err := s.Capabilities(ctx); err != nil {
		return types.NativeHistoryIdentity{}, err
	}
	return s.journal.AppendIdempotent(ctx, event)
}

// ReadAfter performs bounded, read-only source replay through the binding.
func (s *OwnerStore) ReadAfter(ctx context.Context, cursor *types.OpaqueCursor, limit int) (types.NativeEventPage, error) {
	if s == nil || s.journal == nil {
		return types.NativeEventPage{}, types.ErrSourceUnavailable
	}
	if _, err := s.Capabilities(ctx); err != nil {
		return types.NativeEventPage{}, err
	}
	return s.journal.ReadAfter(ctx, s.source, cursor, limit)
}

// ReconcileProjection repairs only the explicitly bound projection ledger.
func (s *OwnerStore) ReconcileProjection(ctx context.Context, event types.NativeExecutionEvent) (types.ProjectionRecoveryResult, error) {
	if s == nil || s.journal == nil || s.ledger == nil {
		return types.ProjectionRecoveryResult{}, types.ErrHistoryUnsupported
	}
	if _, err := s.Capabilities(ctx); err != nil {
		return types.ProjectionRecoveryResult{}, err
	}
	return s.journal.ReconcileProjection(ctx, event)
}

// Close releases an explicitly bound coordinator when it supports lifecycle
// closure. It never changes source facts or projection records.
func (s *OwnerStore) Close() error {
	if s == nil || s.coordinator == nil {
		return nil
	}
	if closer, ok := s.coordinator.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
