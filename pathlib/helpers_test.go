package pathlib

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestCase[I any, E any] struct {
	Name   string
	Input  I
	Expect E
	Error  bool
}

func platformNativeUNC(posixForm string) string {
	if runningOnWindows {
		return toWindowsSeparators(posixForm)
	}
	return posixForm
}

// onWindows returns windowsVal on Windows, posixVal on other platforms.
func onWindows[T any](posixVal, windowsVal T) T {
	if runningOnWindows {
		return windowsVal
	}
	return posixVal
}

func runForResultsE[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expect E, expectError bool)) {
	for _, test := range cases {

		caseName := test.Name
		if strings.TrimSpace(caseName) == "" {
			caseName = fmt.Sprintf("case--\"%v\"", test.Input)
		}

		t.Run(fmt.Sprint(caseName), func(t *testing.T) {
			testFunc(t, test.Input, test.Expect, test.Error)
		})
	}
}

func runForResults[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expectError E)) {
	runForResultsE(t, cases, func(t *testing.T, input I, expect E, error bool) {
		testFunc(t, input, expect)
	})
}

func setupTempDir(t *testing.T) *Path {
	tempDir := t.TempDir()
	return NewPath(tempDir)
}

// writeTempFile creates a file with content inside a given root.
// Returns the Path to the created file.
func writeTempFile(t *testing.T, root *Path, relPath string, content string) *Path {
	filePath := root.JoinStrings(relPath)

	err := os.MkdirAll(filePath.Parent().String(), 0755) // Ensure parent directories exist
	require.NoError(t, err)

	err = os.WriteFile(filePath.String(), []byte(content), 0644)
	require.NoError(t, err)

	return filePath
}

// createTempDir creates a directory inside a given root.
// Returns the Path to the created directory.
func createTempDir(t *testing.T, root *Path, relPath string) *Path {
	dirPath := root.JoinStrings(relPath)
	err := os.MkdirAll(dirPath.String(), 0755)
	require.NoError(t, err)
	return dirPath
}

// createTempSymlinkAbs creates a symlink inside a given root.
// targetRelPath is relative to the root.
// linkRelPath is relative to the root.
// Both paths are made absolute for symlink creation.
// Returns the Path to the created symlink.
func createTempSymlinkAbs(t *testing.T, root *Path, targetRelPath string, linkRelPath string) *Path {
	targetPath := root.JoinStrings(targetRelPath)
	linkPath := root.JoinStrings(linkRelPath)
	err := os.MkdirAll(linkPath.Parent().String(), 0755) // Ensure parent dir exists for the link
	require.NoError(t, err)

	// Create absolute symlinks
	err = os.Symlink(targetPath.String(), linkPath.String())
	require.NoError(t, err)
	return linkPath
}

// createTempSymlinkRel creates a symlink inside a given root.
// targetRelPath is relative to the root.
// linkRelPath is relative to the root.
// Only linkRelPath is made absolute for symlink creation. targetRelPath is kept relative.
// Returns the Path to the created symlink.
func createTempSymlinkRel(t *testing.T, root *Path, targetRelPath string, linkRelPath string) *Path {
	targetPath := NewPath(targetRelPath)
	linkPath := root.JoinStrings(linkRelPath)
	err := os.MkdirAll(linkPath.Parent().String(), 0755) // Ensure parent dir exists for the link
	require.NoError(t, err)

	// Create relative symlink
	err = os.Symlink(targetPath.String(), linkPath.String())
	require.NoError(t, err)
	return linkPath
}

// readDirEntries reads all entries (files and directories) recursively within a given Path,
// returning their paths relative to the `root` Path, sorted.
func readDirEntries(t *testing.T, root *Path, dir *Path) []string {
	var entries []string
	if !dir.Exists() {
		return entries
	}
	err := filepath.WalkDir(dir.String(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir.String() && path == root.String() { // Skip the root directory itself if it's the walk start
			return nil
		}
		if path == dir.String() && path != root.String() { // Include the walked directory itself if it's not the root
			relPath, _ := filepath.Rel(root.String(), path)
			entries = append(entries, filepath.ToSlash(relPath))
			return nil
		}
		if path != dir.String() { // For actual children
			relPath, err := filepath.Rel(root.String(), path)
			require.NoError(t, err)
			entries = append(entries, filepath.ToSlash(relPath))
		}
		return nil
	})
	require.NoError(t, err)
	slices.Sort(entries)
	return entries
}

// relPathsSorted collects paths relative to base, sorts them, and returns the sorted slice.
func relPathsSorted(t *testing.T, entries []*Path, base *Path) []string {
	relPaths := make([]string, len(entries))
	for i, e := range entries {
		rel, err := e.RelativeTo(base)
		require.NoError(t, err)
		relPaths[i] = rel.ToPosix()
	}

	slices.Sort(relPaths)
	return relPaths
}
