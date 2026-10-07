package pathlib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateFileWithOptions(t *testing.T) {
	t.Parallel()

	type Input struct {
		RelPath string
		Options FileOptions
	}
	type Expect struct {
		Created bool
		Content string // To verify if truncated or not
	}

	cases := []TestCase[Input, Expect]{
		{
			Name:   "Successful creation",
			Input:  Input{"new_file.txt", DefaultFileOptions()},
			Expect: Expect{Created: true},
			Error:  false,
		},
		{
			Name:   "ExistOk=true, file exists",
			Input:  Input{"existing_file.txt", FileOptions{ExistOk: true, CreateMode: DefaultFileMode()}},
			Expect: Expect{Created: false, Content: "original content"},
			Error:  false,
		},
		{
			Name:   "ExistOk=false, file exists",
			Input:  Input{"existing_file.txt", FileOptions{ExistOk: false, CreateMode: DefaultFileMode()}},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Path exists and is a directory",
			Input:  Input{"existing_dir", DefaultFileOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Parent directory does exist, file does not exist but ExistOk=false",
			Input:  Input{"existent_dir/file.txt", DefaultFileOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Parent directory does not exist",
			Input:  Input{"non_existent_dir/file.txt", DefaultFileOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "CreateMode=0 should default to DefaultFileMode",
			Input:  Input{"file_with_mode0.txt", FileOptions{ExistOk: false, CreateMode: 0}},
			Expect: Expect{Created: true},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		targetPath := root.JoinStrings(input.RelPath)

		// Pre-create some paths for specific test cases
		if strings.Contains(input.RelPath, "existing_file.txt") {
			writeTempFile(t, root, "existing_file.txt", "original content")
		}
		if strings.Contains(input.RelPath, "truncate_me.txt") {
			writeTempFile(t, root, "truncate_me.txt", "original content")
		}
		if strings.Contains(input.RelPath, "existing_dir") {
			createTempDir(t, root, "existing_dir")
		}

		created, err := CreateFileWithOptions(targetPath, input.Options)

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			return
		}

		require.NoError(t, err)
		require.Equal(t, expect.Created, created)
		require.True(t, targetPath.Exists())
		require.True(t, targetPath.IsFile())

		info, err := targetPath.Stat()
		require.NoError(t, err)

		expectedMode := input.Options.CreateMode
		if expectedMode == 0 {
			expectedMode = DefaultFileMode()
		}
		require.Equal(t, effectiveFileMode(expectedMode).Perm(), info.Mode().Perm())

		if expect.Content != "" || (input.RelPath == "truncate_me.txt") {
			// Check content for truncation test, or if specific content is
			// expected
			content, err := os.ReadFile(targetPath.String())
			require.NoError(t, err)
			require.Equal(t, expect.Content, string(content))
		}
	})
}

// updateTimesInput selects the options of a creation function in the
// UpdateTimes tests.
type updateTimesInput struct {
	ExistOk     bool
	UpdateTimes bool
}

// updateTimesExpect is the expected result of a creation function for an
// existing entry in the UpdateTimes tests.
type updateTimesExpect struct {
	// Err is the expected error kind, or nil for success.
	Err *PathlibError

	// Updated reports whether the modification time is set to the current
	// time.
	Updated bool
}

// runUpdateTimesCases calls create with the options of every case on an
// existing entry made by setup, whose modification time is pastTime. existErr
// is the kind create returns for an existing entry without ExistOk.
func runUpdateTimesCases(
	t *testing.T,
	setup func(*testing.T, *Path) *Path,
	create func(*Path, updateTimesInput) (bool, error),
	existErr *PathlibError,
) {
	t.Helper()

	cases := []TestCase[updateTimesInput, updateTimesExpect]{
		{
			Name:   "ExistOk and UpdateTimes set the times",
			Input:  updateTimesInput{ExistOk: true, UpdateTimes: true},
			Expect: updateTimesExpect{Updated: true},
		},
		{
			Name:   "ExistOk alone leaves the times untouched",
			Input:  updateTimesInput{ExistOk: true, UpdateTimes: false},
			Expect: updateTimesExpect{Updated: false},
		},
		{
			Name:   "UpdateTimes without ExistOk has no effect",
			Input:  updateTimesInput{ExistOk: false, UpdateTimes: true},
			Expect: updateTimesExpect{Err: existErr, Updated: false},
		},
	}

	runForResults(t, cases, func(t *testing.T, input updateTimesInput, expect updateTimesExpect) {
		t.Helper()

		root := setupTempDir(t)
		p := setup(t, root)
		require.NoError(t, os.Chtimes(p.String(), pastTime(), pastTime()))

		created, err := create(p, input)
		require.False(t, created)
		if expect.Err != nil {
			require.ErrorIs(t, err, expect.Err)
		} else {
			require.NoError(t, err)
		}

		info, err := os.Stat(p.String())
		require.NoError(t, err)
		require.Equal(t, expect.Updated, info.ModTime().After(pastTime()))
	})

	t.Run("Missing entry is created", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		created, err := create(root.JoinStrings("missing"), updateTimesInput{ExistOk: true, UpdateTimes: true})
		require.NoError(t, err)
		require.True(t, created)
	})
}

func TestCreateFileWithOptions_UpdateTimes(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, root *Path) *Path {
		t.Helper()

		return writeTempFile(t, root, "file.txt", "content")
	}
	create := func(p *Path, input updateTimesInput) (bool, error) {
		return CreateFileWithOptions(p, FileOptions{ExistOk: input.ExistOk, UpdateTimes: input.UpdateTimes})
	}

	runUpdateTimesCases(t, setup, create, ErrFileExist)
}

func TestMkDirWithOptions_UpdateTimes(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, root *Path) *Path {
		t.Helper()

		return createTempDir(t, root, "dir")
	}
	create := func(p *Path, input updateTimesInput) (bool, error) {
		return MkDirWithOptions(p, DirOptions{ExistOk: input.ExistOk, UpdateTimes: input.UpdateTimes})
	}

	runUpdateTimesCases(t, setup, create, ErrDirExist)
}

func TestMkDirWithOptions(t *testing.T) {
	t.Parallel()

	type Input struct {
		RelPath string
		Options DirOptions
	}
	type Expect struct {
		Created bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name:   "Successful creation (single level)",
			Input:  Input{"new_dir", DefaultDirOptions()},
			Expect: Expect{Created: true},
			Error:  false,
		},
		{
			Name:   "ExistOk=true, directory exists",
			Input:  Input{"existing_dir", DirOptions{ExistOk: true, CreateMode: DefaultDirMode()}},
			Expect: Expect{Created: false},
			Error:  false,
		},
		{
			Name:   "ExistOk=false, directory exists",
			Input:  Input{"existing_dir", DirOptions{ExistOk: false, CreateMode: DefaultDirMode()}},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "Path exists and is a file",
			Input:  Input{"existing_file.txt", DefaultDirOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "CreateAll=false, parent directory does not exist",
			Input:  Input{"non_existent_parent/new_dir", DefaultDirOptions()},
			Expect: Expect{Created: false},
			Error:  true,
		},
		{
			Name:   "CreateAll=true, parent directory does not exist",
			Input:  Input{"nested/deeply/created_dir", DirOptions{CreateAll: true, CreateMode: DefaultDirMode()}},
			Expect: Expect{Created: true},
			Error:  false,
		},
		{
			Name:   "CreateMode=0 should default to DefaultDirMode",
			Input:  Input{"dir_with_mode0", DirOptions{ExistOk: false, CreateMode: 0}},
			Expect: Expect{Created: true},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		targetPath := root.JoinStrings(input.RelPath)

		// Pre-create some paths for specific test cases
		if strings.Contains(input.RelPath, "existing_dir") {
			createTempDir(t, root, "existing_dir")
		}
		if strings.Contains(input.RelPath, "existing_file.txt") {
			writeTempFile(t, root, "existing_file.txt", "content")
		}

		created, err := MkDirWithOptions(targetPath, input.Options)

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)
			require.Equal(t, expect.Created, created)
			require.True(t, targetPath.Exists())
			require.True(t, targetPath.IsDir())

			info, err := targetPath.Stat()
			require.NoError(t, err)
			expectedMode := input.Options.CreateMode
			if expectedMode == 0 {
				expectedMode = DefaultDirMode()
			}
			require.Equal(t, effectiveDirMode(expectedMode).Perm(), info.Mode().Perm())
		}
	})
}

func TestPath_SymlinkTo(t *testing.T) {
	t.Parallel()

	type Input struct {
		SrcRel      string
		LinkPathRel string
		Setup       func(*testing.T, *Path) // Setup for src/link parents
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Successful symlink to a file",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "my_link.txt",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					writeTempFile(t, root, "target.txt", "content")
				},
			},
			Error: false,
		},
		{
			Name: "Successful symlink to a directory",
			Input: Input{
				SrcRel:      "target_dir",
				LinkPathRel: "my_link_dir",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					createTempDir(t, root, "target_dir")
				},
			},
			Error: false,
		},
		{
			Name: "Source path does not exist",
			Input: Input{
				SrcRel:      "non_existent_target.txt",
				LinkPathRel: "my_link.txt",
				Setup:       func(_ *testing.T, _ *Path) {},
			},
			Error: true,
		},
		{
			Name: "Link path already exists (file)",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "existing_link_file.txt",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					writeTempFile(t, root, "target.txt", "")
					writeTempFile(t, root, "existing_link_file.txt", "")
				},
			},
			Error: true,
		},
		{
			Name: "Link path already exists (directory)",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "existing_link_dir",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					writeTempFile(t, root, "target.txt", "")
					createTempDir(t, root, "existing_link_dir")
				},
			},
			Error: true,
		},
		{
			Name: "Link path parent directory does not exist",
			Input: Input{
				SrcRel:      "target.txt",
				LinkPathRel: "non_existent_dir/my_link.txt",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					writeTempFile(t, root, "target.txt", "")
				},
			},
			Error: true,
		},
		{
			Name: "Relative target, relative link (both in root)",
			Input: Input{
				SrcRel:      "dir/target.txt",
				LinkPathRel: "link.txt",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					writeTempFile(t, root, "dir/target.txt", "content")
				},
			},
			Error: false,
		},
		{
			Name: "Relative target, relative link in subdir",
			Input: Input{
				SrcRel:      "target.txt", // target is root/target.txt
				LinkPathRel: "subdir/link.txt",
				Setup: func(t *testing.T, root *Path) {
					t.Helper()

					writeTempFile(t, root, "target.txt", "content")
					createTempDir(t, root, "subdir")
				},
			},
			Error: false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, _ any, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		input.Setup(t, root)

		srcPath := root.JoinStrings(input.SrcRel)
		linkPath := root.JoinStrings(input.LinkPathRel)

		err := srcPath.SymlinkTo(linkPath)

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)

			require.True(t, linkPath.Exists(), "Link path should exist")
			require.True(t, linkPath.IsSymlink(), "Link path should be a symlink")

			// Verify the symlink target by reading it
			readTarget, err := os.Readlink(linkPath.String())
			require.NoError(t, err)

			// Compare read link target to lib function
			readTargetLibPath, err := linkPath.ReadSymlinkTarget()
			require.NoError(t, err)
			require.Equal(t, readTarget, readTargetLibPath.String())

			// The target passed to os.Symlink is relative from the link's
			// parent. Reconstruct the expected relative target to compare.
			expectedReadTarget := filepath.Join(filepath.Base(filepath.Dir(srcPath.String())), filepath.Base(srcPath.String()))
			// If target is directly in root, it should be just the base
			if filepath.Dir(srcPath.String()) == root.String() {
				expectedReadTarget = filepath.Base(srcPath.String())
			}

			readTargetPath := NewPath(readTarget)
			var readTargetPathAbsolute *Path
			if readTargetPath.IsAbsolute() {
				readTargetPathAbsolute = readTargetPath.Copy()

				readTargetPath, err = readTargetPath.RelativeTo(root)
				require.NoError(t, err)
			}
			readTarget = readTargetPath.String()

			// Test for equal relative paths
			require.Equal(t, expectedReadTarget, readTarget)

			// Test equal absolute paths
			require.Equal(t, srcPath.String(), readTargetPathAbsolute.String(), "Symlink target read should match absolute source path")
			require.True(t, srcPath.EqualsFS(readTargetPathAbsolute), "Symlink target read should point to same file")
		}
	})
}

func TestCreateFile(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)

	filePath := root.JoinStrings("new_file.txt")
	err := CreateFile(filePath)
	require.NoError(t, err)
	require.True(t, filePath.Exists())
	require.True(t, filePath.IsFile())

	info, err := filePath.Stat()
	require.NoError(t, err)
	require.Equal(t, DefaultFileMode().Perm(), info.Mode().Perm())

	// An existing file is neither truncated nor overwritten.
	existingPath := writeTempFile(t, root, "existing.txt", "content")
	err = CreateFile(existingPath)
	require.ErrorIs(t, err, ErrFileExist)

	content, err := os.ReadFile(existingPath.String())
	require.NoError(t, err)
	require.Equal(t, "content", string(content))
}

func TestMkDir(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)

	dirPath := root.JoinStrings("new_dir")
	err := MkDir(dirPath)
	require.NoError(t, err)
	require.True(t, dirPath.Exists())
	require.True(t, dirPath.IsDir())

	info, err := dirPath.Stat()
	require.NoError(t, err)
	require.Equal(t, DefaultDirMode().Perm(), info.Mode().Perm())
}

func TestCreateSymlink(t *testing.T) {
	t.Parallel()

	type Input struct {
		Setup func(*testing.T, *Path) (target *Path, linkPath *Path)
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Successful symlink creation",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				t.Helper()

				target := writeTempFile(t, root, "target.txt", "content")
				return target, root.JoinStrings("link.txt")
			}},
			Error: false,
		},
		{
			Name: "Link path already exists",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				t.Helper()

				target := writeTempFile(t, root, "target.txt", "content")
				existing := writeTempFile(t, root, "existing.txt", "")
				return target, existing
			}},
			Error: true,
		},
		{
			Name: "Link parent directory does not exist",
			Input: Input{Setup: func(t *testing.T, root *Path) (*Path, *Path) {
				t.Helper()

				target := writeTempFile(t, root, "target.txt", "content")
				return target, root.JoinStrings("nonexistent_dir", "link.txt")
			}},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, _ any, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		target, linkPath := input.Setup(t, root)
		err := CreateSymlink(target, linkPath)
		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			return
		}
		require.NoError(t, err)
		require.True(t, linkPath.IsSymlink())
	})
}

func TestCreate_BrokenSymlinks(t *testing.T) {
	t.Parallel()

	t.Run("CreateFileWithOptions does not create the link target", func(t *testing.T) {
		t.Parallel()

		for _, existOk := range []bool{false, true} {
			root := setupTempDir(t)
			link := createTempSymlinkAbs(t, root, "target.txt", "link")

			created, err := CreateFileWithOptions(link, FileOptions{ExistOk: existOk})
			require.ErrorIs(t, err, ErrNotFile, "ExistOk=%v", existOk)
			require.False(t, created)
			requireLExists(t, false, root.JoinStrings("target.txt"), "ExistOk=%v", existOk)
		}
	})

	t.Run("MkDirWithOptions does not create the link target", func(t *testing.T) {
		t.Parallel()

		for _, createAll := range []bool{false, true} {
			root := setupTempDir(t)
			link := createTempSymlinkAbs(t, root, "target_dir", "link")

			created, err := MkDirWithOptions(link, DirOptions{CreateAll: createAll})
			require.ErrorIs(t, err, ErrNotDir, "CreateAll=%v", createAll)
			require.False(t, created)
			requireLExists(t, false, root.JoinStrings("target_dir"), "CreateAll=%v", createAll)
		}
	})

	t.Run("CreateSymlink over a broken symlink", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		link := createTempSymlinkAbs(t, root, "missing", "link")

		err := CreateSymlink(root, link)
		var kind *PathlibError
		require.ErrorAs(t, err, &kind)
		require.Equal(t, ErrExist, kind, "raised by the existence check, not by os.Symlink")
	})
}

func TestCreateFileWithOptions_SymlinkToFile(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	target := writeTempFile(t, root, "target.txt", "content")
	link := createTempSymlinkAbs(t, root, "target.txt", "link")

	// A symlink to a file counts as that file.
	created, err := CreateFileWithOptions(link, FileOptions{ExistOk: true})
	require.NoError(t, err)
	require.False(t, created)

	_, err = CreateFileWithOptions(link, FileOptions{})
	require.ErrorIs(t, err, ErrFileExist)

	content, err := os.ReadFile(target.String())
	require.NoError(t, err)
	require.Equal(t, "content", string(content), "the target is not truncated")
}
