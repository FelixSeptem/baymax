package conformance

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func mustDigest(t *testing.T, projection RequestProjection) string {
	t.Helper()
	digest, err := RequestProjectionDigest(projection)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	return digest
}

func validCaseObserved(t *testing.T, observed RequestProjection) *RequestProjectionCase {
	t.Helper()
	digest := mustDigest(t, observed)
	return &RequestProjectionCase{
		Name:         "case",
		Provider:     "openai",
		Mode:         "run",
		Source:       RequestFacts{Roles: []string{"user"}},
		Observed:     observed,
		FirstDigest:  digest,
		ReplayDigest: digest,
	}
}

func TestRequestProjectionDigestIsDeterministic(t *testing.T) {
	projection := RequestProjection{
		Roles:     []string{"user"},
		Parts:     []string{RequestPartUserText, RequestPartToolResultEnvelope},
		ToolOrder: []string{"read_file"},
	}
	first, err := RequestProjectionDigest(projection)
	if err != nil {
		t.Fatalf("first digest: %v", err)
	}
	second, err := RequestProjectionDigest(projection)
	if err != nil {
		t.Fatalf("second digest: %v", err)
	}
	if first != second {
		t.Fatalf("digest is not deterministic: %q != %q", first, second)
	}
	canonical, err := CanonicalRequestProjection(projection)
	if err != nil {
		t.Fatalf("canonical: %v", err)
	}
	if !json.Valid(canonical) {
		t.Fatalf("canonical projection is not valid json: %s", canonical)
	}
}

func TestRequestProjectionIgnoresUnknownFields(t *testing.T) {
	base := `{"roles":["user"],"parts":["user_text"],"cache_usage":{"available":true,"read_tokens":7,"total_tokens":7,"source_kind":"openai_responses","source_version":"v1"}}`
	withUnknown := `{"roles":["user"],"parts":["user_text"],"cache_usage":{"available":true,"read_tokens":7,"total_tokens":7,"source_kind":"openai_responses","source_version":"v1"},"future_provider_field":{"nested":true},"another_unknown":"x"}`

	var plain, extended RequestProjection
	if err := json.Unmarshal([]byte(base), &plain); err != nil {
		t.Fatalf("decode base: %v", err)
	}
	if err := json.Unmarshal([]byte(withUnknown), &extended); err != nil {
		t.Fatalf("decode extended: %v", err)
	}
	if got, want := mustDigest(t, extended), mustDigest(t, plain); got != want {
		t.Fatalf("unknown fields changed the canonical digest: %q != %q", got, want)
	}
}

func TestRequestProjectionHistoricalFixtureWithoutCacheUsage(t *testing.T) {
	// A fixture written before cache usage existed omits cache_usage entirely.
	raw := `{"roles":["user"],"parts":["user_text"],"tool_result_native":false}`
	var projection RequestProjection
	if err := json.Unmarshal([]byte(raw), &projection); err != nil {
		t.Fatalf("decode historical projection: %v", err)
	}
	if projection.CacheUsage.Available {
		t.Fatal("missing cache_usage must default to unavailable")
	}
	if projection.CacheUsage.ReadTokens != 0 || projection.CacheUsage.WriteTokens != 0 {
		t.Fatalf("missing cache_usage must default to zero tokens, got %+v", projection.CacheUsage)
	}
	if err := ValidateRequestProjection(projection); err != nil {
		t.Fatalf("historical projection must validate: %v", err)
	}
}

func TestRequestProjectionRejectsFabricatedCacheUsage(t *testing.T) {
	projection := RequestProjection{
		CacheUsage: CacheUsageProjection{Available: false, ReadTokens: 12},
	}
	err := ValidateRequestProjection(projection)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonCacheUsageProjectionDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestRequestProjectionRejectsNegativeCacheUsage(t *testing.T) {
	projection := RequestProjection{
		CacheUsage: CacheUsageProjection{Available: true, WriteTokens: -1},
	}
	err := ValidateRequestProjection(projection)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonCacheUsageProjectionDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestCacheUsageProjectionAcceptsNormalizedReadWriteTotalAndSource(t *testing.T) {
	projection := RequestProjection{CacheUsage: CacheUsageProjection{
		Available:     true,
		ReadTokens:    12,
		WriteTokens:   3,
		TotalTokens:   15,
		SourceKind:    CacheUsageSourceOpenAIResponses,
		SourceVersion: "v1",
	}}
	if err := ValidateRequestProjection(projection); err != nil {
		t.Fatalf("expected normalized cache usage to validate: %v", err)
	}
}

func TestCacheUsageProjectionRejectsInconsistentTotal(t *testing.T) {
	err := ValidateRequestProjection(RequestProjection{CacheUsage: CacheUsageProjection{
		Available:     true,
		ReadTokens:    12,
		WriteTokens:   3,
		TotalTokens:   14,
		SourceKind:    CacheUsageSourceOpenAIResponses,
		SourceVersion: "v1",
	}})
	var classified *RequestProjectionError
	if !errors.As(err, &classified) || classified.Code != ReasonCacheUsageProjectionDrift {
		t.Fatalf("expected cache drift for inconsistent total, got %v", err)
	}
}

func TestCacheUsageProjectionRejectsMissingSourceWhenAvailable(t *testing.T) {
	err := ValidateRequestProjection(RequestProjection{CacheUsage: CacheUsageProjection{
		Available:  true,
		ReadTokens: 1,
	}})
	var classified *RequestProjectionError
	if !errors.As(err, &classified) || classified.Code != ReasonCacheUsageProjectionDrift {
		t.Fatalf("expected cache drift for missing source, got %v", err)
	}
}

func TestCacheUsageProjectionRejectsOverflow(t *testing.T) {
	err := ValidateRequestProjection(RequestProjection{CacheUsage: CacheUsageProjection{
		Available:     true,
		ReadTokens:    MaxCacheTokens + 1,
		SourceKind:    CacheUsageSourceGemini,
		SourceVersion: "v1",
	}})
	var classified *RequestProjectionError
	if !errors.As(err, &classified) || classified.Code != ReasonRequestOverflowDrift {
		t.Fatalf("expected cache overflow drift, got %v", err)
	}
}

func TestValidateCacheUsageBaselineUnavailable(t *testing.T) {
	if err := ValidateCacheUsageBaselineUnavailable(RequestProjection{}); err != nil {
		t.Fatalf("zero-value cache usage must satisfy the baseline: %v", err)
	}

	cases := []struct {
		name       string
		projection RequestProjection
	}{
		{
			name:       "claims availability",
			projection: RequestProjection{CacheUsage: CacheUsageProjection{Available: true}},
		},
		{
			name:       "fabricates read tokens",
			projection: RequestProjection{CacheUsage: CacheUsageProjection{ReadTokens: 3}},
		},
		{
			name:       "fabricates write tokens",
			projection: RequestProjection{CacheUsage: CacheUsageProjection{WriteTokens: 3}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCacheUsageBaselineUnavailable(tc.projection)
			var classified *RequestProjectionError
			if !errors.As(err, &classified) {
				t.Fatalf("expected classified error, got %v", err)
			}
			if classified.Code != ReasonCacheUsageProjectionDrift {
				t.Fatalf("unexpected code %q", classified.Code)
			}
		})
	}
}

func TestRequestProjectionBounds(t *testing.T) {
	cases := []struct {
		name       string
		projection RequestProjection
		wantCode   string
	}{
		{
			name:       "parts overflow",
			projection: RequestProjection{Parts: repeatString(RequestPartUserText, MaxRequestParts+1)},
			wantCode:   ReasonRequestOverflowDrift,
		},
		{
			name:       "roles overflow",
			projection: RequestProjection{Roles: repeatString("user", MaxRequestRoles+1)},
			wantCode:   ReasonRequestOverflowDrift,
		},
		{
			name:       "tool order overflow",
			projection: RequestProjection{ToolOrder: repeatString("tool", MaxRequestTools+1)},
			wantCode:   ReasonRequestOverflowDrift,
		},
		{
			name:       "digest overflow",
			projection: RequestProjection{ContentDigest: strings.Repeat("a", MaxRequestIdentifier+1)},
			wantCode:   ReasonRequestOverflowDrift,
		},
		{
			name:       "unsupported part",
			projection: RequestProjection{Parts: []string{"provider_native_block"}},
			wantCode:   ReasonRequestSchemaDrift,
		},
		{
			name:       "unsupported role",
			projection: RequestProjection{Roles: []string{"developer"}},
			wantCode:   ReasonRequestSchemaDrift,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRequestProjection(tc.projection)
			var classified *RequestProjectionError
			if !errors.As(err, &classified) {
				t.Fatalf("expected classified error, got %v", err)
			}
			if classified.Code != tc.wantCode {
				t.Fatalf("got code %q want %q", classified.Code, tc.wantCode)
			}
		})
	}
}

func TestClassifyRequestGaps(t *testing.T) {
	cases := []struct {
		name     string
		source   RequestFacts
		observed RequestProjection
		want     []string
	}{
		{
			name:     "no gap when projection preserves facts",
			source:   RequestFacts{Roles: []string{"system", "user"}, ToolOrder: []string{"read_file"}, Capabilities: []string{"tool_call"}},
			observed: RequestProjection{Roles: []string{"user", "system"}, ToolOrder: []string{"read_file"}, Capabilities: []string{"tool_call"}},
			want:     nil,
		},
		{
			name:     "role loss",
			source:   RequestFacts{Roles: []string{"system", "user"}},
			observed: RequestProjection{Roles: []string{"user"}},
			want:     []string{ReasonRequestRoleProjectionDrift},
		},
		{
			name:     "tool result not native",
			source:   RequestFacts{Roles: []string{"user"}, ToolResults: []RequestToolFact{{CallID: "c1", Name: "read_file"}}},
			observed: RequestProjection{Roles: []string{"user"}},
			want:     []string{ReasonRequestToolResultNativeDrift},
		},
		{
			name:     "stable prefix not projected",
			source:   RequestFacts{Roles: []string{"user"}, PrefixVersion: "v3"},
			observed: RequestProjection{Roles: []string{"user"}},
			want:     []string{ReasonRequestStablePrefixDrift},
		},
		{
			name:     "tool order drift",
			source:   RequestFacts{Roles: []string{"user"}, ToolOrder: []string{"read_file", "write_file"}},
			observed: RequestProjection{Roles: []string{"user"}, ToolOrder: []string{"write_file", "read_file"}},
			want:     []string{ReasonRequestToolOrderDrift},
		},
		{
			name:     "capability not projected",
			source:   RequestFacts{Roles: []string{"user"}, Capabilities: []string{"tool_call", "streaming"}},
			observed: RequestProjection{Roles: []string{"user"}},
			want:     []string{ReasonRequestCapabilityProjectionDrift},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyRequestGaps(tc.source, tc.observed)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestParseRequestProjectionFixtureRejectsUnsupportedVersion(t *testing.T) {
	raw := []byte(`{"version":"provider_request_projection.v0","cases":[{"name":"c","provider":"openai","mode":"run","observed":{"roles":["user"]},"first_digest":"x","replay_digest":"x"}]}`)
	_, err := ParseRequestProjectionFixtureJSON(raw)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestSchemaDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestProjectRequestNativePreservesRoleOrderInputAndToolFacts(t *testing.T) {
	source := RequestFacts{
		Roles:      []string{"system", "user", "assistant"},
		InputBytes: len("final instruction"),
		ToolOrder:  []string{"read_file"},
		ToolResults: []RequestToolFact{{
			CallID: "call-1",
			Name:   "read_file",
		}},
	}
	projected := ProjectRequestNative(source)
	if !reflect.DeepEqual(projected.Roles, source.Roles) {
		t.Fatalf("roles = %v, want %v", projected.Roles, source.Roles)
	}
	wantParts := []string{RequestPartSystemText, RequestPartUserText, RequestPartAssistantText, RequestPartUserText, RequestPartToolResultNative}
	if !reflect.DeepEqual(projected.Parts, wantParts) {
		t.Fatalf("parts = %v, want %v", projected.Parts, wantParts)
	}
	if !projected.ToolResultNative || !projected.ToolResultCorrelated || !reflect.DeepEqual(projected.ToolOrder, source.ToolOrder) {
		t.Fatalf("native tool projection = %#v", projected)
	}
}

func TestParseRequestProjectionFixtureRejectsOversizedPayload(t *testing.T) {
	raw := make([]byte, MaxRequestBytes+1)
	for i := range raw {
		raw[i] = ' '
	}
	_, err := ParseRequestProjectionFixtureJSON(raw)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestOverflowDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestParseRequestProjectionFixtureRejectsEmptyCases(t *testing.T) {
	raw := []byte(`{"version":"` + FixtureVersionRequestV1 + `","cases":[]}`)
	_, err := ParseRequestProjectionFixtureJSON(raw)
	if err == nil {
		t.Fatal("expected error for empty cases")
	}
}

func TestValidateRequestProjectionCaseUndeclaredGapFails(t *testing.T) {
	observed := RequestProjection{Roles: []string{"user"}, Parts: []string{RequestPartUserText}}
	c := validCaseObserved(t, observed)
	c.Source = RequestFacts{Roles: []string{"system", "user"}}
	err := ValidateRequestProjectionCase(c)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestRoleProjectionDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestValidateRequestProjectionCaseDeclaredGapNoLongerReproduces(t *testing.T) {
	observed := RequestProjection{Roles: []string{"user"}, Parts: []string{RequestPartUserText}}
	c := validCaseObserved(t, observed)
	c.Source = RequestFacts{Roles: []string{"system", "user"}}
	c.DeclaredGaps = []string{ReasonRequestRoleProjectionDrift}

	// The gap reproduces while roles are lost.
	if err := ValidateRequestProjectionCase(c); err != nil {
		t.Fatalf("declared gap must reproduce: %v", err)
	}

	// Once the projection preserves roles, the declared gap must be removed
	// explicitly instead of failing silently.
	c.Observed = RequestProjection{Roles: []string{"system", "user"}, Parts: []string{RequestPartSystemText, RequestPartUserText}}
	digest := mustDigest(t, c.Observed)
	c.FirstDigest = digest
	c.ReplayDigest = digest
	err := ValidateRequestProjectionCase(c)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestContractDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestValidateRequestProjectionCaseDigestMustMatch(t *testing.T) {
	observed := RequestProjection{Roles: []string{"user"}, Parts: []string{RequestPartUserText}}
	c := validCaseObserved(t, observed)
	c.FirstDigest = strings.Repeat("0", 64)
	c.ReplayDigest = c.FirstDigest
	err := ValidateRequestProjectionCase(c)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestSchemaDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestValidateRequestProjectionCaseIdempotencyDigestsMustMatch(t *testing.T) {
	observed := RequestProjection{Roles: []string{"user"}, Parts: []string{RequestPartUserText}}
	c := validCaseObserved(t, observed)
	c.ReplayDigest = strings.Repeat("0", 64)
	err := ValidateRequestProjectionCase(c)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestValidateRequestProjectionCaseRunStreamParity(t *testing.T) {
	run := RequestProjection{Roles: []string{"user"}, Parts: []string{RequestPartUserText}, ContentDigest: "abc"}
	stream := RequestProjection{Roles: []string{"user"}, Parts: []string{RequestPartUserText}, ContentDigest: "def"}
	c := validCaseObserved(t, run)
	c.Run = &run
	c.Stream = &stream
	err := ValidateRequestProjectionCase(c)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestRunStreamParityDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestValidateRequestProjectionCaseUnsupportedDeclaredGap(t *testing.T) {
	observed := RequestProjection{Roles: []string{"user"}, Parts: []string{RequestPartUserText}}
	c := validCaseObserved(t, observed)
	c.DeclaredGaps = []string{"provider_made_up_drift"}
	err := ValidateRequestProjectionCase(c)
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestSchemaDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func TestRequestProjectionRejectsNegativeTotalBytes(t *testing.T) {
	err := ValidateRequestProjection(RequestProjection{TotalBytes: -1})
	var classified *RequestProjectionError
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %v", err)
	}
	if classified.Code != ReasonRequestSchemaDrift {
		t.Fatalf("unexpected code %q", classified.Code)
	}
}

func repeatString(value string, count int) []string {
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, fmt.Sprintf("%s_%d", value, i))
	}
	return out
}
