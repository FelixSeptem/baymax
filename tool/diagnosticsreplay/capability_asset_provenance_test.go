package diagnosticsreplay

import (
	"encoding/json"
	"strings"
	"testing"
)

func validCapabilityAsset() CapabilityAsset {
	return CapabilityAsset{
		Kind:       "skill",
		Identity:   "skill.search",
		Version:    "1.2.0",
		Digest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Owner:      "team.search",
		Scope:      "session/project",
		Source:     "host-manifest",
		VerifiedAt: "2026-09-26T00:00:00Z",
		Capabilities: []string{
			"search",
			"summarize",
		},
		Dependencies: []CapabilityAssetReference{{Identity: "tool.fetch", Kind: "tool", Scope: "session/project", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		Consumers:    []CapabilityAssetReference{{Identity: "agent.research", Kind: "agent", Scope: "session/project"}},
	}
}

func TestReplayCapabilityAssetProvenanceNormalizesReferencesAndUnknownFields(t *testing.T) {
	asset := validCapabilityAsset()
	asset.Dependencies = []CapabilityAssetReference{asset.Dependencies[0], asset.Dependencies[0]}
	fixture := CapabilityAssetProvenanceFixture{Version: CapabilityAssetProvenanceVersion, Cases: []CapabilityAssetProvenanceCase{{CaseID: "valid", Asset: asset}}}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	first, err := ReplayCapabilityAssetProvenanceJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := ReplayCapabilityAssetProvenanceJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if len(first.Cases) != 1 || !first.Cases[0].Idempotent || first.Cases[0].Digest != first.Cases[0].ReplayDigest {
		t.Fatalf("replay result = %#v", first)
	}
	if first.Cases[0].Digest != second.Cases[0].Digest || len(first.Cases[0].Projection.Dependencies) != 1 {
		t.Fatalf("normalization is not stable: first=%#v second=%#v", first, second)
	}
	withUnknown := strings.Replace(string(raw), `"version":"capability_asset_provenance.v1"`, `"version":"capability_asset_provenance.v1","future_field":"ignored"`, 1)
	if _, err := ReplayCapabilityAssetProvenanceJSON([]byte(withUnknown)); err != nil {
		t.Fatalf("unknown additive field rejected: %v", err)
	}
}

func TestReplayCapabilityAssetProvenanceRejectsMissingRequiredAndSensitiveInput(t *testing.T) {
	missing := CapabilityAssetProvenanceFixture{Version: CapabilityAssetProvenanceVersion, Cases: []CapabilityAssetProvenanceCase{{CaseID: "missing", Asset: CapabilityAsset{Kind: "skill"}}}}
	raw, _ := json.Marshal(missing)
	if _, err := ReplayCapabilityAssetProvenanceJSON(raw); err == nil || !strings.Contains(err.Error(), ReasonCodeCapabilityAssetSchemaDrift) {
		t.Fatalf("missing field error = %v", err)
	}
	if _, err := ReplayCapabilityAssetProvenanceJSON([]byte(`{"version":"capability_asset_provenance.v1","cases":[{"case_id":"secret","asset":{"kind":"skill","identity":"skill.x","owner":"team","scope":"session","credential":"secret"}}]}`)); err == nil || !strings.Contains(err.Error(), ReasonCodeCapabilityAssetPrivacyOrBoundViolation) {
		t.Fatalf("sensitive field error = %v", err)
	}
}

func TestReplayCapabilityAssetProvenanceClassifiesConflictDriftImpactReplacementAndParity(t *testing.T) {
	asset := validCapabilityAsset()
	conflict := asset
	conflict.Dependencies = append(append([]CapabilityAssetReference(nil), asset.Dependencies...), CapabilityAssetReference{Kind: "tool", Identity: "tool.fetch", Scope: asset.Scope, Digest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"})
	conflictFixture := CapabilityAssetProvenanceFixture{Version: CapabilityAssetProvenanceVersion, Cases: []CapabilityAssetProvenanceCase{{CaseID: "conflict", Asset: conflict}}}
	conflictRaw, _ := json.Marshal(conflictFixture)
	if _, err := ReplayCapabilityAssetProvenanceJSON(conflictRaw); err == nil || !strings.Contains(err.Error(), ReasonCodeCapabilityAssetDuplicateConflict) {
		t.Fatalf("conflicting duplicate error = %v", err)
	}

	observed := asset
	observed.Version = "1.3.0"
	replacement := asset
	replacement.Version = "2.0.0"
	fixture := CapabilityAssetProvenanceFixture{Version: CapabilityAssetProvenanceVersion, Cases: []CapabilityAssetProvenanceCase{
		{CaseID: "drift", Asset: asset, Observed: &observed},
		{CaseID: "withdrawal-incomplete", Asset: CapabilityAsset{Kind: asset.Kind, Identity: asset.Identity, Owner: asset.Owner, Scope: asset.Scope, Source: asset.Source}, Withdraw: true},
		{CaseID: "replacement", Asset: asset, Replacement: &replacement},
		{CaseID: "parity", Asset: asset, Run: &asset, Stream: &asset},
	}}
	raw, _ := json.Marshal(fixture)
	result, err := ReplayCapabilityAssetProvenanceJSON(raw)
	if err != nil {
		t.Fatalf("boundary replay: %v", err)
	}
	byID := make(map[string]CapabilityAssetProvenanceCaseResult, len(result.Cases))
	for _, item := range result.Cases {
		byID[item.CaseID] = item
	}
	if !containsCapabilityString(byID["drift"].Findings, ReasonCodeCapabilityAssetProvenanceDrift) {
		t.Fatalf("drift findings = %#v", byID["drift"].Findings)
	}
	if !containsCapabilityString(byID["withdrawal-incomplete"].Findings, ReasonCodeCapabilityAssetImpactIncomplete) || byID["withdrawal-incomplete"].ImpactComplete {
		t.Fatalf("withdrawal result = %#v", byID["withdrawal-incomplete"])
	}
	if !containsCapabilityString(byID["replacement"].Findings, ReasonCodeCapabilityAssetReplacementCompatible) {
		t.Fatalf("replacement findings = %#v", byID["replacement"].Findings)
	}
	if byID["parity"].RunStreamParity != "equivalent" {
		t.Fatalf("parity = %#v", byID["parity"])
	}
}

func TestReplayCapabilityAssetProvenanceEnforcesReplacementVersionRangeAndDependents(t *testing.T) {
	asset := validCapabilityAsset()
	asset.Version = "1.0.0"
	asset.VersionRange = ">=1.0.0 <2.0.0"
	asset.Dependents = []CapabilityAssetReference{{Kind: "workflow", Identity: "workflow.index", Scope: asset.Scope}}
	replacement := asset
	replacement.Version = "2.1.0"
	fixture := CapabilityAssetProvenanceFixture{
		Version: CapabilityAssetProvenanceVersion,
		Cases:   []CapabilityAssetProvenanceCase{{CaseID: "range-and-dependents", Asset: asset, Withdraw: true, Replacement: &replacement}},
	}
	raw, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ReplayCapabilityAssetProvenanceJSON(raw)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	item := result.Cases[0]
	if !containsCapabilityString(item.Findings, ReasonCodeCapabilityAssetReplacementIncompatible) {
		t.Fatalf("replacement findings = %#v", item.Findings)
	}
	if !item.ImpactComplete || len(item.Impact) != 2 || item.Impact[0].Identity != "agent.research" || item.Impact[1].Identity != "workflow.index" {
		t.Fatalf("withdrawal impact = %#v complete=%v", item.Impact, item.ImpactComplete)
	}
}
