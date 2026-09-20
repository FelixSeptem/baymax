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

// budgetProjectionFixturePath is the versioned derived budget projection
// artifact (source-owned budget facts -> bounded read-only remaining view).
const budgetProjectionFixturePath = "tool/diagnosticsreplay/testdata/budget_projection.v1.json"

// budgetProjectionFixtureVersion is the pinned version namespace. A change to it
// is a contract change, not a refactor.
const budgetProjectionFixtureVersion = "budget_projection.v1"

// budgetProjectionDriftCodes is the stable classification vocabulary. It must
// stay identical across context/budgetprojection, tool/diagnosticsreplay, the
// spec, the module documentation and the gate scripts.
var budgetProjectionDriftCodes = []string{
	"budget_projection_schema_drift",
	"budget_projection_unknown_version",
	"budget_facts_missing_iteration_limit",
	"budget_facts_missing_tool_call_limit",
	"budget_facts_missing_time_budget",
	"budget_facts_missing_cost_threshold",
	"budget_projection_negative_remaining",
	"budget_projection_ratio_out_of_range",
	"budget_projection_pressure_level_mismatch",
	"budget_projection_note_unbounded",
	"budget_projection_digest_mismatch",
	"budget_projection_overflow_drift",
	"budget_projection_writeback_shape_detected",
	"budget_benchmark_schema_drift",
	"budget_benchmark_metric_mismatch",
	"budget_benchmark_recovery_recompute_drift",
	"budget_benchmark_overflow_drift",
	"budget_projection_replay_not_idempotent",
}

// budgetProjectionContractDir is the derived projection owner.
var budgetProjectionContractDir = filepath.Join("context", "budgetprojection")

// TestBudgetProjectionContractBoundary guards the derived budget projection
// contract: zero internal dependencies (no second ledger, no runtime coupling),
// absence of raw payload fields, boundedness of the versioned fixture, and
// taxonomy stability across code, docs and gate scripts.
func TestBudgetProjectionContractBoundary(t *testing.T) {
	root := repoRoot(t)

	t.Run("projection contract has no internal dependencies", func(t *testing.T) {
		assertNoBaymaxImports(t, root, budgetProjectionContractDir)
	})

	t.Run("projection contract introduces no budget ledger keys", func(t *testing.T) {
		forbidden := []string{
			"runtime.admission.",
			"runtime.react.",
			"runtime.budget.",
			"budget_ledger",
			"remaining_budget_config",
		}
		entries, err := os.ReadDir(filepath.Join(root, budgetProjectionContractDir))
		if err != nil {
			t.Fatalf("read projection contract dir: %v", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			source := mustRead(t, filepath.Join(root, budgetProjectionContractDir, entry.Name()))
			for _, marker := range forbidden {
				if strings.Contains(source, marker) {
					t.Fatalf("%s introduces budget ledger marker %q", entry.Name(), marker)
				}
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
			filepath.Join(budgetProjectionContractDir, "projection.go"),
			filepath.Join(budgetProjectionContractDir, "benchmark.go"),
			filepath.Join(budgetProjectionContractDir, "fixture.go"),
			filepath.Join("tool", "diagnosticsreplay", "budget_projection.go"),
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
		path := filepath.Join(root, filepath.FromSlash(budgetProjectionFixturePath))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("read budget projection fixture: %v", err)
		}
		if info.Size() > 2<<20 {
			t.Fatalf("budget projection fixture exceeds 2 MiB: %d bytes", info.Size())
		}

		raw := mustRead(t, path)
		var document struct {
			Version string `json:"version"`
			Cases   []struct {
				CaseID       string   `json:"case_id"`
				DeclaredGaps []string `json:"declared_gaps"`
			} `json:"cases"`
			Benchmarks []struct {
				BenchmarkID string `json:"benchmark_id"`
			} `json:"benchmarks"`
		}
		if err := json.Unmarshal([]byte(raw), &document); err != nil {
			t.Fatalf("decode budget projection fixture: %v", err)
		}
		if document.Version != budgetProjectionFixtureVersion {
			t.Fatalf("fixture version drifted: got %q want %q", document.Version, budgetProjectionFixtureVersion)
		}
		if len(document.Cases) == 0 {
			t.Fatal("fixture contains no projection cases")
		}
		if len(document.Benchmarks) == 0 {
			t.Fatal("fixture contains no benchmark cases")
		}
		for _, item := range document.Cases {
			if strings.TrimSpace(item.CaseID) == "" {
				t.Fatal("fixture contains a case without an id")
			}
			for _, gap := range item.DeclaredGaps {
				if !slices.Contains(budgetProjectionDriftCodes, gap) {
					t.Fatalf("case %q declares unknown gap %q", item.CaseID, gap)
				}
			}
		}
	})

	t.Run("taxonomy is documented in context README", func(t *testing.T) {
		doc := mustRead(t, filepath.Join(root, "context", "README.md"))
		for _, code := range budgetProjectionDriftCodes {
			if !strings.Contains(doc, code) {
				t.Fatalf("context/README.md does not document budget projection classification %q", code)
			}
		}
		if !strings.Contains(doc, budgetProjectionFixtureVersion) {
			t.Fatalf("context/README.md does not document fixture version %q", budgetProjectionFixtureVersion)
		}
	})

	t.Run("gate is wired into the quality gate", func(t *testing.T) {
		for _, relative := range []string{
			filepath.Join("scripts", "check-budget-aware-context-projection-contract.sh"),
			filepath.Join("scripts", "check-budget-aware-context-projection-contract.ps1"),
		} {
			if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
				t.Fatalf("missing gate script %s: %v", filepath.ToSlash(relative), err)
			}
		}
		for _, relative := range []string{
			filepath.Join("scripts", "check-quality-gate.sh"),
			filepath.Join("scripts", "check-quality-gate.ps1"),
		} {
			source := mustRead(t, filepath.Join(root, relative))
			if !strings.Contains(source, "check-budget-aware-context-projection-contract") {
				t.Fatalf("%s does not wire the budget projection gate", filepath.ToSlash(relative))
			}
		}

		shQualityGate := strings.ReplaceAll(mustRead(t, filepath.Join(root, "scripts", "check-quality-gate.sh")), "\r\n", "\n")
		const shInvocation = "echo \"[quality-gate] derived budget projection contract\"\n" +
			"if ! bash scripts/check-budget-aware-context-projection-contract.sh; then"
		if !strings.Contains(shQualityGate, shInvocation) {
			t.Fatalf("scripts/check-quality-gate.sh does not invoke the budget projection gate as a standalone shell command")
		}
	})
}

// assertNoBaymaxImports fails when any non-generated Go file under the given
// package directory imports another Baymax package. The derived projection must
// stay a pure contract layer: no runtime, scheduler, model or observability
// coupling, and therefore no second budget ledger.
func assertNoBaymaxImports(t *testing.T, root string, relativeDir string) {
	t.Helper()
	dir := filepath.Join(root, relativeDir)
	walkErr := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range file.Imports {
			pkg := strings.TrimSpace(strings.Trim(imp.Path.Value, `"`))
			if pkg == "github.com/FelixSeptem/baymax" || strings.HasPrefix(pkg, "github.com/FelixSeptem/baymax/") {
				t.Fatalf("derived projection must stay dependency free: %s imports %q",
					filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))), pkg)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan %s imports failed: %v", filepath.ToSlash(relativeDir), walkErr)
	}
}
