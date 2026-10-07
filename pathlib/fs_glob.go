// fs_glob.go holds globbing and listing, which find the entries below a
// directory by pattern.

package pathlib

import (
	"cmp"
	"context"
	"io/fs"
	"slices"
	"strings"
)

// FilterFunc reports whether a glob includes the entry at path.
type FilterFunc func(path *Path) bool

// GlobOptions configures a glob. The zero value matches case-sensitively,
// includes files and directories, stops at a directory that cannot be read,
// and has no limit.
type GlobOptions struct {
	// CaseSensitivity selects how the pattern treats casing. The zero value
	// is CaseSensitive.
	CaseSensitivity CompareOption

	// Filter selects the types of the included entries. The zero value is
	// GlobOptionFilterAll.
	Filter GlobOptionFilterType

	// FilterFunc decides which entries are included, in place of Filter. Nil
	// leaves the decision to Filter.
	FilterFunc FilterFunc

	// SkipOnDirError skips a directory that cannot be opened or read. If it
	// is false, the glob stops and returns the error.
	SkipOnDirError bool

	// Limit is the maximum number of returned entries. Zero or less means no
	// limit.
	Limit int
}

// ListOptions configures a listing. It embeds the options of the glob that
// finds the entries.
type ListOptions struct {
	// GlobOptions configures the glob that finds the entries.
	GlobOptions

	// Recursive includes the entries of subdirectories.
	Recursive bool
}

// DefaultGlobOptions returns the options [Path.Glob] uses, which equal the
// zero value of [GlobOptions].
func DefaultGlobOptions() GlobOptions {
	return GlobOptions{
		Limit:           0,
		CaseSensitivity: CaseSensitive,
		Filter:          GlobOptionFilterAll,
		FilterFunc:      nil,
		SkipOnDirError:  false,
	}
}

// DefaultListOptions returns the options of a listing of the direct entries of
// a directory with [DefaultGlobOptions].
func DefaultListOptions() ListOptions {
	return ListOptions{GlobOptions: DefaultGlobOptions()}
}

// Glob returns the entries below this Path that match pattern, as
// [Path.GlobWithOptions] does with [DefaultGlobOptions].
func (p *Path) Glob(pattern string) ([]*Path, error) {
	return p.GlobContext(context.Background(), pattern)
}

// GlobContext returns the entries below this Path that match pattern, as
// [Path.GlobWithOptionsContext] does with DefaultGlobOptions.
func (p *Path) GlobContext(ctx context.Context, pattern string) ([]*Path, error) {
	return p.GlobWithOptionsContext(ctx, pattern, DefaultGlobOptions())
}

// GlobWithOptions returns the entries below this Path whose path relative to
// this Path matches pattern, with the syntax of [Path.MatchesPatternE]. This
// Path must be a directory. An error during the walk returns the entries
// matched until then.
//
// The tree is walked as [Path.WalkR] walks it, so a symlink to a directory
// inside the tree is an entry, and its contents are not visited.
// GlobOptions.Filter judges a symlink by its target. A symlink to a directory
// counts as a directory, and a broken symlink as neither a file nor a
// directory.
//
// Directories below the deepest level pattern can match are not read. "*" and
// "?" never match "/", so a pattern without "**" and character classes only
// matches paths with as many names as the pattern. "*" reads this directory
// alone, and "src/*.go" reads this directory and its direct subdirectories.
//
// A missing path returns [ErrNotExist], and an existing non-directory returns
// [ErrNotDir]. An empty pattern returns [ErrEmptyPattern], a malformed pattern
// returns [ErrBadPattern], and an unknown GlobOptions.Filter returns
// [ErrInvalidFilter]. Denied access returns [ErrPermissionDenied], also for a
// directory inside the tree. Any other failure to check the path returns
// [ErrStat], and any other failure to open or read a directory returns
// [ErrOpen] or [ErrReadDir]. GlobOptions.SkipOnDirError skips a directory that
// cannot be opened or read.
func (p *Path) GlobWithOptions(pattern string, options GlobOptions) ([]*Path, error) {
	return p.GlobWithOptionsContext(context.Background(), pattern, options)
}

// GlobWithOptionsContext returns the entries below this Path that match
// pattern, as [Path.GlobWithOptions] does. A done ctx stops the glob, which
// returns the entries matched until then and ctx.Err() as the cause of
// [ErrWalk].
func (p *Path) GlobWithOptionsContext(ctx context.Context, pattern string, options GlobOptions) ([]*Path, error) {
	err := validateGlob(p, pattern, options)
	if err != nil {
		return nil, err
	}

	var entries []*Path

	limit := max(0, options.Limit)
	limitExists := limit != 0

	// Directories at the deepest level the pattern can match are not entered.
	// 0 means the pattern can match at any depth.
	maxDepth := patternMaxDepth(pattern)

	err = p.walkRContext(ctx, func(entry *Path, dirEntry fs.DirEntry, localDirError error) error {
		if localDirError != nil {
			if options.SkipOnDirError {
				return SkipDir
			}

			return localDirError
		}

		matched, matchErr := matchGlobEntry(p, entry, dirEntry, pattern, options, maxDepth)
		if matched {
			entries = append(entries, entry)
		}

		if limitExists && len(entries) >= limit {
			return SkipAll
		}

		return matchErr
	})

	return entries, err
}

// HasGlobMatchE reports whether an entry below this Path matches pattern. It
// globs with GlobOptions{Limit: 1}, so the match is case-sensitive, and the
// errors of [Path.GlobWithOptions] apply.
func (p *Path) HasGlobMatchE(pattern string) (bool, error) {
	matches, err := p.GlobWithOptions(pattern, GlobOptions{Limit: 1})
	if err != nil {
		return false, err
	}

	return len(matches) != 0, nil
}

// HasGlobMatch reports whether an entry below this Path matches pattern, as
// [Path.HasGlobMatchE] does. It returns false on an error.
func (p *Path) HasGlobMatch(pattern string) bool {
	contains, err := p.HasGlobMatchE(pattern)
	if err != nil {
		return false
	}

	return contains
}

// List returns the entries of this directory, sorted by their Posix form. It
// globs with options and the pattern "*", or "**" if ListOptions.Recursive is
// set. [Path.Walk] and [Path.WalkR] give finer control.
//
// An error returns no entries. The errors of [Path.GlobWithOptions] apply.
func (p *Path) List(options ListOptions) ([]*Path, error) {
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

// ListFiles returns the files of this directory, sorted by their Posix form,
// as [Path.List] does. If recursive is set, the files of subdirectories are
// included.
func (p *Path) ListFiles(recursive bool) ([]*Path, error) {
	options := DefaultListOptions()
	options.Filter = GlobOptionFilterFiles
	options.Recursive = recursive

	return p.List(options)
}

// ListDirs returns the directories of this directory, sorted by their Posix
// form, as [Path.List] does. If recursive is set, the directories of
// subdirectories are included.
func (p *Path) ListDirs(recursive bool) ([]*Path, error) {
	options := DefaultListOptions()
	options.Filter = GlobOptionFilterDirectories
	options.Recursive = recursive

	return p.List(options)
}

// patternMaxDepth returns the number of names the paths matched by pattern can
// have at most, or 0 if there is no limit. "*" and "?" never match "/", so
// only "**" and character classes, which can match "/", match across
// separators.
func patternMaxDepth(pattern string) int {
	if strings.Contains(pattern, "**") || strings.Contains(pattern, "[") {
		return 0
	}

	return strings.Count(pattern, canonicalPathSeparator) + 1
}

// sortSliceOfPaths sorts paths in place by their Posix form and returns them.
func sortSliceOfPaths(paths []*Path) []*Path {
	slices.SortFunc(paths, func(a, b *Path) int {
		return cmp.Compare(a.ToPosix(), b.ToPosix())
	})

	return paths
}

// validateGlob checks the arguments of a glob below p. It returns the errors
// of [requireDir], [ErrEmptyPattern] for an empty pattern, [ErrBadPattern] for
// a malformed pattern, and [ErrInvalidFilter] for an unknown filter.
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

// matchGlobEntry reports whether entry, found while globbing root, passes the
// filter of options and matches pattern. It returns [SkipDir] next to the
// result when nothing below entry can match, because entry lies at maxDepth.
func matchGlobEntry(
	root, entry *Path, dirEntry fs.DirEntry, pattern string, options GlobOptions, maxDepth int,
) (bool, error) {
	included, err := globIncludes(entry, options)
	if err != nil {
		return false, err
	}

	// The pattern matches the path of the entry relative to the root.
	entryForMatching, err := entry.RelativeTo(root)
	if err != nil {
		return false, err
	}

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

// globIncludes reports whether entry passes the filter of options, which is
// GlobOptions.FilterFunc if set, or GlobOptions.Filter otherwise. An unknown
// filter returns [ErrInvalidFilter].
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
