package config

import (
	"strings"
	"testing"
)

func TestResolveArbitrationRuleVersionMatrix(t *testing.T) {
	base := RuntimeArbitrationVersionConfig{
		Enabled:       true,
		Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
		CompatWindow:  1,
		OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
		OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
	}
	tests := []struct {
		name            string
		cfg             RuntimeArbitrationVersionConfig
		requested       string
		wantErrCode     string
		wantEffective   string
		wantSource      string
		wantPolicy      string
		wantUnsupported int
		wantMismatch    int
	}{
		{
			name:          "default-when-request-absent",
			cfg:           base,
			requested:     "",
			wantEffective: RuntimeArbitrationRuleVersionExplainabilityV1,
			wantSource:    RuntimeArbitrationVersionSourceDefault,
			wantPolicy:    RuntimeArbitrationPolicyActionNone,
		},
		{
			name:          "requested-supported-version",
			cfg:           base,
			requested:     RuntimeArbitrationRuleVersionPrimaryReasonV1,
			wantEffective: RuntimeArbitrationRuleVersionPrimaryReasonV1,
			wantSource:    RuntimeArbitrationVersionSourceRequested,
			wantPolicy:    RuntimeArbitrationPolicyActionNone,
		},
		{
			name:            "unsupported-request-fail-fast",
			cfg:             base,
			requested:       "a77.v9",
			wantErrCode:     ArbitrationRuleVersionErrorUnsupported,
			wantEffective:   "",
			wantSource:      RuntimeArbitrationVersionSourceRequested,
			wantPolicy:      RuntimeArbitrationPolicyActionFailFastUnsupported,
			wantUnsupported: 1,
		},
		{
			name: "compat-window-mismatch-fail-fast",
			cfg: RuntimeArbitrationVersionConfig{
				Enabled:       true,
				Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
				CompatWindow:  0,
				OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
				OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
			},
			requested:     RuntimeArbitrationRuleVersionPrimaryReasonV1,
			wantErrCode:   ArbitrationRuleVersionErrorMismatch,
			wantEffective: "",
			wantSource:    RuntimeArbitrationVersionSourceRequested,
			wantPolicy:    RuntimeArbitrationPolicyActionFailFastMismatch,
			wantMismatch:  1,
		},
		{
			name: "governance-disabled",
			cfg: RuntimeArbitrationVersionConfig{
				Enabled:       false,
				Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
				CompatWindow:  1,
				OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
				OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
			},
			requested:     "a77.v9",
			wantEffective: RuntimeArbitrationRuleVersionExplainabilityV1,
			wantSource:    RuntimeArbitrationVersionSourceDefault,
			wantPolicy:    RuntimeArbitrationPolicyActionDisabled,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveArbitrationRuleVersion(tc.cfg, tc.requested)
			if tc.wantErrCode == "" {
				if err != nil {
					t.Fatalf("ResolveArbitrationRuleVersion unexpected error: %v", err)
				}
			} else {
				var typed *ArbitrationRuleVersionError
				switch v := err.(type) {
				case *ArbitrationRuleVersionError:
					typed = v
				default:
					t.Fatalf("error type = %T, want *ArbitrationRuleVersionError", err)
				}
				if typed.Code != tc.wantErrCode {
					t.Fatalf("error code = %q, want %q", typed.Code, tc.wantErrCode)
				}
			}
			if strings.TrimSpace(got.EffectiveVersion) != strings.TrimSpace(tc.wantEffective) {
				t.Fatalf("effective_version = %q, want %q", got.EffectiveVersion, tc.wantEffective)
			}
			if strings.TrimSpace(got.VersionSource) != strings.TrimSpace(tc.wantSource) {
				t.Fatalf("version_source = %q, want %q", got.VersionSource, tc.wantSource)
			}
			if strings.TrimSpace(got.PolicyAction) != strings.TrimSpace(tc.wantPolicy) {
				t.Fatalf("policy_action = %q, want %q", got.PolicyAction, tc.wantPolicy)
			}
			if got.UnsupportedTotal != tc.wantUnsupported {
				t.Fatalf("unsupported_total = %d, want %d", got.UnsupportedTotal, tc.wantUnsupported)
			}
			if got.MismatchTotal != tc.wantMismatch {
				t.Fatalf("mismatch_total = %d, want %d", got.MismatchTotal, tc.wantMismatch)
			}
		})
	}
}

func TestValidateRuntimeArbitrationVersionConfigRejectsInvalidValues(t *testing.T) {
	tests := []RuntimeArbitrationVersionConfig{
		{
			Enabled:       true,
			Default:       "a77.v9",
			CompatWindow:  1,
			OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
			OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
		},
		{
			Enabled:       true,
			Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
			CompatWindow:  -1,
			OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
			OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
		},
		{
			Enabled:       true,
			Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
			CompatWindow:  1,
			OnUnsupported: "warn",
			OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
		},
		{
			Enabled:       true,
			Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
			CompatWindow:  1,
			OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
			OnMismatch:    "warn",
		},
	}
	for i := range tests {
		if err := ValidateRuntimeArbitrationVersionConfig(tests[i]); err == nil {
			t.Fatalf("case %d expected validation error", i)
		}
	}
}
