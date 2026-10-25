package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/planbridge"
	"github.com/FelixSeptem/baymax/tool/local"
)

type model struct{ calls int }

func (m *model) Generate(_ context.Context, request types.ModelRequest) (types.ModelResponse, error) {
	m.calls++
	if len(request.ToolResult) != 0 {
		return types.ModelResponse{FinalAnswer: "offline completion"}, nil
	}
	return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "offline-call", Name: "local.echo"}}}, nil
}
func (m *model) Stream(ctx context.Context, request types.ModelRequest, emit func(types.ModelEvent) error) error {
	response, err := m.Generate(ctx, request)
	if err != nil {
		return err
	}
	for _, call := range response.ToolCalls {
		if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &call}); err != nil {
			return err
		}
	}
	if response.FinalAnswer != "" {
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: response.FinalAnswer})
	}
	return nil
}

type echoTool struct{ calls int }

func (*echoTool) Name() string               { return "echo" }
func (*echoTool) Description() string        { return "offline echo" }
func (*echoTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *echoTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	t.calls++
	return types.ToolResult{Content: "offline tool result"}, nil
}

func binding(engine types.Runner) planbridge.Binding {
	return planbridge.Binding{SemanticDigest: "offline-semantics", Direct: &planbridge.DirectBinding{Runner: engine, Request: types.RunRequest{RunID: "offline-run", Input: "offline input", Capabilities: types.CapabilityRequirements{Required: []types.ModelCapability{types.ModelCapabilityStreaming}}}}, Admit: func(context.Context, planbridge.Identity) (planbridge.Admission, error) {
		return planbridge.Admission{AuthorityDigest: "offline-authority", ToolCapabilityDigest: "offline-tools", SemanticDigest: "offline-semantics"}, nil
	}}
}
func descriptor() planbridge.Descriptor {
	return planbridge.Descriptor{SchemaVersion: planbridge.SchemaVersion, PlanID: "offline-plan", PlanVersion: 1, Shape: planbridge.ShapeDirect, Delivery: planbridge.DeliveryInline, Trigger: planbridge.TriggerManual, AuthorityDigest: "offline-authority", ToolCapabilityDigest: "offline-tools", InputRef: "offline-input", InputDigest: "offline-input-digest", HandlerRef: "offline-handler", HandlerDigest: "offline-handler-digest"}
}

func main() {
	base := &model{}
	wrapped, err := modelcapability.Wrap(base, modelcapability.Config{Provider: "offline", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		panic(err)
	}
	tool := &echoTool{}
	registry := local.NewRegistry()
	if _, err = registry.Register(tool); err != nil {
		panic(err)
	}
	engine := runner.New(wrapped, runner.WithLocalRegistry(registry))
	d, b := descriptor(), binding(engine)
	d, err = planbridge.Describe(d, b, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	p, err := planbridge.Compile(d, b, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	if p.Identity().Digest == "" || base.calls != 0 || tool.calls != 0 {
		panic("compile was not side-effect free")
	}
	fmt.Println("plan-compiled")
	run, err := p.Run(context.Background(), planbridge.Execution{})
	if err != nil || run.Direct == nil || run.Direct.TerminalOutcome.State != types.RunStateCompleted {
		panic(fmt.Sprintf("run: %v", err))
	}
	stream, err := p.Stream(context.Background(), planbridge.Execution{})
	if err != nil || stream.Direct == nil || stream.Direct.TerminalOutcome.State != types.RunStateCompleted {
		panic(fmt.Sprintf("stream: %v", err))
	}
	if run.Direct.RunID != stream.Direct.RunID || base.calls == 0 || tool.calls == 0 {
		panic("source owner parity")
	}
	fmt.Println("source-owner-preserved")
	fmt.Println("run-stream-parity")
	mailboxOwner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		panic(err)
	}
	mailboxDescriptor := descriptor()
	mailboxDescriptor.Delivery = planbridge.DeliveryMailbox
	mailboxBinding := planbridge.Binding{
		SemanticDigest: "offline-semantics",
		Admit: func(context.Context, planbridge.Identity) (planbridge.Admission, error) {
			return planbridge.Admission{AuthorityDigest: "offline-authority", ToolCapabilityDigest: "offline-tools", SemanticDigest: "offline-semantics"}, nil
		},
		Mailbox: &planbridge.MailboxBinding{Owner: mailboxOwner, Manifest: mustManifest(), Binding: mailbox.HostPlanBinding{
			ContractVersion: mailbox.HostPlanBindingContractVersion, MessageID: "offline-message", IdempotencyKey: "offline-logical",
			RunID: "offline-run", ParentRunID: "offline-parent", CausationID: "offline-cause", AuthorityDigest: "offline-authority", WriteScope: "offline-scope",
			PlanDigest: "", EffectNamespace: "offline-effect", Occurrence: "offline-occurrence", HostCorrelation: "offline-host", ResourceLimit: 1,
		}},
	}
	mailboxDescriptor, err = planbridge.Describe(mailboxDescriptor, mailboxBinding, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	mailboxPlan, err := planbridge.Compile(mailboxDescriptor, mailboxBinding, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	mailboxRun, err := mailboxPlan.Run(context.Background(), planbridge.Execution{})
	if err != nil || mailboxRun.Mailbox == nil || mailboxRun.Mailbox.Duplicate {
		panic(fmt.Sprintf("mailbox run: %v", err))
	}
	mailboxStream, err := mailboxPlan.Stream(context.Background(), planbridge.Execution{})
	if err != nil || mailboxStream.Mailbox == nil || !mailboxStream.Mailbox.Duplicate {
		panic(fmt.Sprintf("mailbox stream: %v", err))
	}
	fmt.Println("mailbox-logical-admission")
	fmt.Println("mailbox-lifecycle-source-owned")
	continuationDir, err := os.MkdirTemp("", "baymax-host-plan-continuation-")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(continuationDir) }()
	continuationStore, err := mailbox.NewFileStore(filepath.Join(continuationDir, "mailbox.json"), mailbox.Policy{})
	if err != nil {
		panic(err)
	}
	continuationOwner, err := mailbox.New(continuationStore)
	if err != nil {
		panic(err)
	}
	continuationDescriptor := descriptor()
	continuationDescriptor.Delivery = planbridge.DeliveryMailbox
	continuationBinding := mailbox.HostPlanBinding{
		ContractVersion: mailbox.HostPlanBindingContractVersion, MessageID: "offline-continuation-message", IdempotencyKey: "offline-continuation-logical",
		RunID: "offline-continuation-run", ParentRunID: "offline-continuation-parent", CausationID: "offline-continuation-cause", AuthorityDigest: "offline-authority",
		WriteScope: "offline-scope", EffectNamespace: "offline-effect", Occurrence: "offline-occurrence", HostCorrelation: "offline-continuation-host", ResourceLimit: 1,
	}
	continuationPlanBinding := planbridge.Binding{
		SemanticDigest: "offline-semantics",
		Admit: func(context.Context, planbridge.Identity) (planbridge.Admission, error) {
			return planbridge.Admission{AuthorityDigest: "offline-authority", ToolCapabilityDigest: "offline-tools", SemanticDigest: "offline-semantics"}, nil
		},
		Mailbox: &planbridge.MailboxBinding{Owner: continuationOwner, Manifest: mustManifest(), Binding: continuationBinding},
	}
	continuationDescriptor, err = planbridge.Describe(continuationDescriptor, continuationPlanBinding, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	initialContinuationPlan, err := planbridge.Compile(continuationDescriptor, continuationPlanBinding, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	continuationBinding.PlanDigest = initialContinuationPlan.Identity().Digest
	if _, err := continuationOwner.AdmitHostPlan(context.Background(), continuationBinding); err != nil {
		panic(err)
	}
	leased, ok, err := continuationOwner.ConsumeWithLease(context.Background(), "offline-worker", time.Hour, false)
	if err != nil || !ok {
		panic(fmt.Sprintf("continuation lease: %v", err))
	}
	now := time.Now().UTC()
	checkpoint, err := mailbox.SealMailboxContinuation(mailbox.MailboxContinuation{
		Ref: mailbox.MailboxContinuationRef{OwnerNamespace: "mailbox"}, Binding: continuationBinding,
		MessageID: continuationBinding.MessageID, DeliveryID: leased.DeliveryID, LeaseID: leased.LeaseID, AttemptCausation: leased.AttemptCausation, DeliveryAttempt: leased.DeliveryAttempt, LifecycleState: leased.State,
		ParentRunID: continuationBinding.ParentRunID, RunID: continuationBinding.RunID, HostCorrelation: continuationBinding.HostCorrelation, AuthorityDigest: continuationBinding.AuthorityDigest, PlanDigest: continuationBinding.PlanDigest, WriteScope: continuationBinding.WriteScope, EffectNamespace: continuationBinding.EffectNamespace, Occurrence: continuationBinding.Occurrence,
		PauseReason: mailbox.MailboxPauseInputRequired, PendingActionReference: "offline-input", Generation: 1, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		panic(err)
	}
	if err := continuationOwner.SaveHostPlanContinuation(context.Background(), checkpoint); err != nil {
		panic(err)
	}
	continuationManifest, err := planbridge.CapabilityManifestWithMailboxContinuation(context.Background(), continuationOwner)
	if err != nil {
		panic(err)
	}
	continuationReader, err := planbridge.NewCapabilityManifestReader(continuationManifest)
	if err != nil {
		panic(err)
	}
	resume := mailbox.MailboxResumeRequest{Ref: checkpoint.Ref, Binding: continuationBinding, IdempotencyKey: "offline-continuation-resume", Generation: 1, PauseReason: mailbox.MailboxPauseInputRequired, Deployment: types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}}
	continuationPlanBinding.Mailbox = &planbridge.MailboxBinding{Owner: continuationOwner, Manifest: continuationReader, Binding: continuationBinding, Resume: &resume}
	continuationDescriptor, err = planbridge.Describe(continuationDescriptor, continuationPlanBinding, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	continuationPlan, err := planbridge.Compile(continuationDescriptor, continuationPlanBinding, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	continuationRun, err := continuationPlan.Run(context.Background(), planbridge.Execution{})
	if err != nil || continuationRun.Mailbox == nil || continuationRun.Mailbox.MessageID != continuationBinding.MessageID || continuationRun.Mailbox.LifecycleResult != "leased" {
		panic(fmt.Sprintf("continuation run: %v", err))
	}
	continuationStream, err := continuationPlan.Stream(context.Background(), planbridge.Execution{})
	if err != nil || continuationStream.Mailbox == nil || *continuationStream.Mailbox != *continuationRun.Mailbox {
		panic(fmt.Sprintf("continuation stream: %v", err))
	}
	continuationPage, err := continuationOwner.Query(context.Background(), mailbox.QueryRequest{MessageID: continuationBinding.MessageID})
	if err != nil || len(continuationPage.Items) != 1 || continuationPage.Items[0].State != mailbox.StateInFlight || continuationPage.Items[0].DeliveryAttempt != 1 {
		panic(fmt.Sprintf("continuation lifecycle: %v", err))
	}
	fmt.Println("team-mailbox-continuation-owner-configured")
	bad := d
	bad.PlanVersion++
	bad.SourceFacts["run_id"] = "different"
	if _, err = planbridge.Compile(bad, b, planbridge.Limits{}); err == nil {
		panic("invalid binding accepted")
	}
	fmt.Println("invalid-plan-rejected")
	mail := d
	mail.Delivery = planbridge.DeliveryMailbox
	if _, err = planbridge.Compile(mail, b, planbridge.Limits{}); err == nil {
		panic("mailbox accepted")
	}
	fmt.Println("unsupported-capability-rejected")
	if base.calls != 4 || tool.calls != 2 {
		panic("rejection activated source")
	}
	fmt.Println("no-side-effects-before-rejection")
	// Native completion is deliberately not a business write; the host owns its commit gate.
	fmt.Println("host-commit-required")
}

func mustManifest() *planbridge.CapabilityManifestReader {
	manifest, err := planbridge.NewCapabilityManifestReader(func() planbridge.CapabilityManifest {
		m := planbridge.DefaultCapabilityManifest()
		for i := range m.Forms {
			if m.Forms[i].Form == planbridge.ExecutionFormMailboxAsync {
				m.Forms[i].MailboxContinuation = planbridge.CapabilitySupported
				delete(m.Forms[i].UnsupportedReasons, "mailbox_continuation")
			}
		}
		return m
	}())
	if err != nil {
		panic(err)
	}
	return manifest
}
