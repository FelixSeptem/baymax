package memory

import (
	"sort"
	"strings"
)

const (
	ProfileMem0       = "mem0"
	ProfileZep        = "zep"
	ProfileOpenViking = "openviking"
	ProfileGeneric    = "generic"
)

type Profile struct {
	ID               string   `json:"id"`
	Provider         string   `json:"provider"`
	ContractVersion  string   `json:"contract_version"`
	RequiredOps      []string `json:"required_operations"`
	OptionalOps      []string `json:"optional_operations,omitempty"`
	ErrorTaxonomyRef string   `json:"error_taxonomy_ref,omitempty"`
}

var defaultProfilePack = map[string]Profile{
	ProfileMem0: {
		ID:               ProfileMem0,
		Provider:         "mem0",
		ContractVersion:  ContractVersionMemoryV1,
		RequiredOps:      []string{OperationQuery, OperationUpsert, OperationDelete},
		OptionalOps:      []string{"metadata_filter", "ttl"},
		ErrorTaxonomyRef: "memory.v1",
	},
	ProfileZep: {
		ID:               ProfileZep,
		Provider:         "zep",
		ContractVersion:  ContractVersionMemoryV1,
		RequiredOps:      []string{OperationQuery, OperationUpsert, OperationDelete},
		OptionalOps:      []string{"metadata_filter", "namespace_scope"},
		ErrorTaxonomyRef: "memory.v1",
	},
	ProfileOpenViking: {
		ID:               ProfileOpenViking,
		Provider:         "openviking",
		ContractVersion:  ContractVersionMemoryV1,
		RequiredOps:      []string{OperationQuery, OperationUpsert, OperationDelete},
		OptionalOps:      []string{"metadata_filter"},
		ErrorTaxonomyRef: "memory.v1",
	},
	ProfileGeneric: {
		ID:               ProfileGeneric,
		Provider:         "generic",
		ContractVersion:  ContractVersionMemoryV1,
		RequiredOps:      []string{OperationQuery, OperationUpsert, OperationDelete},
		OptionalOps:      []string{"metadata_filter"},
		ErrorTaxonomyRef: "memory.v1",
	},
}

func SupportedProfiles() []string {
	out := make([]string, 0, len(defaultProfilePack))
	for key := range defaultProfilePack {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func ResolveProfile(id string) (Profile, error) {
	normalized := strings.ToLower(strings.TrimSpace(id))
	if normalized == "" {
		return Profile{}, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeProfileUnknown,
			Layer:     LayerSemantic,
			Message:   "memory profile is required",
		}
	}
	profile, ok := defaultProfilePack[normalized]
	if !ok {
		return Profile{}, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeProfileUnknown,
			Layer:     LayerSemantic,
			Message:   "memory profile is unsupported",
			Raw: map[string]any{
				"profile":   normalized,
				"supported": SupportedProfiles(),
			},
		}
	}
	out := profile
	out.RequiredOps = append([]string(nil), profile.RequiredOps...)
	out.OptionalOps = append([]string(nil), profile.OptionalOps...)
	return out, nil
}
