package event

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderProjectsRedactedProviderAdmissionFields(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()
	recorder := NewRuntimeRecorder(mgr)
	recorder.OnEvent(context.Background(), types.Event{
		Type:  "run.finished",
		RunID: "run-provider-catalog",
		Payload: map[string]any{
			"status":                      "success",
			"provider_catalog_version":    "v1",
			"provider_model":              "openai/gpt",
			"provider_capability_outcome": "degraded",
			"provider_credential_status":  "available",
			"provider_fallback":           "openai/vision",
			"provider_admission_reasons":  []any{"adapter.capability.optional_downgraded", "provider.catalog.optional_fallback"},
			"credential":                  "sk-secret-must-not-persist",
			"endpoint":                    "https://secret.example",
		},
	})
	records := mgr.RecentRuns(1)
	if len(records) != 1 {
		t.Fatalf("RecentRuns() len = %d, want 1", len(records))
	}
	record := records[0]
	if record.ProviderCatalogVersion != "v1" || record.ProviderModel != "openai/gpt" || record.ProviderCredentialStatus != "available" {
		t.Fatalf("provider fields = %#v", record)
	}
	if len(record.ProviderAdmissionReasons) != 2 || record.ProviderAdmissionReasons[1] != "provider.catalog.optional_fallback" {
		t.Fatalf("ProviderAdmissionReasons = %#v", record.ProviderAdmissionReasons)
	}
	if record.ProviderFallback != "openai/vision" {
		t.Fatalf("ProviderFallback = %q", record.ProviderFallback)
	}
}
