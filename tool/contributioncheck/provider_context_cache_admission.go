package contributioncheck

import (
	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

// ProviderContextCacheAdmissionSummary is a contribution-time, review-only
// routing result. It contains no mutation hook and no provider SDK value.
type ProviderContextCacheAdmissionSummary struct {
	Version string                              `json:"version"`
	Cases   []ProviderContextCacheAdmissionCase `json:"cases"`
}

type ProviderContextCacheAdmissionCase struct {
	CaseID     string   `json:"case_id"`
	Provider   string   `json:"provider"`
	Verdict    string   `json:"verdict"`
	Reasons    []string `json:"reasons,omitempty"`
	OwnerRoute string   `json:"owner_route,omitempty"`
	ReviewOnly bool     `json:"review_only"`
	Digest     string   `json:"digest"`
}

// AdmitProviderContextCacheEvidence replays a bounded evidence fixture and
// routes confirmed drift to the existing provider owner. The result is
// intentionally advisory: callers receive no method capable of changing code,
// fixtures, runtime configuration, or provider behavior.
func AdmitProviderContextCacheEvidence(raw []byte) (ProviderContextCacheAdmissionSummary, error) {
	replayed, err := diagnosticsreplay.ReplayProviderContextCacheEvidenceJSON(raw)
	if err != nil {
		return ProviderContextCacheAdmissionSummary{}, err
	}
	result := ProviderContextCacheAdmissionSummary{Version: replayed.Version, Cases: make([]ProviderContextCacheAdmissionCase, 0, len(replayed.Cases))}
	for _, item := range replayed.Cases {
		admission := ProviderContextCacheAdmissionCase{
			CaseID: item.CaseID, Provider: item.Provider, Verdict: item.Verdict,
			Reasons: append([]string(nil), item.Reasons...), Digest: item.Digest,
		}
		if item.Verdict == "drift-confirmed" {
			admission.OwnerRoute = "model/" + item.Provider
			admission.ReviewOnly = true
		}
		result.Cases = append(result.Cases, admission)
	}
	return result, nil
}
