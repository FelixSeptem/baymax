package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeReadinessAdmissionReactBlockedDenyRunStreamEquivalentAndNoSideEffects(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a56-react-blocked.yaml")
	writeRuntimeReadinessAdmissionConfig(t, cfgPath, true, runtimeconfig.ReadinessAdmissionDegradedPolicyAllowAndRecord)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A56_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	mgr.SetReactReadinessDependencySnapshot(runtimeconfig.ReactReadinessDependencySnapshot{
		ProviderChecked:              true,
		ProviderName:                 "openai",
		ProviderToolCallingSupported: false,
		ProviderReason:               "tool calling unsupported",
	})

	before, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats before deny failed: %v", err)
	}
	mailboxBefore := len(mgr.RecentMailbox(10))

	runRes, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID: "run-a56-react-blocked-run",
		Input: "blocked-run",
	}, nil)
	if runErr == nil {
		t.Fatal("run should be denied by react readiness admission")
	}
	assertAdmissionContractDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeBlocked)

	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a56-react-blocked-stream",
		Input: "blocked-stream",
	}, nil)
	if streamErr == nil {
		t.Fatal("stream should be denied by react readiness admission")
	}
	assertAdmissionContractDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeBlocked)

	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after deny failed: %v", err)
	}
	if before.QueueTotal != after.QueueTotal || before.ClaimTotal != after.ClaimTotal || before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("deny path should be side-effect free, before=%#v after=%#v", before, after)
	}
	if len(mgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(mgr.RecentMailbox(10)))
	}

	assertAdmissionRunRecord(t, mgr, "run-a56-react-blocked-run", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 1 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionMode != runtimeconfig.ReadinessAdmissionModeFailFast ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeReactProviderToolCallingUnsupported ||
			rec.RuntimePrimaryDomain != runtimeconfig.ReadinessDomainRuntime ||
			rec.RuntimePrimaryCode != runtimeconfig.ReadinessCodeReactProviderToolCallingUnsupported ||
			rec.RuntimePrimarySource != runtimeconfig.RuntimePrimarySourceReadiness ||
			rec.RuntimePrimaryConflictTotal != 0 ||
			rec.RuntimeSecondaryReasonCount != 0 ||
			len(rec.RuntimeSecondaryReasonCodes) != 0 ||
			rec.RuntimeArbitrationRuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionExplainabilityV1 ||
			rec.RuntimeRemediationHintCode != "react.select_tool_calling_provider" ||
			rec.RuntimeRemediationHintDomain != runtimeconfig.ReadinessDomainRuntime {
			t.Fatalf("blocked react run record mismatch: %#v", rec)
		}
	})
	assertAdmissionRunRecord(t, mgr, "run-a56-react-blocked-stream", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 1 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionMode != runtimeconfig.ReadinessAdmissionModeFailFast ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeReactProviderToolCallingUnsupported ||
			rec.RuntimePrimaryDomain != runtimeconfig.ReadinessDomainRuntime ||
			rec.RuntimePrimaryCode != runtimeconfig.ReadinessCodeReactProviderToolCallingUnsupported ||
			rec.RuntimePrimarySource != runtimeconfig.RuntimePrimarySourceReadiness ||
			rec.RuntimePrimaryConflictTotal != 0 ||
			rec.RuntimeSecondaryReasonCount != 0 ||
			len(rec.RuntimeSecondaryReasonCodes) != 0 ||
			rec.RuntimeArbitrationRuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionExplainabilityV1 ||
			rec.RuntimeRemediationHintCode != "react.select_tool_calling_provider" ||
			rec.RuntimeRemediationHintDomain != runtimeconfig.ReadinessDomainRuntime {
			t.Fatalf("blocked react stream record mismatch: %#v", rec)
		}
	})
}

func TestRuntimeReadinessAdmissionReactDegradedPolicyMappingRunStreamEquivalent(t *testing.T) {
	allowCfg := filepath.Join(t.TempDir(), "runtime-a56-react-degraded-allow.yaml")
	writeRuntimeReadinessAdmissionConfig(t, allowCfg, true, runtimeconfig.ReadinessAdmissionDegradedPolicyAllowAndRecord)
	allowMgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  allowCfg,
		EnvPrefix: "BAYMAX_A56_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = allowMgr.Close() }()

	allowMgr.SetReactReadinessDependencySnapshot(runtimeconfig.ReactReadinessDependencySnapshot{
		ToolRegistryChecked:   true,
		ToolRegistryAvailable: false,
		ToolRegistryReason:    "tool registry unavailable",
	})

	allowModel := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	allowDispatcher := event.NewDispatcher(event.NewRuntimeRecorder(allowMgr))
	allowComp, err := composer.NewBuilder(allowModel).
		WithRuntimeManager(allowMgr).
		WithEventHandler(dispatcherHandler{dispatcher: allowDispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	if runRes, runErr := allowComp.Run(context.Background(), types.RunRequest{
		RunID: "run-a56-react-degraded-allow-run",
		Input: "allow-run",
	}, nil); runErr != nil || runRes.Error != nil {
		t.Fatalf("degraded allow run should succeed, err=%v result=%#v", runErr, runRes)
	}
	if streamRes, streamErr := allowComp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a56-react-degraded-allow-stream",
		Input: "allow-stream",
	}, nil); streamErr != nil || streamRes.Error != nil {
		t.Fatalf("degraded allow stream should succeed, err=%v result=%#v", streamErr, streamRes)
	}
	assertAdmissionRunRecord(t, allowMgr, "run-a56-react-degraded-allow-run", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 0 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 1 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimaryDomain != runtimeconfig.ReadinessDomainRuntime ||
			rec.RuntimePrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimarySource != runtimeconfig.RuntimePrimarySourceReadiness ||
			rec.RuntimePrimaryConflictTotal != 0 ||
			rec.RuntimeSecondaryReasonCount != 0 ||
			len(rec.RuntimeSecondaryReasonCodes) != 0 ||
			rec.RuntimeArbitrationRuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionExplainabilityV1 ||
			rec.RuntimeRemediationHintCode != "react.restore_tool_registry" ||
			rec.RuntimeRemediationHintDomain != runtimeconfig.ReadinessDomainRuntime {
			t.Fatalf("degraded allow run record mismatch: %#v", rec)
		}
	})
	assertAdmissionRunRecord(t, allowMgr, "run-a56-react-degraded-allow-stream", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 0 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 1 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimaryDomain != runtimeconfig.ReadinessDomainRuntime ||
			rec.RuntimePrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimarySource != runtimeconfig.RuntimePrimarySourceReadiness ||
			rec.RuntimePrimaryConflictTotal != 0 ||
			rec.RuntimeSecondaryReasonCount != 0 ||
			len(rec.RuntimeSecondaryReasonCodes) != 0 ||
			rec.RuntimeArbitrationRuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionExplainabilityV1 ||
			rec.RuntimeRemediationHintCode != "react.restore_tool_registry" ||
			rec.RuntimeRemediationHintDomain != runtimeconfig.ReadinessDomainRuntime {
			t.Fatalf("degraded allow stream record mismatch: %#v", rec)
		}
	})

	denyCfg := filepath.Join(t.TempDir(), "runtime-a56-react-degraded-deny.yaml")
	writeRuntimeReadinessAdmissionConfig(t, denyCfg, true, runtimeconfig.ReadinessAdmissionDegradedPolicyFailFast)
	denyMgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  denyCfg,
		EnvPrefix: "BAYMAX_A56_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = denyMgr.Close() }()
	denyMgr.SetReactReadinessDependencySnapshot(runtimeconfig.ReactReadinessDependencySnapshot{
		ToolRegistryChecked:   true,
		ToolRegistryAvailable: false,
		ToolRegistryReason:    "tool registry unavailable",
	})

	denyModel := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	denyDispatcher := event.NewDispatcher(event.NewRuntimeRecorder(denyMgr))
	denyComp, err := composer.NewBuilder(denyModel).
		WithRuntimeManager(denyMgr).
		WithEventHandler(dispatcherHandler{dispatcher: denyDispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	before, err := denyComp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats before deny failed: %v", err)
	}
	mailboxBefore := len(denyMgr.RecentMailbox(10))

	runRes, runErr := denyComp.Run(context.Background(), types.RunRequest{
		RunID: "run-a56-react-degraded-deny-run",
		Input: "deny-run",
	}, nil)
	if runErr == nil {
		t.Fatal("degraded fail_fast run should be denied")
	}
	assertAdmissionContractDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeDegradedDeny)

	streamRes, streamErr := denyComp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a56-react-degraded-deny-stream",
		Input: "deny-stream",
	}, nil)
	if streamErr == nil {
		t.Fatal("degraded fail_fast stream should be denied")
	}
	assertAdmissionContractDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeDegradedDeny)

	after, err := denyComp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after deny failed: %v", err)
	}
	if before.QueueTotal != after.QueueTotal || before.ClaimTotal != after.ClaimTotal || before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("deny path should be side-effect free, before=%#v after=%#v", before, after)
	}
	if len(denyMgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(denyMgr.RecentMailbox(10)))
	}

	assertAdmissionRunRecord(t, denyMgr, "run-a56-react-degraded-deny-run", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionMode != runtimeconfig.ReadinessAdmissionModeFailFast ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimaryDomain != runtimeconfig.ReadinessDomainRuntime ||
			rec.RuntimePrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimarySource != runtimeconfig.RuntimePrimarySourceReadiness ||
			rec.RuntimeArbitrationRuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionExplainabilityV1 ||
			rec.RuntimeRemediationHintCode != "react.restore_tool_registry" ||
			rec.RuntimeRemediationHintDomain != runtimeconfig.ReadinessDomainRuntime {
			t.Fatalf("degraded deny run record mismatch: %#v", rec)
		}
	})
	assertAdmissionRunRecord(t, denyMgr, "run-a56-react-degraded-deny-stream", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionMode != runtimeconfig.ReadinessAdmissionModeFailFast ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimaryDomain != runtimeconfig.ReadinessDomainRuntime ||
			rec.RuntimePrimaryCode != runtimeconfig.ReadinessCodeReactToolRegistryUnavailable ||
			rec.RuntimePrimarySource != runtimeconfig.RuntimePrimarySourceReadiness ||
			rec.RuntimeArbitrationRuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionExplainabilityV1 ||
			rec.RuntimeRemediationHintCode != "react.restore_tool_registry" ||
			rec.RuntimeRemediationHintDomain != runtimeconfig.ReadinessDomainRuntime {
			t.Fatalf("degraded deny stream record mismatch: %#v", rec)
		}
	})
}
