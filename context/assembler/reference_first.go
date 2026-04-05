package assembler

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	contextReferenceVersionV1 = "context_reference_first.v1"

	contextRefMissingPolicySkipAndRecord = "skip_and_record"
	contextRefMissingPolicyFailFast      = "fail_fast"
)

func resolveReferenceMissingPolicy(stagePolicy string) string {
	if strings.EqualFold(strings.TrimSpace(stagePolicy), "fail_fast") {
		return contextRefMissingPolicyFailFast
	}
	return contextRefMissingPolicySkipAndRecord
}

func discoverStage2References(
	chunks []string,
	source string,
	maxRefs int,
) (types.ContextReferenceDiscoveryPayload, map[string]string) {
	normalizedSource := normalizeReferenceSource(source)
	if maxRefs <= 0 {
		maxRefs = len(chunks)
	}
	discovery := types.ContextReferenceDiscoveryPayload{
		References:       make([]types.ContextReference, 0, len(chunks)),
		MaxRefsApplied:   maxRefs,
		DiscoverFrom:     normalizedSource,
		ReferenceVersion: contextReferenceVersionV1,
	}
	catalog := make(map[string]string, len(chunks))
	seenLocator := map[string]struct{}{}
	for _, chunk := range chunks {
		content := strings.TrimSpace(chunk)
		if content == "" {
			continue
		}
		id, locator := referenceIdentity(content, normalizedSource)
		if _, exists := seenLocator[locator]; exists {
			discovery.Deduplicated++
			continue
		}
		seenLocator[locator] = struct{}{}
		catalog[locator] = content
		if len(discovery.References) >= maxRefs {
			continue
		}
		discovery.References = append(discovery.References, types.ContextReference{
			ID:      id,
			Type:    "stage2_chunk",
			Locator: locator,
			Source:  normalizedSource,
			Summary: truncateRunes(content, 120),
			Tags:    []string{"stage2", "reference_first"},
		})
	}
	return discovery, catalog
}

func resolveSelectedStage2References(
	selected []types.ContextReference,
	catalog map[string]string,
	maxResolveTokens int,
	missingPolicy string,
) (types.ContextReferenceResolutionPayload, error) {
	payload := types.ContextReferenceResolutionPayload{
		Resolved:  []types.ContextReferenceResolutionItem{},
		Missing:   []types.ContextReference{},
		MaxTokens: maxResolveTokens,
	}
	if len(selected) == 0 {
		return payload, nil
	}
	if maxResolveTokens <= 0 {
		return payload, fmt.Errorf("runtime.context.jit.reference_first.max_resolve_tokens must be > 0")
	}

	seen := map[string]struct{}{}
	for i := range selected {
		ref := selected[i]
		if err := validateContextReference(ref); err != nil {
			return payload, fmt.Errorf("resolve_selected_refs[%d]: %w", i, err)
		}
		if _, exists := seen[ref.Locator]; exists {
			continue
		}
		seen[ref.Locator] = struct{}{}

		content, ok := catalog[ref.Locator]
		if !ok {
			if strings.EqualFold(strings.TrimSpace(missingPolicy), contextRefMissingPolicyFailFast) {
				return payload, fmt.Errorf("selected reference locator not found: %s", ref.Locator)
			}
			payload.Missing = append(payload.Missing, ref)
			continue
		}

		tokens := estimateReferenceTokens(content)
		if payload.BudgetUsedTokens+tokens > maxResolveTokens {
			payload.Truncated = true
			break
		}
		payload.Resolved = append(payload.Resolved, types.ContextReferenceResolutionItem{
			Reference: ref,
			Content:   content,
			Tokens:    tokens,
		})
		payload.BudgetUsedTokens += tokens
	}
	return payload, nil
}

func validateContextReference(ref types.ContextReference) error {
	if strings.TrimSpace(ref.ID) == "" {
		return fmt.Errorf("reference.id is required")
	}
	if strings.TrimSpace(ref.Type) == "" {
		return fmt.Errorf("reference.type is required")
	}
	locator := strings.TrimSpace(ref.Locator)
	if locator == "" {
		return fmt.Errorf("reference.locator is required")
	}
	if !strings.HasPrefix(strings.ToLower(locator), "stage2://") {
		return fmt.Errorf("reference.locator must use stage2:// scheme")
	}
	return nil
}

func referenceIdentity(content string, source string) (id string, locator string) {
	hash := sha1.Sum([]byte(content))
	digest := hex.EncodeToString(hash[:])
	id = "ref-" + digest[:8]
	locator = "stage2://" + normalizeReferenceSource(source) + "/" + digest[:16]
	return id, locator
}

func normalizeReferenceSource(source string) string {
	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == "" {
		return "stage2"
	}
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "/", "_")
	return normalized
}

func estimateReferenceTokens(content string) int {
	runes := len([]rune(strings.TrimSpace(content)))
	if runes <= 0 {
		return 0
	}
	if runes < 4 {
		return 1
	}
	return runes / 4
}

func truncateRunes(content string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= max {
		return content
	}
	return string(runes[:max])
}
