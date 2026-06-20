package pathlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_IsOnCaseSensitiveFs(t *testing.T) {
	// This test is tricky because file system case-sensitivity depends on the OS/FS.
	// We'll create a temp directory and use its behavior to determine the expected outcome.
	// The function's internal logic already handles the detection.

	root := setupTempDir(t)

	// Create a test file
	filePath := writeTempFile(t, root, "TestFile.txt", "content")
	mixedCasePath := root.JoinStrings("testfile.txt") // Attempt a different casing

	// Determine expected behavior by trying to access the same inode
	stat1, err1 := filePath.Stat()
	stat2, err2 := mixedCasePath.Stat()

	// The filesystem is case-sensitive if:
	// - the mixed-case path doesn't exist (err2 != nil), or
	// - both exist but point to different inodes
	isFsCaseSensitiveExpected := err1 == nil && (err2 != nil || !os.SameFile(stat1, stat2))

	cases := []TestCase[*Path, bool]{
		{
			Name:   "Test with file having a letter in base",
			Input:  filePath,
			Expect: isFsCaseSensitiveExpected,
		},
		{
			Name: "Test with directory having a letter in base",
			Input: func() *Path {
				dirPath := createTempDir(t, root, "testDir")
				return dirPath
			}(),
			Expect: isFsCaseSensitiveExpected,
		},
		{
			Name: "Test with file having no letters in base (should create temp)",
			Input: func() *Path {
				// Path with no letters, e.g., only numbers or symbols
				noLetterPath := writeTempFile(t, root, "12345.ext", "content")
				return noLetterPath
			}(),
			Expect: isFsCaseSensitiveExpected,
		},
		{
			Name: "Test with non-existent path (should return false)",
			Input: func() *Path {
				return root.JoinStrings("non_existent_path")
			}(),
			Expect: false, // Should always return false for non-existent path
		},
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect bool) {
		actual := input.IsOnCaseSensitiveFs()
		require.Equal(t, expect, actual)
	})
}

func TestPath_EqualsFs(t *testing.T) {
	type Input struct {
		Setup func(*testing.T, *Path) (*Path, *Path)
	}

	cases := []TestCase[Input, bool]{
		{
			Name: "Same file",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f := writeTempFile(t, root, "file.txt", "content")
				return f, f
			}},
			Expect: true,
		},
		{
			Name: "Symlink and its target",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f := writeTempFile(t, root, "target.txt", "content")
				l := createTempSymlinkAbs(t, root, "target.txt", "link.txt")
				return f, l
			}},
			Expect: true,
		},
		{
			Name: "Different files",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f1 := writeTempFile(t, root, "file1.txt", "content")
				f2 := writeTempFile(t, root, "file2.txt", "content")
				return f1, f2
			}},
			Expect: false,
		},
		{
			Name: "One path does not exist",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				f := writeTempFile(t, root, "file.txt", "content")
				return f, root.JoinStrings("nonexistent")
			}},
			Expect: false,
		},
		{
			Name: "Both paths do not exist",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				return root.JoinStrings("a"), root.JoinStrings("b")
			}},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input Input, expect bool) {
		root := setupTempDir(t)
		p1, p2 := input.Setup(t, root)
		require.Equal(t, expect, p1.EqualsFs(p2))
	})
}
