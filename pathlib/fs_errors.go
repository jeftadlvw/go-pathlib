package pathlib

import (
	"errors"
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
