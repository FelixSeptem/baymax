package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/model/conformance"
)

const requestProjectionFixturePath = "testdata/model_request_projection.v1.json"

// regenRequestProjectionFixtureEnv switches the generator on. Regenerating is
// an explicit, reviewed act: the artifact is a pinned contract, so it must
// never rewrite itself during a normal test run.
const regenRequestProjectionFixtureEnv = "BAYMAX_REGEN_REQUEST_PROJECTION_FIXTURE"

// auditedRequestGaps is the pinned semantic-loss set produced by every current
// adapter for the audited request shape. It is evidence, not a tolerance:
// removing an entry requires an explicit contract change, and adding one
// requires the adapter to actually regress.
func auditedRequestGaps() []string {
	return []string{
		conformance.ReasonRequestCapabilityProjectionDrift,
		conformance.ReasonRequestRoleProjectionDrift,
		conformance.ReasonRequestToolResultNativeDrift,
	}
}

// fixtureAuditPayload reconstructs, provider-neutrally, the text payload the
// current adapters hand to their SDK: the canonical input plus the canonical
// tool-result feedback envelope when tool results were admitted.
//
// The bytes are synthetic by design. Byte-for-byte equivalence with the live
// adapters is owned by model/<provider>/request_projection_test.go; here only
// the normalized projection of that payload is pinned.
func fixtureAuditPayload(withToolResult bool) string {
	payload := "summarize the repository"
	if withToolResult {
		payload += "\nworking on it\n[tool_result_feedback.v1]\n" +
			`{"tool_name":"read_file","call_id":"call-1","content":"file body"}`
	}
	return payload
}

// fixtureSource mirrors what RequestFactsFromModelRequest admits for the
// audited request shape once Skill bundle mapping has appended system-role
// prompt fragments and the ReAct loop has produced tool-result feedback.
func fixtureSource(withToolResult, withCapabilities bool) conformance.RequestFacts {
	source := conformance.RequestFacts{
		Roles:      []string{"system", "user", "assistant"},
		InputBytes: len("summarize the repository"),
	}
	if withToolResult {
		source.ToolResults = []conformance.RequestToolFact{{
			CallID:       "call-1",
			Name:         "read_file",
			Status:       "ok",
			ResultDigest: conformance.RequestContentDigest("file body"),
		}}
		source.ToolOrder = []string{"read_file"}
	}
	if withCapabilities {
		source.Capabilities = []string{"tool_call", "streaming"}
	}
	return source
}

func fixtureObserved(source conformance.RequestFacts) conformance.RequestProjection {
	return conformance.ProjectRequestTextEnvelope(fixtureAuditPayload(len(source.ToolResults) > 0), source.ToolResults)
}

func finalizeRequestProjectionCase(t *testing.T, c conformance.RequestProjectionCase) conformance.RequestProjectionCase {
	t.Helper()
	digest, err := conformance.RequestProjectionDigest(c.Observed)
	if err != nil {
		t.Fatalf("digest %q: %v", c.Name, err)
	}
	c.FirstDigest = digest
	c.ReplayDigest = digest
	return c
}

// finalizeRequestProjectionCaseBestEffort sets the idempotency digests only
// when the observed projection is digestible. Cases that deliberately carry an
// invalid projection are rejected earlier in validation, so an empty digest
// can never mask the classification under test.
func finalizeRequestProjectionCaseBestEffort(c conformance.RequestProjectionCase) conformance.RequestProjectionCase {
	if digest, err := conformance.RequestProjectionDigest(c.Observed); err == nil {
		c.FirstDigest = digest
		c.ReplayDigest = digest
	}
	return c
}

// requestProjectionFixture builds the committed fixture in Go so the artifact
// can never drift from the contract silently: the drift guard below rebuilds
// the same cases and compares bytes.
func requestProjectionFixture(t *testing.T) conformance.RequestProjectionFixture {
	t.Helper()
	cases := make([]conformance.RequestProjectionCase, 0, 16)
	for _, provider := range []string{"openai", "anthropic", "gemini"} {
		for _, mode := range []string{"run", "stream"} {
			source := fixtureSource(true, true)
			observed := fixtureObserved(source)
			expected := observed
			cases = append(cases, finalizeRequestProjectionCase(t, conformance.RequestProjectionCase{
				Name:         provider + "_" + mode + "_canonical_text_envelope",
				Provider:     provider,
				Mode:         mode,
				Source:       source,
				Observed:     observed,
				Expected:     &expected,
				DeclaredGaps: auditedRequestGaps(),
			}))
		}

		// Run/Stream parity: the same admitted request must normalize
		// identically on both lanes.
		paritySource := fixtureSource(true, true)
		parityObserved := fixtureObserved(paritySource)
		runProjection := parityObserved
		streamProjection := parityObserved
		cases = append(cases, finalizeRequestProjectionCase(t, conformance.RequestProjectionCase{
			Name:         provider + "_run_stream_parity",
			Provider:     provider,
			Mode:         "run",
			Source:       paritySource,
			Observed:     runProjection,
			Run:          &runProjection,
			Stream:       &streamProjection,
			DeclaredGaps: auditedRequestGaps(),
		}))

		// No tool results admitted: the native-tool-result gap must disappear
		// while role and capability gaps remain.
		plainSource := fixtureSource(false, true)
		cases = append(cases, finalizeRequestProjectionCase(t, conformance.RequestProjectionCase{
			Name:     provider + "_run_without_tool_results",
			Provider: provider,
			Mode:     "run",
			Source:   plainSource,
			Observed: fixtureObserved(plainSource),
			DeclaredGaps: []string{
				conformance.ReasonRequestCapabilityProjectionDrift,
				conformance.ReasonRequestRoleProjectionDrift,
			},
		}))
	}
	return conformance.RequestProjectionFixture{
		Version: RequestProjectionFixtureV1,
		Cases:   cases,
	}
}

func marshalRequestProjectionFixture(t *testing.T, fixture conformance.RequestProjectionFixture) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return append(raw, '\n')
}

// TestGenerateProviderRequestProjectionFixture regenerates the committed
// artifact. It is skipped by default so a normal test run can never rewrite a
// pinned contract:
//
//	BAYMAX_REGEN_REQUEST_PROJECTION_FIXTURE=1 go test ./tool/diagnosticsreplay -run TestGenerateProviderRequestProjectionFixture
func TestGenerateProviderRequestProjectionFixture(t *testing.T) {
	if os.Getenv(regenRequestProjectionFixtureEnv) != "1" {
		t.Skipf("set %s=1 to regenerate %s", regenRequestProjectionFixtureEnv, requestProjectionFixturePath)
	}
	raw := marshalRequestProjectionFixture(t, requestProjectionFixture(t))
	if err := os.WriteFile(filepath.FromSlash(requestProjectionFixturePath), raw, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

// TestProviderRequestProjectionFixtureMatchesCommittedArtifact is the drift
// guard: the checked-in artifact must be exactly what the contract builder
// produces.
func TestProviderRequestProjectionFixtureMatchesCommittedArtifact(t *testing.T) {
	want := marshalRequestProjectionFixture(t, requestProjectionFixture(t))
	got, err := os.ReadFile(filepath.FromSlash(requestProjectionFixturePath))
	if err != nil {
		t.Fatalf("read committed fixture: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("%s drifted from the contract builder; regenerate it with %s=1",
			requestProjectionFixturePath, regenRequestProjectionFixtureEnv)
	}
}

// TestProviderRequestProjectionFixtureReplays is the success path: a legal,
// digest-consistent fixture replays offline and deterministically.
func TestProviderRequestProjectionFixtureReplays(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash(requestProjectionFixturePath))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := ReplayProviderRequestProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if result.Version != RequestProjectionFixtureV1 {
		t.Fatalf("unexpected version %q", result.Version)
	}
	if len(result.Cases) != len(requestProjectionFixture(t).Cases) {
		t.Fatalf("replay dropped cases: got %d", len(result.Cases))
	}

	providers := map[string]int{}
	parityVerified := 0
	for _, c := range result.Cases {
		if !c.Idempotent || c.Digest != c.ReplayDigest || c.Digest == "" {
			t.Fatalf("case %q is not idempotent: %+v", c.Name, c)
		}
		if c.CacheUsageAvailable {
			t.Fatalf("case %q claims cache usage availability without an accounting source", c.Name)
		}
		providers[c.Provider]++
		if c.RunStreamParityVerified {
			parityVerified++
		}
	}
	for _, provider := range []string{"openai", "anthropic", "gemini"} {
		if providers[provider] == 0 {
			t.Fatalf("provider %q is not covered by the fixture", provider)
		}
	}
	if parityVerified == 0 {
		t.Fatal("fixture contains no verified run/stream parity case")
	}
}

// TestProviderRequestProjectionReplayIsIdempotent proves replay is a pure
// function: same input, same output, no growth.
func TestProviderRequestProjectionReplayIsIdempotent(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash(requestProjectionFixturePath))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	first, err := ReplayProviderRequestProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := ReplayProviderRequestProjectionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if len(first.Cases) != len(second.Cases) {
		t.Fatalf("case count grew: %d -> %d", len(first.Cases), len(second.Cases))
	}
	for i := range first.Cases {
		if first.Cases[i].Name != second.Cases[i].Name ||
			first.Cases[i].Digest != second.Cases[i].Digest ||
			strings.Join(first.Cases[i].Gaps, ",") != strings.Join(second.Cases[i].Gaps, ",") {
			t.Fatalf("case %d is not deterministic", i)
		}
	}
}

// TestProviderRequestProjectionReplayAcceptsHistoricalFixtureWithoutCacheUsage
// pins the additive + nullable + default rule: a fixture written before cache
// accounting existed decodes and replays without error.
func TestProviderRequestProjectionReplayAcceptsHistoricalFixtureWithoutCacheUsage(t *testing.T) {
	c := finalizeRequestProjectionCase(t, conformance.RequestProjectionCase{
		Name:         "openai_run_historical_shape",
		Provider:     "openai",
		Mode:         "run",
		Source:       fixtureSource(true, true),
		Observed:     fixtureObserved(fixtureSource(true, true)),
		DeclaredGaps: auditedRequestGaps(),
	})
	raw := marshalFixtureCase(t, c)

	// Drop cache_usage entirely and prove the historical shape still replays.
	historical := stripCacheUsage(t, raw)
	if bytes.Contains(historical, []byte(`"cache_usage"`)) {
		t.Fatal("test setup failed to strip cache_usage")
	}

	result, err := ReplayProviderRequestProjectionFixtureJSON(historical)
	if err != nil {
		t.Fatalf("historical fixture must replay: %v", err)
	}
	if len(result.Cases) != 1 || result.Cases[0].CacheUsageAvailable {
		t.Fatalf("missing cache_usage must default to unavailable: %+v", result.Cases)
	}
}

// TestProviderRequestProjectionReplayAcceptsUnknownFields proves an added
// provider field cannot break stored fixtures.
func TestProviderRequestProjectionReplayAcceptsUnknownFields(t *testing.T) {
	c := finalizeRequestProjectionCase(t, conformance.RequestProjectionCase{
		Name:         "gemini_stream_unknown_provider_fields",
		Provider:     "gemini",
		Mode:         "stream",
		Source:       fixtureSource(true, true),
		Observed:     fixtureObserved(fixtureSource(true, true)),
		DeclaredGaps: auditedRequestGaps(),
	})
	raw := marshalFixtureCase(t, c)
	raw = bytes.Replace(raw, []byte(`"version":"`),
		[]byte(`"future_fixture_field":{"nested":true},"version":"`), 1)
	raw = bytes.Replace(raw, []byte(`"mode":"stream"`),
		[]byte(`"mode":"stream","future_case_field":[1,2,3]`), 1)

	if _, err := ReplayProviderRequestProjectionFixtureJSON(raw); err != nil {
		t.Fatalf("unknown fields must be ignored, got %v", err)
	}
}

// TestProviderRequestProjectionReplayClassifiesDrift covers the canonical
// classification vocabulary at the replay boundary: each mutation must surface
// its own stable code, never a generic failure.
func TestProviderRequestProjectionReplayClassifiesDrift(t *testing.T) {
	cases := []struct {
		name     string
		wantCode string
		mutate   func(t *testing.T, c *conformance.RequestProjectionCase)
	}{
		{
			name:     "unsupported provider",
			wantCode: ReasonCodeRequestSchemaDrift,
			mutate:   func(t *testing.T, c *conformance.RequestProjectionCase) { c.Provider = "cohere" },
		},
		{
			name:     "unsupported mode",
			wantCode: ReasonCodeRequestSchemaDrift,
			mutate:   func(t *testing.T, c *conformance.RequestProjectionCase) { c.Mode = "batch" },
		},
		{
			name:     "unsupported part kind",
			wantCode: ReasonCodeRequestSchemaDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				c.Observed.Parts = []string{"provider_native_block"}
			},
		},
		{
			name:     "role projection drift",
			wantCode: ReasonCodeRequestRoleProjectionDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				expected := c.Observed
				expected.Roles = []string{"system", "user"}
				c.Expected = &expected
			},
		},
		{
			name:     "tool result native drift",
			wantCode: ReasonCodeRequestToolResultNativeDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				expected := c.Observed
				expected.ToolResultNative = true
				c.Observed.ToolResultNative = false
				c.Expected = &expected
			},
		},
		{
			name:     "part ordering drift",
			wantCode: ReasonCodeRequestPartOrderingDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				expected := c.Observed
				expected.Parts = []string{
					conformance.RequestPartToolResultEnvelope,
					conformance.RequestPartUserText,
				}
				c.Expected = &expected
			},
		},
		{
			name:     "stable prefix drift",
			wantCode: ReasonCodeRequestStablePrefixDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				expected := c.Observed
				expected.StablePrefixDigest = "sha256:stable-prefix"
				c.Expected = &expected
			},
		},
		{
			name:     "tool order drift",
			wantCode: ReasonCodeRequestToolOrderDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				expected := c.Observed
				expected.ToolOrder = []string{"write_file", "read_file"}
				c.Expected = &expected
			},
		},
		{
			name:     "capability projection drift",
			wantCode: ReasonCodeRequestCapabilityProjectionDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				expected := c.Observed
				expected.Capabilities = []string{"tool_call", "streaming"}
				c.Expected = &expected
			},
		},
		{
			name:     "cache usage projection drift",
			wantCode: ReasonCodeCacheUsageProjectionDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				expected := c.Observed
				expected.CacheUsage = conformance.CacheUsageProjection{Available: true, ReadTokens: 128}
				c.Expected = &expected
			},
		},
		{
			name:     "undeclared role gap",
			wantCode: ReasonCodeRequestRoleProjectionDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				c.Name = "openai_run_undeclared_role_gap"
				c.Source = conformance.RequestFacts{Roles: []string{"system", "user"}}
				source := c.Source
				c.Observed = fixtureObserved(source)
				c.Expected = nil
				c.DeclaredGaps = nil
			},
		},
		{
			name:     "undeclared native tool result gap",
			wantCode: ReasonCodeRequestToolResultNativeDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				c.Name = "openai_run_undeclared_tool_result_gap"
				c.Source = conformance.RequestFacts{
					Roles: []string{"user"},
					ToolResults: []conformance.RequestToolFact{{
						CallID: "call-1",
						Name:   "read_file",
						Status: "ok",
					}},
					ToolOrder: []string{"read_file"},
				}
				source := c.Source
				c.Observed = fixtureObserved(source)
				c.Expected = nil
				c.DeclaredGaps = nil
			},
		},
		{
			name:     "declared gap no longer reproduces",
			wantCode: ReasonCodeRequestContractDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				c.Name = "openai_run_declared_gap_fixed"
				// Every source dimension is preserved, so no gap is computed.
				c.Source = conformance.RequestFacts{Roles: []string{"user"}}
				source := c.Source
				c.Observed = fixtureObserved(source)
				c.Expected = nil
				c.DeclaredGaps = []string{conformance.ReasonRequestRoleProjectionDrift}
			},
		},
		{
			name:     "run/stream parity drift",
			wantCode: ReasonCodeRequestRunStreamParityDrift,
			mutate: func(t *testing.T, c *conformance.RequestProjectionCase) {
				c.Name = "openai_run_stream_parity_drift"
				run := c.Observed
				stream := c.Observed
				stream.ContentDigest = conformance.RequestContentDigest("different payload")
				c.Run = &run
				c.Stream = &stream
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := finalizeRequestProjectionCase(t, conformance.RequestProjectionCase{
				Name:         "openai_run_canonical_text_envelope",
				Provider:     "openai",
				Mode:         "run",
				Source:       fixtureSource(true, true),
				Observed:     fixtureObserved(fixtureSource(true, true)),
				DeclaredGaps: auditedRequestGaps(),
			})
			tc.mutate(t, &c)
			c = finalizeRequestProjectionCaseBestEffort(c)

			_, err := ReplayProviderRequestProjectionFixtureJSON(marshalFixtureCase(t, c))
			var validation *ValidationError
			if err == nil {
				t.Fatalf("expected replay to fail with %s", tc.wantCode)
			}
			if !errors.As(err, &validation) {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			if validation.Code != tc.wantCode {
				t.Fatalf("got code %q want %q (%s)", validation.Code, tc.wantCode, validation.Message)
			}
		})
	}
}

// TestProviderRequestProjectionReplayRejectsBounds proves overflow is a
// fail-fast classification, never a silent truncation.
func TestProviderRequestProjectionReplayRejectsBounds(t *testing.T) {
	base := finalizeRequestProjectionCase(t, conformance.RequestProjectionCase{
		Name:         "openai_run_canonical_text_envelope",
		Provider:     "openai",
		Mode:         "run",
		Source:       fixtureSource(true, true),
		Observed:     fixtureObserved(fixtureSource(true, true)),
		DeclaredGaps: auditedRequestGaps(),
	})

	t.Run("fixture exceeds byte bound", func(t *testing.T) {
		raw := append([]byte(`{"version":"`+RequestProjectionFixtureV1+`","cases":[`), bytes.Repeat([]byte(" "), conformance.MaxRequestBytes)...)
		_, err := ReplayProviderRequestProjectionFixtureJSON(raw)
		var validation *ValidationError
		if !errors.As(err, &validation) || validation.Code != ReasonCodeRequestOverflowDrift {
			t.Fatalf("expected overflow classification, got %v", err)
		}
	})

	t.Run("case count exceeds bound", func(t *testing.T) {
		overflow := conformance.RequestProjectionFixture{
			Version: RequestProjectionFixtureV1,
			Cases:   make([]conformance.RequestProjectionCase, conformance.MaxRequestCases+1),
		}
		for i := range overflow.Cases {
			overflow.Cases[i] = base
		}
		raw, err := json.Marshal(overflow)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		_, replayErr := ReplayProviderRequestProjectionFixtureJSON(raw)
		var validation *ValidationError
		if !errors.As(replayErr, &validation) || validation.Code != ReasonCodeRequestSchemaDrift {
			t.Fatalf("expected case-count schema failure, got %v", replayErr)
		}
	})

	t.Run("part overflow classifies as overflow drift", func(t *testing.T) {
		parts := make([]string, conformance.MaxRequestParts+1)
		for i := range parts {
			parts[i] = conformance.RequestPartUserText
		}
		mutated := base
		mutated.Observed.Parts = parts
		mutated = finalizeRequestProjectionCaseBestEffort(mutated)

		_, err := ReplayProviderRequestProjectionFixtureJSON(marshalFixtureCase(t, mutated))
		var validation *ValidationError
		if !errors.As(err, &validation) || validation.Code != ReasonCodeRequestOverflowDrift {
			t.Fatalf("expected overflow classification, got %v", err)
		}
	})
}

// TestProviderRequestProjectionTaxonomyIsStable pins the replay vocabulary to
// the contract vocabulary. Because replay aliases the conformance constants
// rather than copying them, this test is a compile-time-equivalent guard
// against a future refactor re-introducing a second, drifting source.
func TestProviderRequestProjectionTaxonomyIsStable(t *testing.T) {
	replayCodes := map[string]string{
		"provider_request_schema_drift":                ReasonCodeRequestSchemaDrift,
		"provider_request_role_projection_drift":       ReasonCodeRequestRoleProjectionDrift,
		"provider_request_tool_result_native_drift":    ReasonCodeRequestToolResultNativeDrift,
		"provider_request_part_ordering_drift":         ReasonCodeRequestPartOrderingDrift,
		"provider_request_stable_prefix_drift":         ReasonCodeRequestStablePrefixDrift,
		"provider_request_tool_order_drift":            ReasonCodeRequestToolOrderDrift,
		"provider_request_capability_projection_drift": ReasonCodeRequestCapabilityProjectionDrift,
		"provider_request_run_stream_parity_drift":     ReasonCodeRequestRunStreamParityDrift,
		"provider_cache_usage_projection_drift":        ReasonCodeCacheUsageProjectionDrift,
		"provider_request_overflow_drift":              ReasonCodeRequestOverflowDrift,
		"provider_request_contract_drift":              ReasonCodeRequestContractDrift,
	}

	contractCodes := map[string]string{
		"provider_request_schema_drift":                conformance.ReasonRequestSchemaDrift,
		"provider_request_role_projection_drift":       conformance.ReasonRequestRoleProjectionDrift,
		"provider_request_tool_result_native_drift":    conformance.ReasonRequestToolResultNativeDrift,
		"provider_request_part_ordering_drift":         conformance.ReasonRequestPartOrderingDrift,
		"provider_request_stable_prefix_drift":         conformance.ReasonRequestStablePrefixDrift,
		"provider_request_tool_order_drift":            conformance.ReasonRequestToolOrderDrift,
		"provider_request_capability_projection_drift": conformance.ReasonRequestCapabilityProjectionDrift,
		"provider_request_run_stream_parity_drift":     conformance.ReasonRequestRunStreamParityDrift,
		"provider_cache_usage_projection_drift":        conformance.ReasonCacheUsageProjectionDrift,
		"provider_request_overflow_drift":              conformance.ReasonRequestOverflowDrift,
		"provider_request_contract_drift":              conformance.ReasonRequestContractDrift,
	}

	if len(replayCodes) != len(contractCodes) {
		t.Fatalf("classifications are missing: replay=%d contract=%d", len(replayCodes), len(contractCodes))
	}
	for code, value := range contractCodes {
		if value != code {
			t.Fatalf("contract constant drifted: key %q carries value %q", code, value)
		}
		got, ok := replayCodes[code]
		if !ok {
			t.Fatalf("replay no longer exposes classification %q", code)
		}
		if got != value {
			t.Fatalf("taxonomy drift for %q: replay %q contract %q", code, got, value)
		}
	}
	if RequestProjectionFixtureV1 != conformance.FixtureVersionRequestV1 {
		t.Fatalf("fixture version drifted: %q != %q", RequestProjectionFixtureV1, conformance.FixtureVersionRequestV1)
	}
}

// marshalFixtureCase wraps one case in a single-case fixture document.
func marshalFixtureCase(t *testing.T, c conformance.RequestProjectionCase) []byte {
	t.Helper()
	raw, err := json.Marshal(conformance.RequestProjectionFixture{
		Version: RequestProjectionFixtureV1,
		Cases:   []conformance.RequestProjectionCase{c},
	})
	if err != nil {
		t.Fatalf("marshal fixture case %q: %v", c.Name, err)
	}
	return raw
}

// stripCacheUsage rewrites a fixture document as it would have been written
// before cache accounting existed: every cache_usage field is removed,
// wherever it appears.
func stripCacheUsage(t *testing.T, raw []byte) []byte {
	t.Helper()
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode fixture for cache-usage stripping: %v", err)
	}
	removeCacheUsageFields(document)
	stripped, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode stripped fixture: %v", err)
	}
	return stripped
}

func removeCacheUsageFields(node any) {
	switch typed := node.(type) {
	case map[string]any:
		delete(typed, "cache_usage")
		for _, value := range typed {
			removeCacheUsageFields(value)
		}
	case []any:
		for _, value := range typed {
			removeCacheUsageFields(value)
		}
	}
}
