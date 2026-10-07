package pathlib

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:maintidx // The table of cases is long by design.
func TestCopy(t *testing.T) {
	t.Parallel()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

					writeTempFile(t, root, "src/file.txt", "")
				},
			},
			Expect: Expect{},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		input.Setup(t, root)

		srcPath := root.JoinStrings(input.SrcRel)
		dstPath := root.JoinStrings(input.DstRel)

		err := Copy(srcPath, dstPath)

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)

			// Verify the state of the *entire* root to capture all changes
			// correctly.
			finalRootState := readDirEntries(t, root, root)
			// The cases share their slices, so a sorted clone is compared.
			expect.State = slices.Clone(expect.State)
			slices.Sort(expect.State)
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
				require.True(t, originalTarget.EqualsFS(copiedTarget), "Symlinks do not point to same file on fs")
			}
		}
	})
}

func TestCopy_RelativeSymlinkTarget(t *testing.T) {
	t.Parallel()

	type Input struct {
		SrcRel      string // the copied path, relative to the root
		DstRel      string // the destination, relative to the root
		LinkRel     string // the copied symlink, relative to the root
		AbsoluteSrc bool   // pass the source as an absolute path
		AbsoluteDst bool   // pass the destination as an absolute path
	}

	cases := []TestCase[Input, string]{
		{
			Name:   "Relative source and destination",
			Input:  Input{SrcRel: "src/link", DstRel: "dst/link", LinkRel: "dst/link"},
			Expect: "../target.txt",
		},
		{
			Name:   "Relative source and absolute destination",
			Input:  Input{SrcRel: "src/link", DstRel: "dst/link", LinkRel: "dst/link", AbsoluteDst: true},
			Expect: "../target.txt",
		},
		{
			Name:   "Absolute source and relative destination",
			Input:  Input{SrcRel: "src/link", DstRel: "dst/link", LinkRel: "dst/link", AbsoluteSrc: true},
			Expect: "../target.txt",
		},
		{
			Name:   "Destination at another depth",
			Input:  Input{SrcRel: "src/link", DstRel: "dst/deep/link", LinkRel: "dst/deep/link"},
			Expect: "../../target.txt",
		},
		{
			Name:   "Symlink inside a copied directory",
			Input:  Input{SrcRel: "src", DstRel: "dst/copy", LinkRel: "dst/copy/link"},
			Expect: "../../target.txt",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			root := setupRelativeTempDir(t)
			writeTempFile(t, root, "target.txt", "content")
			createTempSymlinkRel(t, root, "../target.txt", "src/link")
			createTempDir(t, root, "dst/deep")

			absoluteRoot, err := root.MakeAbsolute()
			require.NoError(t, err)

			srcRoot := root
			if c.Input.AbsoluteSrc {
				srcRoot = absoluteRoot
			}
			dstRoot := root
			if c.Input.AbsoluteDst {
				dstRoot = absoluteRoot
			}

			err = Copy(srcRoot.JoinStrings(c.Input.SrcRel), dstRoot.JoinStrings(c.Input.DstRel))
			require.NoError(t, err)

			link := root.JoinStrings(c.Input.LinkRel)
			target, err := link.ReadSymlinkTarget()
			require.NoError(t, err)
			require.Equal(t, c.Expect, target.ToPosix())

			content, err := ReadFileToString(link)
			require.NoError(t, err)
			require.Equal(t, "content", content, "the copied symlink resolves to the original target")
		})
	}
}

func TestMove(t *testing.T) {
	t.Parallel()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

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
		t.Helper()

		root := setupTempDir(t)
		input.Setup(t, root) // Setup initial state

		srcPath := root.JoinStrings(input.SrcRel)
		dstPath := root.JoinStrings(input.DstRel)

		err := Move(srcPath, dstPath)

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)
		}

		finalRootState := readDirEntries(t, root, root)
		// The cases share their slices, so a sorted clone is compared.
		expect.FinalState = slices.Clone(expect.FinalState)
		slices.Sort(expect.FinalState)
		require.Equal(t, expect.FinalState, finalRootState)
	})
}

func TestRename(t *testing.T) {
	t.Parallel()

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
					t.Helper()

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
					t.Helper()

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
					t.Helper()

					writeTempFile(t, root, "existing.txt", "existing")
					return writeTempFile(t, root, "original.txt", "original")
				},
			},
			Expect: Expect{FinalEntries: []string{"existing.txt", "original.txt"}},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.Setup(t, root)

		err := Rename(p, input.NewName)
		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)
		}

		finalState := readDirEntries(t, root, root)
		// The cases share their slices, so a sorted clone is compared.
		expect.FinalEntries = slices.Clone(expect.FinalEntries)
		slices.Sort(expect.FinalEntries)
		require.Equal(t, expect.FinalEntries, finalState)
	})
}

func TestCopy_BrokenSymlinks(t *testing.T) {
	t.Parallel()

	t.Run("broken symlink is copied as symlink", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		src := createTempSymlinkRel(t, root, "missing", "link")
		dst := root.JoinStrings("copy")

		require.NoError(t, Copy(src, dst))
		require.True(t, dst.IsSymlink())
		target, err := dst.ReadSymlinkTarget()
		require.NoError(t, err)
		require.True(t, target.EqualsString("missing"))
	})

	t.Run("directory containing a broken symlink is copied", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		src := createTempDir(t, root, "src")
		writeTempFile(t, src, "file.txt", "")
		createTempSymlinkRel(t, root, "missing", "src/link")
		dst := root.JoinStrings("dst")

		require.NoError(t, Copy(src, dst))
		require.True(t, dst.JoinStrings("file.txt").IsFile())
		require.True(t, dst.JoinStrings("link").IsSymlink())
	})

	t.Run("file is not copied through a broken symlink", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		src := writeTempFile(t, root, "src.txt", "content")
		link := createTempSymlinkAbs(t, root, "target.txt", "link")

		err := Copy(src, link)
		require.ErrorIs(t, err, ErrExist)
		require.True(t, link.IsSymlink(), "the link is left in place")
		requireLExists(t, false, root.JoinStrings("target.txt"), "the link target is not created")
	})

	t.Run("directory is not copied onto a broken symlink", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		src := createTempDir(t, root, "src")
		link := createTempSymlinkAbs(t, root, "target_dir", "link")

		err := Copy(src, link)
		require.ErrorIs(t, err, ErrExist)
		require.True(t, link.IsSymlink(), "the link is left in place")
		requireLExists(t, false, root.JoinStrings("target_dir"), "the link target is not created")
	})
}

func TestMove_BrokenSymlinks(t *testing.T) {
	t.Parallel()

	t.Run("broken symlink is moved", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		src := createTempSymlinkRel(t, root, "missing", "link")
		dst := root.JoinStrings("moved")

		require.NoError(t, Move(src, dst))
		requireLExists(t, false, src)
		require.True(t, dst.IsSymlink())
	})

	t.Run("file is not moved onto a broken symlink", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		src := writeTempFile(t, root, "src.txt", "content")
		link := createTempSymlinkAbs(t, root, "missing", "link")

		require.ErrorIs(t, Move(src, link), ErrExist)
		require.True(t, src.IsFile(), "the source is left in place")
		require.True(t, link.IsSymlink(), "the link is not replaced")
	})
}
