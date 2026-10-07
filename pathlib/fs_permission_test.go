package pathlib

import (
	"io/fs"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// permissionFuncs returns functions that pass a mode to every function that
// accepts a permission. Each returns the path the mode was meant for and
// whether the path existed before.
func permissionFuncs() map[string]func(t *testing.T, root *Path, mode fs.FileMode) (*Path, bool, error) {
	return map[string]func(t *testing.T, root *Path, mode fs.FileMode) (*Path, bool, error){
		"OpenFileWithOptions": func(_ *testing.T, root *Path, mode fs.FileMode) (*Path, bool, error) {
			p := root.JoinStrings("file")
			file, err := OpenFileWithOptions(p, OpenOptions{CreateIfNotExists: true, Permission: mode, Mode: "w"})
			if file != nil {
				_ = file.Close()
			}
			return p, false, err
		},
		"CreateFileWithOptions": func(_ *testing.T, root *Path, mode fs.FileMode) (*Path, bool, error) {
			p := root.JoinStrings("file")
			_, err := CreateFileWithOptions(p, FileOptions{Mode: mode})
			return p, false, err
		},
		"WriteBytesWithOptions": func(_ *testing.T, root *Path, mode fs.FileMode) (*Path, bool, error) {
			p := root.JoinStrings("file")
			_, err := WriteBytesWithOptions(p, []byte("content"), FileOptions{Mode: mode})
			return p, false, err
		},
		"MkDirWithOptions": func(_ *testing.T, root *Path, mode fs.FileMode) (*Path, bool, error) {
			p := root.JoinStrings("dir")
			_, err := MkDirWithOptions(p, DirOptions{Mode: mode})
			return p, false, err
		},
		"SetPermission": func(t *testing.T, root *Path, mode fs.FileMode) (*Path, bool, error) {
			t.Helper()

			p := writeTempFile(t, root, "file", "")
			return p, true, SetPermission(p, mode)
		},
	}
}

func TestPermissionBits_RefusesOtherBits(t *testing.T) {
	t.Parallel()

	modes := map[string]fs.FileMode{
		"directory type bit":          fs.ModeDir | 0755,
		"symlink type bit":            fs.ModeSymlink | 0644,
		"Unix octal setuid (0o4755)":  0o4755,
		"Unix octal sticky (0o1644)":  0o1644,
		"permission above 0o777 only": 0o1000,
	}

	for funcName, apply := range permissionFuncs() {
		for modeName, mode := range modes {
			t.Run(funcName+"/"+modeName, func(t *testing.T) {
				t.Parallel()

				root := setupTempDir(t)
				p, existed, err := apply(t, root, mode)

				require.ErrorIs(t, err, ErrInvalidPermission)
				require.ErrorIs(t, err, ErrInvalidPermission)
				var permErr *PermissionError
				require.ErrorAs(t, err, &permErr)
				require.Equal(t, mode, permErr.Perm())

				if !existed {
					requireLExists(t, false, p, "nothing is created")
				}
			})
		}
	}
}

func TestPermissionBits_AcceptsSpecialBits(t *testing.T) {
	t.Parallel()

	modes := map[string]fs.FileMode{
		"setuid": fs.ModeSetuid | 0755,
		"setgid": fs.ModeSetgid | 0755,
		"sticky": fs.ModeSticky | 0755,
	}

	for funcName, apply := range permissionFuncs() {
		for modeName, mode := range modes {
			t.Run(funcName+"/"+modeName, func(t *testing.T) {
				t.Parallel()

				p, _, err := apply(t, setupTempDir(t), mode)
				require.NoError(t, err)
				requireLExists(t, true, p)
			})
		}
	}
}

func TestSetPermission(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	root := setupTempDir(t)

	t.Run("file", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

		nonExistent := root.JoinStrings("does_not_exist")
		err := SetPermission(nonExistent, 0644)
		require.ErrorIs(t, err, ErrNotExist)
	})
}

func TestSetPermission_SetsSetuidReliably(t *testing.T) {
	t.Parallel()

	if runningOnWindows {
		t.Skip("Windows has no setuid bit")
	}

	// Creating a file may drop the setuid bit (e.g. on macOS), chmod does not.
	file := writeTempFile(t, setupTempDir(t), "file", "")
	require.NoError(t, SetPermission(file, fs.ModeSetuid|0755))

	info, err := file.Stat()
	require.NoError(t, err)
	require.Equal(t, fs.ModeSetuid|0755, info.Mode()&PermissionBits)
}
