package mailbox

import (
	"context"
	"strings"
)

// AdmitHostPlan delegates one fully bound logical message to the existing
// mailbox publish owner. It neither runs a consumer nor creates a separate
// queue, lease, retry, DLQ or business-commit state machine.
func (m *Mailbox) AdmitHostPlan(ctx context.Context, binding HostPlanBinding) (HostPlanOutcome, error) {
	normalized, err := normalizeHostPlanBinding(binding)
	if err != nil {
		return HostPlanOutcome{}, err
	}
	published, err := m.Publish(ctx, Envelope{
		DispatchBinding: hostPlanDispatchBinding(normalized),
		HostPlanBinding: &normalized,
		MessageID:       normalized.MessageID,
		IdempotencyKey:  normalized.IdempotencyKey,
		CorrelationID:   normalized.HostCorrelation,
		Kind:            KindCommand,
		RunID:           normalized.RunID,
	})
	if err != nil {
		if err == ErrDispatchBindingMismatch {
			return HostPlanOutcome{}, ErrHostPlanBindingConflict
		}
		return HostPlanOutcome{}, err
	}
	return hostPlanOutcome(normalized, published), nil
}

// HostPlanOutcomeFromRecord projects the source record without changing its
// lifecycle. It is useful after consume/ack/retry/DLQ operations.
func HostPlanOutcomeFromRecord(record Record) (HostPlanOutcome, error) {
	if record.Envelope.HostPlanBinding == nil {
		return HostPlanOutcome{}, ErrHostPlanBindingConflict
	}
	return hostPlanOutcome(*record.Envelope.HostPlanBinding, PublishResult{Record: record}), nil
}

func hostPlanOutcome(binding HostPlanBinding, published PublishResult) HostPlanOutcome {
	record := published.Record
	deliveryLeaseID := ""
	if record.DeliveryID != "" || record.LeaseID != "" {
		deliveryLeaseID = record.DeliveryID + ":" + record.LeaseID
	}
	lifecycleResult, terminalOutcome := lifecycleProjection(record)
	return HostPlanOutcome{
		MessageID:        record.Envelope.MessageID,
		RunID:            binding.RunID,
		DeliveryID:       record.DeliveryID,
		LeaseID:          record.LeaseID,
		DeliveryLeaseID:  deliveryLeaseID,
		AttemptCausation: record.AttemptCausation,
		ParentRunID:      binding.ParentRunID,
		CausationID:      binding.CausationID,
		AuthorityDigest:  binding.AuthorityDigest,
		PlanDigest:       binding.PlanDigest,
		WriteScope:       binding.WriteScope,
		EffectNamespace:  binding.EffectNamespace,
		Occurrence:       binding.Occurrence,
		HostCorrelation:  binding.HostCorrelation,
		DeliveryAttempt:  record.DeliveryAttempt,
		LifecycleState:   record.State,
		LifecycleResult:  lifecycleResult,
		LifecycleReason:  firstNonEmpty(record.DeadLetterReason, record.LastError),
		TerminalOutcome:  terminalOutcome,
		Duplicate:        published.Duplicate,
	}
}

func lifecycleProjection(record Record) (string, string) {
	switch record.State {
	case StateQueued:
		if NormalizeLifecycleReason(record.LastError) == LifecycleReasonLeaseExpired {
			return "lease_lost", ""
		}
		if strings.TrimSpace(record.LastError) != "" {
			return "retry", ""
		}
		return "admitted", ""
	case StateInFlight:
		return "leased", ""
	case StateAcked:
		return "ack", "acked"
	case StateNacked:
		if NormalizeLifecycleReason(record.LastError) == LifecycleReasonLeaseExpired {
			return "lease_lost", ""
		}
		return "nack", ""
	case StateDeadLetter:
		return "dlq", "dead_letter"
	case StateExpired:
		return "timeout", "expired"
	default:
		return "", ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
