package pathlib

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// modeFuncs returns functions that pass a mode to every function that accepts
// one. Each returns the path the mode was meant for and whether the path
// existed before.
func modeFuncs() map[string]func(t *testing.T, root *Path, mode FileMode) (*Path, bool, error) {
	return map[string]func(t *testing.T, root *Path, mode FileMode) (*Path, bool, error){
		"OpenFileWithOptions": func(_ *testing.T, root *Path, mode FileMode) (*Path, bool, error) {
			p := root.JoinStrings("file")
			file, err := OpenFileWithOptions(p, OpenOptions{CreateIfNotExists: true, CreateMode: mode, OpenMode: OpenWrite})
			if file != nil {
				_ = file.Close()
			}
			return p, false, err
		},
		"CreateFileWithOptions": func(_ *testing.T, root *Path, mode FileMode) (*Path, bool, error) {
			p := root.JoinStrings("file")
			_, err := CreateFileWithOptions(p, FileOptions{CreateMode: mode})
			return p, false, err
		},
		"WriteBytesWithOptions": func(_ *testing.T, root *Path, mode FileMode) (*Path, bool, error) {
			p := root.JoinStrings("file")
			_, err := WriteBytesWithOptions(p, []byte("content"), FileOptions{CreateMode: mode})
			return p, false, err
		},
		"MkDirWithOptions": func(_ *testing.T, root *Path, mode FileMode) (*Path, bool, error) {
			p := root.JoinStrings("dir")
			_, err := MkDirWithOptions(p, DirOptions{CreateMode: mode})
			return p, false, err
		},
		"SetMode": func(t *testing.T, root *Path, mode FileMode) (*Path, bool, error) {
			t.Helper()

			p := writeTempFile(t, root, "file", "")
			return p, true, SetMode(p, mode)
		},
	}
}

func TestFileMode_RefusesOtherBits(t *testing.T) {
	t.Parallel()

	modes := map[string]FileMode{
		"directory type bit":          ModeDir | 0o755,
		"symlink type bit":            ModeSymlink | 0o644,
		"Unix octal setuid (0o4755)":  0o4755,
		"Unix octal sticky (0o1644)":  0o1644,
		"permission above 0o777 only": 0o1000,
	}

	for funcName, apply := range modeFuncs() {
		for modeName, mode := range modes {
			t.Run(funcName+"/"+modeName, func(t *testing.T) {
				t.Parallel()

				root := setupTempDir(t)
				p, existed, err := apply(t, root, mode)

				require.ErrorIs(t, err, ErrInvalidFileMode)
				var fileModeErr *FileModeError
				require.ErrorAs(t, err, &fileModeErr)
				require.Equal(t, mode, fileModeErr.FileMode())

				if !existed {
					requireLExists(t, false, p, "nothing is created")
				}
			})
		}
	}
}

func TestFileMode_AcceptsSpecialBits(t *testing.T) {
	t.Parallel()

	modes := map[string]FileMode{
		"setuid": ModeSetuid | 0o755,
		"setgid": ModeSetgid | 0o755,
		"sticky": ModeSticky | 0o755,
	}

	for funcName, apply := range modeFuncs() {
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

func TestSetMode(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	root := setupTempDir(t)

	t.Run("file", func(t *testing.T) {
		t.Parallel()

		filePath := writeTempFile(t, root, "file.txt", "content")

		err := SetMode(filePath, 0o600)
		require.NoError(t, err)

		info, err := filePath.Stat()
		require.NoError(t, err)
		require.Equal(t, FileMode(0o600), info.Mode().Perm())

		// Change again
		err = SetMode(filePath, 0o755)
		require.NoError(t, err)

		info, err = filePath.Stat()
		require.NoError(t, err)
		require.Equal(t, FileMode(0o755), info.Mode().Perm())
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()

		dirPath := createTempDir(t, root, "permdir")

		err := SetMode(dirPath, 0o700)
		require.NoError(t, err)

		info, err := dirPath.Stat()
		require.NoError(t, err)
		require.Equal(t, FileMode(0o700), info.Mode().Perm())

		err = SetMode(dirPath, 0o755)
		require.NoError(t, err)

		info, err = dirPath.Stat()
		require.NoError(t, err)
		require.Equal(t, FileMode(0o755), info.Mode().Perm())
	})

	t.Run("non-existent path", func(t *testing.T) {
		t.Parallel()

		nonExistent := root.JoinStrings("does_not_exist")
		err := SetMode(nonExistent, 0o644)
		require.ErrorIs(t, err, ErrNotExist)
	})
}

func TestSetMode_SetsSetuidReliably(t *testing.T) {
	t.Parallel()

	if runningOnWindows {
		t.Skip("Windows has no setuid bit")
	}

	// Creating a file may drop the setuid bit (e.g. on macOS), chmod does not.
	file := writeTempFile(t, setupTempDir(t), "file", "")
	require.NoError(t, SetMode(file, ModeSetuid|0o755))

	info, err := file.Stat()
	require.NoError(t, err)
	require.Equal(t, ModeSetuid|0o755, info.Mode()&(ModePerm|ModeSpecial))
}
