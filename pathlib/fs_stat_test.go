package pathlib

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_Resolve(t *testing.T) {
	t.Parallel()

	type Input struct {
		PathStr string
		Setup   func(*testing.T, *Path) *Path // Function to set up the path in the temp dir
	}

	cases := []TestCase[Input, string]{
		{
			Name: "Regular file",
			Input: Input{PathStr: "file.txt", Setup: func(t *testing.T, root *Path) *Path {
				t.Helper()
				return writeTempFile(t, root, "file.txt", "content")
			}},
			Expect: "file.txt",
			Error:  false,
		},
		{
			Name: "Directory",
			Input: Input{PathStr: "dir", Setup: func(t *testing.T, root *Path) *Path {
				t.Helper()
				return createTempDir(t, root, "dir")
			}},
			Expect: "dir",
			Error:  false,
		},
		{
			Name:  "Non-existent path",
			Input: Input{PathStr: "nonexistent", Setup: func(_ *testing.T, root *Path) *Path { return root.JoinStrings("nonexistent") }},
			Error: true,
		},
		{
			Name: "Symlink to file",
			Input: Input{
				PathStr: "linkToFile.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
		t.Helper()

		root := setupTempDir(t)
		p := input.Setup(t, root)

		resolvedPath, err := p.Resolve()

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			require.Nil(t, resolvedPath)
			return
		}

		require.NoError(t, err)
		require.NotNil(t, resolvedPath)

		// Ensure the resolved path is absolute
		require.True(t, resolvedPath.IsAbsolute())

		// Resolve the root too so both paths are in canonical form. This
		// handles platform-specific path rewriting that Resolve performs, e.g.
		// /var -> /private/var on macOS, or 8.3 short names (RUNNER~1 ->
		// runneradmin) on Windows.
		resolvedRoot, err := root.Resolve()
		require.NoError(t, err)

		// Compare relative to root for easier testing
		relPath, err := resolvedPath.RelativeTo(resolvedRoot)
		require.NoError(t, err)

		require.Equal(t, expect, relPath.ToPosix())
	})
}

func TestPath_IsFile(t *testing.T) {
	t.Parallel()

	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: true,
		},
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempDir(t, root, "dir")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(_ *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				writeTempFile(t, root, "target.txt", "content")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsFile())
	})
}

func TestPath_IsDir(t *testing.T) {
	t.Parallel()

	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempDir(t, root, "dir")
			},
			Expect: true,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(_ *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: true,
		},
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				writeTempFile(t, root, "target.txt", "content")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsDir())
	})
}

func TestPath_IsEmptyDir(t *testing.T) {
	t.Parallel()

	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Empty directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempDir(t, root, "emptyDir")
			},
			Expect: true,
		},
		{
			Name: "Non-empty directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				dir := createTempDir(t, root, "nonEmptyDir")
				writeTempFile(t, dir, "file.txt", "content")
				return dir
			},
			Expect: false,
		},
		{
			Name: "Directory with subdirectory only",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				dir := createTempDir(t, root, "dirWithSubdir")
				createTempDir(t, dir, "subdir")
				return dir
			},
			Expect: false,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(_ *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsEmptyDir())
	})
}

func TestPath_Exists(t *testing.T) {
	t.Parallel()

	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Existing file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: true,
		},
		{
			Name: "Existing directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempDir(t, root, "dir")
			},
			Expect: true,
		},
		{
			Name: "Non-existent path",
			Input: func(_ *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to existing file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				writeTempFile(t, root, "target.txt", "")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Broken symlink",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempSymlinkAbs(t, root, "nonexistent_target", "broken_link")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.Exists())
	})
}

func TestPath_IsSymlink(t *testing.T) {
	t.Parallel()

	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				writeTempFile(t, root, "target.txt", "")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: true,
		},
		{
			Name: "Broken symlink",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempSymlinkAbs(t, root, "nonexistent", "broken_link")
			},
			Expect: true,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "")
			},
			Expect: false,
		},
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempDir(t, root, "dir")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(_ *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsSymlink())
	})
}

func TestPath_Stat(t *testing.T) {
	t.Parallel()

	type Input struct {
		Setup func(*testing.T, *Path) *Path
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Stat regular file",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "hello")
			}},
			Error: false,
		},
		{
			Name: "Stat directory",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempDir(t, root, "dir")
			}},
			Error: false,
		},
		{
			Name: "Stat non-existent path",
			Input: Input{Setup: func(_ *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			}},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, _ any, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.Setup(t, root)
		info, err := p.Stat()
		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			return
		}
		require.NoError(t, err)
		require.NotNil(t, info)
	})
}

func TestPath_Lstat(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)

	writeTempFile(t, root, "target.txt", "content")
	link := createTempSymlinkAbs(t, root, "target.txt", "link.txt")

	info, err := link.Lstat()
	require.NoError(t, err)
	require.NotNil(t, info)
	require.NotEqual(t, 0, info.Mode()&ModeSymlink, "Lstat should report symlink mode")
}

func TestDeviceMethods(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	filePath := writeTempFile(t, root, "regular.txt", "content")
	dirPath := createTempDir(t, root, "subdir")
	nonExistent := root.JoinStrings("does_not_exist")

	t.Run("IsBlockDevice", func(t *testing.T) {
		t.Parallel()

		require.False(t, filePath.IsBlockDevice(), "regular file is not a block device")
		require.False(t, dirPath.IsBlockDevice(), "directory is not a block device")
		require.False(t, nonExistent.IsBlockDevice(), "non-existent path is not a block device")
	})

	t.Run("IsCharDevice", func(t *testing.T) {
		t.Parallel()

		require.False(t, filePath.IsCharDevice(), "regular file is not a char device")
		require.False(t, dirPath.IsCharDevice(), "directory is not a char device")
		require.False(t, nonExistent.IsCharDevice(), "non-existent path is not a char device")

		if !runningOnWindows {
			devNull := NewPath("/dev/null")
			require.True(t, devNull.IsCharDevice(), "/dev/null should be a char device")
		}
	})

	t.Run("IsFIFO", func(t *testing.T) {
		t.Parallel()

		require.False(t, filePath.IsFIFO(), "regular file is not a FIFO pipe")
		require.False(t, dirPath.IsFIFO(), "directory is not a FIFO pipe")
		require.False(t, nonExistent.IsFIFO(), "non-existent path is not a FIFO pipe")
	})

	t.Run("IsSocket", func(t *testing.T) {
		t.Parallel()

		require.False(t, filePath.IsSocket(), "regular file is not a socket")
		require.False(t, dirPath.IsSocket(), "directory is not a socket")
		require.False(t, nonExistent.IsSocket(), "non-existent path is not a socket")
	})
}

func TestFSErrorsAreWrapped(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	missing := root.JoinStrings("missing")

	_, err := missing.Stat()
	require.ErrorIs(t, err, ErrNotExist)
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.ErrorAs(t, err, new(*PathlibError))

	_, err = missing.Lstat()
	require.ErrorIs(t, err, ErrNotExist)

	err = SetPermission(missing, 0644)
	require.ErrorIs(t, err, ErrNotExist)
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = MkDirWithOptions(root.JoinStrings("a", "b"), DirOptions{})
	require.ErrorIs(t, err, ErrNotExist)
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.ErrorAs(t, err, new(*PathlibError))
}

func TestPath_LExists(t *testing.T) {
	t.Parallel()

	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "File",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()
				return writeTempFile(t, root, "file.txt", "")
			},
			Expect: true,
		},
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()
				return createTempDir(t, root, "dir")
			},
			Expect: true,
		},
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				writeTempFile(t, root, "target.txt", "")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Broken symlink",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempSymlinkAbs(t, root, "nonexistent_target", "broken_link")
			},
			Expect: true,
		},
		{
			Name:   "Missing path",
			Input:  func(_ *testing.T, root *Path) *Path { return root.JoinStrings("missing") },
			Expect: false,
		},
		{
			Name: "Path below a file",
			Input: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "").JoinStrings("child")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.LExists())
	})
}

func TestPath_ReadSymlinkTarget(t *testing.T) {
	t.Parallel()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
				Setup:      func(_ *testing.T, _ *Path) {},
			},
			Expect: Expect{},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, _ Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		input.Setup(t, root)

		symlinkPath := root.JoinStrings(input.SymlinkRel)
		readTargetPath, err := symlinkPath.ReadSymlinkTarget()

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			require.Nil(t, readTargetPath)
		} else {
			require.NoError(t, err)
			require.NotNil(t, readTargetPath)

			require.NotEmpty(t, input.TargetRel, "Test is setup wrongly. No target path defined.")
			targetPath := root.JoinStrings(input.TargetRel)

			// The returned target is a Path created from the raw string from
			// os.Readlink. `createTempSymlinkAbs` uses a relative path
			// (relative to link's parent).
			require.Equal(t, targetPath.ToPosix(), readTargetPath.ToPosix())
		}
	})
}

func TestLexists(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	file := writeTempFile(t, root, "file.txt", "")

	cases := []TestCase[*Path, bool]{
		{Name: "Existing file", Input: file, Expect: true},
		{Name: "Broken symlink", Input: createTempSymlinkAbs(t, root, "missing", "link"), Expect: true},
		{Name: "Missing path", Input: root.JoinStrings("missing"), Expect: false},
		{Name: "Path below a file", Input: file.JoinStrings("child"), Expect: false},
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect bool) {
		t.Helper()

		exists, err := lexists(input)
		require.NoError(t, err)
		require.Equal(t, expect, exists)
	})
}

func TestLexists_UncheckablePathIsAnError(t *testing.T) {
	t.Parallel()

	dir := createTempDir(t, setupTempDir(t), "locked")
	file := writeTempFile(t, dir, "file.txt", "")
	lockDir(t, dir)

	exists, err := lexists(file)
	require.False(t, exists)
	require.ErrorIs(t, err, ErrPermissionDenied)
	require.ErrorIs(t, err, fs.ErrPermission)
	require.NotErrorIs(t, err, ErrNotExist)

	require.False(t, file.LExists(), "LExists cannot report the error and returns false")
}

func TestRequireDir(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	missing := root.JoinStrings("missing")

	err := requireDir(missing)
	var kind *PathlibError
	require.ErrorAs(t, err, &kind)
	require.Equal(t, ErrNotExist, kind)

	var cause *PathError
	require.ErrorAs(t, err, &cause)
	require.Equal(t, []Path{*missing}, cause.Paths())
	require.ErrorIs(t, cause.Unwrap(), fs.ErrNotExist, "the os cause is kept")

	require.NoError(t, requireDir(root))
	require.NoError(t, requireDir(createTempSymlinkAbs(t, root, ".", "link")))
}
