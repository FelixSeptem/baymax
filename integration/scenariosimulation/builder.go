package scenariosimulation

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
)

// Builder maps a validated scenario onto the repository's existing test fakes.
// It is intentionally only an input adapter: runtime code owns execution,
// ordering, cancellation, completion, and terminal semantics.
type Builder struct{ scenario Scenario }

// Dependencies are the controlled test inputs derived from a canonical
// scenario. They are data/fakes only; none owns runtime ordering or terminal
// decisions.
type Dependencies struct {
	Approval     fakes.Approval
	Cancellation fakes.Cancellation
	StreamFault  *fakes.StreamFault
	Recovery     *fakes.Recovery
}

func NewBuilder(s Scenario) (*Builder, error) {
	normalized, _, err := NormalizeScenario(s)
	if err != nil {
		return nil, err
	}
	if s.ExecutionMode == "live" || s.ExecutionMode == "network" || s.ExecutionMode == "workspace" || s.ExecutionMode == "git" {
		return nil, fmt.Errorf("%s", ReasonOfflineScope)
	}
	return &Builder{scenario: normalized}, nil
}

func (b *Builder) Model(steps []fakes.ModelStep, stream []types.ModelEvent, streamErr error) *fakes.Model {
	model := fakes.NewModel(steps)
	model.SetStream(stream, streamErr)
	return model
}

func (b *Builder) Tool(name string, invoke func(args map[string]any) (types.ToolResult, error)) *fakes.Tool {
	if invoke == nil {
		return &fakes.Tool{NameValue: name}
	}
	return &fakes.Tool{NameValue: name, InvokeFn: func(_ context.Context, args map[string]any) (types.ToolResult, error) { return invoke(args) }}
}

// PlannedEvents returns a defensive copy for test setup; it does not schedule
// or execute the events.
func (b *Builder) PlannedEvents() []PlannedEvent {
	return append([]PlannedEvent(nil), b.scenario.Events...)
}

// Dependencies projects approval, cancellation, truncation and recovery
// actions into the existing integration fakes. Unknown event kinds remain
// valid additive plan entries and are not interpreted by the builder.
func (b *Builder) Dependencies() (Dependencies, error) {
	deps := Dependencies{Approval: fakes.Approval{Decision: fakes.ApprovalPending}}
	for _, event := range b.scenario.Events {
		switch event.Kind {
		case EventApproval:
			decision := event.Decision
			if decision == "" && event.Expected {
				decision = fakes.ApprovalGranted
			}
			if decision == "" {
				decision = fakes.ApprovalPending
			}
			if decision != fakes.ApprovalGranted && decision != fakes.ApprovalDenied && decision != fakes.ApprovalPending {
				return Dependencies{}, fmt.Errorf("%s", ReasonScenarioSchemaDrift)
			}
			deps.Approval = fakes.Approval{Decision: decision}
		case EventCancel:
			deps.Cancellation.Requested = true
		case EventStreamTruncation:
			if deps.StreamFault != nil {
				return Dependencies{}, fmt.Errorf("%s", ReasonScenarioSchemaDrift)
			}
			reason := event.Reason
			if reason == "" {
				reason = "planned_stream_truncation"
			}
			deps.StreamFault = &fakes.StreamFault{AfterSequence: event.Sequence, Reason: reason}
		case EventRecovery:
			if deps.Recovery != nil {
				return Dependencies{}, fmt.Errorf("%s", ReasonScenarioSchemaDrift)
			}
			if event.Reference.ID == "" {
				return Dependencies{}, fmt.Errorf("%s", ReasonScenarioSchemaDrift)
			}
			deps.Recovery = &fakes.Recovery{ReferenceID: event.Reference.ID, Replayed: event.Expected}
		}
	}
	return deps, nil
}
