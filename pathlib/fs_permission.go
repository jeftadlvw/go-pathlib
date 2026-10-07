// fs_permission.go holds the default modes, the check of the bits the library
// accepts in a mode, and the change of the mode of a path.

package pathlib

import (
	"os"
)

const (
	// posixDefaultFileMode is the default mode of a file outside Windows.
	posixDefaultFileMode FileMode = 0o644

	// posixDefaultDirMode is the default mode of a directory outside Windows.
	posixDefaultDirMode FileMode = 0o755

	// windowsWritableFileMode is the mode Windows applies to a writable file.
	windowsWritableFileMode FileMode = 0o666

	// windowsReadOnlyFileMode is the mode Windows applies to a read-only file.
	windowsReadOnlyFileMode FileMode = 0o444

	// windowsDirMode is the mode Windows applies to every directory.
	windowsDirMode FileMode = 0o777
)

// DefaultFileMode returns the default mode of a file. It is 0o644 (rw-r--r--)
// on Unix and 0o666 on Windows, which only knows writable and read-only files.
func DefaultFileMode() FileMode {
	return effectiveFileMode(posixDefaultFileMode)
}

// DefaultDirMode returns the default mode of a directory. It is 0o755
// (rwxr-xr-x) on Unix and 0o777 on Windows, which applies 0o777 to every
// directory.
func DefaultDirMode() FileMode {
	return effectiveDirMode(posixDefaultDirMode)
}

// SetMode sets the mode of path to mode. It wraps [os.Chmod] and follows
// symlinks. The setuid and setgid bits are set reliably.
//
// A mode with bits outside [ModePerm] and [ModeSpecial] returns
// [ErrInvalidFileMode]. A missing path returns [ErrNotExist], and denied access
// returns [ErrPermissionDenied]. Any other failure returns [ErrSetMode].
func SetMode(path *Path, mode FileMode) error {
	err := checkModeBits(mode, path)
	if err != nil {
		return err
	}

	err = os.Chmod(path.String(), mode)
	if err != nil {
		return osErr(ErrSetMode, err, *path)
	}

	return nil
}

// effectiveFileMode returns the mode the operating system applies to a file
// created with mode. Windows knows writable (0o666) and read-only (0o444)
// files alone.
func effectiveFileMode(mode FileMode) FileMode {
	if !runningOnWindows {
		return mode
	}
	if mode&0o200 != 0 {
		return windowsWritableFileMode
	}
	return windowsReadOnlyFileMode
}

// effectiveDirMode returns the mode the operating system applies to a
// directory created with mode. Windows applies 0o777 to every directory.
func effectiveDirMode(mode FileMode) FileMode {
	if !runningOnWindows {
		return mode
	}
	return windowsDirMode
}

// checkModeBits returns [ErrInvalidFileMode] for path if mode has bits outside
// [ModePerm] and [ModeSpecial].
func checkModeBits(mode FileMode, path *Path) error {
	if mode&^(ModePerm|ModeSpecial) != 0 {
		return fileModeErr(mode, *path)
	}

	return nil
}
