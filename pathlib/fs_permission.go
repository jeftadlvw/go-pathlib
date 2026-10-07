// fs_permission.go holds file permissions: the default modes, the permission
// bits the library accepts, and their change.

package pathlib

import (
	"os"
)

// PermissionBits are the mode bits accepted wherever a permission is passed.
// They are the Unix permission bits [ModePerm] plus [ModeSetuid],
// [ModeSetgid], and [ModeSticky].
//
// Any other bit returns [ErrInvalidPermission]. This includes file type bits,
// such as [ModeDir], and the Unix octal notation of the special bits, such as
// 0o4755, which the os package silently drops.
//
// Whether the setuid and setgid bits survive the creation of a file or
// directory depends on the operating system. macOS drops them, for example.
// [SetPermission] sets them reliably. On Windows, the special bits have no
// effect.
const PermissionBits = ModePerm | ModeSetuid | ModeSetgid | ModeSticky

const (
	// posixDefaultFileMode is the default mode of a file outside Windows.
	posixDefaultFileMode FileMode = 0644

	// posixDefaultDirMode is the default mode of a directory outside Windows.
	posixDefaultDirMode FileMode = 0755

	// windowsWritableFileMode is the mode Windows applies to a writable file.
	windowsWritableFileMode FileMode = 0666

	// windowsReadOnlyFileMode is the mode Windows applies to a read-only file.
	windowsReadOnlyFileMode FileMode = 0444

	// windowsDirMode is the mode Windows applies to every directory.
	windowsDirMode FileMode = 0777
)

// DefaultFileMode returns the default permission of a file. It is 0644
// (rw-r--r--) on Unix and 0666 on Windows, which only knows writable and
// read-only files.
func DefaultFileMode() FileMode {
	return effectiveFileMode(posixDefaultFileMode)
}

// DefaultDirMode returns the default permission of a directory. It is 0755
// (rwxr-xr-x) on Unix and 0777 on Windows, which applies 0777 to every
// directory.
func DefaultDirMode() FileMode {
	return effectiveDirMode(posixDefaultDirMode)
}

// SetPermission sets the permission of path to mode. It wraps [os.Chmod] and
// follows symlinks. The setuid and setgid bits are set reliably.
//
// A mode with bits outside [PermissionBits] returns [ErrInvalidPermission]. A
// missing path returns [ErrNotExist], and denied access returns
// [ErrPermissionDenied]. Any other failure returns [ErrSetPermission].
func SetPermission(path *Path, mode FileMode) error {
	err := checkPermission(mode, path)
	if err != nil {
		return err
	}

	err = os.Chmod(path.String(), mode)
	if err != nil {
		return osErr(ErrSetPermission, err, *path)
	}

	return nil
}

// effectiveFileMode returns the permission the operating system applies to a
// file created with mode. Windows knows writable (0666) and read-only (0444)
// files alone.
func effectiveFileMode(mode FileMode) FileMode {
	if !runningOnWindows {
		return mode
	}
	if mode&0200 != 0 {
		return windowsWritableFileMode
	}
	return windowsReadOnlyFileMode
}

// effectiveDirMode returns the permission the operating system applies to a
// directory created with mode. Windows applies 0777 to every directory.
func effectiveDirMode(mode FileMode) FileMode {
	if !runningOnWindows {
		return mode
	}
	return windowsDirMode
}

// checkPermission returns [ErrInvalidPermission] for path if mode has bits
// outside [PermissionBits].
func checkPermission(mode FileMode, path *Path) error {
	if mode&^PermissionBits != 0 {
		return permRangeErr(mode, *path)
	}

	return nil
}
