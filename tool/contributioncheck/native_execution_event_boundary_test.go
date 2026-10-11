package contributioncheck

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

// Check executable declarations rather than comments discussing non-goals.
func nativeExecutionBoundary(source string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "adapter.go", source, 0)
	if err != nil {
		return []string{"parse"}
	}
	var violations []string
	for _, item := range file.Imports {
		name, _ := strconv.Unquote(item.Path.Value)
		if name != "github.com/FelixSeptem/baymax/core/types" && (strings.Contains(name, ".") || name == "os" || strings.HasPrefix(name, "net") || strings.HasPrefix(name, "database")) {
			violations = append(violations, "owner/IO dependency")
		}
	}
	for _, decl := range file.Decls {
		if global, ok := decl.(*ast.GenDecl); ok && global.Tok == token.VAR {
			for _, spec := range global.Specs {
				for _, name := range spec.(*ast.ValueSpec).Names {
					if name.Name != "_" { // A compile-time interface assertion has no state.
						violations = append(violations, "global state")
					}
				}
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.GoStmt, *ast.ChanType:
			violations = append(violations, "worker/queue")
		case *ast.CallExpr:
			if call, ok := value.Fun.(*ast.SelectorExpr); ok {
				switch call.Sel.Name {
				case "Run", "Stream", "Resume", "Cancel", "Retry", "Record", "AppendEvent":
					violations = append(violations, "execution/diagnostic control")
				}
			}
		}
		return true
	})
	return violations
}

func TestNativeExecutionOwnershipBoundary(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"core/types/native_execution_event.go", "host/eventadapter/observer.go", "tool/diagnosticsreplay/native_execution_event.go"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if violations := nativeExecutionBoundary(string(body)); len(violations) != 0 {
			t.Fatalf("%s: %v", name, violations)
		}
	}
	for _, mutation := range []string{
		`package adapter; import "net/http"`,
		`package adapter; import "github.com/FelixSeptem/baymax/core/runner"`,
		`package adapter; var terminalState = "completed"`,
		`package adapter; func emit() { go deliver() }`,
		`package adapter; type queue chan int`,
		`package adapter; func emit() { engine.Resume() }`,
		`package adapter; func emit() { recorder.Record() }`,
	} {
		if len(nativeExecutionBoundary(mutation)) == 0 {
			t.Fatalf("forbidden ownership mutation admitted: %s", mutation)
		}
	}
}
