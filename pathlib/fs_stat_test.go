package pathlib

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_Resolve(t *testing.T) {
	type Input struct {
		PathStr string
		Setup   func(*testing.T, *Path) *Path // Function to set up the path in the temp dir
	}

	cases := []TestCase[Input, string]{
		{
			Name:   "Regular file",
			Input:  Input{PathStr: "file.txt", Setup: func(t *testing.T, root *Path) *Path { return writeTempFile(t, root, "file.txt", "content") }},
			Expect: "file.txt",
			Error:  false,
		},
		{
			Name:   "Directory",
			Input:  Input{PathStr: "dir", Setup: func(t *testing.T, root *Path) *Path { return createTempDir(t, root, "dir") }},
			Expect: "dir",
			Error:  false,
		},
		{
			Name:  "Non-existent path",
			Input: Input{PathStr: "nonexistent", Setup: func(t *testing.T, root *Path) *Path { return root.JoinStrings("nonexistent") }},
			Error: true,
		},
		{
			Name: "Symlink to file",
			Input: Input{
				PathStr: "linkToFile.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "target.txt", "content")
					return createTempSymlinkAbs(t, root, "target.txt", "linkToFile.txt")
				},
			},
			Expect: "target.txt",
			Error:  false,
		},
		{
			Name: "Symlink to directory",
			Input: Input{
				PathStr: "linkToDir",
				Setup: func(t *testing.T, root *Path) *Path {
					createTempDir(t, root, "targetDir")
					return createTempSymlinkAbs(t, root, "targetDir", "linkToDir")
				},
			},
			Expect: "targetDir",
			Error:  false,
		},
		{
			Name: "Chained symlinks",
			Input: Input{
				PathStr: "link1",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "final.txt", "content")
					createTempSymlinkAbs(t, root, "final.txt", "link2")
					return createTempSymlinkAbs(t, root, "link2", "link1")
				},
			},
			Expect: "final.txt",
			Error:  false,
		},
		{
			Name: "Broken symlink",
			Input: Input{
				PathStr: "brokenLink",
				Setup: func(t *testing.T, root *Path) *Path {
					return createTempSymlinkAbs(t, root, "nonexistentTarget", "brokenLink")
				},
			},
			Error: true,
		},
		{
			Name: "Relative path should resolve to absolute",
			Input: Input{
				PathStr: "dir/../file.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "file.txt", "content")
					createTempDir(t, root, "dir")
					return root.JoinStrings("dir/../file.txt")
				},
			},
			Expect: "file.txt",
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect string, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)

		resolvedPath, err := p.Resolve()

		if expectError {
			require.Error(t, err)
			require.Nil(t, resolvedPath)
			return
		}

		require.NoError(t, err)
		require.NotNil(t, resolvedPath)

		// Ensure the resolved path is absolute
		require.True(t, resolvedPath.IsAbsolute())

		// Cut "/private" from path so that RelativeTo will succeed
		if runtime.GOOS == "darwin" {
			resolvedPath.path = strings.TrimPrefix(resolvedPath.path, "/private")
		}

		// Compare relative to root for easier testing
		relPath, err := resolvedPath.RelativeTo(root)
		require.NoError(t, err)

		require.Equal(t, expect, relPath.ToPosix())
	})
}

func TestPath_IsFile(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: true,
		},
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "content")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsFile())
	})
}

func TestPath_IsDir(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: true,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: true,
		},
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "content")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsDir())
	})
}

func TestPath_IsEmptyDir(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Empty directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "emptyDir")
			},
			Expect: true,
		},
		{
			Name: "Non-empty directory",
			Input: func(t *testing.T, root *Path) *Path {
				dir := createTempDir(t, root, "nonEmptyDir")
				writeTempFile(t, dir, "file.txt", "content")
				return dir
			},
			Expect: false,
		},
		{
			Name: "Directory with subdirectory only",
			Input: func(t *testing.T, root *Path) *Path {
				dir := createTempDir(t, root, "dirWithSubdir")
				createTempDir(t, dir, "subdir")
				return dir
			},
			Expect: false,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsEmptyDir())
	})
}

func TestPath_Exists(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Existing file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			},
			Expect: true,
		},
		{
			Name: "Existing directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: true,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
		{
			Name: "Symlink to existing file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Broken symlink",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempSymlinkAbs(t, root, "nonexistent_target", "broken_link")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.Exists())
	})
}

func TestPath_IsSymlink(t *testing.T) {
	cases := []TestCase[func(*testing.T, *Path) *Path, bool]{
		{
			Name: "Symlink to file",
			Input: func(t *testing.T, root *Path) *Path {
				writeTempFile(t, root, "target.txt", "")
				return createTempSymlinkAbs(t, root, "target.txt", "link.txt")
			},
			Expect: true,
		},
		{
			Name: "Symlink to directory",
			Input: func(t *testing.T, root *Path) *Path {
				createTempDir(t, root, "targetDir")
				return createTempSymlinkAbs(t, root, "targetDir", "linkDir")
			},
			Expect: true,
		},
		{
			Name: "Broken symlink",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempSymlinkAbs(t, root, "nonexistent", "broken_link")
			},
			Expect: true,
		},
		{
			Name: "Regular file",
			Input: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "")
			},
			Expect: false,
		},
		{
			Name: "Directory",
			Input: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			},
			Expect: false,
		},
		{
			Name: "Non-existent path",
			Input: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			},
			Expect: false,
		},
	}

	runForResults(t, cases, func(t *testing.T, input func(*testing.T, *Path) *Path, expect bool) {
		root := setupTempDir(t)
		p := input(t, root)
		require.Equal(t, expect, p.IsSymlink())
	})
}

func TestPath_Stat(t *testing.T) {
	type Input struct {
		Setup func(*testing.T, *Path) *Path
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Stat regular file",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "hello")
			}},
			Error: false,
		},
		{
			Name: "Stat directory",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "dir")
			}},
			Error: false,
		},
		{
			Name: "Stat non-existent path",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			}},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		info, err := p.Stat()
		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.NotNil(t, info)
	})
}

func TestPath_Lstat(t *testing.T) {
	root := setupTempDir(t)

	writeTempFile(t, root, "target.txt", "content")
	link := createTempSymlinkAbs(t, root, "target.txt", "link.txt")

	info, err := link.Lstat()
	require.NoError(t, err)
	require.NotNil(t, info)
	require.True(t, info.Mode()&os.ModeSymlink != 0, "Lstat should report symlink mode")
}

func TestDeviceMethods(t *testing.T) {
	root := setupTempDir(t)
	filePath := writeTempFile(t, root, "regular.txt", "content")
	dirPath := createTempDir(t, root, "subdir")
	nonExistent := root.JoinStrings("does_not_exist")

	t.Run("IsBlockDevice", func(t *testing.T) {
		require.False(t, filePath.IsBlockDevice(), "regular file is not a block device")
		require.False(t, dirPath.IsBlockDevice(), "directory is not a block device")
		require.False(t, nonExistent.IsBlockDevice(), "non-existent path is not a block device")
	})

	t.Run("IsCharDevice", func(t *testing.T) {
		require.False(t, filePath.IsCharDevice(), "regular file is not a char device")
		require.False(t, dirPath.IsCharDevice(), "directory is not a char device")
		require.False(t, nonExistent.IsCharDevice(), "non-existent path is not a char device")

		if notRunningOnWindows {
			devNull := NewPath("/dev/null")
			require.True(t, devNull.IsCharDevice(), "/dev/null should be a char device")
		}
	})

	t.Run("IsFiFoPipe", func(t *testing.T) {
		require.False(t, filePath.IsFiFoPipe(), "regular file is not a FIFO pipe")
		require.False(t, dirPath.IsFiFoPipe(), "directory is not a FIFO pipe")
		require.False(t, nonExistent.IsFiFoPipe(), "non-existent path is not a FIFO pipe")
	})

	t.Run("IsSocket", func(t *testing.T) {
		require.False(t, filePath.IsSocket(), "regular file is not a socket")
		require.False(t, dirPath.IsSocket(), "directory is not a socket")
		require.False(t, nonExistent.IsSocket(), "non-existent path is not a socket")
	})
}
