package pathlib

import (
	"cmp"
	"context"
	"io/fs"
	"slices"
	"strings"
)

/*
Glob returns all entries matching the given pattern within this Path's Posix representation.

If an error is returned, all entries until that error are returned.

This Path must be a directory.

This function uses GlobWithOptions with DefaultGlobOptions. The same behaviors apply.
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

This Path must be a directory. A missing path returns [ErrNotExist], and an
existing non-directory returns [ErrNotDir]. An empty pattern returns
[ErrEmptyPattern], a malformed pattern returns [ErrBadPattern], and an unknown
GlobOptions.Filter returns [ErrInvalidFilter]. Denied access returns
[ErrPermissionDenied], also for a directory inside the tree. Any other failure
to check the path returns [ErrStat], and any other failure to open or read a
directory returns a kind below [ErrAccess]. GlobOptions.SkipOnDirError skips a
directory that cannot be opened or read.

The tree is walked with WalkR, so symlinks to directories inside the tree are matched
as entries, but their contents are not visited. GlobOptions.Filter judges a symlink by
its target, so a symlink to a directory counts as a directory and a broken symlink
is neither a file nor a directory.

Directories below the deepest level the pattern can match are not read. A pattern
without "**" and without character classes matches only paths with as many parts
as the pattern, so e.g. "*" reads only this directory and "src/*.go" reads only
this directory and its direct subdirectories. Character classes are excluded because
they can match "/".
*/
func (p *Path) GlobWithOptions(pattern string, options GlobOptions) ([]*Path, error) {
	return p.GlobWithOptionsContext(context.Background(), pattern, options)
}

/*
GlobWithOptionsContext is GlobWithOptions with support for cancellation through
ctx. The entries collected before cancellation are returned alongside ctx.Err(),
wrapped as ErrWalk.
*/
func (p *Path) GlobWithOptionsContext(ctx context.Context, pattern string, options GlobOptions) ([]*Path, error) {
	err := validateGlob(p, pattern, options)
	if err != nil {
		return nil, err
	}

	var entries []*Path

	// Initialize GlobOptions.Limit option handling
	limit := max(0, options.Limit)
	limitExists := limit != 0

	// Directories at the deepest level the pattern can match are not entered.
	// 0 means the pattern can match at any depth.
	maxDepth := patternMaxDepth(pattern)

	err = p.walkRContext(ctx, func(entry *Path, dirEntry fs.DirEntry, localDirError error) error {
		if localDirError != nil {
			// Handle GlobOptions.SkipOnDirError option:
			// skip the offending directory and ignore the error.
			if options.SkipOnDirError {
				return SkipDir
			}

			// By default, abort the entire walk and bubble the error up.
			return localDirError
		}

		matched, matchErr := matchGlobEntry(p, entry, dirEntry, pattern, options, maxDepth)
		if matched {
			entries = append(entries, entry)
		}

		// Handle GlobOptions.Limit option.
		// If a limit exists and the number of entries is greater than or equal to the limit,
		// stop any further tree walking.
		if limitExists && len(entries) >= limit {
			return SkipAll
		}

		return matchErr
	})

	return entries, err
}

/*
patternMaxDepth returns the number of path parts the paths matched by pattern can
have at most, or 0 if there is no limit. "*" and "?" never match "/", so only
"**" and character classes, which can match "/", match across separators.
*/
func patternMaxDepth(pattern string) int {
	if strings.Contains(pattern, "**") || strings.Contains(pattern, "[") {
		return 0
	}

	return strings.Count(pattern, canonicalPathSeparator) + 1
}

/*
HasGlobMatchE returns whether the passed pattern exists within this Path's directory.

This function uses GlobWithOptions with GlobOptions{Limit: 1}, so the pattern
syntax (including "**") is the same and matching is case-sensitive. The errors
of [Path.GlobWithOptions] apply.
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

This function uses GlobWithOptions, and the errors of [Path.GlobWithOptions]
apply.
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

/*
validateGlob checks the arguments of a glob below p. It returns the errors of
[requireDir], [ErrEmptyPattern] for an empty pattern, [ErrBadPattern] for an
invalid pattern, and [ErrInvalidFilter] for an unknown filter.
*/
func validateGlob(p *Path, pattern string, options GlobOptions) error {
	err := requireDir(p)
	if err != nil {
		return err
	}

	if pattern == "" {
		return patternErr(ErrEmptyPattern, pattern, nil, *p)
	}

	err = validatePattern(pattern)
	if err != nil {
		return patternErr(ErrBadPattern, pattern, err, *p)
	}

	if options.FilterFunc == nil && !options.Filter.Valid() {
		return pathErr(ErrInvalidFilter, *p)
	}

	return nil
}

/*
matchGlobEntry reports whether entry, found while globbing root, passes the
filter of options and matches pattern. It returns [SkipDir] alongside the result
when nothing below entry can match, because entry lies at maxDepth.
*/
func matchGlobEntry(
	root, entry *Path, dirEntry fs.DirEntry, pattern string, options GlobOptions, maxDepth int,
) (bool, error) {
	included, err := globIncludes(entry, options)
	if err != nil {
		return false, err
	}

	// The entry path is joined with the original path and any subdirectories.
	// For pattern matching, we don't want the original path included, because
	// the pattern is matched starting from the original path.
	entryForMatching, err := entry.RelativeTo(root)
	if err != nil {
		return false, err
	}

	// Test if the current entry matches the given pattern. Also handles GlobOptions.CaseSensitivity option
	matchesPattern, err := entryForMatching.MatchesPatternE(pattern, options.CaseSensitivity)
	if err != nil {
		return false, err
	}

	matched := included && matchesPattern

	// Nothing below this directory can match. dirEntry does not follow
	// symlinks, so SkipDir is only returned where the walk would descend.
	depth := strings.Count(entryForMatching.path, canonicalPathSeparator) + 1
	if maxDepth > 0 && depth >= maxDepth && dirEntry.IsDir() {
		return matched, SkipDir
	}

	return matched, nil
}

/*
globIncludes reports whether entry passes the filter of options, which is
[GlobOptions.FilterFunc] if set, or [GlobOptions.Filter] otherwise. An unknown
filter returns [ErrInvalidFilter].
*/
func globIncludes(entry *Path, options GlobOptions) (bool, error) {
	if options.FilterFunc != nil {
		return options.FilterFunc(entry), nil
	}

	switch options.Filter {
	case GlobOptionFilterFiles:
		return entry.IsFile(), nil
	case GlobOptionFilterDirectories:
		return entry.IsDir(), nil
	case GlobOptionFilterAll:
		return true, nil
	default:
		return false, pathErr(ErrInvalidFilter, *entry)
	}
}
