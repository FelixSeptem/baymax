package host

import (
	"context"

	"github.com/FelixSeptem/baymax/core/types"
)

// resumeDynamicAction admits a source-owned same-Run checkpoint. The command
// response is intentionally limited to admission; runtime events emitted by
// the source remain the authoritative pause/resume/terminal projection.
func (c *Connection) resumeDynamicAction(ctx context.Context, cmd types.HostCommandEnvelope) (types.HostCommandResponse, error) {
	resumer, ok := c.coord.runner.(types.DynamicActionResumer)
	if !ok {
		return types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusRejected, "host.dynamic_action_resume_unavailable")
	}
	decision := types.DynamicActionDecision{
		Decision:          types.DynamicActionDecisionKind(stringPayload(cmd.Payload, "decision")),
		Token:             stringPayload(cmd.Payload, "token"),
		RunID:             cmd.RunID,
		SessionID:         cmd.SessionID,
		CheckpointID:      stringPayload(cmd.Payload, "checkpoint_id"),
		CheckpointVersion: stringPayload(cmd.Payload, "checkpoint_version"),
		CheckpointDigest:  stringPayload(cmd.Payload, "checkpoint_digest"),
		IdempotencyKey:    stringPayload(cmd.Payload, "idempotency_key"),
		Reference:         stringPayload(cmd.Payload, "reference"),
	}
	if err := decision.Validate(); err != nil {
		return reject(cmd, err)
	}
	stream, _ := cmd.Payload["stream"].(bool)
	h := EventHandlerFunc(func(eventCtx context.Context, ev types.Event) { _ = c.emitEvent(eventCtx, cmd, ev) })
	if _, err := resumer.ResumeDynamicAction(ctx, decision, h, stream); err != nil {
		return reject(cmd, err)
	}
	return types.NormalizeHostCommandAdmission(cmd, types.HostAdmissionStatusAccepted, "dynamic_action.resume_admitted")
}
