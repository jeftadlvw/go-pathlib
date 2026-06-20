package pathlib

import (
	"io/fs"
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
