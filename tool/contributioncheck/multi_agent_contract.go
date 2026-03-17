package contributioncheck

import "strings"

type MultiAgentContractSnapshot struct {
	IdentifierDoc             string
	TeamsTimelineSpec         string
	WorkflowTimelineSpec      string
	A2ATimelineSpec           string
	A2ACoreSpec               string
	TeamsRuntimeConfigSpec    string
	WorkflowRuntimeConfigSpec string
	A2ARuntimeConfigSpec      string
	TeamsBoundarySpec         string
	WorkflowBoundarySpec      string
	A2ABoundarySpec           string
}

func ValidateMultiAgentSharedContractSnapshot(snapshot MultiAgentContractSnapshot) []Violation {
	violations := make([]Violation, 0)

	if !strings.Contains(snapshot.IdentifierDoc, "| a2a | `submitted` | `pending` |") {
		violations = append(violations, Violation{
			Code:    "missing_status_mapping_a2a_submitted_pending",
			Message: "identifier model must include mapping a2a submitted -> pending",
		})
	}
	if !strings.Contains(snapshot.A2ACoreSpec, "submitted") || !strings.Contains(snapshot.A2ACoreSpec, "pending") {
		violations = append(violations, Violation{
			Code:    "missing_a2a_submitted_pending_alignment",
			Message: "a2a lifecycle spec must align submitted with pending semantic layer",
		})
	}

	if !strings.Contains(snapshot.IdentifierDoc, "`team.*`") ||
		!strings.Contains(snapshot.IdentifierDoc, "`workflow.*`") ||
		!strings.Contains(snapshot.IdentifierDoc, "`a2a.*`") {
		violations = append(violations, Violation{
			Code:    "missing_reason_namespace_contract",
			Message: "identifier model must define team/workflow/a2a reason namespaces",
		})
	}

	requiredReasons := map[string]string{
		"team.dispatch":      snapshot.TeamsTimelineSpec,
		"team.collect":       snapshot.TeamsTimelineSpec,
		"team.resolve":       snapshot.TeamsTimelineSpec,
		"workflow.schedule":  snapshot.WorkflowTimelineSpec,
		"workflow.retry":     snapshot.WorkflowTimelineSpec,
		"workflow.resume":    snapshot.WorkflowTimelineSpec,
		"a2a.submit":         snapshot.A2ATimelineSpec,
		"a2a.status_poll":    snapshot.A2ATimelineSpec,
		"a2a.callback_retry": snapshot.A2ATimelineSpec,
		"a2a.resolve":        snapshot.A2ATimelineSpec,
	}
	for reason, source := range requiredReasons {
		if !strings.Contains(source, reason) {
			violations = append(violations, Violation{
				Code:    "missing_reason_" + strings.ReplaceAll(reason, ".", "_"),
				Message: "missing required namespaced reason: " + reason,
			})
		}
	}

	if !strings.Contains(snapshot.IdentifierDoc, "`peer_id`") ||
		!strings.Contains(snapshot.A2ATimelineSpec, "`peer_id`") ||
		!strings.Contains(snapshot.A2ARuntimeConfigSpec, "`peer_id`") {
		violations = append(violations, Violation{
			Code:    "missing_peer_id_canonical_naming",
			Message: "peer_id must be used as canonical A2A peer identifier field",
		})
	}

	if strings.Contains(snapshot.IdentifierDoc, "`a2a_peer`") || strings.Contains(snapshot.A2ARuntimeConfigSpec, "`a2a_peer`") {
		violations = append(violations, Violation{
			Code:    "deprecated_a2a_peer_field_detected",
			Message: "deprecated field a2a_peer detected; use peer_id instead",
		})
	}

	if !strings.Contains(snapshot.TeamsRuntimeConfigSpec, "`teams.*`") ||
		!strings.Contains(snapshot.WorkflowRuntimeConfigSpec, "`workflow.*`") ||
		!strings.Contains(snapshot.A2ARuntimeConfigSpec, "`a2a.*`") {
		violations = append(violations, Violation{
			Code:    "missing_domain_scoped_config_namespaces",
			Message: "teams/workflow/a2a runtime config specs must declare domain-scoped namespaces",
		})
	}

	if !strings.Contains(snapshot.TeamsBoundarySpec, "shared multi-agent contract gate") ||
		!strings.Contains(snapshot.WorkflowBoundarySpec, "shared multi-agent contract gate") ||
		!strings.Contains(snapshot.A2ABoundarySpec, "shared multi-agent contract gate") {
		violations = append(violations, Violation{
			Code:    "missing_blocking_shared_contract_gate",
			Message: "teams/workflow/a2a boundary specs must declare blocking shared-contract gate",
		})
	}

	return violations
}
