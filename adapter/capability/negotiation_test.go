package capability

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestNegotiateDefaultFailFastNoOverride(t *testing.T) {
	out, err := Negotiate("", Set{
		Required: []string{"tool.invoke.required_input"},
		Optional: []string{"tool.schema.rich_validation"},
	}, Request{
		Required: []string{"tool.invoke.required_input"},
	})
	if err != nil {
		t.Fatalf("negotiate: %v", err)
	}
	if !out.Accepted || out.AppliedStrategy != StrategyFailFast || out.StrategyOverrideApplied {
		t.Fatalf("unexpected outcome: %#v", out)
	}
	if len(out.Reasons) != 0 {
		t.Fatalf("unexpected reasons: %#v", out.Reasons)
	}
}

func TestNegotiateRequiredMissingFailFast(t *testing.T) {
	out, err := Negotiate(StrategyFailFast, Set{
		Required: []string{"mcp.invoke.required_input"},
		Optional: []string{"mcp.response.normalized"},
	}, Request{
		Required: []string{"mcp.transport.sse"},
	})
	if err != nil {
		t.Fatalf("negotiate: %v", err)
	}
	if out.Accepted {
		t.Fatal("expected fail-fast rejection")
	}
	if len(out.MissingRequired) != 1 || out.MissingRequired[0] != "mcp.transport.sse" {
		t.Fatalf("unexpected missing required: %#v", out.MissingRequired)
	}
	if !contains(out.Reasons, ReasonMissingRequired) {
		t.Fatalf("missing reason code %q in %#v", ReasonMissingRequired, out.Reasons)
	}
}

func TestNegotiateBestEffortOptionalDowngrade(t *testing.T) {
	out, err := Negotiate(StrategyFailFast, Set{
		Required: []string{"model.run_stream.semantic_equivalent"},
		Optional: []string{"model.response.mandatory_fields"},
	}, Request{
		Required:         []string{"model.run_stream.semantic_equivalent"},
		Optional:         []string{"model.capability.token_count"},
		StrategyOverride: StrategyBestEffort,
	})
	if err != nil {
		t.Fatalf("negotiate: %v", err)
	}
	if !out.Accepted || !out.Downgraded {
		t.Fatalf("expected accepted downgrade outcome: %#v", out)
	}
	if !contains(out.Reasons, ReasonOptionalDowngraded) || !contains(out.Reasons, ReasonStrategyOverrideApply) {
		t.Fatalf("unexpected reasons: %#v", out.Reasons)
	}
	if len(out.DowngradedOptional) != 1 || out.DowngradedOptional[0] != "model.capability.token_count" {
		t.Fatalf("unexpected downgraded optional: %#v", out.DowngradedOptional)
	}
}

func TestNegotiateFailFastOptionalRequestedRejected(t *testing.T) {
	out, err := Negotiate(StrategyFailFast, Set{
		Required: []string{"model.run_stream.semantic_equivalent"},
		Optional: []string{"model.response.mandatory_fields"},
	}, Request{
		Required: []string{"model.run_stream.semantic_equivalent"},
		Optional: []string{"model.capability.token_count"},
	})
	if err != nil {
		t.Fatalf("negotiate: %v", err)
	}
	if out.Accepted {
		t.Fatal("expected fail_fast to reject missing optional request")
	}
	if !contains(out.Reasons, ReasonMissingRequired) {
		t.Fatalf("expected missing_required reason, got %#v", out.Reasons)
	}
}

func TestNegotiateDeterministicOutcomeAndReasons(t *testing.T) {
	declared := Set{
		Required: []string{"a.required"},
		Optional: []string{"a.optional"},
	}
	req := Request{
		Required:         []string{"a.required"},
		Optional:         []string{"a.missing_optional"},
		StrategyOverride: StrategyBestEffort,
	}
	a, errA := Negotiate(StrategyFailFast, declared, req)
	b, errB := Negotiate(StrategyFailFast, declared, req)
	if errA != nil || errB != nil {
		t.Fatalf("unexpected errors: %v %v", errA, errB)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("non-deterministic outcome: %#v vs %#v", a, b)
	}
}

func TestNegotiateInvalidStrategy(t *testing.T) {
	_, err := Negotiate("random", Set{}, Request{})
	if err == nil {
		t.Fatal("expected invalid default strategy error")
	}
	var ne *NegotiationError
	if !errors.As(err, &ne) {
		t.Fatalf("expected negotiation error, got %T", err)
	}
	if ne.Code != CodeInvalidStrategy {
		t.Fatalf("unexpected error code: %#v", ne)
	}

	_, err = Negotiate(StrategyFailFast, Set{}, Request{StrategyOverride: "bad"})
	if err == nil {
		t.Fatal("expected invalid override strategy error")
	}
}

func TestDiagnosticsAdditiveNullableDefaultCompatibility(t *testing.T) {
	legacy := []byte(`{"adapter_capability_strategy_applied":"fail_fast"}`)
	var d Diagnostics
	if err := json.Unmarshal(legacy, &d); err != nil {
		t.Fatalf("unmarshal legacy diagnostics: %v", err)
	}
	if d.StrategyApplied != StrategyFailFast {
		t.Fatalf("unexpected strategy: %#v", d)
	}
	if d.StrategyOverrideApplied {
		t.Fatalf("override flag should default false: %#v", d)
	}
	if d.MissingRequired != nil || d.MissingOptional != nil || d.DowngradedOptional != nil || d.ReasonCodes != nil {
		t.Fatalf("nullable fields should remain nil for legacy payload: %#v", d)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
