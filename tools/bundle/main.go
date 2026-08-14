// Command bundle merges the multi-file pathlib source tree into single-file,
// drop-in artifacts under dist/, one per release tier defined in bundle.manifest.
//
// It deliberately uses the pathlib library itself for every path and file
// operation (globbing sources, reading the manifest/license, creating the dist
// tree, writing artifacts) so the build doubles as a real-world exercise of the
// library it ships.
package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jeftadlvw/go-pathlib/pathlib"
)

// Manifest is the bundle.manifest schema: groups (a prefix + dependency edges),
// the release artifacts to emit, and how to render the license preamble.
type Manifest struct {
	Package     string            `json:"package"`
	SourceDir   string            `json:"source_dir"`
	DistDir     string            `json:"dist_dir"`
	License     string            `json:"license"`
	LicenseMode string            `json:"license_mode"` // "full" | "spdx"
	LicenseFile string            `json:"license_file"`
	Groups      map[string]string `json:"groups"` // group name -> filename prefix
	Artifacts   []Artifact        `json:"artifacts"`
}

// Artifact is one emitted single-file bundle: the union of its Groups' files,
// written to <dist>/<Name>/pathlib.go.
type Artifact struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
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

		fmt.Printf("bundled %-5s  %2d files  %6d bytes  ->  %s  (groups: %s)\n",
			art.Name, len(files), len(content), outFile.String(), strings.Join(art.Groups, "+"))
	}
}

// gatherFiles globs the non-test source files for every named group, de-duplicated
// and sorted for reproducible output.
func gatherFiles(srcDir *pathlib.Path, m *Manifest, groups []string) ([]*pathlib.Path, error) {
	seen := map[string]bool{}
	var files []*pathlib.Path

	for _, g := range groups {
		prefix, ok := m.Groups[g]
		if !ok {
			return nil, fmt.Errorf("unknown group %q", g)
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

	sort.Slice(files, func(i, j int) bool { return files[i].Base() < files[j].Base() })
	return files, nil
}

// merge concatenates the bodies of all files (verbatim, after their package
// clause and imports), unions their imports, and prepends the license preamble
// and package doc. The whole-file granularity guarantees every unioned import is
// still used, so no import pruning is needed.
func merge(files []*pathlib.Path, m *Manifest) (string, error) {
	fset := token.NewFileSet()
	imports := map[string]string{} // import path (quoted) -> alias ("" if none)
	pkgDoc := ""
	var bodies []string

	for _, fp := range files {
		src, err := pathlib.ReadFileToString(fp)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fp.Base(), err)
		}
		if strings.Contains(src, "//go:build") || strings.Contains(src, "// +build") {
			return "", fmt.Errorf("%s carries build constraints, cannot merge into a single file", fp.Base())
		}

		af, err := parser.ParseFile(fset, fp.String(), src, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return "", fmt.Errorf("parse %s: %w", fp.Base(), err)
		}

		for _, imp := range af.Imports {
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			imports[imp.Path.Value] = alias
		}

		// The package doc lives above the package clause in exactly one file.
		if af.Doc != nil && pkgDoc == "" {
			pkgDoc = src[offset(fset, af.Doc.Pos()):offset(fset, af.Doc.End())]
		}

		// Everything after the import block (or the package clause, if no
		// imports) is the body, taken verbatim so comments stay byte-identical.
		var bodyStart int
		n := len(af.Decls)
		if n > 0 {
			bodyStart = offset(fset, af.Decls[n-1].End())
		} else {
			bodyStart = offset(fset, af.Name.End())
		}
		bodies = append(bodies, strings.TrimSpace(src[bodyStart:]))
	}

	var b strings.Builder
	b.WriteString(preamble(m))
	b.WriteString("\n")
	if pkgDoc != "" {
		b.WriteString(pkgDoc)
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "package %s\n\n", m.Package)

	paths := make([]string, 0, len(imports))
	for p := range imports {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if len(paths) > 0 {
		b.WriteString("import (\n")
		for _, p := range paths {
			alias := imports[p]
			if alias != "" {
				fmt.Fprintf(&b, "\t%s %s\n", alias, p)
			} else {
				fmt.Fprintf(&b, "\t%s\n", p)
			}
		}
		b.WriteString(")\n")
	}

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

// preamble builds the generated-code marker plus the license header: the full
// LICENSE text in "full" mode (falling back to the SPDX identifier if the file
// is missing), or just the SPDX identifier in "spdx" mode.
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

// writeArtifact creates the artifact's directory and file (via pathlib) and
// writes the bundled source.
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
