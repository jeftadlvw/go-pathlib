package pathlib

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"strings"
	"testing"
)

func TestTempBaseDir(t *testing.T) {
	tempBaseDir := TempBaseDir()

	assert.True(t, tempBaseDir.Equals(NewPath(os.TempDir())))
	assert.True(t, tempBaseDir.IsDir())
}

func TestCreateTempFile(t *testing.T) {
	tempFile, err := CreateTempFile()
	assert.NoError(t, err)
	defaultTempFileTests(t, tempFile)
}

func TestCreateTempFileWithOptions(t *testing.T) {
	testWithOptions(t, CreateTempFileWithOptions, defaultTempFileTests)
}

func TestCreateTempDir(t *testing.T) {
	tempDir, err := CreateTempDir()
	assert.NoError(t, err)
	defaultTempDirTests(t, tempDir)
}

func TestCreateTempDirWithOptions(t *testing.T) {
	testWithOptions(t, CreateTempDirWithOptions, defaultTempDirTests)
}

func testWithOptions(t *testing.T, creationFunc func(*TempPathOptions) (*TempPath, error), defaultTests func(*testing.T, *TempPath)) {
	localTempBaseDir := NewPath(t.TempDir())

	t.Run("nil", func(t *testing.T) {
		tempFile, err := creationFunc(nil)
		assert.NoError(t, err)
		defaultTests(t, tempFile)
	})

	t.Run("empty", func(t *testing.T) {
		tempFile, err := creationFunc(&TempPathOptions{})
		assert.NoError(t, err)
		defaultTests(t, tempFile)

		assert.True(t, tempFile.Parent().Equals(TempBaseDir()))
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
			assert.NoError(t, err)
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
				assert.NoError(t, err)
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
			assert.NoError(t, err)
			defaultTests(t, tempFile)

			defaultTempPathOptionsTests(t, tempFile, options)
		})
	}
}

func defaultTempFileTests(t *testing.T, p *TempPath) {
	assert.True(t, p.IsFile())
	assert.False(t, p.IsDir())

	err := p.Dispose()
	assert.NoError(t, err)

	assert.False(t, p.IsFile())
	assert.False(t, p.IsDir())
	assert.False(t, p.Exists())
}

func defaultTempDirTests(t *testing.T, p *TempPath) {
	assert.True(t, p.IsDir())
	assert.False(t, p.IsFile())

	err := p.Dispose()
	assert.NoError(t, err)

	assert.False(t, p.IsDir())
	assert.False(t, p.IsFile())
	assert.False(t, p.Exists())
}

func defaultTempPathOptionsTests(t *testing.T, p *TempPath, opts *TempPathOptions) {
	expectedBaseDir := TempBaseDir()
	if opts.BaseDir != nil && opts.BaseDir.String() != "" {
		expectedBaseDir = opts.BaseDir
	}

	expectedPrefix := opts.Prefix

	assert.True(t, p.Parent().Equals(expectedBaseDir))
	assert.True(t, strings.HasPrefix(p.Base(), expectedPrefix))
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomString(minLength, maxLength int) string {
	// Generate a random length between minLength and maxLength
	length := rand.Intn(maxLength-minLength+1) + minLength

	// Create a byte slice to store the random string
	result := make([]byte, length)

	// Fill the byte slice with random characters from the charset
	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}
