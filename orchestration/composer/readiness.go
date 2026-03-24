package composer

import (
	"errors"
	"strings"
	"time"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// ReadinessPreflight returns runtime readiness result for managed runtime path.
// The query is read-only and does not mutate scheduler/task state.
func (c *Composer) ReadinessPreflight() (runtimeconfig.ReadinessResult, error) {
	if c == nil {
		return runtimeconfig.ReadinessResult{}, errors.New("composer is nil")
	}
	if c.runtimeMgr == nil {
		return runtimeconfig.ReadinessResult{}, errors.New("runtime manager is not initialized")
	}
	return c.runtimeMgr.ReadinessPreflight(), nil
}

func (c *Composer) publishRuntimeReadinessSnapshot() {
	if c == nil || c.runtimeMgr == nil {
		return
	}
	c.schedulerMu.RLock()
	snapshot := runtimeconfig.RuntimeReadinessComponentSnapshot{
		Scheduler: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: strings.TrimSpace(c.schedulerConfiguredBackend),
			EffectiveBackend:  strings.TrimSpace(c.schedulerBackend),
			Fallback:          c.schedulerFallback,
			FallbackReason:    strings.TrimSpace(c.schedulerFallbackReason),
		},
		Mailbox: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           c.mailboxEnabled,
			ConfiguredBackend: strings.TrimSpace(c.mailboxConfiguredBackend),
			EffectiveBackend:  strings.TrimSpace(c.mailboxBackend),
			Fallback:          c.mailboxFallback,
			FallbackReason:    strings.TrimSpace(c.mailboxFallbackReason),
		},
		Recovery: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           c.recoveryEnabled,
			ConfiguredBackend: strings.TrimSpace(c.recoveryConfiguredBackend),
			EffectiveBackend:  strings.TrimSpace(c.recoveryBackend),
			Fallback:          c.recoveryFallback,
			FallbackReason:    strings.TrimSpace(c.recoveryFallbackReason),
		},
		UpdatedAt: resolveReadinessSnapshotTime(c.now),
	}
	c.schedulerMu.RUnlock()
	c.runtimeMgr.SetReadinessComponentSnapshot(snapshot)
}

func resolveReadinessSnapshotTime(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC()
	}
	return now().UTC()
}
