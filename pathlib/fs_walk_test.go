package pathlib

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_Walk(t *testing.T) {
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
				RootSetup: func(t *testing.T, root *Path) *Path { return createTempDir(t, root, "emptyDir") },
				WalkFunc:  func(p *Path) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{}},
		},
		{
			Name: "Directory with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					return dir
				},
				WalkFunc: func(p *Path) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "file2.log"}},
		},
		{
			Name: "Directory with subdirectories (should not recurse)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file.txt", "")
					subdir := createTempDir(t, dir, "subdir")
					writeTempFile(t, subdir, "nested.txt", "") // Should not be walked
					return dir
				},
				WalkFunc: func(p *Path) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file.txt", "subdir"}}, // subdir itself is walked, but not its contents
		},
		{
			Name: "Non-directory path",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path { return writeTempFile(t, root, "file.txt", "") },
				WalkFunc:  func(p *Path) error { return nil },
			},
			Expect: Expect{Error: true},
		},
		{
			Name: "WalkFunc returns error",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.log", "")
					return dir
				},
				WalkFunc: func(p *Path) error {
					return fmt.Errorf("simulated error")
				},
			},
			Expect: Expect{WalkedPaths: []string{}, Error: true},
		},
		{
			Name: "WalkFunc returns SkipAll",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					writeTempFile(t, dir, "file3.csv", "")
					return dir
				},
				WalkFunc: func() func(*Path) error {
					first := true
					return func(p *Path) error {
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

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
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
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		// Ensure consistent order for comparison
		slices.Sort(walkedEntries)
		slices.Sort(expect.WalkedPaths)

		require.Len(t, walkedEntries, len(expect.WalkedPaths))

		if len(expect.WalkedPaths) > 0 {
			require.Equal(t, expect.WalkedPaths, walkedEntries)
		}
	})
}

func TestPath_WalkR(t *testing.T) {
	type Input struct {
		RootSetup   func(*testing.T, *Path) *Path // Function to set up the directory for walking
		CleanupFunc func(*testing.T, *Path)
		WalkRFunc   func(p *Path, localDirError error) error
	}
	type Expect struct {
		WalkedPaths []string // Relative paths from the rootSetup
		Error       bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Empty directory",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path { return createTempDir(t, root, "emptyDir") },
				WalkRFunc: func(p *Path, localDirError error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{}},
		},
		{
			Name: "Flat directory with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					writeTempFile(t, dir, "file2.log", "")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error) error { return nil },
			},
			Expect: Expect{WalkedPaths: []string{"file1.txt", "file2.log"}},
		},
		{
			Name: "Nested directories with files",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested1.txt", "")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "nested2.log", "")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error) error { return nil },
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
				RootSetup: func(t *testing.T, root *Path) *Path { return writeTempFile(t, root, "file.txt", "") },
				WalkRFunc: func(p *Path, localDirError error) error { return nil },
			},
			Expect: Expect{Error: true},
		},
		{
			Name: "WalkRFunc returns error (aborts entire tree)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					dir := createTempDir(t, root, "testDir")
					subdir := createTempDir(t, dir, "subdir")
					writeTempFile(t, subdir, "file2.log", "")
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error) error {
					if p.Base() == "file2.log" {
						return fmt.Errorf("simulated error")
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
					dir := createTempDir(t, root, "testDir")
					writeTempFile(t, dir, "file1.txt", "")
					subdir1 := createTempDir(t, dir, "subdir1")
					writeTempFile(t, subdir1, "nested1.txt", "") // Should be skipped
					createTempDir(t, subdir1, "subdir2")         // Should be skipped
					writeTempFile(t, dir, "file3.csv", "")       // Should still be walked
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error) error {
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
					dir := createTempDir(t, root, "testDir")
					subdir1 := createTempDir(t, dir, "subdir1")
					subdir2 := createTempDir(t, subdir1, "subdir2")
					writeTempFile(t, subdir2, "deep.txt", "") // Should NOT be walked
					return dir
				},
				WalkRFunc: func(p *Path, localDirError error) error {
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
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					dir := createTempDir(t, root, "root")
					createTempDir(t, dir, "readable_dir")
					writeTempFile(t, dir, "readable_dir/file.txt", "")

					unreadableDir := createTempDir(t, dir, "unreadable_dir")
					writeTempFile(t, unreadableDir, "secret.log", "")

					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)

					createTempDir(t, dir, "another_dir")
					writeTempFile(t, dir, "another_dir/another_file.txt", "")
					return dir
				},
				CleanupFunc: func(t *testing.T, root *Path) {
					os.Chmod(root.JoinStrings("root", "unreadable_dir").String(), 0755)
				},
				WalkRFunc: func(p *Path, localDirError error) error {
					if localDirError != nil {
						return nil
					}
					return nil
				},
			},
			Expect: Expect{
				WalkedPaths: []string{
					"another_dir",
					"another_dir/another_file.txt",
					"readable_dir",
					"readable_dir/file.txt",
					"unreadable_dir",
				},
			},
		},
		{
			Name: "Error reading directory, walkFunc returns error (stops)",
			Input: Input{
				RootSetup: func(t *testing.T, root *Path) *Path {
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					// Entries are walked in lexical order, so naming the siblings
					// a/b/c makes the traversal deterministic: a_readable is walked,
					// b_unreadable raises a directory error that aborts the walk, and
					// c_readable (lexically after the error) is never reached. This
					// demonstrates that returning an error stops the walk.
					dir := createTempDir(t, root, "root")
					createTempDir(t, dir, "a_readable")
					unreadableDir := createTempDir(t, dir, "b_unreadable")
					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)
					createTempDir(t, dir, "c_readable")
					return dir
				},
				CleanupFunc: func(t *testing.T, root *Path) {
					os.Chmod(root.JoinStrings("root", "b_unreadable").String(), 0755)
				},
				WalkRFunc: func(p *Path, localDirError error) error {
					if localDirError != nil {
						return fmt.Errorf("custom error in %s: %w", p.ToPosix(), localDirError)
					}
					return nil
				},
			},
			Expect: Expect{
				// Only a_readable is recorded: b_unreadable raises a directory error
				// that aborts the walk (so it is never recorded), and c_readable is
				// never reached.
				WalkedPaths: []string{
					"a_readable",
				},
				Error: true,
			},
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.RootSetup(t, root)

		defer func() {
			if input.CleanupFunc != nil {
				input.CleanupFunc(t, root)
			}
		}()

		var walkedEntries []string
		customWalkRFunc := func(path *Path, localDirError error) error {
			relPath, err := path.RelativeTo(p)
			if err != nil && !expectError { // Ignore relative path expectError if we expect an overall expectError.
				return err
			}

			var inputWalkFuncErr error = nil

			// Original walkFunc for specific test behavior
			if input.WalkRFunc != nil {
				inputWalkFuncErr = input.WalkRFunc(path, localDirError)
			}

			// Record entries that were visited, including those that signal SkipDir or
			// SkipAll (the entry itself was still visited). Genuine errors are not recorded.
			if inputWalkFuncErr == nil || errors.Is(inputWalkFuncErr, SkipDir) || errors.Is(inputWalkFuncErr, SkipAll) {
				walkedEntries = append(walkedEntries, relPath.ToPosix())
			}

			return inputWalkFuncErr
		}

		err := p.WalkR(customWalkRFunc)

		if expect.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		if len(expect.WalkedPaths) != 0 && len(walkedEntries) != 0 {
			// Ensure consistent order for comparison
			slices.Sort(expect.WalkedPaths)
			slices.Sort(walkedEntries)
			require.Equal(t, expect.WalkedPaths, walkedEntries)
		} else {
			require.Equal(t, len(expect.WalkedPaths), len(walkedEntries))
		}
	})
}

func TestPath_WalkContext_Cancellation(t *testing.T) {
	buildTree := func(t *testing.T) *Path {
		root := setupTempDir(t)
		dir := createTempDir(t, root, "tree")
		writeTempFile(t, dir, "a.txt", "")
		writeTempFile(t, dir, "b.txt", "")
		sub := createTempDir(t, dir, "sub")
		writeTempFile(t, sub, "c.txt", "")
		return dir
	}

	t.Run("WalkContext with cancelled context returns context.Canceled", func(t *testing.T) {
		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var visited int
		err := dir.WalkContext(ctx, func(p *Path) error {
			visited++
			return nil
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, visited, "no entries should be visited after cancellation")
	})

	t.Run("WalkRContext with cancelled context returns context.Canceled", func(t *testing.T) {
		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := dir.WalkRContext(ctx, func(p *Path, localDirError error) error {
			return nil
		})

		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("WalkRContext cancelled mid-walk stops early", func(t *testing.T) {
		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())

		var visited int
		err := dir.WalkRContext(ctx, func(p *Path, localDirError error) error {
			visited++
			cancel() // cancel after the first visited entry
			return nil
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, visited, "walk should stop right after cancellation")
	})

	t.Run("GlobContext with cancelled context returns context.Canceled", func(t *testing.T) {
		dir := buildTree(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		matches, err := dir.GlobContext(ctx, "*.txt")
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, matches)
	})

	t.Run("context variants with Background behave like the plain calls", func(t *testing.T) {
		dir := buildTree(t)

		plain, err := dir.Glob("*.txt")
		require.NoError(t, err)

		withCtx, err := dir.GlobContext(context.Background(), "*.txt")
		require.NoError(t, err)

		require.ElementsMatch(t, plain, withCtx)
	})
}

func TestWalk_DirAccessErrorGrouping(t *testing.T) {
	t.Run("ErrOpen and ErrReadDir are members of the ErrDirAccess group", func(t *testing.T) {
		// Precise matching still works.
		require.ErrorIs(t, ErrOpen, ErrOpen)
		require.ErrorIs(t, ErrReadDir, ErrReadDir)

		// Both are catchable through the shared parent group.
		require.ErrorIs(t, ErrOpen, ErrAccess)
		require.ErrorIs(t, ErrReadDir, ErrAccess)

		// The two remain distinguishable from each other.
		require.NotErrorIs(t, ErrOpen, ErrReadDir)
		require.NotErrorIs(t, ErrReadDir, ErrOpen)
	})

	t.Run("localDirError from a real walk matches ErrDirAccess", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("chmod 0000 does not restrict access on Windows")
		}

		root := setupTempDir(t)
		dir := createTempDir(t, root, "root")
		unreadableDir := createTempDir(t, dir, "unreadable")
		require.NoError(t, os.Chmod(unreadableDir.String(), 0000))
		t.Cleanup(func() { os.Chmod(unreadableDir.String(), 0755) })

		var captured error
		err := dir.WalkR(func(p *Path, localDirError error) error {
			if localDirError != nil {
				captured = localDirError
			}
			return nil // ignore so the walk completes
		})
		require.NoError(t, err)

		require.Error(t, captured)
		require.ErrorIs(t, captured, ErrAccess)
	})
}
