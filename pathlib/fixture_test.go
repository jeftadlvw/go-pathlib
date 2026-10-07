package pathlib

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// randomStringCharset holds the digits and the Latin letters in both cases.
const randomStringCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// errSimulated is an error of a test, such as of a walk callback.
var errSimulated = errors.New("simulated error")

// platformNativeUNC returns posixForm, a UNC path with forward slashes, in the
// native form of the platform.
func platformNativeUNC(posixForm string) string {
	if runningOnWindows {
		return toWindowsSeparators(posixForm)
	}
	return posixForm
}

// onWindows returns windowsVal on Windows, posixVal on other platforms.
//
//nolint:ireturn // T is the type of the caller's values.
func onWindows[T any](posixVal, windowsVal T) T {
	if runningOnWindows {
		return windowsVal
	}
	return posixVal
}

// setupTempDir returns a temporary directory that is removed when the test
// ends.
func setupTempDir(t *testing.T) *Path {
	t.Helper()

	return NewPath(t.TempDir())
}

// setupRelativeTempDir creates a directory below the working directory and
// returns it as a relative Path, so tests can pass relative paths to the
// library. Its name starts with an underscore, so the go tool ignores a
// directory left behind by an interrupted run. It is removed when the test
// ends.
func setupRelativeTempDir(t *testing.T) *Path {
	t.Helper()

	//nolint:usetesting // t.TempDir is absolute, and the directory must be relative.
	dir, err := os.MkdirTemp(".", "_tmp-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	root := NewPath(dir)
	require.True(t, root.IsRelative())
	return root
}

// writeTempFile creates the file relPath with content below root, together
// with its missing parent directories, and returns its path.
func writeTempFile(t *testing.T, root *Path, relPath string, content string) *Path {
	t.Helper()

	filePath := root.JoinStrings(relPath)

	err := os.MkdirAll(filePath.Parent().String(), 0755) //nolint:gosec // Fixtures use common permissions.
	require.NoError(t, err)

	err = os.WriteFile(filePath.String(), []byte(content), 0644) //nolint:gosec // Fixtures use common permissions.
	require.NoError(t, err)

	return filePath
}

// createTempDir creates the directory relPath below root, together with its
// missing parent directories, and returns its path.
func createTempDir(t *testing.T, root *Path, relPath string) *Path {
	t.Helper()

	dirPath := root.JoinStrings(relPath)
	err := os.MkdirAll(dirPath.String(), 0755) //nolint:gosec // Fixtures use common permissions.
	require.NoError(t, err)
	return dirPath
}

// createTempSymlinkAbs creates the symlink linkRelPath below root with the
// absolute target root joined with targetRelPath, and returns its path.
func createTempSymlinkAbs(t *testing.T, root *Path, targetRelPath string, linkRelPath string) *Path {
	t.Helper()

	targetPath := root.JoinStrings(targetRelPath)
	linkPath := root.JoinStrings(linkRelPath)
	err := os.MkdirAll(linkPath.Parent().String(), 0755) //nolint:gosec // Fixtures use common permissions.
	require.NoError(t, err)

	err = os.Symlink(targetPath.String(), linkPath.String())
	require.NoError(t, err)
	return linkPath
}

// createTempSymlinkRel creates the symlink linkRelPath below root with the
// relative target targetRelPath, and returns its path.
func createTempSymlinkRel(t *testing.T, root *Path, targetRelPath string, linkRelPath string) *Path {
	t.Helper()

	targetPath := NewPath(targetRelPath)
	linkPath := root.JoinStrings(linkRelPath)
	err := os.MkdirAll(linkPath.Parent().String(), 0755) //nolint:gosec // Fixtures use common permissions.
	require.NoError(t, err)

	err = os.Symlink(targetPath.String(), linkPath.String())
	require.NoError(t, err)
	return linkPath
}

// readDirEntries returns the entries below dir, recursively and sorted, as
// paths relative to root with forward slashes. dir itself is included unless
// it is root.
func readDirEntries(t *testing.T, root *Path, dir *Path) []string {
	t.Helper()

	var entries []string
	if !dir.Exists() {
		return entries
	}
	err := filepath.WalkDir(dir.String(), func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir.String() && path == root.String() {
			return nil
		}
		relPath, err := filepath.Rel(root.String(), path)
		require.NoError(t, err)
		entries = append(entries, filepath.ToSlash(relPath))
		return nil
	})
	require.NoError(t, err)
	slices.Sort(entries)
	return entries
}

// relPathsSorted returns entries relative to base in Posix form, sorted.
func relPathsSorted(t *testing.T, entries []*Path, base *Path) []string {
	t.Helper()

	relPaths := make([]string, len(entries))
	for i, e := range entries {
		rel, err := e.RelativeTo(base)
		require.NoError(t, err)
		relPaths[i] = rel.ToPosix()
	}

	slices.Sort(relPaths)
	return relPaths
}

// lockDir removes all permissions from dir, so entries below it cannot be
// checked, and restores them when the test ends. The test is skipped where
// directory permissions are not enforced, on Windows or for the root user.
func lockDir(t *testing.T, dir *Path) {
	t.Helper()
	if runningOnWindows || os.Geteuid() == 0 {
		t.Skip("directory permissions are not enforced")
	}

	require.NoError(t, os.Chmod(dir.String(), 0000))
	t.Cleanup(func() { _ = os.Chmod(dir.String(), 0755) }) //nolint:gosec // Fixtures use common permissions.
}

// requireLExists asserts whether p exists, without following symlinks.
func requireLExists(t *testing.T, expect bool, p *Path, msgAndArgs ...any) {
	t.Helper()
	_, err := os.Lstat(p.String())
	require.Equal(t, expect, err == nil, msgAndArgs...)
}

// generateRandomString returns a string of random letters and digits whose
// length lies between minLength and maxLength.
func generateRandomString(minLength, maxLength int) string {
	//nolint:gosec // The strings are no secrets.
	length := rand.IntN(maxLength-minLength+1) + minLength

	result := make([]byte, length)
	for i := range length {
		//nolint:gosec // The strings are no secrets.
		result[i] = randomStringCharset[rand.IntN(len(randomStringCharset))]
	}

	return string(result)
}
