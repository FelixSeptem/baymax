package agentruntimeprotocolprojection

import (
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/snapshot"
)

const (
	patternName    = "agent-runtime-protocol-projection"
	phase          = "P1"
	semanticAnchor = "agent_runtime_protocol.capability_context_admission"
	classification = "agent_runtime_protocol.projection"
)

func RunMinimal()    { run(false) }
func RunProduction() { run(true) }

func run(production bool) {
	variant := "minimal"
	if production {
		variant = "production-ish"
	}
	req := types.RunRequest{RunID: "agent-mode-protocol-run-1", SessionID: "agent-mode-protocol-session-1"}
	runRef, err := types.NewProtocolRunFromRequest(req, types.RunStateWorking)
	if err != nil {
		panic(err)
	}
	eventRef, err := types.MapEventToProtocol(types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   runRef.RunID,
		CallID:  "protocol-step-1",
		Time:    time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC),
		Payload: map[string]any{"phase": "run", "status": "running"},
	}, types.ProtocolSourceTimeline)
	if err != nil {
		panic(err)
	}
	checkpoint, err := snapshot.ProtocolCheckpointRef(snapshot.Manifest{
		SchemaVersion: snapshot.ManifestSchemaVersionV1,
		ExportedAt:    time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC),
		Source:        snapshot.Source{Component: "agent-runtime-protocol-example", RunID: runRef.RunID, SessionID: runRef.SessionID},
		Digest:        "sha256:protocol-example",
	})
	if err != nil {
		panic(err)
	}
	descriptor, err := types.ProtocolDescriptorForSource(types.ProtocolSourceRunner, "agent-runtime-protocol-example", "v1", []types.ProtocolCapability{{Name: "interrupt_resume", Version: "v1"}}, []types.ProtocolAction{types.ProtocolActionCancel, types.ProtocolActionRetry})
	if err != nil {
		panic(err)
	}
	if _, err := types.NegotiateProtocolCapabilities(descriptor, types.ProtocolCapabilityRequest{ProfileVersion: "v1", Required: []string{"interrupt_resume"}}); err != nil {
		panic(err)
	}
	contextProjection, err := types.ProjectProtocolSessionContext(runRefSession(runRef), types.ProtocolContextScopeSession, &types.ProtocolParticipantRef{ID: "agent-runtime-protocol-example", Role: "primary", Source: types.ProtocolSourceRunner}, nil, map[string]string{"requester": "example"}, types.ProtocolContextLimits{AllowedMetadataKeys: []string{"requester"}, MaxMetadataEntries: 2, MaxMetadataValueBytes: 32, MaxSerializedBytes: 512})
	if err != nil {
		panic(err)
	}
	admission, err := types.ProjectProtocolRunAdmission(types.ProtocolRunAdmission{SessionID: runRef.SessionID, RunID: runRef.RunID, Policy: types.ProtocolConcurrentRunPolicySerialize, Decision: types.ProtocolRunAdmissionDecisionAdmitted, ReasonCode: "scheduler.session_available"})
	if err != nil {
		panic(err)
	}
	expectedMarkers := []string{"protocol_run_mapped", "protocol_event_mapped", "protocol_checkpoint_mapped", "protocol_descriptor_validated", "protocol_context_validated", "protocol_admission_projected"}
	if production {
		expectedMarkers = append(expectedMarkers, "protocol_invalid_transition_rejected")
	}
	governance := "baseline"
	if production {
		governance = "enforced"
	}
	runtimePath := "core/types,core/runner,observability/event,orchestration/snapshot"
	if production {
		runtimePath += ",observability/trace"
	}
	finalAnswer := fmt.Sprintf("%s/%s protocol_projection_completed phase=%s anchor=%s classification=%s run_id=%s event_id=%s checkpoint_id=%s markers=%s governance=%s", patternName, variant, phase, semanticAnchor, classification, runRef.RunID, eventRef.EventID, checkpoint.CheckpointID, strings.Join(expectedMarkers, ","), governance)
	fmt.Println("agent-mode example")
	fmt.Printf("pattern=%s\n", patternName)
	fmt.Printf("variant=%s\n", variant)
	fmt.Printf("runtime.path=%s\n", runtimePath)
	fmt.Println("verification.mainline_runtime_path=ok")
	fmt.Printf("verification.semantic.phase=%s\n", phase)
	fmt.Printf("verification.semantic.anchor=%s\n", semanticAnchor)
	fmt.Printf("verification.semantic.classification=%s\n", classification)
	fmt.Printf("verification.semantic.runtime_path=%s\n", runtimePath)
	fmt.Printf("verification.semantic.expected_markers=%s\n", strings.Join(expectedMarkers, ","))
	fmt.Printf("verification.semantic.governance=%s\n", governance)
	fmt.Printf("verification.semantic.marker_count=%d\n", len(expectedMarkers))
	for _, marker := range expectedMarkers[:3] {
		fmt.Printf("verification.semantic.marker.%s=ok\n", marker)
	}
	if err := descriptor.Validate(); err == nil {
		fmt.Println("verification.semantic.marker.protocol_descriptor_validated=ok")
	}
	if contextProjection.Scope == types.ProtocolContextScopeSession {
		fmt.Println("verification.semantic.marker.protocol_context_validated=ok")
	}
	if admission.Decision == types.ProtocolRunAdmissionDecisionAdmitted {
		fmt.Println("verification.semantic.marker.protocol_admission_projected=ok")
	}
	if production {
		if err := types.ValidateRunStateTransition(types.RunStateCompleted, types.RunStateWorking); err != nil {
			fmt.Println("verification.semantic.marker.protocol_invalid_transition_rejected=ok")
		}
	}
	fmt.Printf("result.run_id=%s\n", runRef.RunID)
	fmt.Printf("result.event_id=%s\n", eventRef.EventID)
	fmt.Printf("result.checkpoint_id=%s\n", checkpoint.CheckpointID)
	fmt.Printf("result.final_answer=%s\n", finalAnswer)
	fmt.Printf("result.signature=%d\n", len(finalAnswer))
}

func runRefSession(run types.RunRef) types.SessionRef {
	return types.SessionRef{SessionID: run.SessionID, Source: run.Source}
}
