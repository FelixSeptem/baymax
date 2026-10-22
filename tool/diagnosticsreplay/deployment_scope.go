package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

const DeploymentScopeReplayVersion = "deployment_scope_replay.v1"

type DeploymentScopeReplayFixture struct {
	Version string                      `json:"version"`
	Cases   []DeploymentScopeReplayCase `json:"cases"`
}

type DeploymentScopeReplayCase struct {
	CaseID            string `json:"case_id"`
	AdvertisedProfile string `json:"advertised_profile"`
	RequestedProfile  string `json:"requested_profile"`
	ExpectedOutcome   string `json:"expected_outcome"`
	SideEffects       int    `json:"side_effects"`
	ExpectedReason    string `json:"expected_reason,omitempty"`
}

type DeploymentScopeReplayResult struct {
	CaseID       string `json:"case_id"`
	Outcome      string `json:"outcome"`
	Reason       string `json:"reason,omitempty"`
	SideEffects  int    `json:"side_effects"`
	ReplayDigest string `json:"replay_digest"`
}

// ReplayDeploymentScopeFixtureJSON is reference-only. It validates recorded
// owner evidence and never constructs a store, advances a cursor, or claims a
// checkpoint.
func ReplayDeploymentScopeFixtureJSON(raw []byte) ([]DeploymentScopeReplayResult, error) {
	var fixture DeploymentScopeReplayFixture
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil || fixture.Version != DeploymentScopeReplayVersion || len(fixture.Cases) == 0 {
		return nil, fmt.Errorf("deployment_scope.replay_schema_drift")
	}
	results := make([]DeploymentScopeReplayResult, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if strings.TrimSpace(item.CaseID) == "" || item.SideEffects != 0 {
			return nil, fmt.Errorf("deployment_scope.side_effect_before_rejection")
		}
		scope := types.SameHostSingleWriterScope("fixture-owner", "fixture:deployment-scope.v1")
		request := types.DeploymentRequest{}
		if item.RequestedProfile != "" {
			request.Profile = types.DeploymentProfile(item.RequestedProfile)
		}
		err := scope.Check(request)
		outcome, reason := "admitted", ""
		if err != nil {
			outcome = "unsupported_capability"
			reason = "unsupported_deployment_scope"
		}
		if outcome != item.ExpectedOutcome || (item.ExpectedReason != "" && reason != item.ExpectedReason) {
			return nil, fmt.Errorf("deployment_scope.replay_drift:%s", item.CaseID)
		}
		results = append(results, DeploymentScopeReplayResult{CaseID: item.CaseID, Outcome: outcome, Reason: reason, ReplayDigest: item.AdvertisedProfile + "|" + item.RequestedProfile + "|" + outcome})
	}
	return results, nil
}
