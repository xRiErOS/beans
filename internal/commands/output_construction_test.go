package commands

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"testing"
)

// TestNoDirectResponseLiteralConstruction guards D12: internal/output is the
// only place response shapes are constructed. A command file reaching past
// output's constructors (output.SuccessSingle, output.SuccessMultiple, a
// dedicated internal/output constructor, ...) to build an output.Response{}
// literal by hand is exactly the drift D12 forbids — this scans every
// non-test source file of this package's own AST for that literal.
//
// rename.go, check.go and graphql.go build their own JSON shapes entirely
// outside output.Response (T11/T07, open items outside this container), so
// they naturally pass this guard rather than needing an exemption from it.
func TestNoDirectResponseLiteralConstruction(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		name := fi.Name()
		return len(name) < 8 || name[len(name)-8:] != "_test.go"
	}, 0)
	if err != nil {
		t.Fatalf("parser.ParseDir(.) error = %v", err)
	}

	var violations []string
	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				sel, ok := lit.Type.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || pkgIdent.Name != "output" || sel.Sel.Name != "Response" {
					return true
				}
				pos := fset.Position(lit.Pos())
				violations = append(violations, filename+":"+pos.String())
				return true
			})
		}
	}

	if len(violations) > 0 {
		t.Errorf("output.Response{} constructed directly outside internal/output: %v; use an internal/output constructor instead", violations)
	}
}
