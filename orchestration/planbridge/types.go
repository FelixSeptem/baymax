// Package planbridge compiles bounded host descriptions and delegates to native
// owners. A compiled digest is consistency evidence, never authorization.
package planbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/eventadapter"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const SchemaVersion = "host_execution_plan.v1"

type Shape string

const (
	ShapeDirect   Shape = "direct"
	ShapeSubagent Shape = "subagent"
	ShapeTeam     Shape = "team"
	ShapeWorkflow Shape = "workflow"
)

type Delivery string

const (
	DeliveryInline  Delivery = "inline"
	DeliveryMailbox Delivery = "mailbox"
)

type Trigger string

const (
	TriggerManual    Trigger = "manual"
	TriggerScheduled Trigger = "scheduled"
	TriggerCallback  Trigger = "callback"
	TriggerResumed   Trigger = "resumed"
)

type Descriptor struct {
	SchemaVersion        string                   `json:"schema_version"`
	PlanID               string                   `json:"plan_id"`
	PlanVersion          int                      `json:"plan_version"`
	Shape                Shape                    `json:"shape"`
	Delivery             Delivery                 `json:"delivery"`
	Trigger              Trigger                  `json:"trigger"`
	AuthorityDigest      string                   `json:"authority_digest"`
	ToolCapabilityDigest string                   `json:"tool_capability_digest"`
	InputRef             string                   `json:"input_ref"`
	InputDigest          string                   `json:"input_digest"`
	HandlerRef           string                   `json:"handler_ref"`
	HandlerDigest        string                   `json:"handler_digest"`
	CheckpointRef        string                   `json:"checkpoint_ref,omitempty"`
	Deployment           *types.DeploymentRequest `json:"deployment,omitempty"`
	RequiredCapabilities []string                 `json:"required_capabilities,omitempty"`
	SourceFacts          map[string]any           `json:"source_facts"`
}
type Limits struct{ DescriptorBytes, StructuralItems, Dependencies, ReferenceBytes, NestingDepth, BindingBytes int }
type Code string

const (
	UnsupportedVersion    Code = "unsupported_version"
	InvalidPlan           Code = "invalid_plan"
	LimitExceeded         Code = "limit_exceeded"
	BindingMismatch       Code = "binding_mismatch"
	UnsupportedCapability Code = "unsupported_capability"
	SourceUnavailable     Code = "source_unavailable"
)

type Error struct {
	Code  Code
	Field string
	// Limitation identifies a stable unsupported boundary independently of
	// the human-readable error. Empty retains the original v1 error format.
	Limitation string
	cause      error
}

func (e *Error) Error() string {
	if e.Limitation != "" {
		return fmt.Sprintf("host execution plan: %s at %s (%s)", e.Code, e.Field, e.Limitation)
	}
	return fmt.Sprintf("host execution plan: %s at %s", e.Code, e.Field)
}
func (e *Error) Unwrap() error { return e.cause }
func fail(code Code, field string, cause error) error {
	return &Error{Code: code, Field: field, cause: cause}
}

type Identity struct {
	PlanID      string `json:"plan_id"`
	PlanVersion int    `json:"plan_version"`
	Digest      string `json:"digest"`
}

// Admission is supplied fresh by the host immediately before delegation. The
// host must check current business authority/revision/lease itself, not merely
// copy the descriptor's declarations. Native admission still runs afterward.
type Admission struct {
	AuthorityDigest, ToolCapabilityDigest, SemanticDigest string
	ParentDepth, ParentActiveChildren                     int
	ParentRemainingBudget                                 time.Duration
}
type Binding struct {
	SemanticDigest string
	Admit          func(context.Context, Identity) (Admission, error)
	Direct         *DirectBinding
	Team           *TeamBinding
	Workflow       *WorkflowBinding
	Child          *ChildBinding
	Mailbox        *MailboxBinding
}
type DirectBinding struct {
	Runner  types.Runner
	Request types.RunRequest
}
type TeamBinding struct {
	Plan         teams.Plan
	TerminalOnly bool
}
type WorkflowBinding struct {
	Request            workflow.RunRequest
	Adapter            workflow.DispatchAdapter
	Checkpoints        workflow.CheckpointStore
	DefaultStepTimeout time.Duration
	TerminalOnly       bool
}

// ChildBinding deliberately cannot accept an external scheduler/store/composer.
// Every activation owns a fresh isolated scheduler for exactly one dispatch.
type ChildBinding struct {
	Request           composer.ChildDispatchRequest
	Runtime           *runtimeconfig.Manager
	TerminalOnly      bool
	Resume            *composer.ChildResumeRequest
	ContinuationOwner composer.ChildContinuationOwner
	Manifest          *CapabilityManifestReader
}

type MailboxBinding struct {
	Owner    MailboxOwner
	Binding  mailbox.HostPlanBinding
	Manifest *CapabilityManifestReader
}

type MailboxOwner interface {
	AdmitHostPlan(context.Context, mailbox.HostPlanBinding) (mailbox.HostPlanOutcome, error)
}
type Execution struct {
	Handler         types.EventHandler
	Sink            eventadapter.Sink
	ObserverContext context.Context
	DeliveryTimeout time.Duration
	HostReferences  types.NativeExecutionHostReferences
	TeamEvent       func(teams.StreamEvent) error
	WorkflowEvent   func(workflow.StreamEvent) error
}
type Result struct {
	Identity    Identity
	Owner       Shape
	Direct      *types.RunResult
	Team        *teams.Result
	Workflow    *workflow.RunResult
	Child       *composer.ChildDispatchResult
	Mailbox     *mailbox.HostPlanOutcome
	Observation *eventadapter.Status
}
type Compiled struct {
	identity          Identity
	descriptor        Descriptor
	binding           Binding
	canonical         []byte
	limits            Limits
	childConfigDigest string
}

func (p *Compiled) Identity() Identity { return p.identity }
func (p *Compiled) Canonical() []byte  { return append([]byte(nil), p.canonical...) }

// ContinuationError keeps the source's pause/checkpoint result available to the
// host while preventing terminal-only task adapters from reporting success.
type ContinuationError struct{ Result types.RunResult }

func (e *ContinuationError) Error() string {
	return "host execution plan: unsupported_capability at continuation"
}
func (e *ContinuationError) Unwrap() error { return fail(UnsupportedCapability, "continuation", nil) }
func TerminalOnly(result types.RunResult, err error) (types.RunResult, error) {
	if result.TerminalOutcome != nil && result.TerminalOutcome.State == types.RunStateInputRequired {
		return result, &ContinuationError{Result: result}
	}
	return result, err
}
