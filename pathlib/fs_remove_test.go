package pathlib

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveAll(t *testing.T) {
	t.Parallel()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
				Setup:   func(_ *testing.T, _ *Path) {},
			},
			Error: false,
		},
		{
			Name: "Remove a file",
			Input: Input{
				RelPath: "a_file.txt",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					writeTempFile(t, root, "a_file.txt", "")
				},
			},
			Error: false,
		},
		{
			Name: "Remove a broken symlink",
			Input: Input{
				RelPath: "broken_link",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					createTempSymlinkAbs(t, root, "missing", "broken_link")
				},
			},
			Error: false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, _ any, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		targetPath := root.JoinStrings(input.RelPath)
		input.Setup(t, root) // Set up the path to be removed

		err := RemoveAll(targetPath)

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)
			requireLExists(t, false, targetPath, "Path should not exist after RemoveAll")
		}
	})
}

func TestRemove(t *testing.T) {
	t.Parallel()

	type Input struct {
		Setup func(*testing.T, *Path) *Path
	}

	cases := []TestCase[Input, *PathlibError]{
		{
			Name: "Remove existing file",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "content")
			}},
			Error: false,
		},
		{
			Name: "Remove empty directory",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				t.Helper()

				return createTempDir(t, root, "emptyDir")
			}},
			Error: false,
		},
		{
			Name: "Remove non-existent path",
			Input: Input{Setup: func(_ *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			}},
			Expect: ErrNotExist,
			Error:  true,
		},
		{
			Name: "Remove non-empty directory",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				t.Helper()

				dir := createTempDir(t, root, "nonEmptyDir")
				writeTempFile(t, dir, "file.txt", "")
				return dir
			}},
			Expect: ErrNotEmptyDir,
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect *PathlibError, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.Setup(t, root)
		err := Remove(p)
		if expectError {
			require.ErrorIs(t, err, expect)
			return
		}
		require.NoError(t, err)
		require.False(t, p.Exists())
	})
}

func TestRemoveWithOptions(t *testing.T) {
	t.Parallel()

	type Input struct {
		Setup   func(*testing.T, *Path) *Path
		Options RemoveOptions
	}

	missing := func(_ *testing.T, root *Path) *Path {
		return root.JoinStrings("missing")
	}
	nonEmptyDir := func(t *testing.T, root *Path) *Path {
		t.Helper()

		dir := createTempDir(t, root, "dir")
		writeTempFile(t, dir, "sub/file.txt", "")
		return dir
	}
	brokenSymlink := func(t *testing.T, root *Path) *Path {
		t.Helper()

		return createTempSymlinkAbs(t, root, "missing", "link")
	}

	cases := []TestCase[Input, *PathlibError]{
		{
			Name:   "Missing path is refused by the zero value",
			Input:  Input{Setup: missing, Options: RemoveOptions{}},
			Expect: ErrNotExist,
			Error:  true,
		},
		{
			Name:  "Missing path is accepted with MissingOk",
			Input: Input{Setup: missing, Options: RemoveOptions{MissingOk: true}},
			Error: false,
		},
		{
			Name:   "Missing path is refused with Recursive alone",
			Input:  Input{Setup: missing, Options: RemoveOptions{Recursive: true}},
			Expect: ErrNotExist,
			Error:  true,
		},
		{
			Name:   "Non-empty directory is refused without Recursive",
			Input:  Input{Setup: nonEmptyDir, Options: RemoveOptions{MissingOk: true}},
			Expect: ErrNotEmptyDir,
			Error:  true,
		},
		{
			Name:  "Non-empty directory is removed with Recursive",
			Input: Input{Setup: nonEmptyDir, Options: RemoveOptions{Recursive: true}},
			Error: false,
		},
		{
			Name:  "Broken symlink exists for the zero value",
			Input: Input{Setup: brokenSymlink, Options: RemoveOptions{}},
			Error: false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect *PathlibError, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.Setup(t, root)
		existed := p.LExists()

		err := RemoveWithOptions(p, input.Options)
		if expectError {
			require.ErrorIs(t, err, expect)
			requireLExists(t, existed, p, "a refused path is left untouched")
			return
		}
		require.NoError(t, err)
		requireLExists(t, false, p)
	})
}

func TestRemoveWithOptions_NotEmptyDirMatchesExist(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "file.txt", "")

	err := Remove(dir)
	require.ErrorIs(t, err, ErrNotEmptyDir)
	require.ErrorIs(t, err, ErrExist)
	require.ErrorIs(t, err, fs.ErrExist)
}

func TestDefaultRemoveOptions(t *testing.T) {
	t.Parallel()

	require.Equal(t, RemoveOptions{}, DefaultRemoveOptions())
}

func TestRemoveAll_SymlinkToDirectoryRemovesOnlyLink(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	target := createTempDir(t, root, "target_dir")
	file := writeTempFile(t, target, "file.txt", "")
	link := createTempSymlinkAbs(t, root, "target_dir", "link_to_dir")

	require.NoError(t, RemoveAll(link))

	_, err := os.Lstat(link.String())
	require.ErrorIs(t, err, fs.ErrNotExist, "the symlink is removed")
	require.True(t, target.IsDir(), "the target directory is kept")
	require.True(t, file.IsFile(), "the target's content is kept")
}

func TestRemove_Symlinks(t *testing.T) {
	t.Parallel()

	t.Run("broken symlink is removed", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		link := createTempSymlinkAbs(t, root, "missing", "link")

		require.NoError(t, Remove(link))
		requireLExists(t, false, link)
	})

	t.Run("symlink to file removes only the link", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		target := writeTempFile(t, root, "target.txt", "content")
		link := createTempSymlinkAbs(t, root, "target.txt", "link")

		require.NoError(t, Remove(link))
		requireLExists(t, false, link)
		require.True(t, target.IsFile())
	})
}

func TestRemove_PathBelowFileIsMissing(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	file := writeTempFile(t, root, "file.txt", "")

	require.ErrorIs(t, Remove(file.JoinStrings("child")), ErrNotExist)
	require.NoError(t, RemoveAll(file.JoinStrings("child")))
	require.True(t, file.IsFile())
}

func TestRemove_UncheckablePathIsAnError(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	dir := createTempDir(t, root, "locked")
	file := writeTempFile(t, dir, "file.txt", "")
	subdir := createTempDir(t, dir, "subdir")
	lockDir(t, dir)

	// A path that cannot be checked is no missing path, so even RemoveAll,
	// which accepts a missing path, returns the error.
	err := Remove(file)
	require.ErrorIs(t, err, ErrPermissionDenied)
	require.ErrorIs(t, err, fs.ErrPermission)

	err = RemoveAll(subdir)
	require.ErrorIs(t, err, ErrPermissionDenied)
	require.ErrorIs(t, err, fs.ErrPermission)
}

func TestRemoveAll_SymlinkToFileRemovesOnlyLink(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	target := writeTempFile(t, root, "target.txt", "content")
	link := createTempSymlinkAbs(t, root, "target.txt", "link")

	require.NoError(t, RemoveAll(link))
	requireLExists(t, false, link, "the symlink is removed")
	require.True(t, target.IsFile(), "the target is kept")
}
