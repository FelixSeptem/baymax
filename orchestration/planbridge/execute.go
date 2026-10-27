package planbridge

import (
	"context"
	"strconv"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/eventadapter"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func (p *Compiled) Run(ctx context.Context, execution Execution) (Result, error) {
	return p.execute(ctx, execution, false)
}
func (p *Compiled) Stream(ctx context.Context, execution Execution) (Result, error) {
	return p.execute(ctx, execution, true)
}

type eventHandlerFunc func(context.Context, types.Event)

func (f eventHandlerFunc) OnEvent(c context.Context, e types.Event) { f(c, e) }
func (p *Compiled) execute(ctx context.Context, x Execution, stream bool) (out Result, err error) {
	if p == nil {
		return Result{}, fail(InvalidPlan, "compiled", nil)
	}
	out = Result{Identity: p.identity, Owner: p.descriptor.Shape}
	current, err := p.binding.Admit(ctx, p.identity)
	if err != nil {
		return out, err
	}
	if current.AuthorityDigest != p.descriptor.AuthorityDigest || current.ToolCapabilityDigest != p.descriptor.ToolCapabilityDigest || current.SemanticDigest != p.binding.SemanticDigest {
		return out, fail(BindingMismatch, "activation.recompile_required", nil)
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if err = validateBoundContinuationRequirements(ctx, p.descriptor, p.binding); err != nil {
		return out, err
	}
	if p.descriptor.Delivery == DeliveryMailbox {
		if p.binding.Mailbox == nil || p.binding.Mailbox.Owner == nil || p.binding.Mailbox.Manifest == nil {
			return out, fail(SourceUnavailable, "activation.mailbox.owner", nil)
		}
		if err = p.binding.Mailbox.Manifest.ValidateCompatibility(CapabilityManifestContractVersion, CapabilityManifestAPIVersion); err != nil {
			return out, err
		}
		capability, capabilityErr := p.binding.Mailbox.Manifest.Capability(ExecutionFormMailboxAsync)
		if capabilityErr != nil {
			return out, capabilityErr
		}
		if capability.Admission != CapabilitySupported || capability.MailboxContinuation != CapabilitySupported {
			return out, fail(UnsupportedCapability, "activation.capability_manifest.mailbox_async", nil)
		}
		if p.binding.Mailbox.Resume != nil {
			owner, ok := p.binding.Mailbox.Owner.(MailboxContinuationOwner)
			if !ok {
				return out, fail(UnsupportedCapability, "activation.mailbox.continuation", nil)
			}
			if p.binding.Mailbox.Resume.Ref.Digest == "" || p.binding.Mailbox.Resume.Ref.MessageID != p.binding.Mailbox.Binding.MessageID {
				return out, fail(BindingMismatch, "activation.mailbox.continuation", nil)
			}
			mailboxResult, e := owner.ResumeHostPlanContinuation(ctx, *p.binding.Mailbox.Resume)
			if e != nil {
				return out, e
			}
			out.Mailbox = &mailboxResult
			return out, nil
		}
		binding := p.binding.Mailbox.Binding
		if binding.PlanDigest != p.identity.Digest {
			return out, fail(BindingMismatch, "activation.mailbox.plan_digest", nil)
		}
		mailboxResult, e := p.binding.Mailbox.Owner.AdmitHostPlan(ctx, binding)
		if e != nil {
			return out, e
		}
		out.Mailbox = &mailboxResult
		return out, nil
	}
	source := types.ProtocolSourceRunner
	runID := ""
	switch p.descriptor.Shape {
	case ShapeDirect:
		runID = p.binding.Direct.Request.RunID
	case ShapeTeam:
		source = types.ProtocolSourceTeams
		runID = p.binding.Team.Plan.RunID
	case ShapeWorkflow:
		source = types.ProtocolSourceWorkflow
		runID = p.binding.Workflow.Request.RunID
	case ShapeSubagent:
		source = types.ProtocolSourceScheduler
		runID = p.binding.Child.Request.Task.RunID
		if !p.binding.Child.TerminalOnly {
			break
		}
		cfg := childConfig(*p.binding.Child)
		if childConfigDigest(cfg) != p.childConfigDigest {
			return out, fail(BindingMismatch, "activation.recompile_required", nil)
		}
		if current.ParentDepth < 0 || current.ParentActiveChildren < 0 {
			return out, fail(InvalidPlan, "activation.parent", nil)
		}
		if current.ParentRemainingBudget <= 0 {
			return out, &scheduler.BudgetError{Code: scheduler.BudgetRejectParentBudgetExhausted, Message: "live parent budget exhausted"}
		}
		if err = guardrails(cfg).ValidateSpawn(scheduler.SpawnRequest{ParentDepth: current.ParentDepth, ParentActiveChildren: current.ParentActiveChildren}); err != nil {
			return out, err
		}
	}
	var observer *eventadapter.Observer
	if x.Sink != nil {
		refs := x.HostReferences
		refs.PlanVersion = strconv.Itoa(p.identity.PlanVersion)
		refs.PlanDigest = p.identity.Digest
		observer, err = eventadapter.New(eventadapter.Options{Projection: types.NativeExecutionOptions{Source: source, ExpectedRunID: runID, HostReferences: refs}, Sink: x.Sink, Context: x.ObserverContext, DeliveryTimeout: x.DeliveryTimeout})
		if err != nil {
			return out, err
		}
		defer func() { status := observer.Status(); out.Observation = &status; observer.Close() }()
	}
	handler := eventHandlerFunc(func(c context.Context, e types.Event) {
		if x.Handler != nil {
			x.Handler.OnEvent(c, e)
		}
		if observer != nil {
			observer.OnEvent(c, e)
		}
	})
	switch p.descriptor.Shape {
	case ShapeDirect:
		b := p.binding.Direct
		req, e := snapshot(b.Request, p.limits.BindingBytes, p.limits.NestingDepth)
		if e != nil {
			return out, e
		}
		var r types.RunResult
		if stream {
			r, err = b.Runner.Stream(ctx, req, handler)
		} else {
			r, err = b.Runner.Run(ctx, req, handler)
		}
		out.Direct = &r
	case ShapeTeam:
		plan, e := snapshot(p.binding.Team.Plan, p.limits.BindingBytes, p.limits.NestingDepth)
		if e != nil {
			return out, e
		}
		owner := teams.New(teams.WithTimelineEmitter(handler))
		if p.binding.Team.ContinuationOwner != nil {
			owner = p.binding.Team.ContinuationOwner
		}
		var r teams.Result
		switch {
		case p.binding.Team.Resume != nil:
			r, err = owner.ResumeContinuation(ctx, plan, *p.binding.Team.Resume, x.TeamEvent)
		case stream:
			r, err = owner.Stream(ctx, plan, x.TeamEvent)
		default:
			r, err = owner.Run(ctx, plan)
		}
		out.Team = &r
	case ShapeWorkflow:
		b := p.binding.Workflow
		req, e := snapshot(b.Request, p.limits.BindingBytes, p.limits.NestingDepth)
		if e != nil {
			return out, e
		}
		opts := []workflow.Option{workflow.WithStepAdapter(b.Adapter), workflow.WithTimelineEmitter(handler)}
		if b.Checkpoints != nil {
			opts = append(opts, workflow.WithCheckpointStore(b.Checkpoints))
		}
		if b.Resume != nil {
			opts = append(opts, workflow.WithWorkflowContinuationStore(b.Continuations, b.Resume.Ref.OwnerNamespace))
		}
		owner := workflow.New(opts...)
		var r workflow.RunResult
		switch {
		case b.Resume != nil:
			var onEvent func(workflow.StreamEvent) error
			if stream {
				onEvent = x.WorkflowEvent
			}
			r, err = owner.ResumeContinuation(ctx, *b.Resume, onEvent)
		case stream:
			r, err = owner.Stream(ctx, req, x.WorkflowEvent)
		default:
			r, err = owner.Run(ctx, req)
		}
		out.Workflow = &r
	case ShapeSubagent:
		b := p.binding.Child
		if !b.TerminalOnly {
			if b.Resume == nil || b.ContinuationOwner == nil {
				return out, fail(UnsupportedCapability, "activation.child.continuation", nil)
			}
			result, resumeErr := b.ContinuationOwner.ResumeChildContinuation(ctx, *b.Resume)
			out.Child = &result
			return out, resumeErr
		}
		cfg := childConfig(*b)
		s, e := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithGuardrails(guardrails(cfg)), scheduler.WithLeaseTimeout(cfg.Scheduler.LeaseTimeout), scheduler.WithTimelineEmitter(handler))
		if e != nil {
			return out, e
		}
		// Composer only uses its runner for parent Run/Stream. Injecting a sentinel
		// prevents model construction without adding another execution owner.
		opts := []composer.Option{composer.WithScheduler(s), composer.WithRunner(unusedRunner{})}
		if b.Runtime != nil {
			opts = append(opts, composer.WithRuntimeManager(b.Runtime))
		}
		owner, e := composer.New(nil, opts...)
		if e != nil {
			return out, e
		}
		req := b.Request
		req.Task, e = snapshot(req.Task, p.limits.BindingBytes, p.limits.NestingDepth)
		if e != nil {
			return out, e
		}
		req.ParentDepth = current.ParentDepth
		req.ParentActiveChildren = current.ParentActiveChildren
		req.ParentRemainingBudget = current.ParentRemainingBudget
		var r composer.ChildDispatchResult
		r, err = owner.DispatchChild(ctx, req)
		out.Child = &r
	}
	return out, err
}

type unusedRunner struct{}

func (unusedRunner) Run(context.Context, types.RunRequest, types.EventHandler) (types.RunResult, error) {
	return types.RunResult{}, fail(UnsupportedCapability, "child.parent_runner", nil)
}
func (unusedRunner) Stream(context.Context, types.RunRequest, types.EventHandler) (types.RunResult, error) {
	return types.RunResult{}, fail(UnsupportedCapability, "child.parent_runner", nil)
}
