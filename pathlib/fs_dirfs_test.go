package pathlib

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestPath_FS(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	writeTempFile(t, root, "a.txt", "a")
	writeTempFile(t, root, "dir/b.txt", "b")
	createTempDir(t, root, "dir/empty")

	t.Run("passes the checks of fstest", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, fstest.TestFS(root.FS(), "a.txt", "dir/b.txt", "dir/empty"))
	})

	t.Run("reads a nested file", func(t *testing.T) {
		t.Parallel()

		content, err := fs.ReadFile(root.FS(), "dir/b.txt")
		require.NoError(t, err)
		require.Equal(t, "b", string(content))
	})

	t.Run("takes a name made from a Path below", func(t *testing.T) {
		t.Parallel()

		file := root.JoinStrings("dir", "b.txt")
		name, err := file.RelativeTo(root)
		require.NoError(t, err)

		content, err := fs.ReadFile(root.FS(), name.ToPosix())
		require.NoError(t, err)
		require.Equal(t, "b", string(content))
	})

	t.Run("refuses names outside the tree", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"../a.txt", "/a.txt", "dir/../../a.txt"} {
			_, err := fs.ReadFile(root.FS(), name)
			require.ErrorIs(t, err, fs.ErrInvalid, name)
		}
	})

	t.Run("reads a symlink without following it", func(t *testing.T) {
		t.Parallel()

		dir := setupTempDir(t)
		writeTempFile(t, dir, "a.txt", "a")
		createTempSymlinkRel(t, dir, "a.txt", "link")

		target, err := fs.ReadLink(dir.FS(), "link")
		require.NoError(t, err)
		require.Equal(t, "a.txt", target)

		info, err := fs.Lstat(dir.FS(), "link")
		require.NoError(t, err)
		require.Equal(t, fs.ModeSymlink, info.Mode().Type())
	})

	t.Run("reports a missing directory on use", func(t *testing.T) {
		t.Parallel()

		missing := root.JoinStrings("missing").FS()
		_, err := fs.ReadFile(missing, "a.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)
	})
}
