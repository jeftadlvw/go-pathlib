package pathlib

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	type Input struct {
		SrcRel string
		DstRel string
		Setup  func(*testing.T, *Path) // Setup for src/dst parents
	}

	type Expect struct {
		State []string // Expected final state of the overall root directory (relative paths to root)
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Copy regular file",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/new_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "hello world")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_file.txt", "src", "src/file.txt"},
			},
			Error: false,
		},
		{
			Name: "Copy file to existing file (should error)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/existing_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "src content")
					writeTempFile(t, root, "dst/existing_file.txt", "dst content")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Copy file to existing directory (should error)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst_dir",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "content")
					createTempDir(t, root, "dst_dir")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Copy empty directory",
			Input: Input{
				SrcRel: "src_empty_dir",
				DstRel: "dst/new_empty_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "src_empty_dir")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_empty_dir", "src_empty_dir"},
			},
			Error: false,
		},
		{
			Name: "Copy directory with files",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/new_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file1.txt", "content1")
					writeTempFile(t, srcDir, "file2.log", "content2")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_dir", "dst/new_dir/file1.txt", "dst/new_dir/file2.log", "src_dir", "src_dir/file1.txt", "src_dir/file2.log"},
			},
			Error: false,
		},
		{
			Name: "Copy directory with nested structure",
			Input: Input{
				SrcRel: "src_parent",
				DstRel: "dst/copied_parent",
				Setup: func(t *testing.T, root *Path) {
					srcParent := createTempDir(t, root, "src_parent")
					writeTempFile(t, srcParent, "root_file.txt", "")
					subdir1 := createTempDir(t, srcParent, "subdir1")
					writeTempFile(t, subdir1, "nested_file.md", "")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "deep_file.json", "")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{
					"dst",
					"dst/copied_parent",
					"dst/copied_parent/root_file.txt",
					"dst/copied_parent/subdir1",
					"dst/copied_parent/subdir1/nested_file.md",
					"dst/copied_parent/subdir1/subdir2",
					"dst/copied_parent/subdir1/subdir2/deep_file.json",
					"src_parent",
					"src_parent/root_file.txt",
					"src_parent/subdir1",
					"src_parent/subdir1/nested_file.md",
					"src_parent/subdir1/subdir2",
					"src_parent/subdir1/subdir2/deep_file.json",
				},
			},
			Error: false,
		},
		{
			Name: "Copy directory to existing non-empty directory (should error)",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file.txt", "")
					dstDir := createTempDir(t, root, "dst_dir")
					writeTempFile(t, dstDir, "existing_file.txt", "")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Copy absolute symlink to file",
			Input: Input{
				SrcRel: "link_to_file",
				DstRel: "dst/new_link",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "symlink target content")
					createTempSymlinkAbs(t, root, "target.txt", "link_to_file")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_link", "link_to_file", "target.txt"},
			},
			Error: false,
		},
		{
			Name: "Copy relative symlink to file",
			Input: Input{
				SrcRel: "link_to_file",
				DstRel: "dst/new_link",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "target.txt", "symlink target content")
					createTempSymlinkRel(t, root, "target.txt", "link_to_file")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_link", "link_to_file", "target.txt"},
			},
			Error: false,
		},
		{
			Name: "Copy symlink to directory",
			Input: Input{
				SrcRel: "link_to_dir",
				DstRel: "dst/new_link_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "target_dir")
					createTempSymlinkAbs(t, root, "target_dir", "link_to_dir")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				State: []string{"dst", "dst/new_link_dir", "link_to_dir", "target_dir"},
			},
			Error: false,
		},
		{
			Name: "Source path does not exist",
			Input: Input{
				SrcRel: "non_existent_src",
				DstRel: "dst/target",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Destination parent does not exist",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "non_existent_parent/target.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		input.Setup(t, root)

		srcPath := root.JoinStrings(input.SrcRel)
		dstPath := root.JoinStrings(input.DstRel)

		err := Copy(srcPath, dstPath)

		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)

			// Verify the state of the *entire* root to capture all changes correctly.
			finalRootState := readDirEntries(t, root, root)
			slices.Sort(expect.State) // Ensure expectation is sorted
			require.Equal(t, expect.State, finalRootState)

			// Additional checks for symlinks
			if srcPath.IsSymlink() {
				require.True(t, dstPath.IsSymlink(), "Destination should be a symlink")

				// Test both targets target same file
				originalTarget, err := srcPath.ReadSymlinkTarget()
				require.NoError(t, err)
				originalTarget, err = originalTarget.AbsoluteFrom(srcPath.Parent())
				require.NoError(t, err)

				copiedTarget, err := dstPath.ReadSymlinkTarget()
				require.NoError(t, err)
				copiedTarget, err = copiedTarget.AbsoluteFrom(dstPath.Parent())
				require.NoError(t, err)

				require.Equal(t, originalTarget.ToPosix(), copiedTarget.ToPosix(), "Copied symlink target should match original")
				require.True(t, originalTarget.EqualsFs(copiedTarget), "Symlinks do not point to same file on fs")
			}
		}
	})
}

func TestMove(t *testing.T) {
	type Input struct {
		SrcRel string
		DstRel string
		Setup  func(*testing.T, *Path) // Setup for src/dst parents
	}
	type Expect struct {
		FinalState []string // Expected final state of the overall root directory (relative paths to root)
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Move file to new location",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/moved_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "content")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				FinalState: []string{"dst", "dst/moved_file.txt", "src"}, // src dir remains, but is empty
			},
			Error: false,
		},
		{
			Name: "Move file overwriting existing file (should fail)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst/existing_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "new content")
					writeTempFile(t, root, "dst/existing_file.txt", "old content")
				},
			},
			Expect: Expect{FinalState: []string{"dst", "dst/existing_file.txt", "src", "src/file.txt"}},
			Error:  true,
		},
		{
			Name: "Move empty directory to new location",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/moved_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "src_dir")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				FinalState: []string{"dst", "dst/moved_dir"},
			},
			Error: false,
		},
		{
			Name: "Move directory with content to new location",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/moved_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file.txt", "")
					createTempDir(t, srcDir, "subdir")
					writeTempFile(t, srcDir.JoinStrings("subdir"), "nested.log", "")
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{
				FinalState: []string{
					"dst", "dst/moved_dir", "dst/moved_dir/file.txt",
					"dst/moved_dir/subdir", "dst/moved_dir/subdir/nested.log",
				},
			},
			Error: false,
		},
		{
			Name: "Move directory to existing empty directory",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst/existing_empty_dir",
				Setup: func(t *testing.T, root *Path) {
					srcDir := createTempDir(t, root, "src_dir")
					writeTempFile(t, srcDir, "file.txt", "")
					createTempDir(t, root, "dst/existing_empty_dir")
				},
			},
			Expect: Expect{
				FinalState: []string{"dst", "dst/existing_empty_dir", "dst/existing_empty_dir/file.txt"},
			},
			Error: false,
		},
		{
			Name: "Source path does not exist",
			Input: Input{
				SrcRel: "non_existent_src",
				DstRel: "dst/target",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "dst")
				},
			},
			Expect: Expect{FinalState: []string{"dst"}},
			Error:  true,
		},
		{
			Name: "Destination parent does not exist",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "non_existent_parent/target.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "")
				},
			},
			Expect: Expect{FinalState: []string{"src", "src/file.txt"}},
			Error:  true,
		},
		{
			Name: "Move file to existing non-empty directory (should fail)",
			Input: Input{
				SrcRel: "src/file.txt",
				DstRel: "dst_dir", // This *is* a directory
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "src/file.txt", "content")
					dstDir := createTempDir(t, root, "dst_dir")
					writeTempFile(t, dstDir, "other_file.txt", "") // Make it non-empty
				},
			},
			Expect: Expect{
				FinalState: []string{"dst_dir", "dst_dir/other_file.txt", "src", "src/file.txt"}, // Source should still exist
			},
			Error: true,
		},
		{
			Name: "Move directory to existing non-empty directory (should fail)",
			Input: Input{
				SrcRel: "src_dir",
				DstRel: "dst_dir", // This *is* a directory
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "src_dir")
					writeTempFile(t, root.JoinStrings("src_dir"), "file.txt", "")
					dstDir := createTempDir(t, root, "dst_dir")
					writeTempFile(t, dstDir, "other_file.txt", "") // Make it non-empty
				},
			},
			Expect: Expect{
				FinalState: []string{"dst_dir", "dst_dir/other_file.txt", "src_dir", "src_dir/file.txt"}, // Source should still exist
			},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		input.Setup(t, root) // Setup initial state

		srcPath := root.JoinStrings(input.SrcRel)
		dstPath := root.JoinStrings(input.DstRel)

		err := Move(srcPath, dstPath)

		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		finalRootState := readDirEntries(t, root, root)
		slices.Sort(expect.FinalState)
		require.Equal(t, expect.FinalState, finalRootState)
	})
}

func TestRemoveAll(t *testing.T) {
	type Input struct {
		RelPath string
		Setup   func(*testing.T, *Path) // Setup for the path to remove
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Remove empty directory",
			Input: Input{
				RelPath: "empty_dir",
				Setup: func(t *testing.T, root *Path) {
					createTempDir(t, root, "empty_dir")
				},
			},
			Error: false,
		},
		{
			Name: "Remove directory with files",
			Input: Input{
				RelPath: "dir_with_files",
				Setup: func(t *testing.T, root *Path) {
					dir := createTempDir(t, root, "dir_with_files")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
				},
			},
			Error: false,
		},
		{
			Name: "Remove directory with nested structure",
			Input: Input{
				RelPath: "nested_dir",
				Setup: func(t *testing.T, root *Path) {
					dir := createTempDir(t, root, "nested_dir")
					writeTempFile(t, dir, "file.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested.txt", "")
					createTempDir(t, subdir1, "subdir2")
				},
			},
			Error: false,
		},
		{
			Name: "Remove non-existent path (no-op)",
			Input: Input{
				RelPath: "non_existent_path",
				Setup:   func(t *testing.T, root *Path) { /* no setup */ },
			},
			Error: false,
		},
		{
			Name: "Remove a file (should error)",
			Input: Input{
				RelPath: "a_file.txt",
				Setup: func(t *testing.T, root *Path) {
					writeTempFile(t, root, "a_file.txt", "")
				},
			},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		targetPath := root.JoinStrings(input.RelPath)
		input.Setup(t, root) // Set up the path to be removed

		err := RemoveAll(targetPath)

		if expectError {
			require.Error(t, err)
			// For RemoveAll, if it errors because it's not a directory, the path *should* still exist.
			if errors.Is(err, ErrNotDir) {
				require.True(t, targetPath.Exists())
			}
		} else {
			require.NoError(t, err)
			require.False(t, targetPath.Exists(), "Path should not exist after RemoveAll")
		}
	})
}

func TestRemove(t *testing.T) {
	type Input struct {
		Setup func(*testing.T, *Path) *Path
	}

	cases := []TestCase[Input, any]{
		{
			Name: "Remove existing file",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return writeTempFile(t, root, "file.txt", "content")
			}},
			Error: false,
		},
		{
			Name: "Remove empty directory",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return createTempDir(t, root, "emptyDir")
			}},
			Error: false,
		},
		{
			Name: "Remove non-existent path (no-op)",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				return root.JoinStrings("nonexistent")
			}},
			Error: false,
		},
		{
			Name: "Remove non-empty directory (should error)",
			Input: Input{Setup: func(t *testing.T, root *Path) *Path {
				dir := createTempDir(t, root, "nonEmptyDir")
				writeTempFile(t, dir, "file.txt", "")
				return dir
			}},
			Error: true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect any, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		err := Remove(p)
		if expectError {
			require.Error(t, err)
			require.True(t, p.Exists())
			return
		}
		require.NoError(t, err)
		require.False(t, p.Exists())
	})
}

func TestRename(t *testing.T) {
	type Input struct {
		NewName string
		Setup   func(*testing.T, *Path) *Path
	}

	type Expect struct {
		FinalEntries []string // relative to root
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Rename file",
			Input: Input{
				NewName: "renamed.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					return writeTempFile(t, root, "original.txt", "content")
				},
			},
			Expect: Expect{FinalEntries: []string{"renamed.txt"}},
			Error:  false,
		},
		{
			Name: "Rename directory",
			Input: Input{
				NewName: "renamed_dir",
				Setup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "original_dir")
					writeTempFile(t, dir, "child.txt", "")
					return dir
				},
			},
			Expect: Expect{FinalEntries: []string{"renamed_dir", "renamed_dir/child.txt"}},
			Error:  false,
		},
		{
			Name: "Rename to existing name (should error)",
			Input: Input{
				NewName: "existing.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					writeTempFile(t, root, "existing.txt", "existing")
					return writeTempFile(t, root, "original.txt", "original")
				},
			},
			Expect: Expect{FinalEntries: []string{"existing.txt", "original.txt"}},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)

		err := Rename(p, input.NewName)
		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		finalState := readDirEntries(t, root, root)
		slices.Sort(expect.FinalEntries)
		require.Equal(t, expect.FinalEntries, finalState)
	})
}
