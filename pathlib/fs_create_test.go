package pathlib

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
