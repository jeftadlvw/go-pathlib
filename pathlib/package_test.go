package pathlib

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLibraryDoesNotPrint guards that the library never writes to stdout or stderr
// and does not import "testing" outside of tests. A test cannot observe output
// that is suppressed while testing, so the source is checked instead.
func TestLibraryDoesNotPrint(t *testing.T) {
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)

	fileSet := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		parsed, err := parser.ParseFile(fileSet, file, nil, parser.SkipObjectResolution)
		require.NoError(t, err)

		for _, imp := range parsed.Imports {
			require.NotEqual(t, `"testing"`, imp.Path.Value, "%s imports testing", file)
		}

		ast.Inspect(parsed, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "os" && (selector.Sel.Name == "Stdout" || selector.Sel.Name == "Stderr") {
				t.Errorf("%s uses os.%s", fileSet.Position(selector.Pos()), selector.Sel.Name)
			}
			if ok && pkg.Name == "fmt" && strings.HasPrefix(selector.Sel.Name, "Print") {
				t.Errorf("%s uses fmt.%s", fileSet.Position(selector.Pos()), selector.Sel.Name)
			}
			return true
		})
	}
}
