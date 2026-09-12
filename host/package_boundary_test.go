package host

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

func TestHostPackagesKeepTransportAndRuntimeOwnershipBoundaries(t *testing.T) {
	t.Parallel()

	for _, directory := range []string{".", "jsonl"} {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatalf("read %s: %v", directory, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			filename := filepath.Join(directory, entry.Name())
			file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", filename, err)
			}
			for _, spec := range file.Imports {
				path, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Fatalf("unquote import in %s: %v", filename, err)
				}
				for _, forbidden := range []string{
					"github.com/FelixSeptem/baymax/runtime/diagnostics",
					"github.com/FelixSeptem/baymax/mcp/",
					"github.com/FelixSeptem/baymax/model/",
					"database/sql",
					"net/http",
					"github.com/gorilla/websocket",
				} {
					if path == strings.TrimSuffix(forbidden, "/") || strings.HasPrefix(path, forbidden) {
						t.Errorf("%s imports forbidden owner/transport dependency %q", filepath.ToSlash(filename), path)
					}
				}
			}
			for _, declaration := range file.Decls {
				general, ok := declaration.(*ast.GenDecl)
				if !ok || general.Tok != token.VAR {
					continue
				}
				for _, raw := range general.Specs {
					value := raw.(*ast.ValueSpec)
					if isMutableRegistryType(value.Type) || containsMutableRegistryValue(value.Values) {
						t.Errorf("%s declares package-global mutable map/channel %q; host correlation must remain connection-scoped", filepath.ToSlash(filename), value.Names[0].Name)
					}
				}
			}
		}
	}
}

func isMutableRegistryType(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.MapType, *ast.ChanType:
		return true
	default:
		return false
	}
}

func containsMutableRegistryValue(values []ast.Expr) bool {
	for _, value := range values {
		switch expression := value.(type) {
		case *ast.CompositeLit:
			if isMutableRegistryType(expression.Type) {
				return true
			}
		case *ast.CallExpr:
			identifier, ok := expression.Fun.(*ast.Ident)
			if ok && identifier.Name == "make" && len(expression.Args) > 0 && isMutableRegistryType(expression.Args[0]) {
				return true
			}
		}
	}
	return false
}
