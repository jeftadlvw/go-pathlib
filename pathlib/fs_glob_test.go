package pathlib

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:maintidx // The table of cases is long by design.
func TestPath_GlobWithOptions(t *testing.T) {
	t.Parallel()

	type Input struct {
		Pattern string
		Options GlobOptions
		Setup   func(*testing.T, *Path) *Path // Setup returns the path to perform glob on
	}

	type Expect struct {
		FoundPaths []string // Relative paths from the glob root, sorted
	}

	directories := []string{
		"data",
		"data/logs",
		"data/docs",
		"archive",
	}

	files := []string{
		"data/report.txt",
		"data/image.png",
		"data/logs/app.log",
		"data/logs/error.log",
		"data/docs/README.md",
		"data/docs/summary.txt",
		"archive/old.zip",
		"archive/data.tar.gz",
		"config.json",
		".gitignore",
		"Temp.txt",
		"Foo.TXT",
		"dump.JSON",
	}

	items := slices.Concat(files, directories)
	topLevelItems := slices.Concat(topLevelPaths(files), topLevelPaths(directories))

	setupComplexDir := func(t *testing.T, root *Path) *Path {
		t.Helper()

		for _, dir := range directories {
			createTempDir(t, root, dir)
		}

		for _, file := range files {
			writeTempFile(t, root, file, "")
		}

		return root
	}

	cases := []TestCase[Input, Expect]{
		// Basic non-recursive
		{
			Name: "Non-recursive: all files and dirs (*)",
			Input: Input{
				Pattern: "*", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: topLevelItems},
			Error:  false,
		},
		{
			Name: "Non-recursive: *.txt",
			Input: Input{
				Pattern: "*.txt", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Temp.txt"}}, // Non-recursive only matches top-level, case-sensitive by default
			Error:  false,
		},
		{
			Name: "Non-recursive: starts with 'd' (d*)",
			Input: Input{
				Pattern: "d*", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data", "dump.JSON"}},
			Error:  false,
		},
		{
			Name: "Non-recursive: no match",
			Input: Input{
				Pattern: "nonexistent", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{}},
			Error:  false,
		},

		// Recursive
		{
			Name: "Recursive: all files and dirs (**) - should match all",
			Input: Input{
				Pattern: "**", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: items},
			Error:  false,
		},
		{
			Name: "Recursive: all .txt files (**/*.txt)",
			Input: Input{
				Pattern: "**/*.txt", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/report.txt", "data/docs/summary.txt", "Temp.txt"}},
			Error:  false,
		},
		{
			Name: "Recursive: specific subdirectory files (data/*.txt)",
			Input: Input{
				Pattern: "data/*.txt", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/report.txt"}},
			Error:  false,
		},
		{
			Name: "Recursive: all subdirectory files (data/**/*.txt)",
			Input: Input{
				Pattern: "data/**/*.txt", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/report.txt", "data/docs/summary.txt"}},
			Error:  false,
		},

		// Limit - note: order is not guaranteed, so we just check length
		{
			Name: "Limit=1",
			Input: Input{
				Pattern: "*", Options: GlobOptions{Limit: 1},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: nil}, // Will check length instead
			Error:  false,
		},
		{
			Name: "Limit=2, Recursive",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{Limit: 2},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: nil}, // Will check length instead
			Error:  false,
		},

		// CaseSensitivity
		{
			Name: "CaseSensitive, *.txt - matches only lowercase .txt",
			Input: Input{
				Pattern: "*.txt", Options: GlobOptions{CaseSensitivity: CaseSensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Temp.txt"}},
			Error:  false,
		},
		{
			Name: "CaseSensitive, *.TXT - matches only uppercase .TXT",
			Input: Input{
				Pattern: "*.TXT", Options: GlobOptions{CaseSensitivity: CaseSensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Foo.TXT"}},
			Error:  false,
		},
		{
			Name: "CaseInsensitive, *.txt - matches every casing",
			Input: Input{
				Pattern: "*.txt", Options: GlobOptions{CaseSensitivity: CaseInsensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Temp.txt", "Foo.TXT"}},
			Error:  false,
		},
		{
			Name: "CaseInsensitive, **/*.TXT - matches every casing recursively",
			Input: Input{
				Pattern: "**/*.TXT", Options: GlobOptions{CaseSensitivity: CaseInsensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/report.txt", "data/docs/summary.txt", "Temp.txt", "Foo.TXT"}},
			Error:  false,
		},
		{
			Name: "Zero value GlobOptions is case-sensitive",
			Input: Input{
				Pattern: "*.TXT", Options: GlobOptions{},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Foo.TXT"}},
			Error:  false,
		},

		// Filter
		{
			Name: "Filter: Files only",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{Filter: GlobOptionFilterFiles},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: files},
			Error:  false,
		},
		{
			Name: "Filter: Directories only",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{Filter: GlobOptionFilterDirectories},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: directories},
			Error:  false,
		},

		// FilterFunc
		{
			Name: "FilterFunc: only .md files (recursive)",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{
					FilterFunc: func(p *Path) bool {
						return p.Extension() == ".md"
					},
				},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data/docs/README.md"}},
			Error:  false,
		},

		// SkipOnDirError
		{
			Name: "SkipOnDirError=true with unreadable dir",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{SkipOnDirError: true},
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					setupComplexDir(t, root)
					unreadableDir := createTempDir(t, root, "data/unreadable_logs")
					writeTempFile(t, unreadableDir, "secret.log", "")
					lockDir(t, unreadableDir)
					return root
				},
			},
			// The unreadable directory itself exists and matches, only its
			// contents are skipped.
			Expect: Expect{FoundPaths: append(slices.Clone(items), "data/unreadable_logs")},
			Error:  false,
		},
		{
			Name: "SkipOnDirError=false (default) with unreadable dir",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{SkipOnDirError: false},
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					setupComplexDir(t, root)
					unreadableDir := createTempDir(t, root, "data/unreadable_logs")
					writeTempFile(t, unreadableDir, "secret.log", "")
					lockDir(t, unreadableDir)
					return root
				},
			},
			Expect: Expect{
				FoundPaths: nil,
			},
			Error: true,
		},

		// Edge Cases
		{
			Name: "Non-directory path",
			Input: Input{
				Pattern: "*", Options: DefaultGlobOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					return writeTempFile(t, root, "a_file.txt", "") // Return the FILE path to glob on
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Empty pattern (should error)",
			Input: Input{
				Pattern: "", Options: DefaultGlobOptions(),
				Setup: func(_ *testing.T, root *Path) *Path {
					return root
				},
			},
			Expect: Expect{},
			Error:  true,
		},
		{
			Name: "Invalid filter option (should error)",
			Input: Input{
				Pattern: "*", Options: GlobOptions{Filter: 999},
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					return writeTempFile(t, root, "temp.txt", "") // Return the FILE path to glob on
				},
			},
			Expect: Expect{FoundPaths: []string{}},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		globPath := input.Setup(t, root)

		foundPaths, err := globPath.GlobWithOptions(input.Pattern, input.Options)

		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			return
		}
		require.NoError(t, err)

		// Convert found paths to relative for comparison
		actualRelPaths := make([]string, len(foundPaths))
		for i, p := range foundPaths {
			relPath, err := p.RelativeTo(globPath)
			require.NoError(t, err)
			actualRelPaths[i] = relPath.ToPosix()
		}
		slices.Sort(actualRelPaths)

		// Special handling for Limit tests - just check the count
		if input.Options.Limit > 0 && expect.FoundPaths == nil {
			require.Len(t, actualRelPaths, input.Options.Limit)
			return
		}

		// For error cases with partial results, skip exact comparison
		if expect.FoundPaths == nil {
			return
		}

		// The cases share their slices, so a sorted clone is compared.
		expect.FoundPaths = slices.Clone(expect.FoundPaths)
		slices.Sort(expect.FoundPaths)
		require.Equal(t, expect.FoundPaths, actualRelPaths)
	})
}

func TestPath_Glob(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	dir := createTempDir(t, root, "globDir")
	writeTempFile(t, dir, "a.txt", "")
	writeTempFile(t, dir, "b.txt", "")
	writeTempFile(t, dir, "c.log", "")

	results, err := dir.Glob("*.txt")
	require.NoError(t, err)

	relPaths := make([]string, len(results))
	for i, p := range results {
		rel, err := p.RelativeTo(dir)
		require.NoError(t, err)
		relPaths[i] = rel.ToPosix()
	}
	slices.Sort(relPaths)
	require.Equal(t, []string{"a.txt", "b.txt"}, relPaths)
}

func TestPath_HasGlobMatchE(t *testing.T) {
	t.Parallel()

	type Input struct {
		Pattern string
		Setup   func(*testing.T, *Path) *Path
	}

	type Expect struct {
		HasMatch bool
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "Match exists",
			Input: Input{
				Pattern: "*.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					return dir
				},
			},
			Expect: Expect{HasMatch: true},
			Error:  false,
		},
		{
			Name: "No match",
			Input: Input{
				Pattern: "*.csv",
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					return dir
				},
			},
			Expect: Expect{HasMatch: false},
			Error:  false,
		},
		{
			Name: "Empty directory",
			Input: Input{
				Pattern: "*",
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					return createTempDir(t, root, "emptyDir")
				},
			},
			Expect: Expect{HasMatch: false},
			Error:  false,
		},
		{
			Name: "Double asterisk matches nested entry",
			Input: Input{
				Pattern: "**/deep.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "a/b/deep.txt", "")
					return dir
				},
			},
			Expect: Expect{HasMatch: true},
			Error:  false,
		},
		{
			Name: "Matching is case-sensitive",
			Input: Input{
				Pattern: "*.txt",
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "FILE.TXT", "")
					return dir
				},
			},
			Expect: Expect{HasMatch: false},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.Setup(t, root)
		hasMatch, err := p.HasGlobMatchE(input.Pattern)
		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			return
		}
		require.NoError(t, err)
		require.Equal(t, expect.HasMatch, hasMatch)
	})
}

func TestPath_HasGlobMatch(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "file.txt", "")

	require.True(t, dir.HasGlobMatch("*.txt"))
	require.False(t, dir.HasGlobMatch("*.csv"))
}

func TestPath_List(t *testing.T) {
	t.Parallel()

	type Input struct {
		Options ListOptions
		Setup   func(*testing.T, *Path) *Path
	}

	type Expect struct {
		Entries []string
	}

	cases := []TestCase[Input, Expect]{
		{
			Name: "List all entries non-recursive",
			Input: Input{
				Options: DefaultListOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "b.txt", "")
					writeTempFile(t, dir, "a.txt", "")
					createTempDir(t, dir, "subdir")
					writeTempFile(t, dir.JoinStrings("subdir"), "nested.txt", "")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"a.txt", "b.txt", "subdir"}},
			Error:  false,
		},
		{
			Name: "List all entries recursive",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Recursive = true
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "b.txt", "")
					writeTempFile(t, dir, "a.txt", "")
					createTempDir(t, dir, "subdir")
					writeTempFile(t, dir.JoinStrings("subdir"), "nested.txt", "")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"a.txt", "b.txt", "subdir", "subdir/nested.txt"}},
			Error:  false,
		},
		{
			Name: "List with limit",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Limit = 2
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "a.txt", "")
					writeTempFile(t, dir, "b.txt", "")
					writeTempFile(t, dir, "c.txt", "")
					return dir
				},
			},
			Expect: Expect{Entries: nil}, // check length only
			Error:  false,
		},
		{
			Name: "List files only",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Filter = GlobOptionFilterFiles
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					createTempDir(t, dir, "subdir")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"file.txt"}},
			Error:  false,
		},
		{
			Name: "List directories only",
			Input: Input{
				Options: func() ListOptions {
					o := DefaultListOptions()
					o.Filter = GlobOptionFilterDirectories
					return o
				}(),
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					dir := createTempDir(t, root, "dir")
					writeTempFile(t, dir, "file.txt", "")
					createTempDir(t, dir, "subdir")
					return dir
				},
			},
			Expect: Expect{Entries: []string{"subdir"}},
			Error:  false,
		},
		{
			Name: "List empty directory",
			Input: Input{
				Options: DefaultListOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
					t.Helper()

					return createTempDir(t, root, "emptyDir")
				},
			},
			Expect: Expect{Entries: []string{}},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		t.Helper()

		root := setupTempDir(t)
		p := input.Setup(t, root)
		entries, err := p.List(input.Options)
		if expectError {
			require.ErrorIs(t, err, ErrPathlib)
			return
		}
		require.NoError(t, err)

		relPaths := relPathsSorted(t, entries, p)

		if input.Options.Limit > 0 && expect.Entries == nil {
			require.Len(t, relPaths, input.Options.Limit)
			return
		}

		// The cases share their slices, so a sorted clone is compared.
		expect.Entries = slices.Clone(expect.Entries)
		slices.Sort(expect.Entries)
		require.Equal(t, expect.Entries, relPaths)
	})
}

func TestPath_ListFiles(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "a.txt", "")
	writeTempFile(t, dir, "b.log", "")
	subdir := createTempDir(t, dir, "subdir")
	writeTempFile(t, subdir, "nested.txt", "")

	t.Run("non-recursive", func(t *testing.T) {
		t.Parallel()

		files, err := dir.ListFiles(false)
		require.NoError(t, err)
		require.Equal(t, []string{"a.txt", "b.log"}, relPathsSorted(t, files, dir))
	})

	t.Run("recursive", func(t *testing.T) {
		t.Parallel()

		files, err := dir.ListFiles(true)
		require.NoError(t, err)
		require.Equal(t, []string{"a.txt", "b.log", "subdir/nested.txt"}, relPathsSorted(t, files, dir))
	})
}

func TestPath_ListDirs(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "a.txt", "")
	subdir1 := createTempDir(t, dir, "subdir1")
	createTempDir(t, dir, "subdir2")
	createTempDir(t, subdir1, "nested")

	t.Run("non-recursive", func(t *testing.T) {
		t.Parallel()

		dirs, err := dir.ListDirs(false)
		require.NoError(t, err)
		require.Equal(t, []string{"subdir1", "subdir2"}, relPathsSorted(t, dirs, dir))
	})

	t.Run("recursive", func(t *testing.T) {
		t.Parallel()

		dirs, err := dir.ListDirs(true)
		require.NoError(t, err)
		require.Equal(t, []string{"subdir1", "subdir1/nested", "subdir2"}, relPathsSorted(t, dirs, dir))
	})
}

func TestGlob_Symlinks(t *testing.T) {
	t.Parallel()

	// dir contains a regular file, a symlink to a directory outside of dir
	// and a broken symlink. The symlinked directory is never entered.
	root := setupTempDir(t)
	writeTempFile(t, root, "outside/hidden.txt", "")
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "file.txt", "")
	createTempSymlinkAbs(t, root, "outside", "dir/linkDir")
	createTempSymlinkAbs(t, root, "nonexistent", "dir/brokenLink")

	t.Run("double asterisk does not enter symlinked directory", func(t *testing.T) {
		t.Parallel()

		entries, err := dir.Glob("**")
		require.NoError(t, err)
		require.Equal(t, []string{"brokenLink", "file.txt", "linkDir"}, relPathsSorted(t, entries, dir))
	})

	t.Run("symlink to directory counts as directory", func(t *testing.T) {
		t.Parallel()

		dirs, err := dir.ListDirs(true)
		require.NoError(t, err)
		require.Equal(t, []string{"linkDir"}, relPathsSorted(t, dirs, dir))
	})

	t.Run("broken symlink is neither file nor directory", func(t *testing.T) {
		t.Parallel()

		files, err := dir.ListFiles(true)
		require.NoError(t, err)
		require.Equal(t, []string{"file.txt"}, relPathsSorted(t, files, dir))
	})
}

func TestGlob_InvalidInputFailsBeforeWalking(t *testing.T) {
	t.Parallel()

	// The directory is empty, so the walk callback never runs.
	dir := setupTempDir(t)

	_, err := dir.Glob("[")
	require.ErrorIs(t, err, ErrBadPattern)

	_, err = dir.GlobWithOptions("*", GlobOptions{Filter: 99})
	require.ErrorIs(t, err, ErrInvalidFilter)
}

func TestGlob_DepthPruning(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) *Path {
		t.Helper()

		root := setupTempDir(t)
		writeTempFile(t, root, "top.txt", "")
		writeTempFile(t, root, "a/mid.txt", "")
		writeTempFile(t, root, "a/b/deep.txt", "")
		writeTempFile(t, root, "a/b/c/deeper.txt", "")
		return root
	}

	// FilterFunc is called for every entry the walk visits, so it shows which
	// directories were read.
	visitedWith := func(t *testing.T, root *Path, pattern string) ([]string, []string) {
		t.Helper()

		var visited []string
		options := DefaultGlobOptions()
		options.FilterFunc = func(p *Path) bool {
			rel, err := p.RelativeTo(root)
			require.NoError(t, err)
			visited = append(visited, rel.ToPosix())
			return true
		}

		matches, err := root.GlobWithOptions(pattern, options)
		require.NoError(t, err)
		slices.Sort(visited)
		return relPathsSorted(t, matches, root), visited
	}

	t.Run("single level pattern reads only the root", func(t *testing.T) {
		t.Parallel()

		root := setup(t)
		matches, visited := visitedWith(t, root, "*")
		require.Equal(t, []string{"a", "top.txt"}, matches)
		require.Equal(t, []string{"a", "top.txt"}, visited)
	})

	t.Run("two level pattern reads one level below the root", func(t *testing.T) {
		t.Parallel()

		root := setup(t)
		matches, visited := visitedWith(t, root, "a/*.txt")
		require.Equal(t, []string{"a/mid.txt"}, matches)
		require.Equal(t, []string{"a", "a/b", "a/mid.txt", "top.txt"}, visited)
	})

	t.Run("double asterisk reads the whole tree", func(t *testing.T) {
		t.Parallel()

		root := setup(t)
		matches, visited := visitedWith(t, root, "**/*.txt")
		require.Equal(t, []string{"a/b/c/deeper.txt", "a/b/deep.txt", "a/mid.txt", "top.txt"}, matches)
		require.Len(t, visited, 7)
	})

	t.Run("character class is not pruned", func(t *testing.T) {
		t.Parallel()

		// "[^x]" matches "/", so the pattern matches a path with two parts.
		root := setup(t)
		matches, _ := visitedWith(t, root, "a[^x]mid.txt")
		require.Equal(t, []string{"a/mid.txt"}, matches)
	})
}

func TestGlob_DepthPruningKeepsSymlinkSiblings(t *testing.T) {
	t.Parallel()

	// A symlink to a directory is no directory for the walk, so its contents are
	// never visited. Returning SkipDir for it would skip the rest of its
	// directory.
	root := setupTempDir(t)
	createTempDir(t, root, "target")
	dir := createTempDir(t, root, "dir")
	createTempSymlinkAbs(t, root, "target", "dir/a_link")
	writeTempFile(t, dir, "b.txt", "")

	matches, err := dir.Glob("*")
	require.NoError(t, err)
	require.Equal(t, []string{"a_link", "b.txt"}, relPathsSorted(t, matches, dir))
}

func TestList_NonRecursiveIgnoresUnreadableSubdirectories(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	writeTempFile(t, root, "top.txt", "")
	locked := createTempDir(t, root, "locked")
	deepLocked := createTempDir(t, root, "a/deep_locked")
	lockDir(t, locked)
	lockDir(t, deepLocked)

	// Before, every non-recursive listing walked the whole tree and failed
	// here.
	entries, err := root.List(DefaultListOptions())
	require.NoError(t, err)
	require.Equal(t, []string{"a", "locked", "top.txt"}, relPathsSorted(t, entries, root))

	dirs, err := root.ListDirs(false)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "locked"}, relPathsSorted(t, dirs, root))

	files, err := root.ListFiles(false)
	require.NoError(t, err)
	require.Equal(t, []string{"top.txt"}, relPathsSorted(t, files, root))

	require.False(t, root.IsEmptyDir())

	// A recursive listing still reports the unreadable directories.
	options := DefaultListOptions()
	options.Recursive = true
	_, err = root.List(options)
	require.ErrorIs(t, err, ErrPermissionDenied)
}

// topLevelPaths returns the paths without a separator, in their order.
func topLevelPaths(paths []string) []string {
	var topLevel []string
	for _, p := range paths {
		if !strings.Contains(p, "/") {
			topLevel = append(topLevel, p)
		}
	}
	return topLevel
}

func TestPatternMaxDepth(t *testing.T) {
	t.Parallel()

	cases := map[string]int{
		"*":         1,
		"*.txt":     1,
		"a/*.txt":   2,
		"a/b/c":     3,
		`a\/b`:      2, // an escaped separator still matches only "/"
		"**":        0,
		"a/**/b":    0,
		"**.txt":    0,
		"a[bc]":     0, // character classes can match "/"
		"a/[^x]/b":  0,
		"Plugins/*": 2,
	}

	for pattern, expect := range cases {
		require.Equal(t, expect, patternMaxDepth(pattern), "pattern %q", pattern)
	}
}
