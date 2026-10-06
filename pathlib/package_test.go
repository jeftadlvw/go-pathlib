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
	t.Parallel()

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

			name, prints := printingSelector(selector)
			if prints {
				t.Errorf("%s uses %s", fileSet.Position(selector.Pos()), name)
			}
			return true
		})
	}
}

// printingSelector returns the qualified name of selector and whether it writes
// to stdout or stderr, as os.Stdout or fmt.Println do.
func printingSelector(selector *ast.SelectorExpr) (string, bool) {
	pkg, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	name := pkg.Name + "." + selector.Sel.Name
	switch pkg.Name {
	case "os":
		return name, selector.Sel.Name == "Stdout" || selector.Sel.Name == "Stderr"
	case "fmt":
		return name, strings.HasPrefix(selector.Sel.Name, "Print")
	default:
		return name, false
	}
}
