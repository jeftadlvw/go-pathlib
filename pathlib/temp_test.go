package pathlib

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempBaseDir(t *testing.T) {
	tempBaseDir := TempBaseDir()

	require.True(t, tempBaseDir.Equals(NewPath(os.TempDir()), CaseSensitive))
	require.True(t, tempBaseDir.IsDir())
}

func TestCreateTempFile(t *testing.T) {
	tempFile, err := CreateTempFile()
	require.NoError(t, err)
	defaultTempFileTests(t, tempFile)
}

func TestCreateTempFileWithOptions(t *testing.T) {
	testWithOptions(t, CreateTempFileWithOptions, defaultTempFileTests)
}

func TestCreateTempDir(t *testing.T) {
	tempDir, err := CreateTempDir()
	require.NoError(t, err)
	defaultTempDirTests(t, tempDir)
}

func TestCreateTempDirWithOptions(t *testing.T) {
	testWithOptions(t, CreateTempDirWithOptions, defaultTempDirTests)
}

func testWithOptions(t *testing.T, creationFunc func(*TempPathOptions) (*TempPath, error), defaultTests func(*testing.T, *TempPath)) {
	localTempBaseDir := NewPath(t.TempDir())

	t.Run("nil", func(t *testing.T) {
		tempFile, err := creationFunc(nil)
		require.NoError(t, err)
		defaultTests(t, tempFile)
	})

	t.Run("empty", func(t *testing.T) {
		tempFile, err := creationFunc(&TempPathOptions{})
		require.NoError(t, err)
		defaultTests(t, tempFile)

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
			options := &TempPathOptions{
				BaseDir: baseDir,
			}

			tempFile, err := creationFunc(options)
			require.NoError(t, err)
			defaultTests(t, tempFile)

			defaultTempPathOptionsTests(t, tempFile, options)
		})

		for prefixIdx, prefix := range prefixCases {
			t.Run(fmt.Sprintf("baseDirIdx-%d_prefixIdx-%d", baseDirIdx, prefixIdx), func(t *testing.T) {
				options := &TempPathOptions{
					BaseDir: baseDir,
					Prefix:  prefix,
				}

				tempFile, err := creationFunc(options)
				require.NoError(t, err)
				defaultTests(t, tempFile)

				defaultTempPathOptionsTests(t, tempFile, options)
			})
		}
	}

	for prefixIdx, prefix := range prefixCases {
		t.Run(fmt.Sprintf("prefixIdx-%d_only", prefixIdx), func(t *testing.T) {
			options := &TempPathOptions{
				Prefix: prefix,
			}

			tempFile, err := creationFunc(options)
			require.NoError(t, err)
			defaultTests(t, tempFile)

			defaultTempPathOptionsTests(t, tempFile, options)
		})
	}
}

func defaultTempFileTests(t *testing.T, p *TempPath) {
	preDisposeStat, err := os.Stat(p.String())
	require.NoError(t, err)
	require.False(t, preDisposeStat.IsDir())

	err = p.Dispose()
	require.NoError(t, err)

	postDisposeStat, err := os.Stat(p.String())
	require.Error(t, err)
	require.Nil(t, postDisposeStat)
}

func defaultTempDirTests(t *testing.T, p *TempPath) {
	preDisposeStat, err := os.Stat(p.String())
	require.NoError(t, err)
	require.True(t, preDisposeStat.IsDir())

	err = p.Dispose()
	require.NoError(t, err)

	postDisposeStat, err := os.Stat(p.String())
	require.Error(t, err)
	require.Nil(t, postDisposeStat)
}

func defaultTempPathOptionsTests(t *testing.T, p *TempPath, opts *TempPathOptions) {
	expectedBaseDir := TempBaseDir()
	if opts.BaseDir != nil && opts.BaseDir.String() != "" {
		expectedBaseDir = opts.BaseDir
	}

	expectedPrefix := opts.Prefix

	require.True(t, p.Parent().Equals(expectedBaseDir, CaseSensitive))
	require.True(t, strings.HasPrefix(p.Base(), expectedPrefix))
}
