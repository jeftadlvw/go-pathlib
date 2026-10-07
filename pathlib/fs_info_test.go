package pathlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileInfo_Size(t *testing.T) {
	t.Parallel()

	type Input struct {
		// Stat returns the file info of an entry created below root.
		Stat func(*testing.T, *Path) (*FileInfo, error)
	}

	cases := []TestCase[Input, int64]{
		{
			Name: "Regular file has its length",
			Input: Input{Stat: func(t *testing.T, root *Path) (*FileInfo, error) {
				t.Helper()

				return writeTempFile(t, root, "file.txt", "content").Stat()
			}},
			Expect: int64(len("content")),
		},
		{
			Name: "Symlink to a file has the length of the file with Stat",
			Input: Input{Stat: func(t *testing.T, root *Path) (*FileInfo, error) {
				t.Helper()

				writeTempFile(t, root, "file.txt", "content")
				return createTempSymlinkAbs(t, root, "file.txt", "link").Stat()
			}},
			Expect: int64(len("content")),
		},
		{
			Name: "Symlink has no size with Lstat",
			Input: Input{Stat: func(t *testing.T, root *Path) (*FileInfo, error) {
				t.Helper()

				writeTempFile(t, root, "file.txt", "content")
				return createTempSymlinkAbs(t, root, "file.txt", "link").Lstat()
			}},
			Expect: 0,
		},
		{
			Name: "Directory has no size",
			Input: Input{Stat: func(t *testing.T, root *Path) (*FileInfo, error) {
				t.Helper()

				dir := createTempDir(t, root, "dir")
				writeTempFile(t, dir, "file.txt", "content")
				return dir.Stat()
			}},
			Expect: 0,
		},
	}

	runForResults(t, cases, func(t *testing.T, input Input, expect int64) {
		t.Helper()

		info, err := input.Stat(t, setupTempDir(t))
		require.NoError(t, err)
		require.Equal(t, expect, info.Size())
	})
}

func TestFileInfo_MatchesOS(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	file := writeTempFile(t, root, "file.txt", "content")

	info, err := file.Stat()
	require.NoError(t, err)
	osInfo, err := os.Stat(file.String())
	require.NoError(t, err)

	require.Equal(t, osInfo.Name(), info.Name())
	require.Equal(t, osInfo.Mode(), info.Mode())
	require.Equal(t, osInfo.ModTime(), info.ModTime())
	require.Equal(t, osInfo.IsDir(), info.IsDir())
	require.Equal(t, osInfo.Sys(), info.Sys())
}

func TestFileInfo_SameFile(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	file := writeTempFile(t, root, "file.txt", "content")
	other := writeTempFile(t, root, "other.txt", "content")
	link := createTempSymlinkAbs(t, root, "file.txt", "link")

	fileInfo, err := file.Stat()
	require.NoError(t, err)
	otherInfo, err := other.Stat()
	require.NoError(t, err)
	linkInfo, err := link.Stat()
	require.NoError(t, err)

	require.True(t, fileInfo.SameFile(fileInfo))
	require.True(t, fileInfo.SameFile(linkInfo), "a followed symlink is its target")
	require.False(t, fileInfo.SameFile(otherInfo))
	require.False(t, fileInfo.SameFile(nil))
}
