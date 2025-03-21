package pathlib

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestFsDefaults(t *testing.T) {
	require.Equal(t, 0644, defaultOpenPermission, "default open permission mismatch")
	require.Equal(t, 0755, int(defaultDirMode), "default directory mode mismatch")
}

func TestPathExistings(t *testing.T) {
	tempDirStr := t.TempDir()

	fileName := "tempfile.lol"
	existingFileStr := filepath.Join(tempDirStr, fileName)
	existingFilePath := NewPath(existingFileStr)
	file, err := os.OpenFile(existingFileStr, os.O_RDONLY|os.O_CREATE, 0666)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	randomStr := "f8safkjh3asd09aslkja23shj87asds34"
	nonExistingPathStr := filepath.Join(tempDirStr, randomStr)

	tempDirPath := NewPath(tempDirStr)
	nonExistingPath := NewPath(nonExistingPathStr)

	// test temporary directory
	t.Run("temporary directory", func(t *testing.T) {
		require.True(t, tempDirPath.Exists())
		require.True(t, tempDirPath.IsDir())
		require.False(t, tempDirPath.IsFile())
	})

	// exist existing file
	t.Run("existing file", func(t *testing.T) {
		require.True(t, existingFilePath.Exists())
		require.False(t, existingFilePath.IsDir())
		require.True(t, existingFilePath.IsFile())
	})

	// test non-existing path
	t.Run("non-existing file", func(t *testing.T) {
		require.False(t, nonExistingPath.Exists())
		require.False(t, nonExistingPath.IsDir())
		require.False(t, nonExistingPath.IsFile())
	})
}

func TestPath_Resolve(t *testing.T) {
	cwdPath, err := NewCwd()
	require.NoError(t, err)

	// temporary directory
	tempPath := NewPath(t.TempDir()) // creates temporary directory at /var

	// temporary file paths
	originalFile := "originalFile"
	symlinkFile := "symlinkedFile"

	originalFilePath := NewPath("/private").JoinStrings(tempPath.String(), originalFile) // /var is actually a symlink to /private/var
	symlinkFilePath := tempPath.JoinStrings(symlinkFile)

	// create original file
	file, err := os.OpenFile(originalFilePath.String(), os.O_RDONLY|os.O_CREATE, 0666)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	// create symlink
	err = os.Symlink(originalFilePath.String(), symlinkFilePath.String())
	require.NoError(t, err)

	cases := []TestCase[*Path, *Path]{
		{Input: NewPath("."), Expect: cwdPath},
		{Input: NewPath("./i_do_not_exist"), Error: true},
		{Input: NewPath("/"), Expect: NewPath("/")},
		{Input: originalFilePath, Expect: originalFilePath},
		{Input: symlinkFilePath, Expect: originalFilePath},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResultsE(t, cases, func(t *testing.T, input *Path, expect *Path, error bool) {
		resolvedPath, err := input.Resolve()
		fmt.Println(resolvedPath)

		require.Equal(t, error, err != nil)

		if !error {
			require.Equal(t, expect, resolvedPath)
		}
	})
}

func TestPath_GlobContains(t *testing.T) {

	// NOTICE:
	// This test only tests the existence check mechanism and
	// matched filepath string conversions

	// temporary directory
	tempPath := NewPath(t.TempDir()) // creates temporary directory at /var

	// create a file
	existingFile := "foo"
	existingFilePath := tempPath.JoinStrings(existingFile)
	file, err := os.OpenFile(existingFilePath.String(), os.O_RDONLY|os.O_CREATE, 0666)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	// create subdirectory
	existingSubdirectory := "bar"
	existingSubdirectoryPath := tempPath.JoinStrings(existingSubdirectory)
	err = os.Mkdir(existingSubdirectoryPath.String(), 0777)
	require.NoError(t, err)

	// create file in subdirectory
	existingSubdirFile := "baz"
	existingSubdirFilePath := existingSubdirectoryPath.JoinStrings(existingSubdirFile)
	subdirFile, err := os.OpenFile(existingSubdirFilePath.String(), os.O_RDONLY|os.O_CREATE, 0666)
	require.NoError(t, err)

	err = subdirFile.Close()
	require.NoError(t, err)

	// test cases; the first string is the root path
	// starting at the temporary directory, the second
	// string is the pattern to search for

	// TODO support double asterisks globbing, without adding dependencies
	// see https://github.com/golang/go/issues/11862
	cases := []TestCase[[]string, int]{
		{Input: []string{"", ""}, Error: true},
		{Input: []string{"", "  "}, Error: true},
		{Input: []string{"", "  \t"}, Error: true},
		{Input: []string{"", " \t \n  "}, Error: true},
		{Input: []string{"", "*"}, Expect: 2},
		{Input: []string{"", "/*"}, Expect: 2},
		{Input: []string{"", "**"}, Expect: 2},
		{Input: []string{"", "*/*"}, Expect: 1},
		{Input: []string{"", "bar/*"}, Expect: 1},
		{Input: []string{"", "bar/bar"}, Expect: 0},
		{Input: []string{"", "bar/baz"}, Expect: 1},
		{Input: []string{"", "bar/*z"}, Expect: 1},
		{Input: []string{"", "bat/*z"}, Expect: 0},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResultsE(t, cases, func(t *testing.T, input []string, expect int, error bool) {
		require.Len(t, input, 2)
		require.GreaterOrEqual(t, expect, 0)

		path := tempPath.JoinStrings(input[0])
		pattern := input[1]

		matches, globErr := path.Glob(pattern)
		contains, containsErr := path.Contains(pattern)
		containsB := path.BContains(pattern)

		require.Equal(t, error, globErr != nil)
		require.Equal(t, error, containsErr != nil)

		if !error {
			require.Len(t, matches, expect)
			require.Equal(t, expect != 0, contains)
			require.Equal(t, contains, containsB)
		}
	})
}
