package offline

import (
	"reflect"
	"testing"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
)

func TestOfflineModelAdapterNegotiationFallbackAndOverride(t *testing.T) {
	declared := adaptercap.Set{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional: []string{"model.capability.token_count"},
	}

	// default strategy is fail_fast.
	defaultOutcome, err := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, adaptercap.Request{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional: []string{"model.capability.token_count"},
	})
	if err != nil {
		t.Fatalf("default negotiation failed: %v", err)
	}
	if defaultOutcome.Accepted {
		t.Fatal("expected fail_fast to reject missing optional request")
	}

	// request-level override hook to best_effort.
	overrideOutcome, err := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, adaptercap.Request{
		Required:         []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional:         []string{"model.capability.token_count"},
		StrategyOverride: adaptercap.StrategyBestEffort,
	})
	if err != nil {
		t.Fatalf("override negotiation failed: %v", err)
	}
	if !overrideOutcome.Accepted || !overrideOutcome.Downgraded {
		t.Fatalf("expected best_effort downgrade path, got %#v", overrideOutcome)
	}
	if !containsReason(overrideOutcome.Reasons, adaptercap.ReasonOptionalDowngraded) ||
		!containsReason(overrideOutcome.Reasons, adaptercap.ReasonStrategyOverrideApply) {
		t.Fatalf("unexpected override reasons: %#v", overrideOutcome.Reasons)
	}
}

func TestOfflineModelAdapterNegotiationRunStreamEquivalent(t *testing.T) {
	declared := adaptercap.Set{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional: []string{"model.capability.token_count"},
	}
	req := adaptercap.Request{
		Required:         []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional:         []string{"model.capability.token_count"},
		StrategyOverride: adaptercap.StrategyBestEffort,
	}
	runOutcome, runErr := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, req)
	streamOutcome, streamErr := adaptercap.Negotiate(adaptercap.StrategyFailFast, declared, req)
	if runErr != nil || streamErr != nil {
		t.Fatalf("unexpected run/stream negotiation error runErr=%v streamErr=%v", runErr, streamErr)
	}
	if !reflect.DeepEqual(runOutcome, streamOutcome) {
		t.Fatalf("run/stream negotiation mismatch run=%#v stream=%#v", runOutcome, streamOutcome)
	}
}

func containsReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}
