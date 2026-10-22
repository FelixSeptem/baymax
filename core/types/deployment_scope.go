package types

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const DeploymentScopeVersion = "deployment_scope.v1"

type DeploymentProfile string

const (
	DeploymentProfileSameHostSingleWriterSequentialReopen DeploymentProfile = "same_host_single_writer_sequential_reopen"
	DeploymentProfileMultiWriter                          DeploymentProfile = "multi_writer"
	DeploymentProfileCrossMachine                         DeploymentProfile = "cross_machine"
	DeploymentProfileMultiInstance                        DeploymentProfile = "multi_instance"
)

type WriterScope string

const (
	WriterScopeSingleWriter WriterScope = "single_writer"
	WriterScopeMultiWriter  WriterScope = "multi_writer"
)

type FencingScope string

const (
	FencingScopeSourceOwner FencingScope = "source_owner"
	FencingScopeDistributed FencingScope = "distributed"
)

type TransactionScope string

const (
	TransactionScopeSequentialReopen   TransactionScope = "sequential_reopen"
	TransactionScopeSingleWriterAtomic TransactionScope = "single_writer_atomic"
	TransactionScopeMultiWriterCAS     TransactionScope = "multi_writer_cas"
	TransactionScopeDistributed        TransactionScope = "distributed"
)

var (
	ErrDeploymentUnsupported = errors.New("deployment scope unsupported")
	ErrOwnerUnavailable      = errors.New("source owner unavailable")
	ErrForeignWriter         = errors.New("foreign source writer")
	ErrStaleWriter           = errors.New("stale source writer")
	ErrDeploymentConflict    = errors.New("deployment scope conflict")
)

// DeploymentScope is detached source-owned evidence. It describes where a
// store is safe to use; it is not a coordinator and never grants authority.
type DeploymentScope struct {
	Version           string            `json:"version"`
	Profile           DeploymentProfile `json:"deployment_profile"`
	WriterScope       WriterScope       `json:"writer_scope"`
	FencingScope      FencingScope      `json:"fencing_scope"`
	TransactionScope  TransactionScope  `json:"transaction_scope"`
	OwnerNamespace    string            `json:"owner_namespace"`
	WriterFence       string            `json:"writer_fence"`
	EvidenceReference string            `json:"evidence_reference"`
	Supported         bool              `json:"supported"`
	UnsupportedReason string            `json:"unsupported_reason,omitempty"`
}

// DeploymentRequest is the host's requested deployment/concurrency boundary.
// An empty profile means the source's advertised profile, never a broader one.
type DeploymentRequest struct {
	Profile        DeploymentProfile `json:"deployment_profile,omitempty"`
	WriterMode     WriterScope       `json:"writer_scope,omitempty"`
	Fencing        FencingScope      `json:"fencing_scope,omitempty"`
	Transaction    TransactionScope  `json:"transaction_scope,omitempty"`
	OwnerNamespace string            `json:"owner_namespace,omitempty"`
	WriterFence    string            `json:"writer_fence,omitempty"`
}

func SameHostSingleWriterScope(owner, evidence string) DeploymentScope {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		owner = "source-owner"
	}
	return DeploymentScope{
		Version:           DeploymentScopeVersion,
		Profile:           DeploymentProfileSameHostSingleWriterSequentialReopen,
		WriterScope:       WriterScopeSingleWriter,
		FencingScope:      FencingScopeSourceOwner,
		TransactionScope:  TransactionScopeSequentialReopen,
		OwnerNamespace:    owner,
		WriterFence:       "fence:" + owner,
		EvidenceReference: strings.TrimSpace(evidence),
		Supported:         true,
	}
}

func (s DeploymentScope) Validate() error {
	if s.Version != DeploymentScopeVersion {
		return fmt.Errorf("deployment scope version: %w", ErrDeploymentUnsupported)
	}
	for name, value := range map[string]string{
		"owner_namespace":    s.OwnerNamespace,
		"writer_fence":       s.WriterFence,
		"evidence_reference": s.EvidenceReference,
	} {
		if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || len(value) > MaxHistoryReferenceBytes {
			return fmt.Errorf("deployment scope %s is invalid: %w", name, ErrDeploymentConflict)
		}
	}
	if s.Profile == "" || s.WriterScope == "" || s.FencingScope == "" || s.TransactionScope == "" {
		return fmt.Errorf("deployment scope dimensions are required: %w", ErrDeploymentConflict)
	}
	if !s.Supported {
		if strings.TrimSpace(s.UnsupportedReason) == "" || len(s.UnsupportedReason) > MaxHistoryReferenceBytes {
			return fmt.Errorf("unsupported deployment scope reason is required: %w", ErrDeploymentUnsupported)
		}
		return nil
	}
	if s.Profile != DeploymentProfileSameHostSingleWriterSequentialReopen || s.WriterScope != WriterScopeSingleWriter || s.FencingScope != FencingScopeSourceOwner || s.TransactionScope != TransactionScopeSequentialReopen {
		return fmt.Errorf("deployment scope has no source evidence: %w", ErrDeploymentUnsupported)
	}
	return nil
}

func (s DeploymentScope) ValidateWriter(owner, fence string) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(owner) != s.OwnerNamespace {
		return ErrForeignWriter
	}
	if strings.TrimSpace(fence) != s.WriterFence {
		return ErrStaleWriter
	}
	return nil
}

func (s DeploymentScope) Check(request DeploymentRequest) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if !s.Supported {
		return ErrDeploymentUnsupported
	}
	if request.Profile != "" && request.Profile != s.Profile {
		return ErrDeploymentUnsupported
	}
	if request.WriterMode != "" && request.WriterMode != s.WriterScope {
		return ErrDeploymentUnsupported
	}
	if request.Fencing != "" && request.Fencing != s.FencingScope {
		return ErrDeploymentUnsupported
	}
	if request.Transaction != "" && request.Transaction != s.TransactionScope {
		return ErrDeploymentUnsupported
	}
	if request.OwnerNamespace != "" || request.WriterFence != "" {
		return s.ValidateWriter(request.OwnerNamespace, request.WriterFence)
	}
	return nil
}

func (s DeploymentScope) Supports(request DeploymentRequest) bool { return s.Check(request) == nil }

// DefaultDeploymentScope is used only when a source has not yet attached
// evidence. It is explicitly unsupported, so absence can never imply a wider
// deployment guarantee.
func DefaultDeploymentScope(owner, reason string) DeploymentScope {
	s := SameHostSingleWriterScope(owner, "fixture:deployment-scope.v1")
	s.Supported = false
	s.UnsupportedReason = strings.TrimSpace(reason)
	if s.UnsupportedReason == "" {
		s.UnsupportedReason = "source deployment evidence is unavailable"
	}
	return s
}
