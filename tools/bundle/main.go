// Command bundle merges the multi-file pathlib source tree into single-file,
// drop-in artifacts under dist/, one per artifact defined in bundle.json.
//
// It deliberately uses the pathlib library itself for every path and file
// operation, such as globbing sources, reading the manifest and the license,
// creating the dist tree, and writing artifacts. So the build doubles as a
// real-world exercise of the library it ships.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/jeftadlvw/go-pathlib/pathlib"
)

var (
	// errUnknownGroup is returned for an artifact that names a group the
	// manifest does not define.
	errUnknownGroup = errors.New("unknown group")
	// errBuildConstraints is returned for a source file with build constraints,
	// which cannot hold for a single-file bundle.
	errBuildConstraints = errors.New("carries build constraints, cannot merge into a single file")
)

// Manifest is the schema of bundle.json. It names the groups with their
// filename prefixes, the artifacts to emit, and how to render the license
// preamble.
type Manifest struct {
	// Package is the package name of every artifact.
	Package string `json:"package"`

	// SourceDir is the directory of the source files.
	SourceDir string `json:"sourceDir"`

	// DistDir is the directory the artifacts are written to.
	DistDir string `json:"distDir"`

	// License is the SPDX identifier of the license.
	License string `json:"license"`

	// LicenseMode is "full" for the whole license text, or "spdx" for its
	// identifier alone.
	LicenseMode string `json:"licenseMode"`

	// LicenseFile is the file of the license text.
	LicenseFile string `json:"licenseFile"`

	// Groups maps each group name to the filename prefix of its files.
	Groups map[string]string `json:"groups"`

	// Artifacts are the bundles to emit.
	Artifacts []Artifact `json:"artifacts"`
}

// Artifact is one emitted single-file bundle. It holds the files of its groups
// and is written to <dist>/<Name>/pathlib.go.
type Artifact struct {
	// Name is the directory of the artifact below the dist directory.
	Name string `json:"name"`

	// Groups are the groups whose files the artifact holds.
	Groups []string `json:"groups"`
}

// source is the part of one source file that goes into a bundle.
type source struct {
	// imports maps each quoted import path to its alias, or "" for none.
	imports map[string]string
	// doc is the package doc of the file, or "" if it has none.
	doc string
	// body is everything after the import block, verbatim.
	body string
}

func main() {
	manifestPath := "bundle.json"
	if len(os.Args) > 1 {
		manifestPath = os.Args[1]
	}

	raw, err := pathlib.ReadFileToString(pathlib.NewPath(manifestPath))
	if err != nil {
		log.Fatalf("read manifest: %v", err)
	}

	var m Manifest
	err = json.Unmarshal([]byte(raw), &m)
	if err != nil {
		log.Fatalf("parse manifest: %v", err)
	}

	srcDir := pathlib.NewPath(m.SourceDir)
	distDir := pathlib.NewPath(m.DistDir)

	for _, art := range m.Artifacts {
		files, err := gatherFiles(srcDir, &m, art.Groups)
		if err != nil {
			log.Fatalf("artifact %q: gather files: %v", art.Name, err)
		}
		if len(files) == 0 {
			log.Fatalf("artifact %q: no source files matched groups %v", art.Name, art.Groups)
		}

		content, err := merge(files, &m)
		if err != nil {
			log.Fatalf("artifact %q: %v", art.Name, err)
		}

		outFile := distDir.JoinStrings(art.Name, "pathlib.go")
		err = writeArtifact(outFile, content)
		if err != nil {
			log.Fatalf("artifact %q: write: %v", art.Name, err)
		}

		//nolint:forbidigo // The command reports its progress on stdout.
		fmt.Printf("bundled %-5s  %2d files  %6d bytes  ->  %s  (groups: %s)\n",
			art.Name, len(files), len(content), outFile.String(), strings.Join(art.Groups, "+"))
	}
}

// gatherFiles globs the non-test source files of every named group,
// de-duplicated and sorted for reproducible output.
func gatherFiles(srcDir *pathlib.Path, m *Manifest, groups []string) ([]*pathlib.Path, error) {
	seen := map[string]bool{}
	var files []*pathlib.Path

	for _, g := range groups {
		prefix, ok := m.Groups[g]
		if !ok {
			return nil, fmt.Errorf("%w %q", errUnknownGroup, g)
		}

		pattern := prefix + "*.go"
		matches, err := srcDir.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", pattern, err)
		}

		for _, match := range matches {
			base := match.Base()
			if strings.HasSuffix(base, "_test.go") || seen[base] {
				continue
			}
			seen[base] = true
			files = append(files, match)
		}
	}

	slices.SortFunc(files, func(a, b *pathlib.Path) int { return strings.Compare(a.Base(), b.Base()) })
	return files, nil
}

// merge concatenates the bodies of all files, which follow their package
// clause and imports verbatim. It unions their imports and prepends the license
// preamble and the package doc. Whole files are merged, so every unioned import
// is still used, and no import needs pruning.
func merge(files []*pathlib.Path, m *Manifest) (string, error) {
	fset := token.NewFileSet()
	imports := map[string]string{} // import path (quoted) -> alias ("" if none)
	pkgDoc := ""
	var bodies []string

	for _, fp := range files {
		src, err := parseSource(fset, fp)
		if err != nil {
			return "", err
		}

		for path, alias := range src.imports {
			imports[path] = alias
		}

		// The package doc lives above the package clause in exactly one file.
		if pkgDoc == "" {
			pkgDoc = src.doc
		}

		bodies = append(bodies, src.body)
	}

	var b strings.Builder
	b.WriteString(preamble(m))
	b.WriteString("\n")
	if pkgDoc != "" {
		b.WriteString(pkgDoc)
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "package %s\n\n", m.Package)

	writeImports(&b, imports)

	for _, body := range bodies {
		b.WriteString("\n")
		b.WriteString(body)
		b.WriteString("\n")
	}

	out, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", fmt.Errorf("gofmt merged output: %w", err)
	}
	return string(out), nil
}

// parseSource reads the source file fp and splits it into its imports, package
// doc, and body. A file with build constraints returns errBuildConstraints.
func parseSource(fset *token.FileSet, fp *pathlib.Path) (source, error) {
	src, err := pathlib.ReadFileToString(fp)
	if err != nil {
		return source{}, fmt.Errorf("read %s: %w", fp.Base(), err)
	}
	if strings.Contains(src, "//go:build") || strings.Contains(src, "// +build") {
		return source{}, fmt.Errorf("%s %w", fp.Base(), errBuildConstraints)
	}

	af, err := parser.ParseFile(fset, fp.String(), src, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return source{}, fmt.Errorf("parse %s: %w", fp.Base(), err)
	}

	parsed := source{imports: map[string]string{}}
	for _, imp := range af.Imports {
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		parsed.imports[imp.Path.Value] = alias
	}

	if af.Doc != nil {
		parsed.doc = src[offset(fset, af.Doc.Pos()):offset(fset, af.Doc.End())]
	}

	// Everything after the import block (or the package clause, if no
	// imports) is the body, taken verbatim so comments stay byte-identical.
	bodyStart := offset(fset, af.Name.End())
	n := len(af.Decls)
	if n > 0 {
		bodyStart = offset(fset, af.Decls[n-1].End())
	}
	parsed.body = strings.TrimSpace(src[bodyStart:])

	return parsed, nil
}

// writeImports writes the import block for imports to b, sorted by path. It
// writes nothing if there are no imports.
func writeImports(b *strings.Builder, imports map[string]string) {
	if len(imports) == 0 {
		return
	}

	paths := make([]string, 0, len(imports))
	for p := range imports {
		paths = append(paths, p)
	}
	slices.Sort(paths)

	b.WriteString("import (\n")
	for _, p := range paths {
		alias := imports[p]
		if alias != "" {
			fmt.Fprintf(b, "\t%s %s\n", alias, p)
		} else {
			fmt.Fprintf(b, "\t%s\n", p)
		}
	}
	b.WriteString(")\n")
}

// preamble builds the generated-code marker and the license header. The header
// is the full license text in "full" mode, and the SPDX identifier in "spdx"
// mode or for a missing license file.
func preamble(m *Manifest) string {
	var b strings.Builder
	b.WriteString("//\n")
	b.WriteString("// Code generated by go-pathlib's tools/bundle. DO NOT EDIT.\n")
	b.WriteString("//\n")
	b.WriteString("\n")

	if m.LicenseMode == "full" {
		text, err := pathlib.ReadFileToString(pathlib.NewPath(m.LicenseFile))
		if err == nil && strings.TrimSpace(text) != "" {
			b.WriteString("//\n")
			for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
				if line == "" {
					b.WriteString("//\n")
				} else {
					fmt.Fprintf(&b, "// %s\n", line)
				}
			}
			return b.String()
		}
		log.Printf("license: %q unavailable, falling back to SPDX identifier", m.LicenseFile)
	}

	fmt.Fprintf(&b, "// SPDX-License-Identifier: %s\n", m.License)
	return b.String()
}

// writeArtifact creates the directory and the file of the artifact with
// pathlib and writes the bundled source.
func writeArtifact(outFile *pathlib.Path, content string) error {
	_, err := pathlib.MkDirWithOptions(outFile.Parent(), pathlib.DirOptions{ExistOk: true, CreateAll: true})
	if err != nil {
		return err
	}

	_, err = pathlib.CreateFileWithOptions(outFile, pathlib.FileOptions{ExistOk: true})
	if err != nil {
		return err
	}
	_, err = pathlib.WriteString(outFile, content)
	return err
}

// offset converts a token.Pos to a byte offset in its file.
func offset(fset *token.FileSet, pos token.Pos) int {
	return fset.Position(pos).Offset
}
