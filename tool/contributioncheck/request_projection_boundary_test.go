package contributioncheck

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// requestProjectionFixturePath is the versioned request-side conformance
// artifact (ModelRequest -> provider SDK request).
const requestProjectionFixturePath = "tool/diagnosticsreplay/testdata/model_request_projection.v1.json"

// requestProjectionFixtureVersion is the pinned version namespace. A change to
// it is a contract change, not a refactor.
const requestProjectionFixtureVersion = "provider_request_projection.v1"

// requestProjectionDriftCodes is the stable classification vocabulary. It must
// stay identical across model/conformance, tool/diagnosticsreplay, and the
// module documentation.
var requestProjectionDriftCodes = []string{
	"provider_request_schema_drift",
	"provider_request_role_projection_drift",
	"provider_request_tool_result_native_drift",
	"provider_request_part_ordering_drift",
	"provider_request_stable_prefix_drift",
	"provider_request_tool_order_drift",
	"provider_request_capability_projection_drift",
	"provider_request_run_stream_parity_drift",
	"provider_cache_usage_projection_drift",
	"provider_request_overflow_drift",
	"provider_request_contract_drift",
}

// providerSDKPrefixes are the provider SDK module paths that the
// provider-neutral contract and diagnostics packages must never depend on.
var providerSDKPrefixes = []string{
	"github.com/openai/openai-go",
	"github.com/anthropics/anthropic-sdk-go",
	"google.golang.org/genai",
}

// TestProviderRequestProjectionContractBoundary guards the request-side
// projection contract: provider neutrality of the contract/diagnostics layer,
// adapter ownership of the SDK request construction, boundedness of the
// versioned fixture, and absence of raw payload fields.
func TestProviderRequestProjectionContractBoundary(t *testing.T) {
	root := repoRoot(t)

	t.Run("contract and replay packages stay provider neutral", func(t *testing.T) {
		for _, relative := range []string{
			filepath.Join("model", "conformance"),
			filepath.Join("tool", "diagnosticsreplay"),
		} {
			assertNoProviderSDKImports(t, root, relative)
		}
	})

	t.Run("adapters keep the SDK-neutral request interpretation entrypoint", func(t *testing.T) {
		expectations := map[string]string{
			filepath.Join("model", "openai", "client.go"):    "toolcontract.InterpretRequest",
			filepath.Join("model", "anthropic", "client.go"): "toolcontract.InterpretRequest",
			filepath.Join("model", "gemini", "client.go"):    "toolcontract.InterpretRequest",
		}
		for relative, symbol := range expectations {
			source := mustRead(t, filepath.Join(root, relative))
			if !strings.Contains(source, symbol) {
				t.Fatalf("%s no longer projects the request through %s", filepath.ToSlash(relative), symbol)
			}
		}
	})

	t.Run("contract files carry no raw payload fields", func(t *testing.T) {
		forbidden := []string{
			`json:"raw_prompt`,
			`json:"raw_reasoning`,
			`json:"raw_payload`,
			`json:"prompt_text`,
			`json:"transcript`,
			`json:"credentials`,
		}
		for _, relative := range []string{
			filepath.Join("model", "conformance", "request_projection.go"),
			filepath.Join("model", "conformance", "request_facts.go"),
			filepath.Join("tool", "diagnosticsreplay", "request_projection.go"),
		} {
			source := mustRead(t, filepath.Join(root, relative))
			for _, field := range forbidden {
				if strings.Contains(source, field) {
					t.Fatalf("%s introduces a raw payload field %q", filepath.ToSlash(relative), field)
				}
			}
		}
	})

	t.Run("fixture is versioned, bounded, and classified", func(t *testing.T) {
		path := filepath.Join(root, filepath.FromSlash(requestProjectionFixturePath))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("read request projection fixture: %v", err)
		}
		if info.Size() > 2<<20 {
			t.Fatalf("request projection fixture exceeds 2 MiB: %d bytes", info.Size())
		}

		raw := mustRead(t, path)
		var document struct {
			Version string `json:"version"`
			Cases   []struct {
				Name         string   `json:"name"`
				Provider     string   `json:"provider"`
				Mode         string   `json:"mode"`
				DeclaredGaps []string `json:"declared_gaps"`
			} `json:"cases"`
		}
		if err := json.Unmarshal([]byte(raw), &document); err != nil {
			t.Fatalf("decode request projection fixture: %v", err)
		}
		if document.Version != requestProjectionFixtureVersion {
			t.Fatalf("fixture version drifted: got %q want %q", document.Version, requestProjectionFixtureVersion)
		}
		if len(document.Cases) == 0 {
			t.Fatal("fixture contains no cases")
		}

		covered := map[string]map[string]bool{}
		for _, c := range document.Cases {
			if c.Provider != "openai" && c.Provider != "anthropic" && c.Provider != "gemini" {
				t.Fatalf("case %q declares unsupported provider %q", c.Name, c.Provider)
			}
			if c.Mode != "run" && c.Mode != "stream" {
				t.Fatalf("case %q declares unsupported mode %q", c.Name, c.Mode)
			}
			if covered[c.Provider] == nil {
				covered[c.Provider] = map[string]bool{}
			}
			covered[c.Provider][c.Mode] = true
			for _, gap := range c.DeclaredGaps {
				if !slices.Contains(requestProjectionDriftCodes, gap) {
					t.Fatalf("case %q declares unknown gap %q", c.Name, gap)
				}
			}
		}
		for _, provider := range []string{"openai", "anthropic", "gemini"} {
			for _, mode := range []string{"run", "stream"} {
				if !covered[provider][mode] {
					t.Fatalf("fixture does not cover provider %q mode %q", provider, mode)
				}
			}
		}
	})

	t.Run("taxonomy is documented exactly once and completely", func(t *testing.T) {
		doc := mustRead(t, filepath.Join(root, "model", "README.md"))
		for _, code := range requestProjectionDriftCodes {
			if !strings.Contains(doc, code) {
				t.Fatalf("model/README.md does not document request projection classification %q", code)
			}
		}
		if !strings.Contains(doc, requestProjectionFixtureVersion) {
			t.Fatalf("model/README.md does not document fixture version %q", requestProjectionFixtureVersion)
		}
	})
}

// assertNoProviderSDKImports fails when any non-generated Go file under the
// given package directory imports a provider SDK.
func assertNoProviderSDKImports(t *testing.T, root, relativeDir string) {
	t.Helper()
	dir := filepath.Join(root, relativeDir)
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range file.Imports {
			pkg := strings.TrimSpace(strings.Trim(imp.Path.Value, `"`))
			for _, prefix := range providerSDKPrefixes {
				if pkg == prefix || strings.HasPrefix(pkg, prefix+"/") {
					t.Fatalf("provider neutrality violation: %s imports %q",
						filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))), pkg)
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan %s imports failed: %v", filepath.ToSlash(relativeDir), walkErr)
	}
}
