package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/FelixSeptem/baymax/host/sourceadapter"
)

const SourceAdapterLifecycleReplayVersion = "source_adapter_lifecycle.v1"
const ReasonSourceAdapterReplayDrift = "source_adapter.replay_drift"

type SourceAdapterLifecycleFixture struct {
	Version  string                       `json:"version"`
	Metadata sourceadapter.Metadata       `json:"metadata"`
	Steps    []SourceAdapterLifecycleStep `json:"steps"`
}

// SourceAdapterLifecycleStep contains synthetic references only. No payload,
// source, consumer, store or executable effect handle is admitted by replay.
type SourceAdapterLifecycleStep struct {
	ID              string `json:"id"`
	ExternalID      string `json:"external_id"`
	Reference       string `json:"reference"`
	PayloadDigest   string `json:"payload_digest"`
	Cursor          string `json:"cursor"`
	Outcome         string `json:"outcome"`
	ExpectedCursor  string `json:"expected_cursor"`
	ExpectedPending string `json:"expected_pending"`
	ExpectedReason  string `json:"expected_reason"`
}

func sourceAdapterDrift() error {
	return &ValidationError{Code: ReasonSourceAdapterReplayDrift, Message: "source adapter lifecycle conformance failed"}
}

func sourceAdapterRef(s string) bool {
	if s == "" || len(s) > 256 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func ReplaySourceAdapterLifecycleJSON(raw []byte) error {
	if len(raw) == 0 || len(raw) > 256<<10 || !utf8.Valid(raw) {
		return sourceAdapterDrift()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var f SourceAdapterLifecycleFixture
	if d.Decode(&f) != nil || d.Decode(new(any)) != io.EOF || f.Version != SourceAdapterLifecycleReplayVersion || len(f.Steps) == 0 || len(f.Steps) > 256 {
		return sourceAdapterDrift()
	}
	s, err := sourceadapter.InitialState(f.Metadata)
	if err != nil {
		return sourceAdapterDrift()
	}
	seen := make(map[string]bool, len(f.Steps))
	for _, step := range f.Steps {
		if !sourceAdapterRef(step.ID) || seen[step.ID] || !sourceAdapterRef(step.Cursor) || (step.ExpectedCursor != "" && !sourceAdapterRef(step.ExpectedCursor)) || (step.ExpectedPending != "" && !sourceAdapterRef(step.ExpectedPending)) {
			return sourceAdapterDrift()
		}
		seen[step.ID] = true
		switch step.Outcome {
		case "sent", "gone", "retryable", "interrupted", "confirm_failed":
		default:
			return sourceAdapterDrift()
		}
		r, err := sourceadapter.ReferenceReceipt(f.Metadata, step.ExternalID, step.Reference, step.PayloadDigest, sourceadapter.Cursor{Metadata: f.Metadata, Token: step.Cursor})
		if err != nil {
			return sourceAdapterDrift()
		}
		next, err := sourceadapter.PrepareReceipt(s, r)
		reason := ""
		if err != nil {
			reason = err.Error()
		} else {
			s = next
			switch step.Outcome {
			case "interrupted":
				reason = "source_adapter.interrupted"
			case "confirm_failed":
				reason = sourceadapter.ErrDependency.Error()
			default:
				next, err = sourceadapter.Confirm(s, sourceadapter.Disposition(step.Outcome))
				if err != nil {
					reason = err.Error()
				} else {
					s = next
				}
			}
		}
		cursor, pending := "", ""
		if s.Confirmed != nil {
			cursor = s.Confirmed.Token
		}
		if s.Pending != nil {
			pending = s.Pending.ExternalID
		}
		if cursor != step.ExpectedCursor || pending != step.ExpectedPending || reason != step.ExpectedReason {
			return sourceAdapterDrift()
		}
	}
	return nil
}
