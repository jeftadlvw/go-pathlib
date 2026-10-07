package pathlib

import (
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// pastTime returns a time in the past with whole seconds, which every tested
// filesystem stores without rounding.
func pastTime() time.Time {
	return time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC)
}

// requireModTime asserts that the modification time of p, following
// symlinks, is expect in whole seconds.
func requireModTime(t *testing.T, expect time.Time, p *Path) {
	t.Helper()

	info, err := os.Stat(p.String())
	require.NoError(t, err)
	require.Equal(t, expect.Unix(), info.ModTime().Unix())
}

func TestSetTimes(t *testing.T) {
	t.Parallel()

	type Input struct {
		// Setup returns the path to pass and the path whose times change.
		Setup func(*testing.T, *Path) (*Path, *Path)
	}

	cases := []TestCase[Input, any]{
		{
			Name: "File",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				t.Helper()

				file := writeTempFile(t, root, "file.txt", "content")
				return file, file
			}},
		},
		{
			Name: "Directory",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				t.Helper()

				dir := createTempDir(t, root, "dir")
				return dir, dir
			}},
		},
		{
			Name: "Symlink is followed",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				t.Helper()

				target := writeTempFile(t, root, "target.txt", "content")
				link := createTempSymlinkAbs(t, root, "target.txt", "link")
				return link, target
			}},
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, _ any, _ bool) {
		t.Helper()

		root := setupTempDir(t)
		p, changed := input.Setup(t, root)

		require.NoError(t, SetTimes(p, pastTime(), pastTime()))
		requireModTime(t, pastTime(), changed)
	})
}

func TestSetTimes_ZeroTimeKeepsTime(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	file := writeTempFile(t, root, "file.txt", "content")
	require.NoError(t, SetTimes(file, pastTime(), pastTime()))

	require.NoError(t, SetTimes(file, time.Now(), time.Time{}))
	requireModTime(t, pastTime(), file)
}

func TestSetTimes_MissingPath(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)

	err := SetTimes(root.JoinStrings("missing"), pastTime(), pastTime())
	require.ErrorIs(t, err, ErrNotExist)
	require.ErrorIs(t, err, fs.ErrNotExist)
}
