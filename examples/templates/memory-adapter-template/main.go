package main

import (
	"encoding/json"
	"fmt"

	memorypkg "github.com/FelixSeptem/baymax/memory"
)

type onboardingTemplate struct {
	ProfileID         string         `json:"profile_id"`
	ConformanceCaseID string         `json:"conformance_case_id"`
	Runtime           map[string]any `json:"runtime"`
	Manifest          map[string]any `json:"manifest"`
}

func main() {
	profiles := []string{
		memorypkg.ProfileMem0,
		memorypkg.ProfileZep,
		memorypkg.ProfileOpenViking,
		memorypkg.ProfileGeneric,
	}
	for _, profileID := range profiles {
		template, err := externalSPITemplate(profileID)
		if err != nil {
			panic(err)
		}
		printTemplate(template)
	}
	printTemplate(builtinFilesystemTemplate())
}

func externalSPITemplate(profileID string) (onboardingTemplate, error) {
	profile, err := memorypkg.ResolveProfile(profileID)
	if err != nil {
		return onboardingTemplate{}, err
	}
	return onboardingTemplate{
		ProfileID:         profile.ID,
		ConformanceCaseID: "memory-" + profile.ID + "-matrix",
		Runtime: map[string]any{
			"memory": map[string]any{
				"mode": "external_spi",
				"external": map[string]any{
					"provider":         profile.Provider,
					"profile":          profile.ID,
					"contract_version": profile.ContractVersion,
				},
				"fallback": map[string]any{
					"policy": memorypkg.FallbackPolicyDegradeToBuiltin,
				},
			},
		},
		Manifest: map[string]any{
			"memory": map[string]any{
				"provider":         profile.Provider,
				"profile":          profile.ID,
				"contract_version": profile.ContractVersion,
				"operations": map[string]any{
					"required": profile.RequiredOps,
					"optional": profile.OptionalOps,
				},
				"fallback": map[string]any{
					"supported": true,
				},
			},
		},
	}, nil
}

func builtinFilesystemTemplate() onboardingTemplate {
	return onboardingTemplate{
		ProfileID:         "builtin_filesystem",
		ConformanceCaseID: "memory-builtin-filesystem-switch",
		Runtime: map[string]any{
			"memory": map[string]any{
				"mode": "builtin_filesystem",
				"builtin": map[string]any{
					"root_dir": ".baymax/memory",
					"compaction": map[string]any{
						"enabled":       true,
						"min_ops":       128,
						"max_wal_bytes": 1048576,
					},
				},
				"fallback": map[string]any{
					"policy": memorypkg.FallbackPolicyFailFast,
				},
			},
		},
		Manifest: map[string]any{
			"memory": map[string]any{
				"provider":         "builtin_filesystem",
				"profile":          "builtin",
				"contract_version": memorypkg.ContractVersionMemoryV1,
				"operations": map[string]any{
					"required": []string{
						memorypkg.OperationQuery,
						memorypkg.OperationUpsert,
						memorypkg.OperationDelete,
					},
					"optional": []string{},
				},
				"fallback": map[string]any{
					"supported": true,
				},
			},
		},
	}
}

func printTemplate(template onboardingTemplate) {
	raw, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
}
