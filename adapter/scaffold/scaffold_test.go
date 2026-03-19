package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestBuildPlanDeterministic(t *testing.T) {
	base := t.TempDir()
	opts := Options{
		Type:    TypeMCP,
		Name:    "deterministic",
		Output:  filepath.Join(base, "out"),
		BaseDir: base,
	}

	plan1, err := BuildPlan(opts)
	if err != nil {
		t.Fatalf("build plan #1: %v", err)
	}
	plan2, err := BuildPlan(opts)
	if err != nil {
		t.Fatalf("build plan #2: %v", err)
	}

	if plan1.OutputDir != plan2.OutputDir {
		t.Fatalf("output dir mismatch: %s vs %s", plan1.OutputDir, plan2.OutputDir)
	}
	if !reflect.DeepEqual(plan1.Files, plan2.Files) {
		t.Fatal("generated plan files are not deterministic")
	}
	if len(plan1.Conflicts) != 0 || len(plan2.Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %#v %#v", plan1.Conflicts, plan2.Conflicts)
	}
}

func TestDefaultOutputPathWhenOutputOmitted(t *testing.T) {
	base := t.TempDir()
	plan, err := BuildPlan(Options{
		Type:    TypeTool,
		Name:    "default-path",
		BaseDir: base,
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	want := filepath.Join(base, "examples", "adapters", "tool-default-path")
	if plan.OutputDir != want {
		t.Fatalf("default output mismatch: got=%s want=%s", plan.OutputDir, want)
	}
}

func TestGenerateConflictFailFastNoPartialWrite(t *testing.T) {
	base := t.TempDir()
	output := filepath.Join(base, "out")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	readmePath := filepath.Join(output, "README.md")
	const original = "existing readme"
	if err := os.WriteFile(readmePath, []byte(original), 0o600); err != nil {
		t.Fatalf("seed conflict file: %v", err)
	}

	_, err := Generate(Options{
		Type:    TypeModel,
		Name:    "conflict",
		Output:  output,
		BaseDir: base,
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !IsConflictError(err) {
		t.Fatalf("expected conflict error, got: %v", err)
	}

	raw, readErr := os.ReadFile(readmePath)
	if readErr != nil {
		t.Fatalf("read conflict file: %v", readErr)
	}
	if string(raw) != original {
		t.Fatalf("conflict file was unexpectedly overwritten: %q", string(raw))
	}
	if _, statErr := os.Stat(filepath.Join(output, "adapter.go")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no partial write for adapter.go, stat err=%v", statErr)
	}
}

func TestGenerateForceOverwrite(t *testing.T) {
	base := t.TempDir()
	output := filepath.Join(base, "out")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	if err := os.WriteFile(filepath.Join(output, "README.md"), []byte("stale"), 0o600); err != nil {
		t.Fatalf("seed readme: %v", err)
	}

	plan, err := Generate(Options{
		Type:    TypeTool,
		Name:    "force",
		Output:  output,
		Force:   true,
		BaseDir: base,
	})
	if err != nil {
		t.Fatalf("generate with force: %v", err)
	}
	if len(plan.Files) != 4 {
		t.Fatalf("unexpected generated file count: %d", len(plan.Files))
	}

	raw, readErr := os.ReadFile(filepath.Join(output, "README.md"))
	if readErr != nil {
		t.Fatalf("read generated readme: %v", readErr)
	}
	if !strings.Contains(string(raw), "tool-invoke-fail-fast") {
		t.Fatalf("readme missing expected mapping hint: %s", string(raw))
	}
}

func TestBuildPlanIncludesCategoryBootstrapHints(t *testing.T) {
	root := repoRoot(t)
	testCases := []struct {
		scaffoldType string
		name         string
		scenarioID   string
		categoryRef  string
	}{
		{scaffoldType: TypeMCP, name: "hintmcp", scenarioID: "mcp-normalization-fail-fast", categoryRef: "CategoryMCP"},
		{scaffoldType: TypeModel, name: "hintmodel", scenarioID: "model-run-stream-downgrade", categoryRef: "CategoryModel"},
		{scaffoldType: TypeTool, name: "hinttool", scenarioID: "tool-invoke-fail-fast", categoryRef: "CategoryTool"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scaffoldType, func(t *testing.T) {
			t.Parallel()
			plan, err := BuildPlan(Options{
				Type:    tc.scaffoldType,
				Name:    tc.name,
				BaseDir: root,
				Output:  filepath.Join(root, "examples", "adapters", "tmp-"+tc.name),
			})
			if err != nil {
				t.Fatalf("build plan: %v", err)
			}
			if len(plan.Files) != 4 {
				t.Fatalf("unexpected file count: %d", len(plan.Files))
			}
			if !sort.StringsAreSorted(planFileNames(plan.Files)) {
				t.Fatalf("plan files are not in deterministic order: %#v", plan.Files)
			}

			bootstrap := findFileContent(t, plan.Files, "conformance_bootstrap_test.go")
			if !strings.Contains(bootstrap, tc.scenarioID) {
				t.Fatalf("bootstrap missing scenario mapping hint %q", tc.scenarioID)
			}
			if !strings.Contains(bootstrap, "adapterconformance."+tc.categoryRef) {
				t.Fatalf("bootstrap missing category mapping %q", tc.categoryRef)
			}
		})
	}
}

func TestGeneratedConformanceBootstrapOfflineExecutable(t *testing.T) {
	root := repoRoot(t)
	parent := filepath.Join(root, "examples", "adapters", "_a23-offline-work")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	workDir, err := os.MkdirTemp(parent, "a23-offline-")
	if err != nil {
		t.Fatalf("mkdir temp scaffold dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	for _, scaffoldType := range []string{TypeMCP, TypeModel, TypeTool} {
		output := filepath.Join(workDir, scaffoldType)
		if _, genErr := Generate(Options{
			Type:    scaffoldType,
			Name:    "offline",
			BaseDir: root,
			Output:  output,
		}); genErr != nil {
			t.Fatalf("generate %s scaffold: %v", scaffoldType, genErr)
		}

		cmd := exec.Command("go", "test", ".", "-run", "^TestConformanceBootstrapAlignment$", "-count=1")
		cmd.Dir = output
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("offline bootstrap test failed for %s: %v\n%s", scaffoldType, runErr, string(out))
		}
	}
}

func TestScaffoldDriftFixtures(t *testing.T) {
	root := repoRoot(t)
	fixtureRoot := filepath.Join(root, "integration", "testdata", "adapter-scaffold")
	testCases := []struct {
		scaffoldType string
		name         string
		fixtureDir   string
	}{
		{scaffoldType: TypeMCP, name: "fixture", fixtureDir: "mcp-fixture"},
		{scaffoldType: TypeModel, name: "fixture", fixtureDir: "model-fixture"},
		{scaffoldType: TypeTool, name: "fixture", fixtureDir: "tool-fixture"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scaffoldType, func(t *testing.T) {
			plan, err := BuildPlan(Options{
				Type:    tc.scaffoldType,
				Name:    tc.name,
				BaseDir: root,
				Output:  filepath.Join(root, "examples", "adapters", "drift-check-"+tc.scaffoldType),
			})
			if err != nil {
				t.Fatalf("build plan: %v", err)
			}
			want, err := readFixtureFiles(filepath.Join(fixtureRoot, tc.fixtureDir))
			if err != nil {
				t.Fatalf("read fixture files: %v", err)
			}
			if len(plan.Files) != len(want) {
				t.Fatalf("fixture count mismatch: got=%d want=%d", len(plan.Files), len(want))
			}
			for _, file := range plan.Files {
				expected, ok := want[file.RelativePath]
				if !ok {
					t.Fatalf("unexpected generated file %q for fixture %s", file.RelativePath, tc.fixtureDir)
				}
				if normalizeLineEndings(file.Content) != normalizeLineEndings(expected) {
					t.Fatalf("fixture drift detected for %s/%s", tc.fixtureDir, file.RelativePath)
				}
			}
		})
	}
}

func findFileContent(t *testing.T, files []File, rel string) string {
	t.Helper()
	for _, file := range files {
		if file.RelativePath == rel {
			return file.Content
		}
	}
	t.Fatalf("missing file %s", rel)
	return ""
}

func planFileNames(files []File) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.RelativePath)
	}
	return names
}

func readFixtureFiles(root string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	return files, err
}

func normalizeLineEndings(in string) string {
	return strings.ReplaceAll(in, "\r\n", "\n")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}
