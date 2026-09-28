package diagnose

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Diagnose no longer needs ADR-0008's temporary shell exception. Guard both
// imports and argv construction so future checks use the typed boundaries.
func TestProductionCommandBoundaries(t *testing.T) {
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		pkg := filepath.Base(filepath.Dir(path))
		if pkg == "shell" || pkg == "shelltest" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		boundary := pkg == "git" || pkg == "pr"
		for _, imp := range file.Imports {
			name, _ := strconv.Unquote(imp.Path.Value)
			if name == "github.com/victorhsb/branchless-pr/internal/shell" && !boundary && pkg != "cli" && pkg != "invocation" {
				t.Errorf("%s imports shell outside a command boundary or composition root", path)
			}
			if name == "os/exec" && pkg != "pr" {
				t.Errorf("%s imports os/exec outside shell or the gh installation check", path)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if literal, ok := node.(*ast.CompositeLit); ok && !boundary {
				if typ, ok := literal.Type.(*ast.ArrayType); ok {
					if elt, ok := typ.Elt.(*ast.Ident); ok && elt.Name == "string" && len(literal.Elts) > 0 {
						if first, ok := literal.Elts[0].(*ast.BasicLit); ok {
							value, _ := strconv.Unquote(first.Value)
							if value == "git" || value == "gh" {
								t.Errorf("%s constructs raw %s argv", path, value)
							}
						}
					}
				}
			}
			if call, ok := node.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if recv, ok := sel.X.(*ast.Ident); ok && recv.Name == "exec" && sel.Sel.Name != "LookPath" {
						t.Errorf("%s invokes exec.%s outside shell", path, sel.Sel.Name)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
