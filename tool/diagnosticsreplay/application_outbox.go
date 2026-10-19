package diagnosticsreplay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

const ApplicationOutboxReplayVersion = "application_outbox_dispatch.v1"

type ApplicationOutboxReplayFixture struct {
	Version string                        `json:"version"`
	Cases   []ApplicationOutboxReplayCase `json:"cases"`
}
type ApplicationOutboxReplayCase struct {
	CaseID           string `json:"case_id"`
	DescriptorDigest string `json:"descriptor_digest"`
	Target           string `json:"target"`
	DispatchAccepted bool   `json:"dispatch_accepted"`
	EffectCommitted  bool   `json:"effect_committed"`
	OutboxConfirmed  bool   `json:"outbox_confirmed"`
	ExpectedReason   string `json:"expected_reason,omitempty"`
}
type ApplicationOutboxReplayResult struct {
	CaseID string
	Reason string
	Digest string
}

func ReplayApplicationOutboxFixtureJSON(raw []byte) ([]ApplicationOutboxReplayResult, error) {
	if len(raw) > 4*1024*1024 {
		return nil, errors.New("application outbox replay envelope too large")
	}
	var fixture ApplicationOutboxReplayFixture
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&fixture); err != nil {
		return nil, err
	}
	var extra any
	if dec.Decode(&extra) == nil {
		return nil, errors.New("trailing replay data")
	}
	if fixture.Version != ApplicationOutboxReplayVersion || len(fixture.Cases) > 256 {
		return nil, errors.New("invalid application outbox replay fixture")
	}
	out := make([]ApplicationOutboxReplayResult, 0, len(fixture.Cases))
	for _, c := range fixture.Cases {
		if c.CaseID == "" || c.DescriptorDigest == "" || (c.Target != "mailbox" && c.Target != "scheduler") {
			return nil, fmt.Errorf("invalid replay case")
		}
		reason := ""
		if c.DispatchAccepted && c.OutboxConfirmed {
			reason = "dispatch_outbox_conflated"
		}
		if c.ExpectedReason != "" && c.ExpectedReason != reason {
			return nil, fmt.Errorf("replay drift: %s", c.CaseID)
		}
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%t|%t|%t", c.CaseID, c.DescriptorDigest, c.DispatchAccepted, c.EffectCommitted, c.OutboxConfirmed)))
		out = append(out, ApplicationOutboxReplayResult{CaseID: c.CaseID, Reason: reason, Digest: "sha256:" + hex.EncodeToString(sum[:])})
	}
	return out, nil
}
