package pathlib

import (
	"io/fs"
)

const (
	// posixDefaultFileMode is the default mode of a file outside Windows.
	posixDefaultFileMode fs.FileMode = 0644
	// posixDefaultDirMode is the default mode of a directory outside Windows.
	posixDefaultDirMode fs.FileMode = 0755
	// windowsWritableFileMode is the mode Windows applies to a writable file.
	windowsWritableFileMode fs.FileMode = 0666
	// windowsReadOnlyFileMode is the mode Windows applies to a read-only file.
	windowsReadOnlyFileMode fs.FileMode = 0444
	// windowsDirMode is the mode Windows applies to every directory.
	windowsDirMode fs.FileMode = 0777
)

/*
DefaultFileMode returns the default file permission mode.
On Unix this is 0644 (rw-r--r--). On Windows this is 0666 since Windows
does not support Unix-style permission granularity.
*/
func DefaultFileMode() fs.FileMode {
	return effectiveFileMode(posixDefaultFileMode)
}

/*
DefaultDirMode returns the default directory permission mode.
On Unix this is 0755 (rwxr-xr-x). On Windows this is 0777 since Windows
does not support Unix-style permission granularity for directories.
*/
func DefaultDirMode() fs.FileMode {
	return effectiveDirMode(posixDefaultDirMode)
}

// effectiveFileMode returns the permission mode the OS will actually apply to a file.
// On Windows, files are either read-write (0666) or read-only (0444).
func effectiveFileMode(mode fs.FileMode) fs.FileMode {
	if !runningOnWindows {
		return mode
	}
	if mode&0200 != 0 {
		return windowsWritableFileMode
	}
	return windowsReadOnlyFileMode
}

// effectiveDirMode returns the permission mode the OS will actually apply to a directory.
// On Windows, directories always have 0777.
func effectiveDirMode(mode fs.FileMode) fs.FileMode {
	if !runningOnWindows {
		return mode
	}
	return windowsDirMode
}

/*
PermissionBits are the mode bits accepted wherever a permission is passed. The Unix
permission bits (fs.ModePerm) plus fs.ModeSetuid, fs.ModeSetgid and fs.ModeSticky.
Any other bit returns ErrPermissionRange. This includes file type bits such as
fs.ModeDir and the Unix octal notation of the special bits (e.g. 0o4755), which the
os package would silently drop.

Whether the setuid and setgid bits survive the creation of a file or directory
depends on the operating system, e.g. macOS drops them. SetPermission sets them
reliably. On Windows, the special bits have no effect.
*/
const PermissionBits = fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky

// checkPermission returns ErrPermissionRange if mode has bits outside PermissionBits.
func checkPermission(mode fs.FileMode, path *Path) error {
	if mode&^PermissionBits != 0 {
		return permRangeErr(mode, *path)
	}

	return nil
}

/*
FileOptions contains options for file and directory creation and deletion operations.
*/
type FileOptions struct {
	// ExistOk specifies whether it's acceptable if the file/directory already exists
	ExistOk bool

	// Mode specifies the file/directory permission mode. It may only contain
	// PermissionBits.
	Mode fs.FileMode
}

/*
DefaultFileOptions returns the default options for file operations.
*/
func DefaultFileOptions() FileOptions {
	return FileOptions{
		ExistOk: false,
		Mode:    DefaultFileMode(),
	}
}

/*
DirOptions contains options for file and directory creation and deletion operations.
*/
type DirOptions struct {
	// ExistOk specifies whether it's acceptable if the file/directory already exists
	ExistOk bool

	// Mode specifies the file/directory permission mode. It may only contain
	// PermissionBits.
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
		Mode:      DefaultDirMode(),
		CreateAll: false,
	}
}

/*
FilterFunc reports whether the entry at path is included in a glob result.
*/
type FilterFunc func(path *Path) bool

/*
GlobOptions contains options for file globbing.
*/
type GlobOptions struct {
	// CaseSensitivity defines whether pattern matching should be case-sensitive or not.
	// Defaults to CaseSensitive.
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
}

/*
DefaultGlobOptions returns the default options for directory operations.
*/
func DefaultGlobOptions() GlobOptions {
	return GlobOptions{
		Limit:           0,
		CaseSensitivity: CaseSensitive,
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
