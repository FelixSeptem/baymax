package fixture

import (
	"testing"

	"github.com/FelixSeptem/baymax/integration/adapterconformance"
)

func TestConformanceBootstrapAlignment(t *testing.T) {
	if err := adapterconformance.ValidateMinimumMatrix(adapterconformance.MinimumMatrix); err != nil {
		t.Fatalf("minimum matrix invalid: %v", err)
	}

	// Adapter conformance mapping hint for this scaffold category.
	const expectedScenarioID = "mcp-normalization-fail-fast"
	found := false
	for _, row := range adapterconformance.MinimumMatrix {
		if row.ID != expectedScenarioID {
			continue
		}
		found = true
		if row.Category != adapterconformance.CategoryMCP {
			t.Fatalf("category mismatch for %s: got=%s", expectedScenarioID, row.Category)
		}
		break
	}
	if !found {
		t.Fatalf("missing adapter conformance scenario mapping: %s", expectedScenarioID)
	}

	if err := adapterconformance.ValidateManifestProfileAlignmentForScaffold(".", expectedScenarioID); err != nil {
		t.Fatalf("manifest-profile alignment failed: %v", err)
	}
}
