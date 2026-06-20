package pathlib

import (
	"cmp"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// Error sentinels raised by pathlib_fs.go (filesystem operations). They are
// matched with errors.Is and exposed through PathlibError.Kind.
var (
	// ErrNotExist is the broad group for "a required path does not exist".
	// Match it with errors.Is to catch every not-exist case below.
	ErrNotExist = errors.New("path does not exist")

	// ErrParentNotExist is raised when a required parent directory is missing.
	// It is a member of the ErrNotExist group.
	ErrParentNotExist = subKind(ErrNotExist, "parent directory does not exist")

	// ErrExist is the broad group for "a path already exists and would be
	// overwritten". Match it with errors.Is to catch every exist case below.
	ErrExist = errors.New("path already exists")

	// ErrFileExist is raised when the conflicting path is a file.
	// It is a member of the ErrExist group.
	ErrFileExist = subKind(ErrExist, "file already exists")

	// ErrDirExist is raised when the conflicting path is a directory.
	// It is a member of the ErrExist group.
	ErrDirExist = subKind(ErrExist, "directory already exists")

	// ErrNotFile is returned when a path is expected to be a regular file but is not.
	ErrNotFile = errors.New("path is not a file")

	// ErrNotDir is returned when a path is expected to be a directory but is not.
	ErrNotDir = errors.New("path is not a directory")

	// ErrNotSymlink is returned when a path is expected to be a symlink but is not.
	ErrNotSymlink = errors.New("path is not a symlink")

	// ErrNotEmptyDir is returned when a directory is expected to be empty but is not.
	ErrNotEmptyDir = errors.New("directory is not empty")

	// ErrCopyType is returned when a path's file type cannot be copied.
	ErrCopyType = errors.New("no copy operation defined for this file type")

	// ErrTypeMismatch is returned when the source and destination of a copy or
	// move have incompatible file types.
	ErrTypeMismatch = errors.New("source and destination types are incompatible")

	// ErrOpen is returned when a path could not be opened; wraps the os cause.
	ErrOpen = errors.New("could not open path")

	// ErrReadDir is returned when a directory entry could not be read; wraps the os cause.
	ErrReadDir = errors.New("could not read directory entry")

	// ErrWalk is returned when walking a path fails; wraps the underlying cause.
	ErrWalk = errors.New("error walking path")

	// errInvalidFilterOption is an internal guard for an unknown glob filter option.
	errInvalidFilterOption = errors.New("invalid filter option")
)

/*
DefaultFileMode is the default file permission mode.
On Unix this is 0644 (rw-r--r--). On Windows this is 0666 since Windows
does not support Unix-style permission granularity.
*/
var DefaultFileMode = effectiveFileMode(0644)

/*
DefaultDirMode is the default directory permission mode.
On Unix this is 0755 (rwxr-xr-x). On Windows this is 0777 since Windows
does not support Unix-style permission granularity for directories.
*/
var DefaultDirMode = effectiveDirMode(0755)

// effectiveFileMode returns the permission mode the OS will actually apply to a file.
// On Windows, files are either read-write (0666) or read-only (0444).
func effectiveFileMode(mode fs.FileMode) fs.FileMode {
	if !runningOnWindows {
		return mode
	}
	if mode&0200 != 0 {
		return 0666
	}
	return 0444
}

// effectiveDirMode returns the permission mode the OS will actually apply to a directory.
// On Windows, directories always have 0777.
func effectiveDirMode(mode fs.FileMode) fs.FileMode {
	if !runningOnWindows {
		return mode
	}
	return 0777
}

/*
FileOptions contains options for file and directory creation and deletion operations
*/
type FileOptions struct {
	// ExistOk specifies whether it's acceptable if the file/directory already exists
	ExistOk bool

	// Mode specifies the file/directory permission mode
	Mode fs.FileMode
}

/*
DefaultFileOptions returns the default options for file operations.
*/
func DefaultFileOptions() FileOptions {
	return FileOptions{
		ExistOk: false,
		Mode:    DefaultFileMode,
	}
}

/*
DirOptions contains options for file and directory creation and deletion operations
*/
type DirOptions struct {
	// ExistOk specifies whether it's acceptable if the file/directory already exists
	ExistOk bool

	// Mode specifies the file/directory permission mode
	Mode fs.FileMode

	// CreateAll creates all missing directories.
	CreateAll bool
}

/*
DefaultDirOptions returns the default options for directory operations.
*/
func DefaultDirOptions() DirOptions {
	return DirOptions{
		ExistOk:   false,
		Mode:      DefaultDirMode,
		CreateAll: false,
	}
}

/*
SkipDir and SkipAll are control signals returned from a WalkFunc or WalkRFunc.

They are aliases of io/fs.SkipDir and io/fs.SkipAll, so they interoperate with
the standard library's walk sentinels.
*/
var (

	/*
	 	SkipDir skips the remaining entries of the current directory. If it's returned
	  	for a directory entry, it skips that directory's contents. If it's returned for a file, it
	   	skips the remaining entries of the containing directory.

	    Aliases io/fs.SkipDir.
	*/
	SkipDir = fs.SkipDir

	/*
	 	SkipAll stops the entire walk. Any other non-nil error aborts the walk and is returned
	  	to the caller.

	   Aliases io/fs.SkipAll.
	*/
	SkipAll = fs.SkipAll
)

/*
WalkFunc is called by Walk for every entry, receiving a Path joined with the
walked directory.

Return SkipDir or SkipAll to stop walking. Return any other non-nil error to abort
and have Walk return it. The error will be wrapped as ErrWalk.
*/
type WalkFunc func(p *Path) error

/*
WalkRFunc is called by WalkR for every entry.

Return SkipDir to skip the rest of the current directory or a directory's content.
Return SkipAll to stop the entire walk, or return any other non-nil error to abort
and have WalkR return it.

If the current directory could not be opened or read, localDirError is non-nil.
Users may inspect it and act upon it by e.g. ignoring it (return nil), bubbling
it (return error) or returning a different error instead (e.g. SkipDir).
*/
type WalkRFunc func(p *Path, localDirError error) error

type FilterFunc func(path *Path) bool

type GlobOptionFilterType uint

const (
	GlobOptionFilterAll         GlobOptionFilterType = 0
	GlobOptionFilterFiles       GlobOptionFilterType = 1
	GlobOptionFilterDirectories GlobOptionFilterType = 2
)

/*
GlobOptions contains options for file globbing.
*/
type GlobOptions struct {
	// CaseSensitivity defines whether pattern matching should be case-sensitive or not.
	// Defaults to CaseInsensitive.
	CaseSensitivity CompareOption

	// Filter defines which type of entry to glob:
	//  - GlobOptionFilterAll allows both files and directories.
	//  - GlobOptionFilterFiles only allows files.
	//  - GlobOptionFilterDirectories only allows directories.
	// Defaults to GlobOptionFilterAll.
	Filter GlobOptionFilterType

	// FilterFunc is a custom filter function used instead of Filter if non-nil.
	// Defaults to nil.
	FilterFunc FilterFunc

	// SkipOnDirError defines whether to skip globbing a directory if reading that directory
	// caused an error. If set to false, an error is returned and globbing stops. If set to true,
	// globbing continues silently. Defaults to false.
	SkipOnDirError bool

	// Limit sets a limit. Globbing will return at most this number of entries.
	// If <= 0, no limit is used. Defaults to 0.
	Limit int

	// skipPatternMatchCheck is a cheeky flag to disable the pattern matching inside the globbing function.
	// This behavior totally defeats the reason why this struct exists, but is useful when the same powerful globbing
	// function is used for other things and wants to eliminate the extra pattern matching overhead.
	//
	// ONLY FOR INTERNAL USE!
	skipPatternMatchCheck bool
}

/*
DefaultGlobOptions returns the default options for directory operations.
*/
func DefaultGlobOptions() GlobOptions {
	return GlobOptions{
		Limit:           0,
		CaseSensitivity: CaseInsensitive,
		Filter:          GlobOptionFilterAll,
		FilterFunc:      nil,
		SkipOnDirError:  false,
	}
}

/*
ListOptions is a type derivative for GlobOptions.
*/
type ListOptions struct {
	GlobOptions
	Recursive bool
}

/*
DefaultListOptions returns the same as DefaultGlobOptions, but type cast to ListOptions.
*/
func DefaultListOptions() ListOptions {
	return ListOptions{GlobOptions: DefaultGlobOptions()}
}

/*
IsFile returns whether this Path is an existing file.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) IsFile() bool {
	return pathCheck(p) == pathCheckFile
}

/*
IsDir returns whether this Path is an existing directory.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) IsDir() bool {
	return pathCheck(p) == pathCheckDir
}

/*
IsEmptyDir returns whether this Path is an empty existing directory.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) IsEmptyDir() bool {
	return p.IsDir() && !p.HasGlobMatch("*")
}

/*
Exists returns whether this Path exists.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) Exists() bool {
	return pathCheck(p) != pathCheckNoExistOrUnreadable
}

/*
Resolve resolves all symbolic links and ensures an absolute path representation.

This function uses filepath.EvalSymlinks and MakeAbsolute.
*/
func (p *Path) Resolve() (*Path, error) {
	if !p.Exists() {
		return nil, pathErr(ErrNotExist, *p)
	}

	ep, err := filepath.EvalSymlinks(p.String())
	if err != nil {
		return nil, err
	}

	return NewPath(ep).MakeAbsolute()
}

/*
Walk walks this directory and calls walkFunc for every entry (files, directories, etc.).
This path must be a directory. Symlinks are followed.

walkFunc receives a path joined with this Path.
*/
func (p *Path) Walk(walkFunc WalkFunc) error {
	return p.WalkContext(context.Background(), walkFunc)
}

/*
WalkContext is Walk with support for cancellation through ctx. The walk stops and
returns ctx.Err() as soon as ctx is done, with cancellation checked before each entry.
*/
func (p *Path) WalkContext(ctx context.Context, walkFunc WalkFunc) error {
	if !p.IsDir() {
		return pathErr(ErrNotDir, *p)
	}

	file, err := os.Open(p.String())
	if err != nil {
		return wrapErr(ErrOpen, err, *p)
	}

	defer file.Close()

	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		// os.File.ReadDir returns at most n next entries for every next call.
		// So we call it with n = 1 to yield and process the directory entry by entry
		dirNames, err := file.ReadDir(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return wrapErr(ErrReadDir, err, *p)
			}
		}

		dirName := dirNames[0]
		dirNamePath := p.JoinStrings(dirName.Name())

		walkErr := walkFunc(dirNamePath)
		if walkErr != nil {
			// Walk is not recursive, so SkipDir and SkipAll both simply stop it.
			if errors.Is(walkErr, SkipAll) || errors.Is(walkErr, SkipDir) {
				break
			}
			return wrapErr(ErrWalk, walkErr, *dirNamePath)
		}
	}

	return nil
}

/*
WalkR walks this directory recursively and calls walkFunc for every entry.
This path must be a directory. Symlinks are followed.

walkFunc receives paths that are already joined with this Path.

walkFunc takes the current entry, a function to abort walking the entire tree and
a function to abort walking the current branch.
*/
func (p *Path) WalkR(walkFunc WalkRFunc) error {
	return p.WalkRContext(context.Background(), walkFunc)
}

/*
WalkRContext is WalkR with support for cancellation through ctx. The walk stops
and returns ctx.Err() as soon as ctx is done, with cancellation checked before
each directory and each entry.
*/
func (p *Path) WalkRContext(ctx context.Context, walkFunc WalkRFunc) error {
	err := walkR(ctx, p, nil, walkFunc)

	// SkipAll is a successful early termination, not a failure.
	if errors.Is(err, SkipAll) {
		return nil
	}

	return err
}

/*
walkR calls walkFunc recursively for all path entries in a given root Path.

currentDir must be nil on the initial function call. SkipDir is consumed at the
level that raises it, so it never escapes the current function call frame. SkipAll
is propagated up, so the whole walk unwinds.
*/
func walkR(ctx context.Context, initialDir *Path, currentDir *Path, walkFunc WalkRFunc) error {
	// Set currentDir to initialDir if currentDir is nil (initial call)
	if currentDir == nil {
		currentDir = initialDir
	}

	err := ctx.Err()
	if err != nil {
		return err
	}

	if !currentDir.IsDir() {
		return pathErr(ErrNotDir, *currentDir)
	}

	// Get file descriptor
	file, err := os.Open(currentDir.String())
	if err != nil {
		// Give walkFunc a chance to inspect and ignore the directory error. A nil
		// or SkipDir result skips this directory; anything else aborts.
		handled := walkFunc(currentDir, wrapErr(ErrOpen, err, *currentDir))
		if handled == nil || errors.Is(handled, SkipDir) {
			return nil
		}
		return handled
	}

	defer file.Close()

	// Call walkFunc for the directory itself (except the initial root).
	if !currentDir.Equals(initialDir, CaseInsensitive) {
		walkErr := walkFunc(currentDir, nil)
		if walkErr != nil {
			if errors.Is(walkErr, SkipDir) {
				return nil
			}
			return walkErr
		}
	}

	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		// os.File.ReadDir returns at most n next entries for every next call.
		// So we call it with n = 1 to yield and process the directory entry by entry
		dirNames, err := file.ReadDir(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			handled := walkFunc(currentDir, wrapErr(ErrReadDir, err, *currentDir))
			if handled == nil || errors.Is(handled, SkipDir) {
				return nil
			}
			return handled
		}

		dirName := dirNames[0]
		dirNamePath := currentDir.JoinStrings(dirName.Name())

		// Call walkFunc for non-directory entries.
		if !dirName.IsDir() {
			walkErr := walkFunc(dirNamePath, nil)
			if walkErr != nil {
				if errors.Is(walkErr, SkipDir) {
					return nil
				}
				return walkErr
			}
		}

		// Recurse into subdirectories.
		if dirName.IsDir() {
			subErr := walkR(ctx, initialDir, dirNamePath, walkFunc)
			if subErr != nil {
				return subErr
			}
		}
	}

	return nil
}

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

/*
Stat returns file info for this Path.

This function uses os.Stat.
*/
func (p *Path) Stat() (os.FileInfo, error) {
	return os.Stat(p.String())
}

/*
Lstat returns file info for this Path, not following symbolic links.

This function uses os.Lstat.
*/
func (p *Path) Lstat() (os.FileInfo, error) {
	return os.Lstat(p.String())
}

/*
IsSymlink returns whether this Path is a symbolic link.
*/
func (p *Path) IsSymlink() bool {
	// Symlinks must be checked with Stat instead of Lstat
	return checkFileMode(p, os.ModeSymlink)
}

/*
IsBlockDevice returns whether this Path is a block device.
*/
func (p *Path) IsBlockDevice() bool {
	return checkFileMode(p, os.ModeDevice) && !checkFileMode(p, os.ModeCharDevice)
}

/*
IsCharDevice returns whether this Path is a character device.
*/
func (p *Path) IsCharDevice() bool {
	return checkFileMode(p, os.ModeDevice) && checkFileMode(p, os.ModeCharDevice)
}

/*
IsFiFoPipe returns whether this Path is a FIFO/pipe.
*/
func (p *Path) IsFiFoPipe() bool {
	return checkFileMode(p, os.ModeNamedPipe)
}

/*
IsSocket returns whether this Path is a socket.
*/
func (p *Path) IsSocket() bool {
	return checkFileMode(p, os.ModeSocket)
}

/*
EqualsFs returns whether this Path and another Path point to the same file system inode.
This comparison is performed using file stats, not string comparison.

Symlinks are resolved.
*/
func (p *Path) EqualsFs(other *Path) bool {
	// No need to call Exists for both paths, as Stat() will return an error
	// if the path does not exist.

	stat1, err := p.Stat()
	if err != nil {
		return false
	}

	stat2, err := other.Stat()
	if err != nil {
		return false
	}

	return os.SameFile(stat1, stat2)
}

/*
IsOnCaseSensitiveFs checks if the filesystem at this Path is case-sensitive.

It first tries to check for the given path, toggling the casing of the first encountered letter in the path's base,
checking if the file exists and if both file descriptors point to the same file.

If no letter exists within the path's base, a temporary file is created in the same directory for which
the upper procedure is repeated.

If both attempts result in an invalid state, false is returned.
*/
func (p *Path) IsOnCaseSensitiveFs() bool {
	if !p.Exists() {
		return false
	}

	// Check for case sensitivity by inverting a letter in a copy of this Path's base
	// and check if both paths are EqualsFs()
	pathBase := p.Base()

	firstLetterIndex := findFirstLetterIndex(pathBase)
	hasLetter := firstLetterIndex != -1

	if hasLetter {
		modifiedPathBase := switchCaseAtIndex(pathBase, firstLetterIndex)
		equalOnFilesystem := p.EqualsFs(p.Parent().JoinStrings(modifiedPathBase))

		// If both paths are equal on the filesystem, the unmodified and modified paths both point to the same file.
		// This means the filesystem is NOT case-sensitive.
		return !equalOnFilesystem
	}

	// If the base does not include a letter, create a temporary file and repeat the check.

	// Get a directory to test in
	var testDir *Path
	if p.IsDir() {
		testDir = p
	} else {
		testDir = p.Parent()
	}

	// os.CreateTemp ensures a unique filename
	// The passed pattern is to ensure a single letter at a defined index
	tempFile, err := os.CreateTemp(testDir.String(), "a-*")
	if err != nil {
		return false
	}

	// Get file name, close and defer removal
	tempFilePathStr := tempFile.Name()
	_ = tempFile.Close()
	defer os.Remove(tempFilePathStr)

	tempFilePath := NewPath(tempFilePathStr)

	modifiedTempFilePathBase := switchCaseAtIndex(tempFilePath.Base(), firstLetterIndex)
	return !tempFilePath.EqualsFs(tempFilePath.Parent().JoinStrings(modifiedTempFilePathBase))
}

// findFirstLetterIndex returns the index of the first letter within a string.
// It correctly identifies letters from various scripts using unicode.IsLetter.
// If no letter is found, it returns -1.
func findFirstLetterIndex(s string) int {
	// strings.IndexFunc finds the first index where the provided function returns true.
	// unicode.IsLetter is the function that checks if a rune is a letter
	// in any language or script.
	return strings.IndexFunc(s, unicode.IsLetter)
}

// switchCaseAtIndex switches the case of the letter at the specified index in the string.
// If the index is out of bounds or the character at the index is not a letter,
// the original string is returned.
func switchCaseAtIndex(s string, index int) string {
	// Convert the string to a slice of runes. This is important because Go
	// strings are UTF-8 encoded, and a character (rune) might take up
	// more than one byte. Working with runes ensures we handle characters correctly.
	runes := []rune(s)

	// Check if the index is valid (within the bounds of the rune slice)
	if index < 0 || index >= len(runes) {
		// Index out of bounds, return the original string unchanged
		return s
	}

	// Get the rune at the specified index
	r := runes[index]

	// Check if the character is a letter
	if !unicode.IsLetter(r) {
		// Not a letter, return the original string unchanged
		return s
	}

	// Switch the case of the letter
	if unicode.IsUpper(r) {
		// If it's an uppercase letter, convert it to lowercase
		runes[index] = unicode.ToLower(r)
	} else if unicode.IsLower(r) {
		// If it's a lowercase letter, convert it to uppercase
		runes[index] = unicode.ToUpper(r)
	} else {
		// It's a letter, but neither upper nor lower (e.g., titlecase).
		// In this function's scope, we only handle upper/lower switching.
		// Return the original string as no standard case switch occurred.
		return s
	}

	// Convert the modified slice of runes back into a string
	return string(runes)
}

/*
SymlinkTo creates a symbolic link at the source path pointing to the target path.

This path must exist.

This function uses CreateSymlink.
*/
func (p *Path) SymlinkTo(linkPath *Path) error {
	if !p.Exists() {
		return pathErr(ErrNotExist, *p)
	}

	return CreateSymlink(p, linkPath)
}

/*
ReadSymlinkTarget reads the target path for this Path.

This Path must be a symlink.
*/
func (p *Path) ReadSymlinkTarget() (*Path, error) {
	if !p.IsSymlink() {
		return nil, pathErr(ErrNotSymlink, *p)
	}

	target, err := os.Readlink(p.String())
	if err != nil {
		return nil, err
	}

	return NewPath(target), nil
}

/*
SetPermission sets the permission mode for the specified path.
*/
func SetPermission(path *Path, mode fs.FileMode) error {
	return os.Chmod(path.String(), mode)
}

/*
CreateFile creates the file at the defined path with mode 0644.
If the file already exists, it will be truncated.

Parent directories must exist.
*/
func CreateFile(path *Path) error {
	_, err := CreateFileWithOptions(path, DefaultFileOptions())
	return err
}

/*
CreateFileWithOptions creates the file at the defined path with given options.

If FileOptions.ExistOk is true and the file already exists, no action is taken and false is returned.
Parent directories must exist.

FileOptions.Mode can never be set explicitly to 0000. This is not allowed by the operating system
and defaults to DefaultFileMode.

Returns true if a new file was created, false otherwise.
*/
func CreateFileWithOptions(path *Path, options FileOptions) (bool, error) {
	if path.Exists() {
		if !path.IsFile() {
			return false, pathErr(ErrNotFile, *path)
		}
		if options.ExistOk {
			return false, nil
		}
		return false, pathErr(ErrFileExist, *path)
	}

	if options.Mode == 0 {
		options.Mode = DefaultFileMode
	}

	file, err := os.OpenFile(path.String(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, options.Mode)
	if err != nil {
		return false, err
	}

	err = file.Close()

	return true, err
}

/*
MkDir creates the directory at the defined path with mode 0755.
Parent directories must exist.
*/
func MkDir(path *Path) error {
	_, err := MkDirWithOptions(path, DefaultDirOptions())
	return err
}

/*
MkDirWithOptions creates the directory at the defined path with given options.
If ExistOk is true and the directory already exists, no action is taken.

DirOptions.Mode can never be set explicitly to 0000. This is not allowed by the operating system
and defaults to DefaultDirMode.

Returns true if a new directory was created, false otherwise.
*/
func MkDirWithOptions(path *Path, options DirOptions) (bool, error) {
	if path.Exists() {
		if !path.IsDir() {
			return false, pathErr(ErrNotDir, *path)
		}
		if options.ExistOk {
			return false, nil
		}
		return false, pathErr(ErrDirExist, *path)
	}

	if options.Mode == 0 {
		options.Mode = DefaultDirMode
	}

	var err error
	if options.CreateAll {
		err = os.MkdirAll(path.String(), options.Mode)
	} else {
		err = os.Mkdir(path.String(), options.Mode)
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

/*
CreateSymlink creates a symlink at the symlinkPath that points to symlinkTarget.

symlinkTarget may be relative or absolute.

symlinkPath may not exist, but parent directory should.
*/
func CreateSymlink(symlinkTarget, symlinkPath *Path) error {
	if symlinkPath.Exists() {
		return pathErr(ErrExist, *symlinkPath)
	}

	if !symlinkPath.Parent().Exists() {
		return pathErr(ErrParentNotExist, *symlinkPath)
	}

	return os.Symlink(symlinkTarget.String(), symlinkPath.String())
}

/*
Copy copies the source path to the destination path.

If the source path is a directory, the whole directory tree is copied. All other files
and file types are copied as-is.

Copying a directory requires the target directory to be empty.

The source path must exist. Destination parent directories must exist.
*/
func Copy(src *Path, destination *Path) error {
	if !src.Exists() {
		return pathErr(ErrNotExist, *src)
	}

	// Ensure parent directories exist
	if !destination.Parent().Exists() {
		return pathErr(ErrParentNotExist, *destination)
	}

	if src.IsSymlink() { // A symlink is a special file, thus checking that first
		return copySymlink(src, destination)
	} else if src.IsFile() {
		return copyFile(src, destination)
	} else if src.IsDir() {
		return copyDir(src, destination)
	} else {
		return pathErr(ErrCopyType, *src)
	}
}

/*
copyFile copies a file.

Assumes source and destination parent directories exist.
*/
func copyFile(source *Path, destination *Path) error {
	// Get source file info for permissions
	srcInfo, err := source.Stat()
	if err != nil {
		return err
	}

	// Ensure the target file does not exist
	if destination.Exists() {
		if destination.IsFile() {
			return pathErr(ErrFileExist, *destination)
		} else if destination.IsDir() {
			return pathErr(ErrTypeMismatch, *source, *destination)
		} else {
			return pathErr(ErrExist, *destination)
		}
	}

	// Open source file
	sourceFile, err := os.Open(source.String())
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.OpenFile(destination.String(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy contents
	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

/*
copySymlink copies a symlink.

Assumes source and destination parent directories exist.
*/
func copySymlink(source *Path, destination *Path) error {
	// All checks should be already done by called functions

	originalSymlinkTarget, err := source.ReadSymlinkTarget()
	if err != nil {
		return err
	}

	// The original target might be relative to source. If so, make it relative to destination.
	if originalSymlinkTarget.IsRelative() {
		originalSymlinkTarget, err = originalSymlinkTarget.AbsoluteFrom(source.Parent())
		if err != nil {
			return err
		}

		originalSymlinkTarget, err = originalSymlinkTarget.RelativeTo(destination.Parent())
		if err != nil {
			return err
		}
	}

	return CreateSymlink(originalSymlinkTarget, destination)
}

/*
copyDir copies a directory recursively.

Assumes source and destination parent directories exist.
*/
func copyDir(src *Path, dst *Path) error {
	// Get source file info for permissions
	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	if dst.Exists() {
		if dst.IsFile() {
			// Ensure the target directory is not a file
			return pathErr(ErrTypeMismatch, *src, *dst)

		} else if dst.IsDir() {
			// Ensure destination directory is empty
			file, openErr := os.Open(dst.String())
			if openErr != nil {
				return openErr
			}

			defer file.Close()

			entries, readDirErr := file.ReadDir(1)
			if readDirErr != nil && readDirErr != io.EOF {
				return readDirErr
			}

			if len(entries) != 0 {
				return pathErr(ErrNotEmptyDir, *dst)
			}
		} else {
			return pathErr(ErrExist, *dst)
		}

	} else {
		// Create the destination directory if it doesn't exist
		if err := os.Mkdir(dst.String(), srcInfo.Mode()); err != nil {
			return err
		}
	}

	// Read directory entries
	entries, err := os.ReadDir(src.String())
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcEntry := src.JoinStrings(entry.Name())
		dstEntry := dst.JoinStrings(entry.Name())

		if err := Copy(srcEntry, dstEntry); err != nil {
			return err
		}
	}

	return nil
}

/*
Move moves the file or directory at the source path to the destination path.
If the destination is on the same filesystem, this is equivalent to a rename operation.

Fails if destination already exists, except if the source path is a directory, and the target path
is an empty directory.

Destination parent directories must exist.
*/
func Move(src *Path, dst *Path) error {
	if !src.Exists() {
		return pathErr(ErrNotExist, *src)
	}

	// Destination path may not exist, except if source is a directory
	// and destination is an empty directory too.
	if src.IsDir() && dst.IsEmptyDir() {
		// do nothing
	} else if dst.Exists() {
		return pathErr(ErrExist, *dst)
	}

	// Try renaming first (works if on same filesystem)
	err := os.Rename(src.String(), dst.String())
	if err == nil {
		return nil
	}

	// If rename fails, try copy and delete
	err = Copy(src, dst)
	if err != nil {
		return err
	}

	if src.IsDir() {
		return RemoveAll(src)
	} else {
		return Remove(src)
	}
}

/*
Rename renames the file at the source path to the destination path.
This is a convenience wrapper for Move.
*/
func Rename(src *Path, name string) error {
	return Move(src, src.Parent().JoinStrings(name))
}

/*
Remove removes the file at the specified path
or removes an empty directory.

Nothing happens if the given path does not exist.
*/
func Remove(path *Path) error {
	if !path.Exists() {
		return nil
	}

	return os.Remove(path.String())
}

/*
RemoveAll recursively removes the directory and all its entries at the specified path.

Nothing happens if the given path does not exist.
*/
func RemoveAll(path *Path) error {
	if !path.Exists() {
		return nil
	}

	if !path.IsDir() {
		return pathErr(ErrNotDir, *path)
	}

	return os.RemoveAll(path.String())
}

/*
pathCheck is a lower level Path existence checker.
It returns 0 if the path does not exist, 1 if it's a file and 2 if it's a directory.
*/
func pathCheck(p *Path) int {
	fileInfo, err := p.Stat()
	if fileInfo == nil || err != nil {
		return pathCheckNoExistOrUnreadable
	}

	if fileInfo.IsDir() {
		return pathCheckDir
	}

	return pathCheckFile
}

/*
checkFileMode is a helper function to check file mode bits on a passed path.

It uses Lstat to get the mode.
*/
func checkFileMode(p *Path, modeMask os.FileMode) bool {
	info, err := p.Lstat()
	if err != nil {
		return false
	}
	return info.Mode()&modeMask != 0
}
