package conformance

import (
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// RequestFactsFromModelRequest derives the provider-neutral source facts that
// the runtime admitted into types.ModelRequest. It is the audited "source" side
// of the request projection contract: it records structure, order, identity and
// bounded counters only, never prompt text, reasoning, or tool output bodies.
//
// Roles are recorded in first-appearance order with duplicates removed: role
// presence and ordering are the audited facts, not raw message bodies.
func RequestFactsFromModelRequest(req types.ModelRequest) RequestFacts {
	facts := RequestFacts{
		InputBytes: len(req.Input),
	}
	seenRole := map[string]struct{}{}
	for _, msg := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			continue
		}
		if _, exists := seenRole[role]; exists {
			continue
		}
		seenRole[role] = struct{}{}
		facts.Roles = append(facts.Roles, role)
		if role == "system" {
			facts.SkillFragments++
		}
	}
	if len(req.ToolResult) > 0 {
		facts.ToolResults = make([]RequestToolFact, 0, len(req.ToolResult))
		for i := range req.ToolResult {
			outcome := req.ToolResult[i]
			status := "ok"
			if outcome.Result.Error != nil {
				status = "error"
			}
			facts.ToolResults = append(facts.ToolResults, RequestToolFact{
				CallID:       strings.TrimSpace(outcome.CallID),
				Name:         strings.TrimSpace(outcome.Name),
				Status:       status,
				ResultDigest: RequestContentDigest(outcome.Result.Content),
			})
			facts.ToolOrder = append(facts.ToolOrder, strings.TrimSpace(outcome.Name))
		}
	}
	if required := req.Capabilities.Required; len(required) > 0 {
		facts.Capabilities = make([]string, 0, len(required))
		for _, capability := range required {
			facts.Capabilities = append(facts.Capabilities, string(capability))
		}
	}
	return facts
}
