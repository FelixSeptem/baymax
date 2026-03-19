package fixture

import (
	"testing"

	"github.com/FelixSeptem/baymax/integration/adapterconformance"
)

func TestConformanceBootstrapAlignment(t *testing.T) {
	if err := adapterconformance.ValidateMinimumMatrix(adapterconformance.MinimumMatrix); err != nil {
		t.Fatalf("minimum matrix invalid: %v", err)
	}

	// A22 mapping hint for this scaffold category.
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
		t.Fatalf("missing A22 scenario mapping: %s", expectedScenarioID)
	}
}
