package evalcontract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFirstErrorAttributionImplementationRemainsPureAndOffline(t *testing.T) {
	path := filepath.Join("first_error_attribution.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatal(err)
	}

	allowedImports := map[string]bool{
		"encoding/json": true,
		"fmt":           true,
		"reflect":       true,
		"sort":          true,
		"strings":       true,
		"unicode":       true,
	}
	for _, imported := range parsed.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if !allowedImports[path] {
			t.Fatalf("first-error attribution imported side-effect-capable dependency %q", path)
		}
	}

	forbiddenCalls := map[string]bool{
		"Open": true, "Create": true, "WriteFile": true, "Remove": true, "Rename": true,
		"Command": true, "Dial": true, "Do": true, "Run": true, "Stream": true,
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch function := call.Fun.(type) {
		case *ast.Ident:
			if forbiddenCalls[function.Name] {
				t.Errorf("first-error attribution calls forbidden function %s", function.Name)
			}
		case *ast.SelectorExpr:
			if forbiddenCalls[function.Sel.Name] {
				t.Errorf("first-error attribution calls forbidden method %s", function.Sel.Name)
			}
		}
		return true
	})

	for _, forbidden := range []string{
		"raw_reasoning", "transcript_body", "provider_response", "tool_output", "memory_body",
		"workspace_body", "credential", "mcp/http", "mcp/stdio", "model/", "runtime restore",
	} {
		if strings.Contains(strings.ToLower(string(source)), forbidden) {
			t.Errorf("first-error attribution contains forbidden body/dependency marker %q", forbidden)
		}
	}
}
