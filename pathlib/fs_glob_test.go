package pathlib

import (
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_GlobWithOptions(t *testing.T) {
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

	var items []string
	items = append(items, files...)
	items = append(items, directories...)

	var topLevelFiles []string
	for _, file := range files {
		if !strings.Contains(file, "/") {
			topLevelFiles = append(topLevelFiles, file)
		}
	}

	var topLevelDirs []string
	for _, dir := range directories {
		if !strings.Contains(dir, "/") {
			topLevelDirs = append(topLevelDirs, dir)
		}
	}

	var topLevelItems []string
	topLevelItems = append(topLevelItems, topLevelFiles...)
	topLevelItems = append(topLevelItems, topLevelDirs...)

	setupComplexDir := func(t *testing.T, root *Path) *Path {
		for _, dir := range directories {
			createTempDir(t, root, dir)
		}

		for _, file := range files {
			writeTempFile(t, root, file, "")
		}

		return root
	}

	cases := []TestCase[Input, Expect]{
		/*
		 * Basic non-recursive
		 */
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
			Expect: Expect{FoundPaths: []string{"Temp.txt", "Foo.TXT"}}, // Non-recursive only matches top-level
			Error:  false,
		},
		{
			Name: "Non-recursive: starts with 'd' (d*)",
			Input: Input{
				Pattern: "d*", Options: DefaultGlobOptions(),
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"data", "dump.JSON"}}, // case-insensitive by default
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

		/*
		 * Recursive
		 */
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
			Expect: Expect{FoundPaths: []string{"data/report.txt", "data/docs/summary.txt", "Temp.txt", "Foo.TXT"}},
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

		/*
		 * Limit - note: order is not guaranteed, so we just check length
		 */
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

		/*
		 * CaseSensitivity
		 */
		{
			Name: "CaseSensitivity=true, *.txt - matches only lowercase .txt",
			Input: Input{
				Pattern: "*.txt", Options: GlobOptions{CaseSensitivity: CaseSensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Temp.txt"}},
			Error:  false,
		},
		{
			Name: "CaseSensitivity=true, *.TXT - no match (no files end in .TXT)",
			Input: Input{
				Pattern: "*.TXT", Options: GlobOptions{CaseSensitivity: CaseSensitive},
				Setup: setupComplexDir,
			},
			Expect: Expect{FoundPaths: []string{"Foo.TXT"}},
			Error:  false,
		},

		/*
		 * Filter
		 */
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

		/*
		 * FilterFunc
		 */
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

		/*
		 * SkipOnDirError
		 */
		{
			Name: "SkipOnDirError=true with unreadable dir",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{SkipOnDirError: true},
				Setup: func(t *testing.T, root *Path) *Path {
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					setupComplexDir(t, root)
					unreadableDir := createTempDir(t, root, "data/unreadable_logs")
					writeTempFile(t, unreadableDir, "secret.log", "")
					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)
					t.Cleanup(func() {
						os.Chmod(unreadableDir.String(), 0755)
					})
					return root
				},
			},
			Expect: Expect{FoundPaths: items},
			Error:  false,
		},
		{
			Name: "SkipOnDirError=false (default) with unreadable dir",
			Input: Input{
				Pattern: "**/*", Options: GlobOptions{SkipOnDirError: false},
				Setup: func(t *testing.T, root *Path) *Path {
					if runtime.GOOS == "windows" {
						t.Skip("chmod 0000 does not restrict access on Windows")
					}
					setupComplexDir(t, root)
					unreadableDir := createTempDir(t, root, "data/unreadable_logs")
					writeTempFile(t, unreadableDir, "secret.log", "")
					err := os.Chmod(unreadableDir.String(), 0000)
					require.NoError(t, err)
					t.Cleanup(func() {
						os.Chmod(unreadableDir.String(), 0755)
					})
					return root
				},
			},
			Expect: Expect{
				FoundPaths: nil,
			},
			Error: true,
		},

		/*
		 * Edge Cases
		 */
		{
			Name: "Non-directory path",
			Input: Input{
				Pattern: "*", Options: DefaultGlobOptions(),
				Setup: func(t *testing.T, root *Path) *Path {
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
				Setup: func(t *testing.T, root *Path) *Path {
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
					return writeTempFile(t, root, "temp.txt", "") // Return the FILE path to glob on
				},
			},
			Expect: Expect{FoundPaths: []string{}},
			Error:  true,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		globPath := input.Setup(t, root)

		foundPaths, err := globPath.GlobWithOptions(input.Pattern, input.Options)

		if expectError {
			require.Error(t, err)
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

		slices.Sort(expect.FoundPaths)
		require.Equal(t, expect.FoundPaths, actualRelPaths)
	})
}

func TestPath_Glob(t *testing.T) {
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
					return createTempDir(t, root, "emptyDir")
				},
			},
			Expect: Expect{HasMatch: false},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		hasMatch, err := p.HasGlobMatchE(input.Pattern)
		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.Equal(t, expect.HasMatch, hasMatch)
	})
}

func TestPath_HasGlobMatch(t *testing.T) {
	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "file.txt", "")

	require.True(t, dir.HasGlobMatch("*.txt"))
	require.False(t, dir.HasGlobMatch("*.csv"))
}

func TestPath_List(t *testing.T) {
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
					return createTempDir(t, root, "emptyDir")
				},
			},
			Expect: Expect{Entries: []string{}},
			Error:  false,
		},
	}

	runForResultsE(t, cases, func(t *testing.T, input Input, expect Expect, expectError bool) {
		root := setupTempDir(t)
		p := input.Setup(t, root)
		entries, err := p.List(input.Options)
		if expectError {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)

		relPaths := relPathsSorted(t, entries, p)

		if input.Options.Limit > 0 && expect.Entries == nil {
			require.Len(t, relPaths, input.Options.Limit)
			return
		}

		slices.Sort(expect.Entries)
		require.Equal(t, expect.Entries, relPaths)
	})
}

func TestPath_ListFiles(t *testing.T) {
	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "a.txt", "")
	writeTempFile(t, dir, "b.log", "")
	subdir := createTempDir(t, dir, "subdir")
	writeTempFile(t, subdir, "nested.txt", "")

	t.Run("non-recursive", func(t *testing.T) {
		files, err := dir.ListFiles(false)
		require.NoError(t, err)
		require.Equal(t, []string{"a.txt", "b.log"}, relPathsSorted(t, files, dir))
	})

	t.Run("recursive", func(t *testing.T) {
		files, err := dir.ListFiles(true)
		require.NoError(t, err)
		require.Equal(t, []string{"a.txt", "b.log", "subdir/nested.txt"}, relPathsSorted(t, files, dir))
	})
}

func TestPath_ListDirs(t *testing.T) {
	root := setupTempDir(t)
	dir := createTempDir(t, root, "dir")
	writeTempFile(t, dir, "a.txt", "")
	subdir1 := createTempDir(t, dir, "subdir1")
	createTempDir(t, dir, "subdir2")
	createTempDir(t, subdir1, "nested")

	t.Run("non-recursive", func(t *testing.T) {
		dirs, err := dir.ListDirs(false)
		require.NoError(t, err)
		require.Equal(t, []string{"subdir1", "subdir2"}, relPathsSorted(t, dirs, dir))
	})

	t.Run("recursive", func(t *testing.T) {
		dirs, err := dir.ListDirs(true)
		require.NoError(t, err)
		require.Equal(t, []string{"subdir1", "subdir1/nested", "subdir2"}, relPathsSorted(t, dirs, dir))
	})
}
