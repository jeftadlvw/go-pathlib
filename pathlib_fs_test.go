package pathlib

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Helper functions for filesystem tests

// setupTempDir takes a testing.T and returns a Path representing the root of the temporary directory.
// t.TempDir() automatically cleans up the directory after the test.
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

func TestPath_Resolve(t *testing.T) {
	type Input struct {
		PathStr string
		Setup   func(*testing.T, *Path) *Path // Function to set up the path in the temp dir
	}

	cases := []TestCase[Input, string]{
		{
			Name:   "Regular file",
			Input:  Input{PathStr: "file.txt", Setup: func(t *testing.T, root *Path) *Path { return writeTempFile(t, root, "file.txt", "content") }},
			Expect: "file.txt",
			Error:  false,
		},
		{
			Name:   "Directory",
			Input:  Input{PathStr: "dir", Setup: func(t *testing.T, root *Path) *Path { return createTempDir(t, root, "dir") }},
			Expect: "dir",
			Error:  false,
		},
		{
			Name:  "Non-existent path",
			Input: Input{PathStr: "nonexistent", Setup: func(t *testing.T, root *Path) *Path { return root.JoinStrings("nonexistent") }},
			Error: true,
		},
		{
			Name: "Symlink to file",
			Input: Input{
				PathStr: "linkToFile.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "target.txt", "content")
					return createTempSymlinkAbs(t, root, "target.txt", "linkToFile.txt")
				},
			},
			Expect: "target.txt",
			Error:  false,
		},
		{
			Name: "Symlink to directory",
			Input: Input{
				PathStr: "linkToDir",
				Setup: func(t *testing.T, root *Path) *Path {
					createTempDir(t, root, "targetDir")
					return createTempSymlinkAbs(t, root, "targetDir", "linkToDir")
				},
			},
			Expect: "targetDir",
			Error:  false,
		},
		{
			Name: "Chained symlinks",
			Input: Input{
				PathStr: "link1",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "final.txt", "content")
					createTempSymlinkAbs(t, root, "final.txt", "link2")
					return createTempSymlinkAbs(t, root, "link2", "link1")
				},
			},
			Expect: "final.txt",
			Error:  false,
		},
		{
			Name: "Broken symlink",
			Input: Input{
				PathStr: "brokenLink",
				Setup: func(t *testing.T, root *Path) *Path {
					return createTempSymlinkAbs(t, root, "nonexistentTarget", "brokenLink")
				},
			},
			Error: true,
		},
		{
			Name: "Relative path should resolve to absolute",
			Input: Input{
				PathStr: "dir/../file.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "file.txt", "content")
					createTempDir(t, root, "dir")
					return root.JoinStrings("dir/../file.txt")
				},
			},
			Expect: "file.txt",
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect string, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)

		resolvedPath, err := p.Resolve()

		if expectError {
			require.Error(t, err)
			require.Nil(t, resolvedPath)
			return
		}

		require.NoError(t, err)
		require.NotNil(t, resolvedPath)

		// Ensure the resolved path is absolute
		require.True(t, resolvedPath.IsAbsolute())

		// Cut "/private" from path so that RelativeTo will succeed
		if runtime.GOOS == "darwin" {
			resolvedPath.path = strings.TrimPrefix(resolvedPath.path, "/private")
		}

		// Compare relative to root for easier testing
		relPath, err := resolvedPath.RelativeTo(root)
		require.NoError(t, err)

		require.Equal(t, expect, relPath.ToPosix())
	})
}

func TestPath_Walk(t *testing.T) {
	type Input struct {
		RootSetup func(*testing.T, *Path) *Path // Function to set up the directory for walking
		WalkFunc  func(p *Path, abort AbortFunc) error
	}

	type Expect struct {
		WalkedPaths []string // Relative paths from the rootSetup
		Error       bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Empty directory",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path { return createTempDir(t, root, "emptyDir") },
				WalkFunc:  func(p *Path, abort AbortFunc) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{}},
		},
		{
			Name: "Directory with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					return dir
				},
				WalkFunc: func(p *Path, abort AbortFunc) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "file2.log"}},
		},
		{
			Name: "Directory with subdirectories (should not recurse)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file.txt", "")
					subdir := createTempDir(t, dir, "subdir")
					writeTempFile(t, subdir, "nested.txt", "") // Should not be walked
					return dir
				},
				WalkFunc: func(p *Path, abort AbortFunc) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file.txt", "subdir"}}, // subdir itself is walked, but not its contents
		},
		{
			Name: "Non-directory path",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path { return writeTempFile(t, root, "file.txt", "") },
				WalkFunc:  func(p *Path, abort AbortFunc) error { return nil },
			},
			Expect: Expect{Error: true},
		},
		{
			Name: "WalkFunc returns error",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.log", "")
					return dir
				},
				WalkFunc: func(p *Path, abort AbortFunc) error {
					return fmt.Errorf("simulated error")
				},
			},
			Expect: Expect{WalkedPaths: []string{}, Error: true},
		},
		{
			Name: "WalkFunc calls abortGlob",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					writeTempFile(t, dir, "file3.csv", "")
					return dir
				},
				WalkFunc: func() func(*Path, AbortFunc) error {
					first := true
					return func(p *Path, abort AbortFunc) error {
						if first {
							first = false
							abort()
						}
						return nil
					}
				}(),
			},
			Expect: Expect{WalkedPaths: []string{}}, // First entry triggers abort; walk stops immediately
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.RootSetup(t, root)

		var walkedEntries []string
		customWalkFunc := func(path *Path, abortFunc AbortFunc) error {
			abort := false
			localAbortFunc := func() {
				abort = true
				abortFunc()
			}

			relPath, err := path.RelativeTo(p)
			require.NoError(t, err)

			var walkFuncErr error
			walkFuncErr = input.WalkFunc(path, localAbortFunc)

			if walkFuncErr == nil && !abort {
				walkedEntries = append(walkedEntries, relPath.ToPosix())
			}

			if walkFuncErr != nil {
				return walkFuncErr
			}

			return nil
		}

		err := p.Walk(customWalkFunc)

		if expect.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		// Ensure consistent order for comparison
		slices.Sort(walkedEntries)
		slices.Sort(expect.WalkedPaths)

		require.Len(t, walkedEntries, len(expect.WalkedPaths))

		if len(expect.WalkedPaths) > 0 {
			require.Equal(t, expect.WalkedPaths, walkedEntries)
		}
	})
}

func TestPath_WalkR(t *testing.T) {
	type Input struct {
		RootSetup   func(*testing.T, *Path) *Path // Function to set up the directory for walking
		CleanupFunc func(*testing.T, *Path)
		WalkRFunc   func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error
	}
	type Expect struct {
		WalkedPaths []string // Relative paths from the rootSetup
		Error       bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Empty directory",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path { return createTempDir(t, root, "emptyDir") },
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{}},
		},
		{
			Name: "Flat directory with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "file2.log"}},
		},
		{
			Name: "Nested directories with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested1.txt", "")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "nested2.log", "")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{
				"file.txt",
				"subdir1",
				"subdir1/nested1.txt",
				"subdir1/subdir2",
				"subdir1/subdir2/nested2.log",
			}},
		},
		{
			Name: "Non-directory initial path",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path { return writeTempFile(t, root, "file.txt", "") },
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error { return nil },
			},
			Expect: Expect{Error: true},
		},
		{
			Name: "WalkRFunc returns error (aborts entire tree)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					subdir := createTempDir(t, dir, "subdir")
					writeTempFile(t, subdir, "file2.log", "")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error {
					if p.Base() == "file2.log" {
						return fmt.Errorf("simulated error")
					}
					return nil
				},
			},
			Expect: Expect{WalkedPaths: []string{"subdir"}, Error: true},
		},
		{
			Name: "WalkRFunc calls abortLocalTree",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested1.txt", "") // Should be skipped
					createTempDir(t, subdir1, "subdir2")         // Should be skipped
					writeTempFile(t, dir, "file3.csv", "")       // Should still be walked
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error {
					if p.Base() == "subdir1" {
						abortLocalTree() // Abort processing subdir1 and its children
					}
					return nil
				},
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "subdir1", "file3.csv"}}, // subdir1 is recorded, but its contents are not recursed
		},
		{
			Name: "WalkRFunc calls abortTree",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					subdir1 := createTempDir(t, dir, "subdir1")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "deep.txt", "") // Should NOT be walked
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error {
					if p.Base() == "subdir2" {
						abortTree() // Abort everything
					}
					return nil
				},
			},
			Expect: Expect{WalkedPaths: []string{"subdir1", "subdir1/subdir2"}},
		},
		{
			Name: "Error reading directory, walkFunc handles (continues)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					dir := createTempDir(t, root, "root")
					createTempDir(t, dir, "readable_dir")
					writeTempFile(t, dir, "readable_dir/file.txt", "")

					unreadableDir := createTempDir(t, dir, "unreadable_dir")
					writeTempFile(t, unreadableDir, "secret.log", "")

					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)

					createTempDir(t, dir, "another_dir")
					writeTempFile(t, dir, "another_dir/another_file.txt", "")
					return dir
				},
				CleanupFunc: func(t *testing.T, root *Path) {
					os.Chmod(root.JoinStrings("root", "unreadable_dir").String(), 0755)
				},
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error {
					if localDirError != nil {
						return nil
					}
					return nil
				},
			},
			Expect: Expect{
				WalkedPaths: []string{
					"another_dir",
					"another_dir/another_file.txt",
					"readable_dir",
					"readable_dir/file.txt",
					"unreadable_dir",
				},
			},
		},
		{
			Name: "Error reading directory, walkFunc returns error (stops)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					dir := createTempDir(t, root, "root")
					createTempDir(t, dir, "readable_dir")
					unreadableDir := createTempDir(t, dir, "unreadable_dir")
					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)
					createTempDir(t, dir, "another_dir")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error {
					if localDirError != nil {
						return fmt.Errorf("custom error in %s: %w", p.ToPosix(), localDirError)
					}
					return nil
				},
			},
			Expect: Expect{
				WalkedPaths: []string{

					// MacOS file walking causes `another_dir` to be walked before
					// `unreadable_dir` and `readable_dir`. `readable_dir` is
					// lexically after `unreadable_dir`, so `readable_dir` is not walked.

					"another_dir",
				},
				Error: true,
			},
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.RootSetup(t, root)

		defer func() {
			if input.CleanupFunc != nil {
				input.CleanupFunc(t, root)
			}
		}()

		var walkedEntries []string
		customWalkRFunc := func(path *Path, localDirError error, abortLocalTree AbortFunc, abortTree AbortFunc) error {
			relPath, err := path.RelativeTo(p)
			if err != nil && !expectError { // Ignore relative path expectError if we expect an overall expectError.
				return err
			}

			var inputWalkFuncErr error = nil

			// Original walkFunc for specific test behavior
			if input.WalkRFunc != nil {
				inputWalkFuncErr = input.WalkRFunc(path, localDirError, abortLocalTree, abortTree)
			}

			if inputWalkFuncErr == nil {
				walkedEntries = append(walkedEntries, relPath.ToPosix())
			}

			return inputWalkFuncErr
		}

		err := p.WalkR(customWalkRFunc)

		if expect.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		if len(expect.WalkedPaths) != 0 && len(walkedEntries) != 0 {
			// Ensure consistent order for comparison
			slices.Sort(expect.WalkedPaths)
			slices.Sort(walkedEntries)
			require.Equal(t, expect.WalkedPaths, walkedEntries)
		} else {
			require.Equal(t, len(expect.WalkedPaths), len(walkedEntries))
		}
	})
}

func TestCreateFileWithOptions(t *testing.T) {
	type Input struct {
		RelPath string
		Options FileOptions
	}
	type Expect struct {
		Created bool
		Content string // To verify if truncated or not
	}

	cases := []TestCase[Input, Expect]{
		{
			Name:   "Successful creation",
			Input:  Input{"new_file.txt", DefaultFileOptions()},
			Expect: Expect{Created: true},
			Error:  false,
		},
		{
			Name:   "ExistOk=true, file exists",
			Input:  Input{"existing_file.txt", FileOptions{ExistOk: true, Mode: DefaultFileMode}},
			Expect: Expect{Created: false, Content: "original content"},
			Error:  false,
		},
		{
			Name:   "ExistOk=false, file exists",
			Input:  Input{"existing_file.txt", FileOptions{ExistOk: false, Mode: DefaultFileMode}},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Path exists and is a directory",
			Input:  Input{"existing_dir", DefaultFileOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Parent directory does exist, file does not exist but ExistOk=false",
			Input:  Input{"existent_dir/file.txt", DefaultFileOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Parent directory does not exist",
			Input:  Input{"non_existent_dir/file.txt", DefaultFileOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Mode=0 should default to DefaultFileMode",
			Input:  Input{"file_with_mode0.txt", FileOptions{ExistOk: false, Mode: 0}},
			Expect: Expect{Created: true},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		targetPath := root.JoinStrings(input.RelPath)

		// Pre-create some paths for specific test cases
		if strings.Contains(input.RelPath, "existing_file.txt") {
			writeTempFile(t, root, "existing_file.txt", "original content")
		}
		if strings.Contains(input.RelPath, "truncate_me.txt") {
			writeTempFile(t, root, "truncate_me.txt", "original content")
		}
		if strings.Contains(input.RelPath, "existing_dir") {
			createTempDir(t, root, "existing_dir")
		}

		created, err := CreateFileWithOptions(targetPath, input.Options)

		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, expect.Created, created)
			require.True(t, targetPath.Exists())
			require.True(t, targetPath.IsFile())

			if targetPath.Exists() { // Only check permissions/content if it was created/exists
				info, err := targetPath.Stat()
				require.NoError(t, err)

				expectedMode := input.Options.Mode
				if expectedMode == 0 {
					expectedMode = DefaultFileMode
				}
				require.Equal(t, effectiveFileMode(expectedMode).Perm(), info.Mode().Perm())

				if expect.Content != "" || (input.RelPath == "truncate_me.txt") {
					// Check content for truncation test, or if specific content is expected
					content, err := os.ReadFile(targetPath.String())
					require.NoError(t, err)
					require.Equal(t, expect.Content, string(content))
				}
			}
		}
	})
}

func TestMkDirWithOptions(t *testing.T) {
	type Input struct {
		RelPath string
		Options DirOptions
	}
	type Expect struct {
		Created bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name:   "Successful creation (single level)",
			Input:  Input{"new_dir", DefaultDirOptions()},
			Expect: Expect{Created: true},
			Error:  false,
		},
		{
			Name:   "ExistOk=true, directory exists",
			Input:  Input{"existing_dir", DirOptions{ExistOk: true, Mode: DefaultDirMode}},
			Expect: Expect{Created: false},
			Error:  false,
		},
		{
			Name:   "ExistOk=false, directory exists",
			Input:  Input{"existing_dir", DirOptions{ExistOk: false, Mode: DefaultDirMode}},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Path exists and is a file",
			Input:  Input{"existing_file.txt", DefaultDirOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "CreateAll=false, parent directory does not exist",
			Input:  Input{"non_existent_parent/new_dir", DefaultDirOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "CreateAll=true, parent directory does not exist",
			Input:  Input{"nested/deeply/created_dir", DirOptions{CreateAll: true, Mode: DefaultDirMode}},
			Expect: Expect{Created: true},
			Error:  false,
		},
		{
			Name:   "Mode=0 should default to DefaultDirMode",
			Input:  Input{"dir_with_mode0", DirOptions{ExistOk: false, Mode: 0}},
			Expect: Expect{Created: true},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		targetPath := root.JoinStrings(input.RelPath)

		// Pre-create some paths for specific test cases
		if strings.Contains(input.RelPath, "existing_dir") {
			createTempDir(t, root, "existing_dir")
		}
		if strings.Contains(input.RelPath, "existing_file.txt") {
			writeTempFile(t, root, "existing_file.txt", "content")
		}

		created, err := MkDirWithOptions(targetPath, input.Options)

		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, expect.Created, created)
			require.True(t, targetPath.Exists())
			require.True(t, targetPath.IsDir())

			info, err := targetPath.Stat()
			require.NoError(t, err)
			expectedMode := input.Options.Mode
			if expectedMode == 0 {
				expectedMode = DefaultDirMode
			}
			require.Equal(t, effectiveDirMode(expectedMode).Perm(), info.Mode().Perm())
		}
	})
}

func TestCopy(t *testing.T) {
	type Input struct {
		SrcRel string
		DstRel string
		Setup  func(*testing.T, *Path) // Setup for src/dst parents
	}

	type Expect struct {
		State []string // Expected final state of the overall root directory (relative paths to root)
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Copy regular file",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/new_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "hello world")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_file.txt", "src", "src/file.txt"},
			},
			Error: false,
		},
		{
			Name: "Copy file to existing file (should error)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/existing_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "src content")
					writeTempFile(t, root, "dst/existing_file.txt", "dst content")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Copy file to existing directory (should error)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst_dir",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "content")
					createTempDir(t, root, "dst_dir")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Copy empty directory",
			Input: Input{
				SrcRel: "src_empty_dir",
				DstRel: "dst/new_empty_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "src_empty_dir")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_empty_dir", "src_empty_dir"},
			},
			Error: false,
		},
		{
			Name: "Copy directory with files",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/new_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file1.txt", "content1")
					writeTempFile(t, srcDir, "file2.log", "content2")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_dir", "dst/new_dir/file1.txt", "dst/new_dir/file2.log", "src_dir", "src_dir/file1.txt", "src_dir/file2.log"},
			},
			Error: false,
		},
		{
			Name: "Copy directory with nested structure",
			Input: Input{
				SrcRel: "src_parent",
				DstRel: "dst/copied_parent",
				Setup: func(t *testing.T, root *Path) {
					srcParent := createTempDir(t, root, "src_parent")
					writeTempFile(t, srcParent, "root_file.txt", "")
					subdir1 := createTempDir(t, srcParent, "subdir1")
					writeTempFile(t, subdir1, "nested_file.md", "")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "deep_file.json", "")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{
					"dst",
					"dst/copied_parent",
					"dst/copied_parent/root_file.txt",
					"dst/copied_parent/subdir1",
					"dst/copied_parent/subdir1/nested_file.md",
					"dst/copied_parent/subdir1/subdir2",
					"dst/copied_parent/subdir1/subdir2/deep_file.json",
					"src_parent",
					"src_parent/root_file.txt",
					"src_parent/subdir1",
					"src_parent/subdir1/nested_file.md",
					"src_parent/subdir1/subdir2",
					"src_parent/subdir1/subdir2/deep_file.json",
				},
			},
			Error: false,
		},
		{
			Name: "Copy directory to existing non-empty directory (should error)",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file.txt", "")
					dstDir := createTempDir(t, root, "dst_dir")
					writeTempFile(t, dstDir, "existing_file.txt", "")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Copy absolute symlink to file",
			Input: Input{
				SrcRel: "link_to_file",
				DstRel: "dst/new_link",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "symlink target content")
					createTempSymlinkAbs(t, root, "target.txt", "link_to_file")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_link", "link_to_file", "target.txt"},
			},
			Error: false,
		},
		{
			Name: "Copy relative symlink to file",
			Input: Input{
				SrcRel: "link_to_file",
				DstRel: "dst/new_link",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "symlink target content")
					createTempSymlinkRel(t, root, "target.txt", "link_to_file")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_link", "link_to_file", "target.txt"},
			},
			Error: false,
		},
		{
			Name: "Copy symlink to directory",
			Input: Input{
				SrcRel: "link_to_dir",
				DstRel: "dst/new_link_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "target_dir")
					createTempSymlinkAbs(t, root, "target_dir", "link_to_dir")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_link_dir", "link_to_dir", "target_dir"},
			},
			Error: false,
		},
		{
			Name: "Source path does not exist",
			Input: Input{
				SrcRel: "non_existent_src",
				DstRel: "dst/target",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Destination parent does not exist",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "non_existent_parent/target.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		input.Setup(t, root)

		srcPath := root.JoinStrings(input.SrcRel)
		dstPath := root.JoinStrings(input.DstRel)

		err := Copy(srcPath, dstPath)

		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)

			// Verify the state of the *entire* root to capture all changes correctly.
			finalRootState := readDirEntries(t, root, root)
			slices.Sort(expect.State) // Ensure expectation is sorted
			require.Equal(t, expect.State, finalRootState)

			// Additional checks for symlinks
			if srcPath.IsSymlink() {
				require.True(t, dstPath.IsSymlink(), "Destination should be a symlink")

				// Test both targets target same file
				originalTarget, err := srcPath.ReadSymlinkTarget()
				require.NoError(t, err)
				originalTarget, err = originalTarget.AbsoluteFrom(srcPath.Parent())
				require.NoError(t, err)

				copiedTarget, err := dstPath.ReadSymlinkTarget()
				require.NoError(t, err)
				copiedTarget, err = copiedTarget.AbsoluteFrom(dstPath.Parent())
				require.NoError(t, err)

				require.Equal(t, originalTarget.ToPosix(), copiedTarget.ToPosix(), "Copied symlink target should match original")
				require.True(t, originalTarget.EqualsFs(copiedTarget), "Symlinks do not point to same file on fs")
			}
		}
	})
}

func TestMove(t *testing.T) {
	type Input struct {
		SrcRel string
		DstRel string
		Setup  func(*testing.T, *Path) // Setup for src/dst parents
	}
	type Expect struct {
		FinalState []string // Expected final state of the overall root directory (relative paths to root)
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Move file to new location",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/moved_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "content")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				FinalState: []string{"dst", "dst/moved_file.txt", "src"}, // src dir remains, but is empty
			},
			Error: false,
		},
		{
			Name: "Move file overwriting existing file (should fail)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/existing_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "new content")
					writeTempFile(t, root, "dst/existing_file.txt", "old content")
				},
			},
			Expect: Expect{FinalState: []string{"dst", "dst/existing_file.txt", "src", "src/file.txt"}},
			Error:  true,
		},
		{
			Name: "Move empty directory to new location",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/moved_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "src_dir")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				FinalState: []string{"dst", "dst/moved_dir"},
			},
			Error: false,
		},
		{
			Name: "Move directory with content to new location",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/moved_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file.txt", "")
					createTempDir(t, srcDir, "subdir")
					writeTempFile(t, srcDir.JoinStrings("subdir"), "nested.log", "")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				FinalState: []string{
					"dst", "dst/moved_dir", "dst/moved_dir/file.txt",
					"dst/moved_dir/subdir", "dst/moved_dir/subdir/nested.log",
				},
			},
			Error: false,
		},
		{
			Name: "Move directory to existing empty directory",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/existing_empty_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file.txt", "")
					createTempDir(t, root, "dst/existing_empty_dir")
				},
			},
			Expect: Expect{
				FinalState: []string{"dst", "dst/existing_empty_dir", "dst/existing_empty_dir/file.txt"},
			},
			Error: false,
		},
		{
			Name: "Source path does not exist",
			Input: Input{
				SrcRel: "non_existent_src",
				DstRel: "dst/target",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{FinalState: []string{"dst"}},
			Error:  true,
		},
		{
			Name: "Destination parent does not exist",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "non_existent_parent/target.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "")
				},
			},
			Expect: Expect{FinalState: []string{"src", "src/file.txt"}},
			Error:  true,
		},
		{
			Name: "Move file to existing non-empty directory (should fail)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst_dir", // This *is* a directory
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "content")
					dstDir := createTempDir(t, root, "dst_dir")
					writeTempFile(t, dstDir, "other_file.txt", "") // Make it non-empty
				},
			},
			Expect: Expect{
				FinalState: []string{"dst_dir", "dst_dir/other_file.txt", "src", "src/file.txt"}, // Source should still exist
			},
			Error: true,
		},
		{
			Name: "Move directory to existing non-empty directory (should fail)",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst_dir", // This *is* a directory
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "src_dir")
					writeTempFile(t, root.JoinStrings("src_dir"), "file.txt", "")
					dstDir := createTempDir(t, root, "dst_dir")
					writeTempFile(t, dstDir, "other_file.txt", "") // Make it non-empty
				},
			},
			Expect: Expect{
				FinalState: []string{"dst_dir", "dst_dir/other_file.txt", "src_dir", "src_dir/file.txt"}, // Source should still exist
			},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		input.Setup(t, root) // Setup initial state

		srcPath := root.JoinStrings(input.SrcRel)
		dstPath := root.JoinStrings(input.DstRel)

		err := Move(srcPath, dstPath)

		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		finalRootState := readDirEntries(t, root, root)
		slices.Sort(expect.FinalState)
		require.Equal(t, expect.FinalState, finalRootState)
	})
}

func TestRemoveAll(t *testing.T) {
	type Input struct {
		RelPath string
		Setup   func(*testing.T, *Path) // Setup for the path to remove
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Remove empty directory",
			Input: Input{
				RelPath: "empty_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "empty_dir")
				},
			},
			Error: false,
		},
		{
			Name: "Remove directory with files",
			Input: Input{
				RelPath: "dir_with_files",
				Setup: func(t *testing.T, root *Path) {
					dir := createTempDir(t, root, "dir_with_files")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
				},
			},
			Error: false,
		},
		{
			Name: "Remove directory with nested structure",
			Input: Input{
				RelPath: "nested_dir",
				Setup: func(t *testing.T, root *Path) {
					dir := createTempDir(t, root, "nested_dir")
					writeTempFile(t, dir, "file.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested.txt", "")
					createTempDir(t, subdir1, "subdir2")
				},
			},
			Error: false,
		},
		{
			Name: "Remove non-existent path (no-op)",
			Input: Input{
				RelPath: "non_existent_path",
				Setup:   func(t *testing.T, root *Path) { /* no setup */ },
			},
			Error: false,
		},
		{
			Name: "Remove a file (should error)",
			Input: Input{
				RelPath: "a_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "a_file.txt", "")
				},
			},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		targetPath := root.JoinStrings(input.RelPath)
		input.Setup(t, root) // Set up the path to be removed

		err := RemoveAll(targetPath)

		if expectError {
			require.Error(t, err)
			// For RemoveAll, if it errors because it's not a directory, the path *should* still exist.
			if err.Error() == "path is not a directory" {
				require.True(t, targetPath.Exists())
			}
		} else {
			require.NoError(t, err)
			require.False(t, targetPath.Exists(), "Path should not exist after RemoveAll")
		}
	})
}

func TestPath_IsOnCaseSensitiveFs(t *testing.T) {
	// This test is tricky because file system case-sensitivity depends on the OS/FS.
	// We'll create a temp directory and use its behavior to determine the expected outcome.
	// The function's internal logic already handles the detection.

	root := setupTempDir(t)

	// Create a test file
	filePath := writeTempFile(t, root, "TestFile.txt", "content")
	mixedCasePath := root.JoinStrings("testfile.txt") // Attempt a different casing

	// Determine expected behavior by trying to access the same inode
	stat1, err1 := filePath.Stat()
	stat2, err2 := mixedCasePath.Stat()

	// The filesystem is case-sensitive if:
	// - the mixed-case path doesn't exist (err2 != nil), or
	// - both exist but point to different inodes
	isFsCaseSensitiveExpected := err1 == nil && (err2 != nil || !os.SameFile(stat1, stat2))

	cases := []TestCase[*Path, bool]{
		{
			Name:   "Test with file having a letter in base",
			Input:  filePath,
			Expect: isFsCaseSensitiveExpected,
		},
		{
			Name: "Test with directory having a letter in base",
			Input: func() *Path {
				dirPath := createTempDir(t, root, "testDir")
				return dirPath
			}(),
			Expect: isFsCaseSensitiveExpected,
		},
		{
			Name: "Test with file having no letters in base (should create temp)",
			Input: func() *Path {
				// Path with no letters, e.g., only numbers or symbols
				noLetterPath := writeTempFile(t, root, "12345.ext", "content")
				return noLetterPath
			}(),
			Expect: isFsCaseSensitiveExpected,
		},
		{
			Name: "Test with non-existent path (should return false)",
			Input: func() *Path {
				return root.JoinStrings("non_existent_path")
			}(),
			Expect: false, // Should always return false for non-existent path
		},
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect bool) {
		actual := input.IsOnCaseSensitiveFs()
		require.Equal(t, expect, actual)
	})
}

func TestPath_SymlinkTo(t *testing.T) {
	type Input struct {
		SrcRel      string
		LinkPathRel string
		Setup       func(*testing.T, *Path) // Setup for src/link parents
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Successful symlink to a file",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "my_link.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "content")
				},
			},
			Error: false,
		},
		{
			Name: "Successful symlink to a directory",
			Input: Input{
				SrcRel:      "target_dir",
				LinkPathRel: "my_link_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "target_dir")
				},
			},
			Error: false,
		},
		{
			Name: "Source path does not exist",
			Input: Input{
				SrcRel:      "non_existent_target.txt",
				LinkPathRel: "my_link.txt",
				Setup:       func(t *testing.T, root *Path) { /* no setup for target */ },
			},
			Error: true,
		},
		{
			Name: "Link path already exists (file)",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "existing_link_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "")
					writeTempFile(t, root, "existing_link_file.txt", "")
				},
			},
			Error: true,
		},
		{
			Name: "Link path already exists (directory)",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "existing_link_dir",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "")
					createTempDir(t, root, "existing_link_dir")
				},
			},
			Error: true,
		},
		{
			Name: "Link path parent directory does not exist",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "non_existent_dir/my_link.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "")
				},
			},
			Error: true,
		},
		{
			Name: "Relative target, relative link (both in root)",
			Input: Input{
				SrcRel:      "dir/target.txt",
				LinkPathRel: "link.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "dir/target.txt", "content")
				},
			},
			Error: false,
		},
		{
			Name: "Relative target, relative link in subdir",
			Input: Input{
				SrcRel:      "target.txt", // target is root/target.txt
				LinkPathRel: "subdir/link.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "content")
					createTempDir(t, root, "subdir")
				},
			},
			Error: false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		input.Setup(t, root)

		srcPath := root.JoinStrings(input.SrcRel)
		linkPath := root.JoinStrings(input.LinkPathRel)

		err := srcPath.SymlinkTo(linkPath)

		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)

			require.True(t, linkPath.Exists(), "Link path should exist")
			require.True(t, linkPath.IsSymlink(), "Link path should be a symlink")

			// Verify the symlink target by reading it
			readTarget, err := os.Readlink(linkPath.String())
			require.NoError(t, err)

			// Compare read link target to lib function
			readTargetLibPath, err := linkPath.ReadSymlinkTarget()
			require.NoError(t, err)
			require.Equal(t, readTarget, readTargetLibPath.String())

			// The target passed to os.Symlink is relative from the link's parent.
			// Reconstruct the expected relative target to compare.
			expectedReadTarget := filepath.Join(filepath.Base(filepath.Dir(srcPath.String())), filepath.Base(srcPath.String()))
			// If target is directly in root, it should be just the base
			if filepath.Dir(srcPath.String()) == root.String() {
				expectedReadTarget = filepath.Base(srcPath.String())
			}

			readTargetPath := NewPath(readTarget)
			var readTargetPathAbsolute *Path
			if readTargetPath.IsAbsolute() {
				readTargetPathAbsolute = readTargetPath.Copy()

				readTargetPath, err = readTargetPath.RelativeTo(root)
				require.NoError(t, err)
			}
			readTarget = readTargetPath.String()

			// Test for equal relative paths
			require.Equal(t, expectedReadTarget, readTarget)

			// Test equal absolute paths
			require.Equal(t, srcPath.String(), readTargetPathAbsolute.String(), "Symlink target read should match absolute source path")
			require.True(t, srcPath.EqualsFs(readTargetPathAbsolute), "Symlink target read should point to same file")
		}
	})
}

func TestPath_ReadSymlinkTarget(t *testing.T) {

	type Input struct {
		SymlinkRel string
		TargetRel  string                  // Only used for setup, defines what the symlink *points to*
		Setup      func(*testing.T, *Path) // Setup for the symlink and target
	}

	type Expect struct {
		TargetRead string // Expected raw string read from symlink
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Path is symlink to file",
			Input: Input{
				SymlinkRel: "link_to_file",
				TargetRel:  "actual_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "actual_file.txt", "")
					createTempSymlinkAbs(t, root, "actual_file.txt", "link_to_file")
				},
			},
			Expect: Expect{
				TargetRead: "actual_file.txt", // Target written as relative
			},
			Error: false,
		},
		{
			Name: "Path is symlink to directory",
			Input: Input{
				SymlinkRel: "link_to_dir",
				TargetRel:  "actual_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "actual_dir")
					createTempSymlinkAbs(t, root, "actual_dir", "link_to_dir")
				},
			},
			Expect: Expect{
				TargetRead: "actual_dir",
			},
			Error: false,
		},
		{
			Name: "Path is broken symlink",
			Input: Input{
				SymlinkRel: "broken_link",
				TargetRel:  "non_existent_target",
				Setup: func(t *testing.T, root *Path) {
					createTempSymlinkAbs(t, root, "non_existent_target", "broken_link")
				},
			},
			Expect: Expect{
				TargetRead: "non_existent_target",
			}, // Readlink still returns target path
			Error: false,
		},
		{
			Name: "Path is not a symlink (regular file)",
			Input: Input{
				SymlinkRel: "regular_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "regular_file.txt", "")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Path is not a symlink (directory)",
			Input: Input{
				SymlinkRel: "regular_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "regular_dir")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Path does not exist",
			Input: Input{
				SymlinkRel: "non_existent_path",
				Setup:      func(t *testing.T, root *Path) { /* no setup */ },
			},
			Expect: Expect{},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, error bool) {
		root := setupTempDir(t)
		input.Setup(t, root)

		symlinkPath := root.JoinStrings(input.SymlinkRel)
		readTargetPath, err := symlinkPath.ReadSymlinkTarget()

		if error {
			require.Error(t, err)
			require.Nil(t, readTargetPath)
		} else {
			require.NoError(t, err)
			require.NotNil(t, readTargetPath)

			require.NotEmpty(t, input.TargetRel, "Test is setup wrongly. No target path defined.")
			targetPath := root.JoinStrings(input.TargetRel)

			// The returned target is a Path created from the raw string from os.Readlink.
			// `createTempSymlinkAbs` uses a relative path (relative to link's parent).
			require.Equal(t, targetPath.ToPosix(), readTargetPath.ToPosix())
		}
	})
}

func TestPath_GlobWithOptions(t *testing.T) {
	type Input struct {
		Pattern string
		Options GlobOptions
		Setup   func(*testing.T, *Path) *Path // Setup returns the path to perform glob on
	}

	type Expect struct {
		FoundPaths []string // Relative paths from the glob root, sorted
	}

	directories := []string{
		"data",
		"data/logs",
		"data/docs",
		"archive",
	}

	files := []string{
		"data/report.txt",
		"data/image.png",
		"data/logs/app.log",
		"data/logs/error.log",
		"data/docs/README.md",
		"data/docs/summary.txt",
		"archive/old.zip",
		"archive/data.tar.gz",
		"config.json",
		".gitignore",
		"Temp.txt",
		"Foo.TXT",
		"dump.JSON",
	}

	var items []string
	items = append(items, files...)
	items = append(items, directories...)

	var topLevelFiles []string
	for _, file := range files {
		if !strings.Contains(file, "/") {
			topLevelFiles = append(topLevelFiles, file)
		}
	}

	var topLevelDirs []string
	for _, dir := range directories {
		if !strings.Contains(dir, "/") {
			topLevelDirs = append(topLevelDirs, dir)
		}
	}

	var topLevelItems []string
	topLevelItems = append(topLevelItems, topLevelFiles...)
	topLevelItems = append(topLevelItems, topLevelDirs...)

	setupComplexDir := func(t *testing.T, root *Path) *Path {
		for _, dir := range directories {
			createTempDir(t, root, dir)
		}

		for _, file := range files {
			writeTempFile(t, root, file, "")
		}

		return root
	}

	cases := []TestCase[Input, Expect]{
		/*
		 * Basic non-recursive
		 */
		{
			Name: "Non-recursive: all files and dirs (*)",
			Input: Input{
				Pattern: "*", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: topLevelItems},
			Error:  false,
		},
		{
			Name: "Non-recursive: *.txt",
			Input: Input{
				Pattern: "*.txt", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Temp.txt", "Foo.TXT"}}, // Non-recursive only matches top-level
			Error:  false,
		},
		{
			Name: "Non-recursive: starts with 'd' (d*)",
			Input: Input{
				Pattern: "d*", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data", "dump.JSON"}}, // case-insensitive by default
			Error:  false,
		},
		{
			Name: "Non-recursive: no match",
			Input: Input{
				Pattern: "nonexistent", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{}},
			Error:  false,
		},

		/*
		 * Recursive
		 */
		{
			Name: "Recursive: all files and dirs (**) - should match all",
			Input: Input{
				Pattern: "**", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: items},
			Error:  false,
		},
		{
			Name: "Recursive: all .txt files (**/*.txt)",
			Input: Input{
				Pattern: "**/*.txt", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/report.txt", "data/docs/summary.txt", "Temp.txt", "Foo.TXT"}},
			Error:  false,
		},
		{
			Name: "Recursive: specific subdirectory files (data/*.txt)",
			Input: Input{
				Pattern: "data/*.txt", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/report.txt"}},
			Error:  false,
		},
		{
			Name: "Recursive: all subdirectory files (data/**/*.txt)",
			Input: Input{
				Pattern: "data/**/*.txt", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/report.txt", "data/docs/summary.txt"}},
			Error:  false,
		},

		/*
		 * Limit - note: order is not guaranteed, so we just check length
		 */
		{
			Name: "Limit=1",
			Input: Input{
				Pattern: "*", Options: GlobOptions{Limit: 1},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: nil}, // Will check length instead
			Error:  false,
		},
		{
			Name: "Limit=2, Recursive",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{Limit: 2},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: nil}, // Will check length instead
			Error:  false,
		},

		/*
		 * CaseSensitivity
		 */
		{
			Name: "CaseSensitivity=true, *.txt - matches only lowercase .txt",
			Input: Input{
				Pattern: "*.txt", Options: GlobOptions{CaseSensitivity: CaseSensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Temp.txt"}},
			Error:  false,
		},
		{
			Name: "CaseSensitivity=true, *.TXT - no match (no files end in .TXT)",
			Input: Input{
				Pattern: "*.TXT", Options: GlobOptions{CaseSensitivity: CaseSensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Foo.TXT"}},
			Error:  false,
		},

		/*
		 * Filter
		 */
		{
			Name: "Filter: Files only",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{Filter: GlobOptionFilterFiles},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: files},
			Error:  false,
		},
		{
			Name: "Filter: Directories only",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{Filter: GlobOptionFilterDirectories},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: directories},
			Error:  false,
		},

		/*
		 * FilterFunc
		 */
		{
			Name: "FilterFunc: only .md files (recursive)",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{
					FilterFunc: func(p *Path) bool {
						return p.Extension() == ".md"
					},
				},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/docs/README.md"}},
			Error:  false,
		},

		/*
		 * SkipOnDirError
		 */
		{
			Name: "SkipOnDirError=true with unreadable dir",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{SkipOnDirError: true},
				Setup: func(t *testing.T, root *Path) *Path {
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					setupComplexDir(t, root)
					unreadableDir := createTempDir(t, root, "data/unreadable_logs")
					writeTempFile(t, unreadableDir, "secret.log", "")
					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)
					t.Cleanup(func() {
						os.Chmod(unreadableDir.String(), 0755)
					})
					return root
				},
			},
			Expect: Expect{FoundPaths: items},
			Error:  false,
		},
		{
			Name: "SkipOnDirError=false (default) with unreadable dir",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{SkipOnDirError: false},
				Setup: func(t *testing.T, root *Path) *Path {
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					setupComplexDir(t, root)
					unreadableDir := createTempDir(t, root, "data/unreadable_logs")
					writeTempFile(t, unreadableDir, "secret.log", "")
					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)
					t.Cleanup(func() {
						os.Chmod(unreadableDir.String(), 0755)
					})
					return root
				},
			},
			Expect: Expect{
				FoundPaths: nil,
			},
			Error: true,
		},

		/*
		 * Edge Cases
		 */
		{
			Name: "Non-directory path",
			Input: Input{
				Pattern: "*", Options: DefaultGlobOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
					return writeTempFile(t, root, "a_file.txt", "") // Return the FILE path to glob on
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Empty pattern (should error)",
			Input: Input{
				Pattern: "", Options: DefaultGlobOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
					return root
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Invalid filter option (should error)",
			Input: Input{
				Pattern: "*", Options: GlobOptions{Filter: 999},
				Setup: func(t *testing.T, root *Path) *Path {
					return writeTempFile(t, root, "temp.txt", "") // Return the FILE path to glob on
				},
			},
			Expect: Expect{FoundPaths: []string{}},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		globPath := input.Setup(t, root)

		foundPaths, err := globPath.GlobWithOptions(input.Pattern, input.Options)

		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)

		// Convert found paths to relative for comparison
		actualRelPaths := make([]string, len(foundPaths))
		for i, p := range foundPaths {
			relPath, err := p.RelativeTo(globPath)
			require.NoError(t, err)
			actualRelPaths[i] = relPath.ToPosix()
		}
		slices.Sort(actualRelPaths)

		// Special handling for Limit tests - just check the count
		if input.Options.Limit > 0 && expect.FoundPaths == nil {
			require.Len(t, actualRelPaths, input.Options.Limit)
			return
		}

		// For error cases with partial results, skip exact comparison
		if expect.FoundPaths == nil {
			return
		}

		slices.Sort(expect.FoundPaths)
		require.Equal(t, expect.FoundPaths, actualRelPaths)
	})
}

func TestPath_IsFile(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: true,
		},
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "content")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsFile())
	})
}

func TestPath_IsDir(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: true,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: true,
		},
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "content")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsDir())
	})
}

func TestPath_IsEmptyDir(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Empty directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "emptyDir")
			},
			Expect: true,
		},
		{
			Name: "Non-empty directory",
			Input: func(t *testing.T, root *Path) *Path {
				dir := createTempDir(t, root, "nonEmptyDir")
				writeTempFile(t, dir, "file.txt", "content")
				return dir
			},
			Expect: false,
		},
		{
			Name: "Directory with subdirectory only",
			Input: func(t *testing.T, root *Path) *Path {
				dir := createTempDir(t, root, "dirWithSubdir")
				createTempDir(t, dir, "subdir")
				return dir
			},
			Expect: false,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsEmptyDir())
	})
}

func TestPath_Exists(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Existing file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: true,
		},
		{
			Name: "Existing directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: true,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to existing file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Broken symlink",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempSymlinkAbs(t, root, "nonexistent_target", "broken_link")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.Exists())
	})
}

func TestPath_IsSymlink(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: true,
		},
		{
			Name: "Broken symlink",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempSymlinkAbs(t, root, "nonexistent", "broken_link")
			},
			Expect: true,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "")
			},
			Expect: false,
		},
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsSymlink())
	})
}

func TestPath_EqualsFs(t *testing.T) {
	type Input struct {
		Setup func(*testing.T, *Path) (*Path, *Path)
	}

	cases := []TestCase[Input, bool]{
		{
			Name: "Same file",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f := writeTempFile(t, root, "file.txt", "content")
				return f, f
			}},
			Expect: true,
		},
		{
			Name: "Symlink and its target",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f := writeTempFile(t, root, "target.txt", "content")
				l := createTempSymlinkAbs(t, root, "target.txt", "link.txt")
				return f, l
			}},
			Expect: true,
		},
		{
			Name: "Different files",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f1 := writeTempFile(t, root, "file1.txt", "content")
				f2 := writeTempFile(t, root, "file2.txt", "content")
				return f1, f2
			}},
			Expect: false,
		},
		{
			Name: "One path does not exist",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f := writeTempFile(t, root, "file.txt", "content")
				return f, root.JoinStrings("nonexistent")
			}},
			Expect: false,
		},
		{
			Name: "Both paths do not exist",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				return root.JoinStrings("a"), root.JoinStrings("b")
			}},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input Input, expect bool) {
		root := setupTempDir(t)
		p1, p2 := input.Setup(t, root)
		require.Equal(t, expect, p1.EqualsFs(p2))
	})
}

func TestPath_Stat(t *testing.T) {
	type Input struct {
		Setup func(*testing.T, *Path) *Path
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Stat regular file",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "hello")
			}},
			Error: false,
		},
		{
			Name: "Stat directory",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			}},
			Error: false,
		},
		{
			Name: "Stat non-existent path",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			}},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		info, err := p.Stat()
		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.NotNil(t, info)
	})
}

func TestPath_Lstat(t *testing.T) {
	root := setupTempDir(t)

	writeTempFile(t, root, "target.txt", "content")
	link := createTempSymlinkAbs(t, root, "target.txt", "link.txt")

	info, err := link.Lstat()
	require.NoError(t, err)
	require.NotNil(t, info)
	require.True(t, info.Mode()&os.ModeSymlink != 0, "Lstat should report symlink mode")
}

func TestPath_Glob(t *testing.T) {
	root := setupTempDir(t)
	dir := createTempDir(t, root, "globDir")
	writeTempFile(t, dir, "a.txt", "")
	writeTempFile(t, dir, "b.txt", "")
	writeTempFile(t, dir, "c.log", "")

	results, err := dir.Glob("*.txt")
	require.NoError(t, err)

	relPaths := make([]string, len(results))
	for i, p := range results {
		rel, err := p.RelativeTo(dir)
		require.NoError(t, err)
		relPaths[i] = rel.ToPosix()
	}
	slices.Sort(relPaths)
	require.Equal(t, []string{"a.txt", "b.txt"}, relPaths)
}

func TestPath_HasGlobMatchE(t *testing.T) {
	type Input struct {
		Pattern string
		Setup   func(*testing.T, *Path) *Path
	}

	type Expect struct {
		HasMatch bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Match exists",
			Input: Input{
				Pattern: "*.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					return dir
				},
			},
			Expect: Expect{HasMatch: true},
			Error:  false,
		},
		{
			Name: "No match",
			Input: Input{
				Pattern: "*.csv",
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					return dir
				},
			},
			Expect: Expect{HasMatch: false},
			Error:  false,
		},
		{
			Name: "Empty directory",
			Input: Input{
				Pattern: "*",
				Setup: func(t *testing.T, root *Path) *Path {
					return createTempDir(t, root, "emptyDir")
				},
			},
			Expect: Expect{HasMatch: false},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		hasMatch, err := p.HasGlobMatchE(input.Pattern)
		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.Equal(t, expect.HasMatch, hasMatch)
	})
}

func TestPath_HasGlobMatch(t *testing.T) {
	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "file.txt", "")

	require.True(t, dir.HasGlobMatch("*.txt"))
	require.False(t, dir.HasGlobMatch("*.csv"))
}

func TestPath_List(t *testing.T) {
	type Input struct {
		Options ListOptions
		Setup   func(*testing.T, *Path) *Path
	}

	type Expect struct {
		Entries []string
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "List all entries non-recursive",
			Input: Input{
				Options: DefaultListOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "b.txt", "")
					writeTempFile(t, dir, "a.txt", "")
					createTempDir(t, dir, "subdir")
					writeTempFile(t, dir.JoinStrings("subdir"), "nested.txt", "")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"a.txt", "b.txt", "subdir"}},
			Error:  false,
		},
		{
			Name: "List all entries recursive",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Recursive = true
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "b.txt", "")
					writeTempFile(t, dir, "a.txt", "")
					createTempDir(t, dir, "subdir")
					writeTempFile(t, dir.JoinStrings("subdir"), "nested.txt", "")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"a.txt", "b.txt", "subdir", "subdir/nested.txt"}},
			Error:  false,
		},
		{
			Name: "List with limit",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Limit = 2
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "a.txt", "")
					writeTempFile(t, dir, "b.txt", "")
					writeTempFile(t, dir, "c.txt", "")
					return dir
				},
			},
			Expect: Expect{Entries: nil}, // check length only
			Error:  false,
		},
		{
			Name: "List files only",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Filter = GlobOptionFilterFiles
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					createTempDir(t, dir, "subdir")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"file.txt"}},
			Error:  false,
		},
		{
			Name: "List directories only",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Filter = GlobOptionFilterDirectories
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					createTempDir(t, dir, "subdir")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"subdir"}},
			Error:  false,
		},
		{
			Name: "List empty directory",
			Input: Input{
				Options: DefaultListOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
					return createTempDir(t, root, "emptyDir")
				},
			},
			Expect: Expect{Entries: []string{}},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		entries, err := p.List(input.Options)
		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)

		relPaths := relPathsSorted(t, entries, p)

		if input.Options.Limit > 0 && expect.Entries == nil {
			require.Len(t, relPaths, input.Options.Limit)
			return
		}

		slices.Sort(expect.Entries)
		require.Equal(t, expect.Entries, relPaths)
	})
}

func TestPath_ListFiles(t *testing.T) {
	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "a.txt", "")
	writeTempFile(t, dir, "b.log", "")
	subdir := createTempDir(t, dir, "subdir")
	writeTempFile(t, subdir, "nested.txt", "")

	t.Run("non-recursive", func(t *testing.T) {
		files, err := dir.ListFiles(false)
		require.NoError(t, err)
		require.Equal(t, []string{"a.txt", "b.log"}, relPathsSorted(t, files, dir))
	})

	t.Run("recursive", func(t *testing.T) {
		files, err := dir.ListFiles(true)
		require.NoError(t, err)
		require.Equal(t, []string{"a.txt", "b.log", "subdir/nested.txt"}, relPathsSorted(t, files, dir))
	})
}

func TestPath_ListDirs(t *testing.T) {
	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "a.txt", "")
	subdir1 := createTempDir(t, dir, "subdir1")
	createTempDir(t, dir, "subdir2")
	createTempDir(t, subdir1, "nested")

	t.Run("non-recursive", func(t *testing.T) {
		dirs, err := dir.ListDirs(false)
		require.NoError(t, err)
		require.Equal(t, []string{"subdir1", "subdir2"}, relPathsSorted(t, dirs, dir))
	})

	t.Run("recursive", func(t *testing.T) {
		dirs, err := dir.ListDirs(true)
		require.NoError(t, err)
		require.Equal(t, []string{"subdir1", "subdir1/nested", "subdir2"}, relPathsSorted(t, dirs, dir))
	})
}

func TestCreateFile(t *testing.T) {
	root := setupTempDir(t)

	filePath := root.JoinStrings("new_file.txt")
	err := CreateFile(filePath)
	require.NoError(t, err)
	require.True(t, filePath.Exists())
	require.True(t, filePath.IsFile())

	info, err := filePath.Stat()
	require.NoError(t, err)
	require.Equal(t, DefaultFileMode.Perm(), info.Mode().Perm())
}

func TestMkDir(t *testing.T) {
	root := setupTempDir(t)

	dirPath := root.JoinStrings("new_dir")
	err := MkDir(dirPath)
	require.NoError(t, err)
	require.True(t, dirPath.Exists())
	require.True(t, dirPath.IsDir())

	info, err := dirPath.Stat()
	require.NoError(t, err)
	require.Equal(t, DefaultDirMode.Perm(), info.Mode().Perm())
}

func TestCreateSymlink(t *testing.T) {
	type Input struct {
		Setup func(*testing.T, *Path) (target *Path, linkPath *Path)
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Successful symlink creation",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				target := writeTempFile(t, root, "target.txt", "content")
				return target, root.JoinStrings("link.txt")
			}},
			Error: false,
		},
		{
			Name: "Link path already exists",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				target := writeTempFile(t, root, "target.txt", "content")
				existing := writeTempFile(t, root, "existing.txt", "")
				return target, existing
			}},
			Error: true,
		},
		{
			Name: "Link parent directory does not exist",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				target := writeTempFile(t, root, "target.txt", "content")
				return target, root.JoinStrings("nonexistent_dir", "link.txt")
			}},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		target, linkPath := input.Setup(t, root)
		err := CreateSymlink(target, linkPath)
		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.True(t, linkPath.IsSymlink())
	})
}

func TestRemove(t *testing.T) {
	type Input struct {
		Setup func(*testing.T, *Path) *Path
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Remove existing file",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			}},
			Error: false,
		},
		{
			Name: "Remove empty directory",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "emptyDir")
			}},
			Error: false,
		},
		{
			Name: "Remove non-existent path (no-op)",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			}},
			Error: false,
		},
		{
			Name: "Remove non-empty directory (should error)",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				dir := createTempDir(t, root, "nonEmptyDir")
				writeTempFile(t, dir, "file.txt", "")
				return dir
			}},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		err := Remove(p)
		if expectError {
			require.Error(t, err)
			require.True(t, p.Exists())
			return
		}
		require.NoError(t, err)
		require.False(t, p.Exists())
	})
}

func TestRename(t *testing.T) {
	type Input struct {
		NewName string
		Setup   func(*testing.T, *Path) *Path
	}

	type Expect struct {
		FinalEntries []string // relative to root
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Rename file",
			Input: Input{
				NewName: "renamed.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					return writeTempFile(t, root, "original.txt", "content")
				},
			},
			Expect: Expect{FinalEntries: []string{"renamed.txt"}},
			Error:  false,
		},
		{
			Name: "Rename directory",
			Input: Input{
				NewName: "renamed_dir",
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "original_dir")
					writeTempFile(t, dir, "child.txt", "")
					return dir
				},
			},
			Expect: Expect{FinalEntries: []string{"renamed_dir", "renamed_dir/child.txt"}},
			Error:  false,
		},
		{
			Name: "Rename to existing name (should error)",
			Input: Input{
				NewName: "existing.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "existing.txt", "existing")
					return writeTempFile(t, root, "original.txt", "original")
				},
			},
			Expect: Expect{FinalEntries: []string{"existing.txt", "original.txt"}},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)

		err := Rename(p, input.NewName)
		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		finalState := readDirEntries(t, root, root)
		slices.Sort(expect.FinalEntries)
		require.Equal(t, expect.FinalEntries, finalState)
	})
}

func TestSetPermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	root := setupTempDir(t)

	t.Run("file", func(t *testing.T) {
		filePath := writeTempFile(t, root, "file.txt", "content")

		err := SetPermission(filePath, 0600)
		require.NoError(t, err)

		info, err := filePath.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600).Perm(), info.Mode().Perm())

		// Change again
		err = SetPermission(filePath, 0755)
		require.NoError(t, err)

		info, err = filePath.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0755).Perm(), info.Mode().Perm())
	})

	t.Run("directory", func(t *testing.T) {
		dirPath := createTempDir(t, root, "permdir")

		err := SetPermission(dirPath, 0700)
		require.NoError(t, err)

		info, err := dirPath.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0700).Perm(), info.Mode().Perm())

		err = SetPermission(dirPath, 0755)
		require.NoError(t, err)

		info, err = dirPath.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0755).Perm(), info.Mode().Perm())
	})

	t.Run("non-existent path", func(t *testing.T) {
		nonExistent := root.JoinStrings("does_not_exist")
		err := SetPermission(nonExistent, 0644)
		require.Error(t, err)
	})
}

func TestDeviceMethods(t *testing.T) {
	root := setupTempDir(t)
	filePath := writeTempFile(t, root, "regular.txt", "content")
	dirPath := createTempDir(t, root, "subdir")
	nonExistent := root.JoinStrings("does_not_exist")

	t.Run("IsBlockDevice", func(t *testing.T) {
		require.False(t, filePath.IsBlockDevice(), "regular file is not a block device")
		require.False(t, dirPath.IsBlockDevice(), "directory is not a block device")
		require.False(t, nonExistent.IsBlockDevice(), "non-existent path is not a block device")
	})

	t.Run("IsCharDevice", func(t *testing.T) {
		require.False(t, filePath.IsCharDevice(), "regular file is not a char device")
		require.False(t, dirPath.IsCharDevice(), "directory is not a char device")
		require.False(t, nonExistent.IsCharDevice(), "non-existent path is not a char device")

		if notRunningOnWindows {
			devNull := NewPath("/dev/null")
			require.True(t, devNull.IsCharDevice(), "/dev/null should be a char device")
		}
	})

	t.Run("IsFiFoPipe", func(t *testing.T) {
		require.False(t, filePath.IsFiFoPipe(), "regular file is not a FIFO pipe")
		require.False(t, dirPath.IsFiFoPipe(), "directory is not a FIFO pipe")
		require.False(t, nonExistent.IsFiFoPipe(), "non-existent path is not a FIFO pipe")
	})

	t.Run("IsSocket", func(t *testing.T) {
		require.False(t, filePath.IsSocket(), "regular file is not a socket")
		require.False(t, dirPath.IsSocket(), "directory is not a socket")
		require.False(t, nonExistent.IsSocket(), "non-existent path is not a socket")
	})
}
