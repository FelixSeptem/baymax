package config

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerReadinessPreflightClassificationMatrix(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_A40_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	ready := mgr.ReadinessPreflight()
	if ready.Status != ReadinessStatusReady {
		t.Fatalf("ready status = %q, want %q", ready.Status, ReadinessStatusReady)
	}
	if len(ready.Findings) != 0 {
		t.Fatalf("ready findings = %#v, want empty", ready.Findings)
	}

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})
	degraded := mgr.ReadinessPreflight()
	if degraded.Status != ReadinessStatusDegraded {
		t.Fatalf("degraded status = %q, want %q", degraded.Status, ReadinessStatusDegraded)
	}
	assertReadinessFindingCode(t, degraded.Findings, ReadinessCodeSchedulerFallback)
	assertReadinessCanonicalFields(t, degraded.Findings)

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Recovery: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			ActivationError:   "permission denied",
		},
	})
	blocked := mgr.ReadinessPreflight()
	if blocked.Status != ReadinessStatusBlocked {
		t.Fatalf("blocked status = %q, want %q", blocked.Status, ReadinessStatusBlocked)
	}
	assertReadinessFindingCode(t, blocked.Findings, ReadinessCodeRecoveryActivationError)
	assertReadinessCanonicalFields(t, blocked.Findings)
}

func TestManagerReadinessPreflightStrictEscalation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A40_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status = %q, want %q", result.Status, ReadinessStatusBlocked)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSchedulerFallback)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeStrictEscalated)

	summary := result.Summary()
	if summary.Status != string(ReadinessStatusBlocked) {
		t.Fatalf("summary status = %q, want %q", summary.Status, ReadinessStatusBlocked)
	}
	if summary.FindingTotal < 2 || summary.BlockingTotal < 1 || summary.DegradedTotal < 1 {
		t.Fatalf("summary counts mismatch: %#v", summary)
	}
}

func TestManagerReadinessPreflightDeterministicForEquivalentSnapshot(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_A40_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
		Mailbox: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "mailbox.backend.file_init_failed",
		},
	})

	first := mgr.ReadinessPreflight()
	second := mgr.ReadinessPreflight()
	if first.Status != second.Status {
		t.Fatalf("status mismatch first=%q second=%q", first.Status, second.Status)
	}
	if readinessSemanticFingerprint(first) != readinessSemanticFingerprint(second) {
		t.Fatalf("semantics changed across equivalent snapshots\nfirst=%s\nsecond=%s", readinessSemanticFingerprint(first), readinessSemanticFingerprint(second))
	}
}

func assertReadinessFindingCode(t *testing.T, findings []ReadinessFinding, code string) {
	t.Helper()
	for i := range findings {
		if strings.TrimSpace(findings[i].Code) == strings.TrimSpace(code) {
			return
		}
	}
	t.Fatalf("finding code %q not found in %#v", code, findings)
}

func assertReadinessCanonicalFields(t *testing.T, findings []ReadinessFinding) {
	t.Helper()
	for i := range findings {
		item := findings[i]
		if strings.TrimSpace(item.Code) == "" {
			t.Fatalf("finding[%d] code is empty: %#v", i, item)
		}
		if strings.TrimSpace(item.Domain) == "" {
			t.Fatalf("finding[%d] domain is empty: %#v", i, item)
		}
		if strings.TrimSpace(item.Severity) == "" {
			t.Fatalf("finding[%d] severity is empty: %#v", i, item)
		}
		if strings.TrimSpace(item.Message) == "" {
			t.Fatalf("finding[%d] message is empty: %#v", i, item)
		}
		if item.Metadata == nil {
			t.Fatalf("finding[%d] metadata is nil: %#v", i, item)
		}
	}
}

func readinessSemanticFingerprint(result ReadinessResult) string {
	payload := struct {
		Status   ReadinessStatus    `json:"status"`
		Findings []ReadinessFinding `json:"findings"`
	}{
		Status:   result.Status,
		Findings: result.Findings,
	}
	blob, _ := json.Marshal(payload)
	return string(blob)
}
