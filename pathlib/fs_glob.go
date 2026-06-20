package pathlib

import (
	"cmp"
	"context"
	"slices"
)

/*
Glob returns all files matching the given pattern within this Path's Posix representation.

If an error is returned, all entries until that error are returned.

This Path must be a directory.
*/
func (p *Path) Glob(pattern string) ([]*Path, error) {
	return p.GlobContext(context.Background(), pattern)
}

/*
GlobContext is Glob with support for cancellation through ctx. The entries
collected before cancellation are returned alongside ctx.Err().
*/
func (p *Path) GlobContext(ctx context.Context, pattern string) ([]*Path, error) {
	return p.GlobWithOptionsContext(ctx, pattern, DefaultGlobOptions())
}

/*
GlobWithOptions returns all entries matching the given pattern within this Path's Posix representation.
If an error is returned, all entries until that error are returned.

This Path must be a directory.
*/
func (p *Path) GlobWithOptions(pattern string, options GlobOptions) ([]*Path, error) {
	return p.GlobWithOptionsContext(context.Background(), pattern, options)
}

/*
GlobWithOptionsContext is GlobWithOptions with support for cancellation through
ctx. The entries collected before cancellation are returned alongside ctx.Err().
*/
func (p *Path) GlobWithOptionsContext(ctx context.Context, pattern string, options GlobOptions) ([]*Path, error) {
	if !p.IsDir() {
		return nil, pathErr(ErrNotDir, *p)
	}

	if pattern == "" {
		return nil, pathErr(ErrEmptyPattern, *p)
	}

	var entries []*Path

	// Initialize GlobOptions.Limit option handling
	limit := max(0, options.Limit)
	limitExists := limit != 0

	applyPatternFunc := func(entry *Path) error {
		addToEntries := false

		if options.FilterFunc == nil {
			// Handle GlobOptions.Filter option
			switch options.Filter {
			case GlobOptionFilterFiles:
				addToEntries = entry.IsFile()
			case GlobOptionFilterDirectories:
				addToEntries = entry.IsDir()
			case GlobOptionFilterAll:
				addToEntries = true
			default:
				return pathErr(errInvalidFilterOption, *entry)
			}
		} else {
			// Handle GlobOptions.FilterFunc option
			addToEntries = options.FilterFunc(entry)
		}

		// Handle GlobOptions.skipPatternMatchCheck
		matchesPattern := options.skipPatternMatchCheck
		if !options.skipPatternMatchCheck {
			// The entry path is joined with the original path and any subdirectories.
			// For pattern matching, we don't want the original path included, because
			// the pattern is matched starting from the original path.
			entryForMatching, err := entry.RelativeTo(p)
			if err != nil {
				return wrapErr(ErrRelImpossible, err, *entry, *p)
			}

			// Test if the current entry matches the given pattern. Also handles GlobOptions.CaseSensitivity option
			localMatchesPattern, matchingErr := entryForMatching.MatchesPatternE(pattern, options.CaseSensitivity)
			if matchingErr != nil {
				return matchingErr
			}

			matchesPattern = localMatchesPattern
		}

		if addToEntries && matchesPattern {
			entries = append(entries, entry)
		}

		// Handle GlobOptions.Limit option.
		// If a limit exists and the number of entries is greater than or equal to the limit,
		// stop any further tree walking.
		if limitExists && len(entries) >= limit {
			return SkipAll
		}

		return nil
	}

	err := p.WalkRContext(ctx, func(entry *Path, localDirError error) error {
		if localDirError != nil {
			// Handle GlobOptions.SkipOnDirError option:
			// skip the offending directory and ignore the error.
			if options.SkipOnDirError {
				return SkipDir
			}

			// By default, abort the entire walk and bubble the error up.
			return localDirError
		}

		return applyPatternFunc(entry)
	})

	return entries, err
}

/*
HasGlobMatchE returns whether the passed pattern exists within this Path's directory.

This function uses filepath.Glob.
*/
func (p *Path) HasGlobMatchE(pattern string) (bool, error) {
	matches, err := p.GlobWithOptions(pattern, GlobOptions{Limit: 1})
	if err != nil {
		return false, err
	}

	return len(matches) != 0, nil
}

/*
HasGlobMatch returns whether the passed pattern exists within this Path's directory.
It wraps HasGlobMatchE and returns the boolean success value or false in case of an error.
*/
func (p *Path) HasGlobMatch(pattern string) bool {
	contains, err := p.HasGlobMatchE(pattern)
	if err != nil {
		return false
	}

	return contains
}

/*
List returns a slice of Path, sorted by Posix representations.
Serves as a simple convenience function for retrieving a sorted enumeration of filtered entries.

If an error occurs, no entries are returned.

Use Walk or WalkR for more fine-grained control.

This function uses GlobWithOptions.
*/
func (p *Path) List(options ListOptions) ([]*Path, error) {
	// Use the extensive and configurable globbing algorithm of GlobWithOptions.
	// This also enables the user to use the same options and features as globbing (like filtering, limits, etc.)
	// The pattern controls recursion depth: "**" matches all levels, "*" matches only the top level.
	pattern := "*"
	if options.Recursive {
		pattern = "**"
	}

	entries, err := p.GlobWithOptions(pattern, options.GlobOptions)
	if err != nil {
		return nil, err
	}

	return sortSliceOfPaths(entries), nil
}

/*
ListFiles returns all files in the Path's directory, sorted by Posix representations.
Serves as a simple convenience function for retrieving a sorted enumeration of files.

This function uses List. The same requirements and behaviors apply.
*/
func (p *Path) ListFiles(recursive bool) ([]*Path, error) {
	options := DefaultListOptions()
	options.Filter = GlobOptionFilterFiles
	options.Recursive = recursive

	return p.List(options)
}

/*
ListDirs returns all directories in the Path's directory, sorted by Posix representations.
Serves as a simple convenience function for retrieving a sorted enumeration of directories.

This function uses List. The same requirements and behaviors apply.
*/
func (p *Path) ListDirs(recursive bool) ([]*Path, error) {
	options := DefaultListOptions()
	options.Filter = GlobOptionFilterDirectories
	options.Recursive = recursive

	return p.List(options)
}

/*
sortSliceOfPaths sorts a slice of Path structs by their Posix representation.

Sorting the slice is actually done in-place by slices.SortFunc. However,
the given slice is also returned for improved downstream code readability.
*/
func sortSliceOfPaths(slice []*Path) []*Path {
	slices.SortFunc(slice, func(a, b *Path) int {
		return cmp.Compare(a.ToPosix(), b.ToPosix())
	})

	return slice
}
