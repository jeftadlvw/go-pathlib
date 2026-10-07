package pathlib

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempBaseDir(t *testing.T) {
	t.Parallel()

	tempBaseDir := TempBaseDir()

	require.True(t, tempBaseDir.Equals(NewPath(os.TempDir()), CaseSensitive))
	require.True(t, tempBaseDir.IsDir())
}

func TestCreateTempFile(t *testing.T) {
	t.Parallel()

	tempFile, dispose, err := CreateTempFile()
	require.NoError(t, err)
	defaultTempFileTests(t, tempFile, dispose)
}

func TestCreateTempFileWithOptions(t *testing.T) {
	t.Parallel()

	testWithOptions(t, CreateTempFileWithOptions, defaultTempFileTests)
}

func TestCreateTempDir(t *testing.T) {
	t.Parallel()

	tempDir, dispose, err := CreateTempDir()
	require.NoError(t, err)
	defaultTempDirTests(t, tempDir, dispose)
}

func TestCreateTempDirWithOptions(t *testing.T) {
	t.Parallel()

	testWithOptions(t, CreateTempDirWithOptions, defaultTempDirTests)
}

func testWithOptions(t *testing.T, creationFunc func(TempPathOptions) (*Path, DisposeFunc, error), defaultTests func(*testing.T, *Path, DisposeFunc)) {
	t.Helper()

	localTempBaseDir := NewPath(t.TempDir())

	t.Run("zero options", func(t *testing.T) {
		t.Parallel()

		tempFile, dispose, err := creationFunc(TempPathOptions{})
		require.NoError(t, err)
		defaultTests(t, tempFile, dispose)

		require.True(t, tempFile.Parent().Equals(TempBaseDir(), CaseSensitive))
	})

	baseDirCases := []*Path{
		nil,
		NewPath(""),
		localTempBaseDir,
	}

	prefixCases := []string{
		"",
		"foo",
		"foo__",
		generateRandomString(0, 20),
		generateRandomString(0, 20),
		generateRandomString(0, 20),
	}

	for baseDirIdx, baseDir := range baseDirCases {
		t.Run(fmt.Sprintf("baseDirIdx-%d_only", baseDirIdx), func(t *testing.T) {
			t.Parallel()

			options := TempPathOptions{
				BaseDir: baseDir,
			}

			tempFile, dispose, err := creationFunc(options)
			require.NoError(t, err)
			defaultTests(t, tempFile, dispose)

			defaultTempPathOptionsTests(t, tempFile, options)
		})

		for prefixIdx, prefix := range prefixCases {
			t.Run(fmt.Sprintf("baseDirIdx-%d_prefixIdx-%d", baseDirIdx, prefixIdx), func(t *testing.T) {
				t.Parallel()

				options := TempPathOptions{
					BaseDir: baseDir,
					Prefix:  prefix,
				}

				tempFile, dispose, err := creationFunc(options)
				require.NoError(t, err)
				defaultTests(t, tempFile, dispose)

				defaultTempPathOptionsTests(t, tempFile, options)
			})
		}
	}

	for prefixIdx, prefix := range prefixCases {
		t.Run(fmt.Sprintf("prefixIdx-%d_only", prefixIdx), func(t *testing.T) {
			t.Parallel()

			options := TempPathOptions{
				Prefix: prefix,
			}

			tempFile, dispose, err := creationFunc(options)
			require.NoError(t, err)
			defaultTests(t, tempFile, dispose)

			defaultTempPathOptionsTests(t, tempFile, options)
		})
	}
}

func defaultTempFileTests(t *testing.T, p *Path, dispose DisposeFunc) {
	t.Helper()

	preDisposeStat, err := os.Stat(p.String())
	require.NoError(t, err)
	require.False(t, preDisposeStat.IsDir())

	err = dispose()
	require.NoError(t, err)

	postDisposeStat, err := os.Stat(p.String())
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.Nil(t, postDisposeStat)
}

func defaultTempDirTests(t *testing.T, p *Path, dispose DisposeFunc) {
	t.Helper()

	preDisposeStat, err := os.Stat(p.String())
	require.NoError(t, err)
	require.True(t, preDisposeStat.IsDir())

	err = dispose()
	require.NoError(t, err)

	postDisposeStat, err := os.Stat(p.String())
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.Nil(t, postDisposeStat)
}

func defaultTempPathOptionsTests(t *testing.T, p *Path, opts TempPathOptions) {
	t.Helper()

	expectedBaseDir := TempBaseDir()
	if opts.BaseDir != nil && opts.BaseDir.String() != "" {
		expectedBaseDir = opts.BaseDir
	}

	expectedPrefix := opts.Prefix

	require.True(t, p.Parent().Equals(expectedBaseDir, CaseSensitive))
	require.True(t, strings.HasPrefix(p.Base(), expectedPrefix))
}

func TestDisposeFunc_IsIdempotent(t *testing.T) {
	t.Parallel()

	for name, create := range map[string]func() (*Path, DisposeFunc, error){
		"CreateTempFile": CreateTempFile,
		"CreateTempDir":  CreateTempDir,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, dispose, err := create()
			require.NoError(t, err)

			require.NoError(t, dispose())
			require.NoError(t, dispose(), "a second call is a no-op")
			requireLExists(t, false, p)
		})
	}
}

func TestDisposeFunc_MovedPathIsNoOp(t *testing.T) {
	t.Parallel()

	for name, create := range map[string]func(TempPathOptions) (*Path, DisposeFunc, error){
		"CreateTempFileWithOptions": CreateTempFileWithOptions,
		"CreateTempDirWithOptions":  CreateTempDirWithOptions,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := setupTempDir(t)
			p, dispose, err := create(TempPathOptions{BaseDir: root})
			require.NoError(t, err)

			destination := root.JoinStrings("moved")
			require.NoError(t, Move(p, destination))

			require.NoError(t, dispose())
			requireLExists(t, true, destination, "the moved path is kept")
		})
	}
}

func TestDisposeFunc_IsNoOpOnError(t *testing.T) {
	t.Parallel()

	missing := NewPath(t.TempDir()).JoinStrings("missing")

	for name, create := range map[string]func(TempPathOptions) (*Path, DisposeFunc, error){
		"CreateTempFileWithOptions": CreateTempFileWithOptions,
		"CreateTempDirWithOptions":  CreateTempDirWithOptions,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, dispose, err := create(TempPathOptions{BaseDir: missing})
			require.ErrorIs(t, err, ErrNotExist)
			require.Nil(t, p)
			require.NotNil(t, dispose)
			require.NoError(t, dispose())
		})
	}
}

func TestDisposeFunc_RemovesSymlinkNotTarget(t *testing.T) {
	t.Parallel()

	// A temporary path replaced by a symlink: disposing must not reach the
	// target.
	root := setupTempDir(t)
	targetFile := writeTempFile(t, root, "keep.txt", "content")
	targetDir := createTempDir(t, root, "keep_dir")
	writeTempFile(t, targetDir, "inner.txt", "")

	t.Run("file", func(t *testing.T) {
		t.Parallel()

		file, dispose, err := CreateTempFile()
		require.NoError(t, err)
		require.NoError(t, os.Remove(file.String()))
		require.NoError(t, os.Symlink(targetFile.String(), file.String()))

		require.NoError(t, dispose())
		requireLExists(t, false, file)
		require.True(t, targetFile.IsFile(), "the symlink target is kept")
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()

		dir, dispose, err := CreateTempDir()
		require.NoError(t, err)
		require.NoError(t, os.Remove(dir.String()))
		require.NoError(t, os.Symlink(targetDir.String(), dir.String()))

		require.NoError(t, dispose())
		requireLExists(t, false, dir)
		require.True(t, targetDir.JoinStrings("inner.txt").IsFile(), "the symlink target's content is kept")
	})
}

func TestDisposeFunc_IsIndependentOfReturnedPath(t *testing.T) {
	t.Parallel()

	// Overwriting the returned Path does not redirect what dispose removes.
	file, dispose, err := CreateTempFile()
	require.NoError(t, err)

	keep := writeTempFile(t, setupTempDir(t), "keep.txt", "content")
	original := file.Copy()
	require.NoError(t, file.UnmarshalText([]byte(keep.ToPosix())))

	require.NoError(t, dispose())
	requireLExists(t, false, original)
	require.True(t, keep.IsFile(), "the file the returned Path now points to is kept")
}
