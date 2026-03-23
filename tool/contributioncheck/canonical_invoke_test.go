package contributioncheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalMailboxInvokeEntrypoints(t *testing.T) {
	root := repoRoot(t)

	syncPath := filepath.Join(root, "orchestration", "invoke", "sync.go")
	asyncPath := filepath.Join(root, "orchestration", "invoke", "async.go")
	syncSource := mustRead(t, syncPath)
	asyncSource := mustRead(t, asyncPath)

	if strings.Contains(syncSource, "func InvokeSync(") {
		t.Fatalf("legacy public invoke entrypoint reintroduced: %s", filepath.ToSlash(syncPath))
	}
	if strings.Contains(asyncSource, "func InvokeAsync(") {
		t.Fatalf("legacy public invoke entrypoint reintroduced: %s", filepath.ToSlash(asyncPath))
	}

	activeUse := make([]string, 0)
	testOnlyUse := make([]string, 0)
	docOnlyUse := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".gocache" || name == "vendor" || name == "openspec" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".go":
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			content := string(raw)
			if !strings.Contains(content, "invoke.InvokeSync(") && !strings.Contains(content, "invoke.InvokeAsync(") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}
			if strings.HasSuffix(strings.ToLower(path), "_test.go") {
				testOnlyUse = append(testOnlyUse, filepath.ToSlash(rel))
			} else {
				activeUse = append(activeUse, filepath.ToSlash(rel))
			}
			return nil
		case ".md":
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			content := string(raw)
			if !strings.Contains(content, "InvokeSync") && !strings.Contains(content, "InvokeAsync") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}
			docOnlyUse = append(docOnlyUse, filepath.ToSlash(rel))
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		t.Fatalf("scan repository failed: %v", err)
	}

	t.Logf("legacy invoke symbol classification: active=%d test_only=%d doc_only=%d", len(activeUse), len(testOnlyUse), len(docOnlyUse))

	if len(activeUse) > 0 {
		t.Fatalf("legacy direct invoke usage detected in active code paths: %#v", activeUse)
	}
}
