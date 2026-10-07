package pathlib

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_Walk(t *testing.T) {
	t.Parallel()

	type Input struct {
		RootSetup func(*testing.T, *Path) *Path // Function to set up the directory for walking
		WalkFunc  func(p *Path) error
	}

	type Expect struct {
		WalkedPaths []string // Relative paths from the rootSetup
		Error       bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Empty directory",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()
					return createTempDir(t, root, "emptyDir")
				},
				WalkFunc: func(_ *Path) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{}},
		},
		{
			Name: "Symlinked root directory is followed",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					writeTempFile(t, root, "realDir/file.txt", "")
					return createTempSymlinkAbs(t, root, "realDir", "linkDir")
				},
				WalkFunc: func(_ *Path) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file.txt"}},
		},
		{
			Name: "Directory with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					return dir
				},
				WalkFunc: func(_ *Path) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "file2.log"}},
		},
		{
			Name: "Directory with subdirectories (should not recurse)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file.txt", "")
					subdir := createTempDir(t, dir, "subdir")
					writeTempFile(t, subdir, "nested.txt", "") // Should not be walked
					return dir
				},
				WalkFunc: func(_ *Path) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file.txt", "subdir"}}, // subdir itself is walked, but not its contents
		},
		{
			Name: "Non-directory path",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()
					return writeTempFile(t, root, "file.txt", "")
				},
				WalkFunc: func(_ *Path) error { return nil },
			},
			Expect: Expect{Error: true},
		},
		{
			Name: "WalkFunc returns error",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.log", "")
					return dir
				},
				WalkFunc: func(_ *Path) error {
					return errSimulated
				},
			},
			Expect: Expect{WalkedPaths: []string{}, Error: true},
		},
		{
			Name: "WalkFunc returns SkipAll",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					writeTempFile(t, dir, "file3.csv", "")
					return dir
				},
				WalkFunc: func() func(*Path) error {
					first := true
					return func(_ *Path) error {
						if first {
							first = false
							return SkipAll
						}
						return nil
					}
				}(),
			},
			Expect: Expect{WalkedPaths: []string{}}, // First entry signals SkipAll, walk stops immediately
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, _ bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.RootSetup(t, root)

		var walkedEntries []string
		customWalkFunc := func(path *Path) error {
			// A non-nil result (SkipAll or a real error) stops the walk,
			// the entry that triggered it is not recorded.
			walkErr := input.WalkFunc(path)
			if walkErr != nil {
				return walkErr
			}

			relPath, err := path.RelativeTo(p)
			require.NoError(t, err)
			walkedEntries = append(walkedEntries, relPath.ToPosix())

			return nil
		}

		err := p.Walk(customWalkFunc)

		if expect.Error {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)
		}

		// Ensure consistent order for comparison
		slices.Sort(walkedEntries)
		// The cases share their slices, so a sorted clone is compared.
		expect.WalkedPaths = slices.Clone(expect.WalkedPaths)
		slices.Sort(expect.WalkedPaths)

		require.Len(t, walkedEntries, len(expect.WalkedPaths))

		if len(expect.WalkedPaths) > 0 {
			require.Equal(t, expect.WalkedPaths, walkedEntries)
		}
	})
}

//nolint:maintidx // The table of cases is long by design.
func TestPath_WalkR(t *testing.T) {
	t.Parallel()

	type Input struct {
		RootSetup func(*testing.T, *Path) *Path // Function to set up the directory for walking
		WalkRFunc func(p *Path, localDirError error) error
	}
	type Expect struct {
		WalkedPaths []string // Relative paths from the rootSetup
		Error       bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Empty directory",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()
					return createTempDir(t, root, "emptyDir")
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{}},
		},
		{
			Name: "Symlinked root directory is followed",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					writeTempFile(t, root, "realDir/sub/file.txt", "")
					return createTempSymlinkAbs(t, root, "realDir", "linkDir")
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"sub", "sub/file.txt"}},
		},
		{
			Name: "Symlinked subdirectory is passed as entry but not entered",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					writeTempFile(t, root, "outside/hidden.txt", "")
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file.txt", "")
					createTempSymlinkAbs(t, root, "outside", "testDir/linkDir")
					return dir
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file.txt", "linkDir"}},
		},
		{
			Name: "Symlink cycle terminates",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					createTempSymlinkAbs(t, root, "testDir", "testDir/self")
					return dir
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"self"}},
		},
		{
			Name: "Broken symlink is passed as entry",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					createTempSymlinkAbs(t, root, "nonexistent", "testDir/brokenLink")
					return dir
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"brokenLink"}},
		},
		{
			Name: "Flat directory with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					return dir
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "file2.log"}},
		},
		{
			Name: "Nested directories with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested1.txt", "")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "nested2.log", "")
					return dir
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{
				"file.txt",
				"subdir1",
				"subdir1/nested1.txt",
				"subdir1/subdir2",
				"subdir1/subdir2/nested2.log",
			}},
		},
		{
			Name: "Non-directory initial path",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()
					return writeTempFile(t, root, "file.txt", "")
				},
				WalkRFunc: func(_ *Path, _ error) error { return nil },
			},
			Expect: Expect{Error: true},
		},
		{
			Name: "WalkRFunc returns error (aborts entire tree)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					subdir := createTempDir(t, dir, "subdir")
					writeTempFile(t, subdir, "file2.log", "")
					return dir
				},
				WalkRFunc: func(p *Path, _ error) error {
					if p.Base() == "file2.log" {
						return errSimulated
					}
					return nil
				},
			},
			Expect: Expect{WalkedPaths: []string{"subdir"}, Error: true},
		},
		{
			Name: "WalkRFunc returns SkipDir",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested1.txt", "") // Should be skipped
					createTempDir(t, subdir1, "subdir2")         // Should be skipped
					writeTempFile(t, dir, "file3.csv", "")       // Should still be walked
					return dir
				},
				WalkRFunc: func(p *Path, _ error) error {
					if p.Base() == "subdir1" {
						return SkipDir // Skip processing subdir1's contents
					}
					return nil
				},
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "subdir1", "file3.csv"}}, // subdir1 is recorded, but its contents are not recursed
		},
		{
			Name: "WalkRFunc returns SkipAll",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "testDir")
					subdir1 := createTempDir(t, dir, "subdir1")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "deep.txt", "") // Should NOT be walked
					return dir
				},
				WalkRFunc: func(p *Path, _ error) error {
					if p.Base() == "subdir2" {
						return SkipAll // Stop the entire walk
					}
					return nil
				},
			},
			Expect: Expect{WalkedPaths: []string{"subdir1", "subdir1/subdir2"}},
		},
		{
			Name: "Error reading directory, walkFunc handles (continues)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "root")
					createTempDir(t, dir, "readable_dir")
					writeTempFile(t, dir, "readable_dir/file.txt", "")

					unreadableDir := createTempDir(t, dir, "unreadable_dir")
					writeTempFile(t, unreadableDir, "secret.log", "")
					lockDir(t, unreadableDir)

					createTempDir(t, dir, "another_dir")
					writeTempFile(t, dir, "another_dir/another_file.txt", "")
					return dir
				},
				// Directory errors are ignored, so the walk continues.
				WalkRFunc: func(_ *Path, _ error) error {
					return nil
				},
			},
			Expect: Expect{
				// unreadable_dir is passed twice: before it is read, and with
				// the directory error after reading it failed.
				WalkedPaths: []string{
					"another_dir",
					"another_dir/another_file.txt",
					"readable_dir",
					"readable_dir/file.txt",
					"unreadable_dir",
					"unreadable_dir",
				},
			},
		},
		{
			Name: "Error reading directory, walkFunc returns error (stops)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					// Entries are walked in lexical order, so naming the
					// siblings a/b/c makes the traversal deterministic:
					// a_readable is walked, b_unreadable raises a directory
					// error that aborts the walk, and c_readable (lexically
					// after the error) is never reached. This demonstrates that
					// returning an error stops the walk.
					dir := createTempDir(t, root, "root")
					createTempDir(t, dir, "a_readable")
					unreadableDir := createTempDir(t, dir, "b_unreadable")
					lockDir(t, unreadableDir)
					createTempDir(t, dir, "c_readable")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error) error {
					if localDirError != nil {
						return fmt.Errorf("custom error in %s: %w", p.ToPosix(), localDirError)
					}
					return nil
				},
			},
			Expect: Expect{
				// b_unreadable is recorded once, before it is read. Reading it
				// raises a directory error that aborts the walk (so the second
				// call is not recorded), and c_readable is never reached.
				WalkedPaths: []string{
					"a_readable",
					"b_unreadable",
				},
				Error: true,
			},
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.RootSetup(t, root)

		var walkedEntries []string
		customWalkRFunc := recordingWalkRFunc(p, input.WalkRFunc, expectError, &walkedEntries)

		err := p.WalkR(customWalkRFunc)

		if expect.Error {
			require.ErrorIs(t, err, ErrPathlib)
		} else {
			require.NoError(t, err)
		}

		require.ElementsMatch(t, expect.WalkedPaths, walkedEntries)
	})
}

// recordingWalkRFunc returns a WalkRFunc that calls walkRFunc, if set, and
// records the paths relative to root that it visits in walked. An entry that
// signals SkipDir or SkipAll was still visited and is recorded, while an entry
// with another error is not. A failure to build a relative path is returned
// unless ignoreRelErr is set.
func recordingWalkRFunc(root *Path, walkRFunc WalkRFunc, ignoreRelErr bool, walked *[]string) WalkRFunc {
	return func(path *Path, localDirError error) error {
		relPath, err := path.RelativeTo(root)
		if err != nil && !ignoreRelErr {
			return err
		}

		var inputWalkFuncErr error
		if walkRFunc != nil {
			inputWalkFuncErr = walkRFunc(path, localDirError)
		}

		if inputWalkFuncErr == nil || isSkip(inputWalkFuncErr) {
			*walked = append(*walked, relPath.ToPosix())
		}

		return inputWalkFuncErr
	}
}

// isSkip reports whether err is SkipDir or SkipAll.
func isSkip(err error) bool {
	return errors.Is(err, SkipDir) || errors.Is(err, SkipAll)
}

func TestPath_WalkContext_Cancellation(t *testing.T) {
	t.Parallel()

	buildTree := func(t *testing.T) *Path {
		t.Helper()

		root := setupTempDir(t)
		dir := createTempDir(t, root, "tree")
		writeTempFile(t, dir, "a.txt", "")
		writeTempFile(t, dir, "b.txt", "")
		sub := createTempDir(t, dir, "sub")
		writeTempFile(t, sub, "c.txt", "")
		return dir
	}

	t.Run("WalkContext with canceled context returns context.Canceled", func(t *testing.T) {
		t.Parallel()

		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var visited int
		err := dir.WalkContext(ctx, func(_ *Path) error {
			visited++
			return nil
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, visited, "no entries should be visited after cancellation")
	})

	t.Run("WalkRContext with canceled context returns context.Canceled", func(t *testing.T) {
		t.Parallel()

		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := dir.WalkRContext(ctx, func(_ *Path, _ error) error {
			return nil
		})

		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("WalkRContext canceled mid-walk stops early", func(t *testing.T) {
		t.Parallel()

		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())

		var visited int
		err := dir.WalkRContext(ctx, func(_ *Path, _ error) error {
			visited++
			cancel() // cancel after the first visited entry
			return nil
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, visited, "walk should stop right after cancellation")
	})

	t.Run("GlobContext with canceled context returns context.Canceled", func(t *testing.T) {
		t.Parallel()

		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		matches, err := dir.GlobContext(ctx, "*.txt")
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, matches)
	})

	t.Run("context variants with Background behave like the plain calls", func(t *testing.T) {
		t.Parallel()

		dir := buildTree(t)

		plain, err := dir.Glob("*.txt")
		require.NoError(t, err)

		withCtx, err := dir.GlobContext(context.Background(), "*.txt")
		require.NoError(t, err)

		require.ElementsMatch(t, plain, withCtx)
	})
}

func TestWalk_DirAccessErrorGrouping(t *testing.T) {
	t.Parallel()

	t.Run("ErrOpen and ErrReadDir are members of the ErrOperation group", func(t *testing.T) {
		t.Parallel()

		// Both are catchable through the shared parent group.
		require.ErrorIs(t, ErrOpen, ErrOperation)
		require.ErrorIs(t, ErrReadDir, ErrOperation)

		// The two remain distinguishable from each other.
		require.NotErrorIs(t, ErrOpen, ErrReadDir)
		require.NotErrorIs(t, ErrReadDir, ErrOpen)
	})

	t.Run("localDirError of a denied directory is ErrPermissionDenied", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		dir := createTempDir(t, root, "root")
		unreadableDir := createTempDir(t, dir, "unreadable")
		lockDir(t, unreadableDir)

		var captured error
		err := dir.WalkR(func(_ *Path, localDirError error) error {
			if localDirError != nil {
				captured = localDirError
			}
			return nil // ignore so the walk completes
		})
		require.NoError(t, err)

		require.ErrorIs(t, captured, ErrPermissionDenied)
		require.ErrorIs(t, captured, fs.ErrPermission)
	})
}

func TestWalkR_CallbackErrors(t *testing.T) {
	t.Parallel()

	t.Run("callback error is wrapped as ErrWalk", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		dir := createTempDir(t, root, "root")
		createTempDir(t, dir, "sub")

		err := dir.WalkR(func(_ *Path, _ error) error {
			return errSimulated
		})
		require.ErrorIs(t, err, ErrWalk)
		require.ErrorIs(t, err, errSimulated)
		require.ErrorAs(t, err, new(*PathlibError))
	})

	t.Run("unchanged localDirError is returned as is", func(t *testing.T) {
		t.Parallel()

		root := setupTempDir(t)
		dir := createTempDir(t, root, "root")
		unreadableDir := createTempDir(t, dir, "unreadable")
		lockDir(t, unreadableDir)

		var captured error
		err := dir.WalkR(func(_ *Path, localDirError error) error {
			if localDirError != nil {
				captured = localDirError
			}
			return localDirError
		})
		require.Same(t, captured, err)
		require.NotErrorIs(t, err, ErrWalk)
	})
}

func TestWalkR_PreOrder(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	writeTempFile(t, root, "a/1.txt", "")
	writeTempFile(t, root, "a/2.txt", "")
	writeTempFile(t, root, "a/sub/x.txt", "")
	writeTempFile(t, root, "b.txt", "")

	walk := func(t *testing.T, skip string) []string {
		t.Helper()

		var visited []string
		err := root.WalkR(func(p *Path, localDirError error) error {
			require.NoError(t, localDirError)
			rel, err := p.RelativeTo(root)
			require.NoError(t, err)
			visited = append(visited, rel.ToPosix())
			if rel.ToPosix() == skip {
				return SkipDir
			}
			return nil
		})
		require.NoError(t, err)
		return visited
	}

	t.Run("directory is visited before its entries", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, []string{"a", "a/1.txt", "a/2.txt", "a/sub", "a/sub/x.txt", "b.txt"}, walk(t, ""))
	})

	t.Run("SkipDir on a directory skips its contents", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, []string{"a", "b.txt"}, walk(t, "a"))
	})

	t.Run("SkipDir on a file skips the rest of its directory", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, []string{"a", "a/1.txt", "b.txt"}, walk(t, "a/1.txt"))
	})
}

func TestWalkR_SkipDirDoesNotReadDirectory(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	locked := createTempDir(t, root, "locked")
	writeTempFile(t, locked, "secret.txt", "")
	writeTempFile(t, root, "visible.txt", "")
	lockDir(t, locked)

	var visited []string
	err := root.WalkR(func(p *Path, localDirError error) error {
		require.NoError(t, localDirError, "the skipped directory is never read")
		visited = append(visited, p.Base())
		if p.Equals(locked) {
			return SkipDir
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"locked", "visible.txt"}, visited)
}

func TestWalkR_UnreadableDirectoryIsPassedTwice(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	locked := createTempDir(t, root, "locked")
	lockDir(t, locked)

	var errs []error
	err := root.WalkR(func(p *Path, localDirError error) error {
		require.True(t, p.Equals(locked))
		errs = append(errs, localDirError)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, errs, 2)
	require.NoError(t, errs[0], "first call happens before reading")
	require.ErrorIs(t, errs[1], ErrPermissionDenied, "second call carries the directory error")
}

func TestWalkR_UnreadableRootIsPassedOnce(t *testing.T) {
	t.Parallel()

	root := createTempDir(t, setupTempDir(t), "root")
	lockDir(t, root)

	calls := 0
	err := root.WalkR(func(p *Path, localDirError error) error {
		calls++
		require.True(t, p.Equals(root))
		require.ErrorIs(t, localDirError, ErrPermissionDenied)
		return localDirError
	})
	require.ErrorIs(t, err, ErrPermissionDenied)
	require.Equal(t, 1, calls, "the root is only passed with its directory error")
}
