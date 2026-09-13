package evalcontract

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/orchestration/snapshot"
)

func ContinuityProjectionFromHandoff(h handoff.Handoff, phase string) (ContinuityProjection, error) {
	if err := h.Validate(handoff.DefaultLimits()); err != nil {
		return ContinuityProjection{}, err
	}
	p := ContinuityProjection{Version: ContinuityComparisonVersionV1, RunID: h.RunID, SessionID: h.SessionID, Phase: phase}
	p.Facts = append(p.Facts, ContinuityFact{Kind: ContinuityKindObjective, Owner: "handoff", ID: "objective", Digest: digestString(h.Objective), Required: true})
	if strings.TrimSpace(h.SourceCheckpointID) != "" {
		p.Facts = append(p.Facts, ContinuityFact{Kind: ContinuityKindCheckpoint, Owner: "handoff", ID: strings.TrimSpace(h.SourceCheckpointID), Required: true})
	}
	for _, pending := range h.Pending {
		if id := strings.TrimSpace(pending); id != "" {
			p.Facts = append(p.Facts, ContinuityFact{Kind: ContinuityKindPendingRequest, Owner: "handoff", ID: id})
		}
	}
	for _, ref := range h.References {
		kind := ""
		switch ref.Kind {
		case handoff.ReferenceArtifact:
			kind = ContinuityKindArtifact
		case handoff.ReferenceCheckpoint:
			kind = ContinuityKindCheckpoint
		}
		if kind != "" {
			p.Facts = append(p.Facts, ContinuityFact{Kind: kind, Owner: "handoff", ID: strings.TrimSpace(ref.ID), Digest: strings.TrimSpace(ref.Digest)})
		}
	}
	_, _, err := NormalizeContinuityProjection(p)
	if err != nil {
		return ContinuityProjection{}, fmt.Errorf("handoff continuity projection: %w", err)
	}
	return p, nil
}

func ContinuityProjectionFromSnapshot(m snapshot.Manifest, phase string) (ContinuityProjection, error) {
	if strings.TrimSpace(m.SchemaVersion) == "" || strings.TrimSpace(m.Source.RunID) == "" || strings.TrimSpace(m.Digest) == "" {
		return ContinuityProjection{}, fmt.Errorf("%s", ReasonContinuitySchemaDrift)
	}
	p := ContinuityProjection{Version: ContinuityComparisonVersionV1, RunID: m.Source.RunID, SessionID: m.Source.SessionID, Phase: phase, Facts: []ContinuityFact{{Kind: ContinuityKindCheckpoint, Owner: "snapshot", ID: strings.TrimSpace(m.Digest), Digest: strings.TrimSpace(m.Digest), Version: strings.TrimSpace(m.SchemaVersion), Required: true}}}
	if _, _, err := NormalizeContinuityProjection(p); err != nil {
		return ContinuityProjection{}, fmt.Errorf("snapshot continuity projection: %w", err)
	}
	return p, nil
}

func digestString(value string) string { d, _ := digestValue(strings.TrimSpace(value)); return d }
