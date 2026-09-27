package diagnosticsreplay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

const CapabilityAssetProvenanceVersion = "capability_asset_provenance.v1"

const (
	ReasonCodeCapabilityAssetSchemaDrift             = "capability_asset_schema_drift"
	ReasonCodeCapabilityAssetUnknownVersion          = "capability_asset_unknown_version"
	ReasonCodeCapabilityAssetPrivacyOrBoundViolation = "capability_asset_privacy_or_bound_violation"
	ReasonCodeCapabilityAssetReferenceIntegrity      = "capability_asset_reference_integrity"
	ReasonCodeCapabilityAssetScopeViolation          = "capability_asset_scope_violation"
	ReasonCodeCapabilityAssetProvenanceDrift         = "capability_asset_provenance_drift"
	ReasonCodeCapabilityAssetDuplicateConflict       = "capability_asset_duplicate_conflict"
	ReasonCodeCapabilityAssetImpactIncomplete        = "capability_asset_impact_incomplete"
	ReasonCodeCapabilityAssetImpactConflict          = "capability_asset_impact_conflict"
	ReasonCodeCapabilityAssetReplacementIncompatible = "capability_asset_replacement_incompatible"
	ReasonCodeCapabilityAssetReplacementCompatible   = "capability_asset_replacement_compatible"
	ReasonCodeCapabilityAssetRunStreamParityDrift    = "capability_asset_run_stream_parity_drift"
)

const (
	capabilityAssetMaxString       = 256
	capabilityAssetMaxItems        = 64
	capabilityAssetMaxFixtureBytes = 1 << 20
)

type CapabilityAssetProvenanceFixture struct {
	Version string                          `json:"version"`
	Cases   []CapabilityAssetProvenanceCase `json:"cases"`
}

type CapabilityAssetProvenanceCase struct {
	CaseID      string           `json:"case_id"`
	Asset       CapabilityAsset  `json:"asset"`
	Observed    *CapabilityAsset `json:"observed,omitempty"`
	Withdraw    bool             `json:"withdraw,omitempty"`
	Replacement *CapabilityAsset `json:"replacement,omitempty"`
	Run         *CapabilityAsset `json:"run,omitempty"`
	Stream      *CapabilityAsset `json:"stream,omitempty"`
}

type CapabilityAsset struct {
	Kind              string                     `json:"kind"`
	Identity          string                     `json:"identity"`
	Version           string                     `json:"version,omitempty"`
	VersionRange      string                     `json:"version_range,omitempty"`
	Digest            string                     `json:"digest,omitempty"`
	Owner             string                     `json:"owner"`
	Scope             string                     `json:"scope"`
	Source            string                     `json:"source"`
	VerifiedAt        string                     `json:"verified_at,omitempty"`
	Capabilities      []string                   `json:"capabilities,omitempty"`
	CompatibleDigests []string                   `json:"compatible_digests,omitempty"`
	Dependencies      []CapabilityAssetReference `json:"dependencies,omitempty"`
	Consumers         []CapabilityAssetReference `json:"consumers,omitempty"`
	Dependents        []CapabilityAssetReference `json:"dependents,omitempty"`
}

type CapabilityAssetReference struct {
	Kind     string `json:"kind"`
	Identity string `json:"identity"`
	Scope    string `json:"scope"`
	Digest   string `json:"digest,omitempty"`
}

type CapabilityAssetProvenanceResult struct {
	Version string                                `json:"version"`
	Cases   []CapabilityAssetProvenanceCaseResult `json:"cases"`
}

type CapabilityAssetProvenanceCaseResult struct {
	CaseID          string                     `json:"case_id"`
	Projection      CapabilityAsset            `json:"projection"`
	Findings        []string                   `json:"findings,omitempty"`
	Impact          []CapabilityAssetReference `json:"impact,omitempty"`
	ImpactComplete  bool                       `json:"impact_complete,omitempty"`
	Digest          string                     `json:"digest"`
	ReplayDigest    string                     `json:"replay_digest"`
	Idempotent      bool                       `json:"idempotent"`
	RunStreamParity string                     `json:"run_stream_parity,omitempty"`
}

// ReplayCapabilityAssetProvenanceJSON evaluates a bounded, offline fixture twice.
func ReplayCapabilityAssetProvenanceJSON(raw []byte) (CapabilityAssetProvenanceResult, error) {
	if len(raw) > capabilityAssetMaxFixtureBytes {
		return CapabilityAssetProvenanceResult{}, capabilityAssetError(ReasonCodeCapabilityAssetPrivacyOrBoundViolation, "fixture exceeds bound")
	}
	if err := scanCapabilityAssetJSON(raw); err != nil {
		return CapabilityAssetProvenanceResult{}, err
	}
	var fixture CapabilityAssetProvenanceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return CapabilityAssetProvenanceResult{}, capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, err.Error())
	}
	if fixture.Version != CapabilityAssetProvenanceVersion {
		return CapabilityAssetProvenanceResult{}, capabilityAssetError(ReasonCodeCapabilityAssetUnknownVersion, fixture.Version)
	}
	if len(fixture.Cases) == 0 || len(fixture.Cases) > capabilityAssetMaxItems {
		return CapabilityAssetProvenanceResult{}, capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, "cases must contain between one and 64 items")
	}
	result := CapabilityAssetProvenanceResult{Version: fixture.Version, Cases: make([]CapabilityAssetProvenanceCaseResult, 0, len(fixture.Cases))}
	for _, item := range fixture.Cases {
		first, err := replayCapabilityAssetCase(item)
		if err != nil {
			return CapabilityAssetProvenanceResult{}, err
		}
		second, err := replayCapabilityAssetCase(item)
		if err != nil {
			return CapabilityAssetProvenanceResult{}, err
		}
		if first.Digest != second.Digest {
			return CapabilityAssetProvenanceResult{}, capabilityAssetError(ReasonCodeCapabilityAssetProvenanceDrift, "replay digest changed")
		}
		first.ReplayDigest = second.Digest
		first.Idempotent = first.Digest == first.ReplayDigest
		result.Cases = append(result.Cases, first)
	}
	return result, nil
}

func replayCapabilityAssetCase(input CapabilityAssetProvenanceCase) (CapabilityAssetProvenanceCaseResult, error) {
	caseID := strings.TrimSpace(input.CaseID)
	if caseID == "" {
		return CapabilityAssetProvenanceCaseResult{}, capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, "case_id is required")
	}
	if err := validateCapabilityAsset(input.Asset); err != nil {
		return CapabilityAssetProvenanceCaseResult{}, err
	}
	projection, err := normalizeCapabilityAsset(input.Asset)
	if err != nil {
		return CapabilityAssetProvenanceCaseResult{}, err
	}
	findings := make([]string, 0)
	if input.Observed != nil {
		if err := validateCapabilityAsset(*input.Observed); err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		observed, err := normalizeCapabilityAsset(*input.Observed)
		if err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		if !reflect.DeepEqual(projection, observed) {
			findings = append(findings, ReasonCodeCapabilityAssetProvenanceDrift)
		}
	}
	if input.Withdraw {
		if input.Asset.Consumers == nil {
			findings = append(findings, ReasonCodeCapabilityAssetImpactIncomplete)
		} else {
			for _, ref := range projection.Consumers {
				if ref.Scope != projection.Scope {
					findings = append(findings, ReasonCodeCapabilityAssetImpactConflict)
					continue
				}
			}
		}
	}
	if input.Replacement != nil {
		if err := validateCapabilityAsset(*input.Replacement); err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		replacement, err := normalizeCapabilityAsset(*input.Replacement)
		if err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		if replacementIsCompatible(projection, replacement) {
			findings = append(findings, ReasonCodeCapabilityAssetReplacementCompatible)
		} else {
			findings = append(findings, ReasonCodeCapabilityAssetReplacementIncompatible)
		}
	}
	parity := "not_applicable"
	if input.Run != nil || input.Stream != nil {
		if input.Run == nil || input.Stream == nil {
			return CapabilityAssetProvenanceCaseResult{}, capabilityAssetError(ReasonCodeCapabilityAssetRunStreamParityDrift, "both run and stream projections are required")
		}
		if err := validateCapabilityAsset(*input.Run); err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		if err := validateCapabilityAsset(*input.Stream); err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		run, err := normalizeCapabilityAsset(*input.Run)
		if err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		stream, err := normalizeCapabilityAsset(*input.Stream)
		if err != nil {
			return CapabilityAssetProvenanceCaseResult{}, err
		}
		if !reflect.DeepEqual(run, stream) {
			findings = append(findings, ReasonCodeCapabilityAssetRunStreamParityDrift)
			parity = "drift"
		} else {
			parity = "equivalent"
		}
	}
	sort.Strings(findings)
	findings = uniqueStrings(findings)
	impact, err := normalizeCapabilityReferences(append(append([]CapabilityAssetReference(nil), projection.Consumers...), projection.Dependents...))
	if err != nil {
		return CapabilityAssetProvenanceCaseResult{}, err
	}
	impactComplete := input.Withdraw && input.Asset.Consumers != nil && input.Asset.Dependents != nil && !containsCapabilityString(findings, ReasonCodeCapabilityAssetImpactConflict)
	digest := digestCapabilityAssetCase(projection, findings, impact, impactComplete, parity)
	return CapabilityAssetProvenanceCaseResult{CaseID: caseID, Projection: projection, Findings: findings, Impact: impact, ImpactComplete: impactComplete, Digest: digest, RunStreamParity: parity}, nil
}

func validateCapabilityAsset(asset CapabilityAsset) error {
	for name, value := range map[string]string{"kind": asset.Kind, "identity": asset.Identity, "owner": asset.Owner, "scope": asset.Scope, "source": asset.Source, "version": asset.Version, "version_range": asset.VersionRange, "digest": asset.Digest, "verified_at": asset.VerifiedAt} {
		if err := validateCapabilityString(name, value); err != nil {
			return err
		}
	}
	if strings.TrimSpace(asset.Kind) == "" || strings.TrimSpace(asset.Identity) == "" || strings.TrimSpace(asset.Owner) == "" || strings.TrimSpace(asset.Scope) == "" || strings.TrimSpace(asset.Source) == "" {
		return capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, "kind, identity, owner, scope, and source are required")
	}
	if asset.Digest != "" && !validCapabilityDigest(asset.Digest) {
		return capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, "digest must be sha256 plus 64 hex characters")
	}
	if asset.VersionRange != "" && !validCapabilityVersionRange(asset.VersionRange) {
		return capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, "version_range is invalid")
	}
	if len(asset.Capabilities) > capabilityAssetMaxItems || len(asset.CompatibleDigests) > capabilityAssetMaxItems || len(asset.Dependencies) > capabilityAssetMaxItems || len(asset.Consumers) > capabilityAssetMaxItems || len(asset.Dependents) > capabilityAssetMaxItems {
		return capabilityAssetError(ReasonCodeCapabilityAssetPrivacyOrBoundViolation, "asset references exceed bound")
	}
	for _, capability := range asset.Capabilities {
		if err := validateCapabilityString("capability", capability); err != nil {
			return err
		}
	}
	for _, digest := range asset.CompatibleDigests {
		if !validCapabilityDigest(digest) {
			return capabilityAssetError(ReasonCodeCapabilityAssetReferenceIntegrity, "compatible digest is invalid")
		}
	}
	for _, refs := range map[string][]CapabilityAssetReference{"dependency": asset.Dependencies, "consumer": asset.Consumers, "dependent": asset.Dependents} {
		for _, ref := range refs {
			if err := validateCapabilityReference(ref); err != nil {
				return err
			}
			if ref.Scope != asset.Scope {
				return capabilityAssetError(ReasonCodeCapabilityAssetScopeViolation, "reference scope differs from asset scope")
			}
		}
	}
	return nil
}

func validateCapabilityReference(ref CapabilityAssetReference) error {
	if err := validateCapabilityString("reference kind", ref.Kind); err != nil {
		return err
	}
	if err := validateCapabilityString("reference identity", ref.Identity); err != nil {
		return err
	}
	if err := validateCapabilityString("reference scope", ref.Scope); err != nil {
		return err
	}
	if err := validateCapabilityString("reference digest", ref.Digest); err != nil {
		return err
	}
	if ref.Kind == "" || ref.Identity == "" || ref.Scope == "" {
		return capabilityAssetError(ReasonCodeCapabilityAssetReferenceIntegrity, "reference kind, identity, and scope are required")
	}
	if ref.Digest != "" && !validCapabilityDigest(ref.Digest) {
		return capabilityAssetError(ReasonCodeCapabilityAssetReferenceIntegrity, "reference digest is invalid")
	}
	return nil
}

func validateCapabilityString(name, value string) error {
	if len(value) > capabilityAssetMaxString {
		return capabilityAssetError(ReasonCodeCapabilityAssetPrivacyOrBoundViolation, name+" exceeds bound")
	}
	if containsCapabilitySensitiveMarker(value) {
		return capabilityAssetError(ReasonCodeCapabilityAssetPrivacyOrBoundViolation, name+" contains sensitive material")
	}
	return nil
}

func normalizeCapabilityAsset(asset CapabilityAsset) (CapabilityAsset, error) {
	asset.Kind = strings.ToLower(strings.TrimSpace(asset.Kind))
	asset.Identity = strings.TrimSpace(asset.Identity)
	asset.Version = strings.TrimSpace(asset.Version)
	asset.VersionRange = strings.TrimSpace(asset.VersionRange)
	asset.Digest = strings.ToLower(strings.TrimSpace(asset.Digest))
	asset.Owner = strings.TrimSpace(asset.Owner)
	asset.Scope = strings.TrimSpace(asset.Scope)
	asset.Source = strings.TrimSpace(asset.Source)
	asset.VerifiedAt = strings.TrimSpace(asset.VerifiedAt)
	asset.Capabilities = uniqueStringsNormalized(asset.Capabilities)
	asset.CompatibleDigests = normalizeCapabilityDigests(asset.CompatibleDigests)
	var err error
	asset.Dependencies, err = normalizeCapabilityReferences(asset.Dependencies)
	if err != nil {
		return CapabilityAsset{}, err
	}
	asset.Consumers, err = normalizeCapabilityReferences(asset.Consumers)
	if err != nil {
		return CapabilityAsset{}, err
	}
	asset.Dependents, err = normalizeCapabilityReferences(asset.Dependents)
	if err != nil {
		return CapabilityAsset{}, err
	}
	return asset, nil
}

func replacementIsCompatible(asset, replacement CapabilityAsset) bool {
	if asset.Kind != replacement.Kind || asset.Identity != replacement.Identity || asset.Scope != replacement.Scope || !reflect.DeepEqual(asset.Capabilities, replacement.Capabilities) || !reflect.DeepEqual(asset.Dependencies, replacement.Dependencies) {
		return false
	}
	if asset.VersionRange != "" && !capabilityVersionInRange(replacement.Version, asset.VersionRange) {
		return false
	}
	return len(asset.CompatibleDigests) == 0 || containsCapabilityString(asset.CompatibleDigests, replacement.Digest)
}

func normalizeCapabilityReferences(refs []CapabilityAssetReference) ([]CapabilityAssetReference, error) {
	byKey := make(map[string]CapabilityAssetReference, len(refs))
	for _, ref := range refs {
		ref.Kind = strings.ToLower(strings.TrimSpace(ref.Kind))
		ref.Identity = strings.TrimSpace(ref.Identity)
		ref.Scope = strings.TrimSpace(ref.Scope)
		ref.Digest = strings.ToLower(strings.TrimSpace(ref.Digest))
		key := ref.Kind + "\x00" + ref.Identity + "\x00" + ref.Scope
		if existing, ok := byKey[key]; ok && existing.Digest != ref.Digest {
			return nil, capabilityAssetError(ReasonCodeCapabilityAssetDuplicateConflict, "reference has conflicting digests")
		}
		byKey[key] = ref
	}
	out := make([]CapabilityAssetReference, 0, len(byKey))
	for _, ref := range byKey {
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool { return capabilityReferenceKey(out[i]) < capabilityReferenceKey(out[j]) })
	return out, nil
}

func capabilityReferenceKey(ref CapabilityAssetReference) string {
	return ref.Kind + "\x00" + ref.Identity + "\x00" + ref.Scope + "\x00" + ref.Digest
}

func digestCapabilityAssetCase(asset CapabilityAsset, findings []string, impact []CapabilityAssetReference, complete bool, parity string) string {
	canonical := struct {
		Asset    CapabilityAsset            `json:"asset"`
		Findings []string                   `json:"findings,omitempty"`
		Impact   []CapabilityAssetReference `json:"impact,omitempty"`
		Complete bool                       `json:"complete"`
		Parity   string                     `json:"parity,omitempty"`
	}{asset, findings, impact, complete, parity}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func scanCapabilityAssetJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := scanCapabilityJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, "multiple JSON values are not allowed")
	}
	return nil
}

func scanCapabilityJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return capabilityAssetError(ReasonCodeCapabilityAssetSchemaDrift, err.Error())
	}
	if delim, ok := token.(json.Delim); ok {
		switch delim {
		case '{':
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				if keyString, ok := key.(string); ok && containsCapabilitySensitiveMarker(keyString) {
					return capabilityAssetError(ReasonCodeCapabilityAssetPrivacyOrBoundViolation, "sensitive JSON field is not allowed")
				}
				if err := scanCapabilityJSONValue(decoder); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := scanCapabilityJSONValue(decoder); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		}
	}
	return nil
}

func containsCapabilitySensitiveMarker(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"credential", "password", "secret", "reasoning", "raw_payload", "transcript", "workspace_content"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func validCapabilityDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func normalizeCapabilityDigests(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return uniqueStrings(out)
}

func validCapabilityVersionRange(value string) bool {
	parts := strings.Fields(value)
	if len(parts) == 0 || len(parts) > 2 {
		return false
	}
	for _, part := range parts {
		if !hasCapabilityVersionOperator(part) {
			return false
		}
		if !validCapabilityVersion(strings.TrimLeft(part, "<>=~")) {
			return false
		}
	}
	return true
}

func hasCapabilityVersionOperator(value string) bool {
	for _, operator := range []string{">=", "<=", ">", "<", "=", "~"} {
		if strings.HasPrefix(value, operator) {
			return true
		}
	}
	return false
}

func validCapabilityVersion(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func capabilityVersionInRange(version, versionRange string) bool {
	if !validCapabilityVersion(version) || !validCapabilityVersionRange(versionRange) {
		return false
	}
	current := capabilityVersionParts(version)
	for _, constraint := range strings.Fields(versionRange) {
		op := "="
		for _, candidate := range []string{">=", "<=", ">", "<", "=", "~"} {
			if strings.HasPrefix(constraint, candidate) {
				op = candidate
				break
			}
		}
		bound := capabilityVersionParts(strings.TrimPrefix(constraint, op))

		comparison := compareCapabilityVersions(current, bound)
		valid := false
		switch op {
		case ">=":
			valid = comparison >= 0
		case "<=":
			valid = comparison <= 0
		case ">":
			valid = comparison > 0
		case "<":
			valid = comparison < 0
		case "~":
			valid = current[0] == bound[0] && current[1] == bound[1] && comparison >= 0
		default:
			valid = comparison == 0
		}
		if !valid {
			return false
		}
	}
	return true
}

func capabilityVersionParts(value string) [3]int {
	var parts [3]int
	for index, part := range strings.Split(value, ".") {
		for _, r := range part {
			parts[index] = parts[index]*10 + int(r-'0')
		}
	}
	return parts
}

func compareCapabilityVersions(left, right [3]int) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; !ok && value != "" {
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}

func uniqueStringsNormalized(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out = append(out, value)
		}
	}
	out = uniqueStrings(out)
	sort.Strings(out)
	return out
}

func containsCapabilityString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func capabilityAssetError(code, message string) error {
	return &ValidationError{Code: code, Message: fmt.Sprintf("%s: %s", code, message)}
}
