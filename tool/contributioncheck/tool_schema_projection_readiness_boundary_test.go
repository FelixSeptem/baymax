package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolSchemaProjectionReadinessBoundaryAndGates(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"scripts/check-tool-schema-projection-readiness-contract.sh", "scripts/check-tool-schema-projection-readiness-contract.ps1"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("missing readiness gate %s: %v", name, err)
		}
	}
	for _, path := range []string{"tool/schemaaudit/readiness.go", "tool/diagnosticsreplay/tool_schema_projection_readiness.go"} {
		body := mustRead(t, filepath.Join(root, path))
		lower := strings.ToLower(body)
		for _, forbidden := range []string{"net/http", "modelrequest", "globalselector", "globalrouter", "dynamicdownload", "marketplaceresolve", "persistrawschema", "persistprompt", "persistmodeloutput", "persisttoolresult", "runtime_recorder"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contains forbidden boundary marker %q", path, forbidden)
			}
		}
	}
}

func TestToolSchemaProjectionReadinessGateIncludesContractMarkers(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"scripts/check-tool-schema-projection-readiness-contract.sh", "scripts/check-tool-schema-projection-readiness-contract.ps1"} {
		text := strings.ToLower(mustRead(t, filepath.Join(root, name)))
		for _, marker := range []string{"tool/schemaaudit", "tool/diagnosticsreplay", "readiness", "parity", "privacy", "overflow", "subset", "offline"} {
			if !strings.Contains(text, marker) {
				t.Fatalf("%s missing gate marker %q", name, marker)
			}
		}
	}
}
