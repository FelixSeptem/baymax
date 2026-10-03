package diagnosticsreplay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const DynamicActionResumeFixtureVersion = "dynamic_action_resume.v1"

type DynamicActionResumeFixture struct {
	Version string                    `json:"version"`
	Cases   []DynamicActionResumeCase `json:"cases"`
}

type DynamicActionResumeCase struct {
	CaseID            string `json:"case_id"`
	RunID             string `json:"run_id"`
	Mode              string `json:"mode"`
	Outcome           string `json:"outcome"`
	TokenDigest       string `json:"token_digest"`
	CheckpointDigest  string `json:"checkpoint_digest"`
	IdempotencyKey    string `json:"idempotency_key"`
	ExpectedAdmission string `json:"expected_admission"`
}

type DynamicActionResumeReplay struct {
	Version string                    `json:"version"`
	Cases   []DynamicActionResumeCase `json:"cases"`
}

func ReplayDynamicActionResumeJSON(raw []byte) (DynamicActionResumeReplay, error) {
	var fixture DynamicActionResumeFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return DynamicActionResumeReplay{}, fmt.Errorf("dynamic_action_resume.invalid_json: %w", err)
	}
	if strings.TrimSpace(fixture.Version) != DynamicActionResumeFixtureVersion {
		return DynamicActionResumeReplay{}, fmt.Errorf("dynamic_action_resume.unsupported_version: %s", fixture.Version)
	}
	if len(fixture.Cases) == 0 {
		return DynamicActionResumeReplay{}, fmt.Errorf("dynamic_action_resume.empty_cases")
	}
	seen := make(map[string]struct{}, len(fixture.Cases))
	for i := range fixture.Cases {
		item := &fixture.Cases[i]
		if strings.TrimSpace(item.CaseID) == "" || strings.TrimSpace(item.RunID) == "" || strings.TrimSpace(item.Mode) == "" || strings.TrimSpace(item.Outcome) == "" || strings.TrimSpace(item.TokenDigest) == "" || strings.TrimSpace(item.CheckpointDigest) == "" || strings.TrimSpace(item.IdempotencyKey) == "" || strings.TrimSpace(item.ExpectedAdmission) == "" {
			return DynamicActionResumeReplay{}, fmt.Errorf("dynamic_action_resume.missing_field: case[%d]", i)
		}
		if _, ok := seen[item.CaseID]; ok {
			return DynamicActionResumeReplay{}, fmt.Errorf("dynamic_action_resume.duplicate_case: %s", item.CaseID)
		}
		seen[item.CaseID] = struct{}{}
		if item.Mode != "run" && item.Mode != "stream" {
			return DynamicActionResumeReplay{}, fmt.Errorf("dynamic_action_resume.invalid_mode: %s", item.Mode)
		}
		if strings.Contains(strings.ToLower(item.TokenDigest), "payload") || strings.Contains(strings.ToLower(item.TokenDigest), "secret") {
			return DynamicActionResumeReplay{}, fmt.Errorf("dynamic_action_resume.privacy_violation: %s", item.CaseID)
		}
		item.TokenDigest = stableDigest(item.TokenDigest)
		item.CheckpointDigest = stableDigest(item.CheckpointDigest)
	}
	return DynamicActionResumeReplay{Version: fixture.Version, Cases: fixture.Cases}, nil
}

func stableDigest(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
