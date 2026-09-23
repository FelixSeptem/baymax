package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaAuditBoundaryGateAndImplementationRemainOffline(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Dir(filepath.Dir(root))
	for _, name := range []string{"scripts/check-tool-schema-audit-contract.sh", "scripts/check-tool-schema-audit-contract.ps1"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("missing schema audit gate %s: %v", name, err)
		}
	}
	for _, path := range []string{"tool/schemaaudit/audit.go", "tool/schemaaudit/replay.go", "tool/diagnosticsreplay/schema_audit.go"} {
		body, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, forbidden := range []string{"github.com/openai/", "anthropics/anthropic-sdk", "google.golang.org/genai", "net/http", "os.Open", "os.WriteFile", "ModelRequest", "registry.Register", "globalSelector", "globalRouter", "dynamicDownload", "marketplaceResolve", "persistRawSchema", "persistPrompt", "persistModelOutput", "persistToolResult"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contains forbidden boundary reference %q", path, forbidden)
			}
		}
	}
}

func TestSchemaAuditGateIncludesPositiveAndNegativeContractChecks(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Dir(filepath.Dir(root))
	for _, name := range []string{"scripts/check-tool-schema-audit-contract.sh", "scripts/check-tool-schema-audit-contract.ps1"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, marker := range []string{"tool/schemaaudit", "tool/diagnosticsreplay", "ReplayJSON", "Run", "Stream", "privacy", "overflow", "pressure", "quality", "strategy"} {
			if !strings.Contains(strings.ToLower(text), strings.ToLower(marker)) {
				t.Fatalf("%s missing gate marker %q", name, marker)
			}
		}
	}
}
