package diagnostics

import (
	"encoding/json"
	"testing"
)

func TestRunRecordProviderAdmissionFieldsAreAdditiveAndNullable(t *testing.T) {
	var legacy RunRecord
	if err := json.Unmarshal([]byte(`{"run_id":"legacy","status":"success"}`), &legacy); err != nil {
		t.Fatalf("legacy decode error = %v", err)
	}
	if legacy.ProviderCatalogVersion != "" || legacy.ProviderModel != "" || legacy.ProviderCredentialStatus != "" || legacy.ProviderAdmissionReasons != nil {
		t.Fatalf("legacy provider fields = %#v, want nullable defaults", legacy)
	}

	record := RunRecord{
		RunID:                     "provider-run",
		Status:                    "success",
		ProviderCatalogVersion:    "v1",
		ProviderModel:             "openai/gpt",
		ProviderCapabilityOutcome: "degraded",
		ProviderCredentialStatus:  "available",
		ProviderFallback:          "openai/vision",
		ProviderAdmissionReasons:  []string{"provider.catalog.optional_fallback"},
	}
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}
	text := string(payload)
	for _, forbidden := range []string{"sk-", "secret", "token", "endpoint"} {
		if containsFold(text, forbidden) {
			t.Fatalf("serialized provider diagnostics contain forbidden material %q: %s", forbidden, text)
		}
	}
	var decoded RunRecord
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	if decoded.ProviderCatalogVersion != record.ProviderCatalogVersion || decoded.ProviderModel != record.ProviderModel || len(decoded.ProviderAdmissionReasons) != 1 {
		t.Fatalf("decoded provider fields = %#v", decoded)
	}
}

func containsFold(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		match := true
		for j := range needle {
			c := value[i+j]
			n := needle[j]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			if n >= 'A' && n <= 'Z' {
				n += 'a' - 'A'
			}
			if c != n {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
