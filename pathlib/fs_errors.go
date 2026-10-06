package pathlib

import (
	"errors"
	"io/fs"
)

// Error sentinels raised by the fs group (filesystem operations). They are
// matched with errors.Is and exposed through PathlibError.Kind().
var (
	// ErrNotExist is the broad group for "a required path does not exist".
	// Match it with errors.Is to catch every not-exist case below.
	//
	// It aliases fs.ErrNotExist (and thus os.ErrNotExist), so both sentinels catch
	// every not-exist error, whether raised by this library's own checks or by the
	// operating system (e.g. ErrRead for a missing file). os.IsNotExist does not
	// unwrap errors and never matches.
	ErrNotExist = aliasKind(fs.ErrNotExist, "path does not exist")

	// ErrParentNotExist is raised when a required parent directory is missing.
	// It is a member of the ErrNotExist group.
	ErrParentNotExist = subKind(ErrNotExist, "parent directory does not exist")

	// ErrExist is the broad group for "a path already exists and would be
	// overwritten". Match it with errors.Is to catch every exist case below.
	//
	// It aliases fs.ErrExist (and thus os.ErrExist), so both sentinels catch every
	// exist error, whether raised by this library's own checks or by the operating
	// system. os.IsExist does not unwrap errors and never matches.
	ErrExist = aliasKind(fs.ErrExist, "path already exists")

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

	// ErrAccess is the broad group for "a path could not be accessed", covering
	// both the open and the read-dir failures below. Walk and WalkR surface these
	// failures (as localDirError, in WalkR's case). Match it with errors.Is to catch
	// every access failure without distinguishing the exact cause.
	ErrAccess = errors.New("could not access path")

	// ErrOpen is returned when a path could not be opened. The os cause is wrapped.
	// It is a member of the ErrAccess group.
	ErrOpen = subKind(ErrAccess, "could not open path")

	// ErrReadDir is returned when a directory entry could not be read. The os cause is wrapped.
	// It is a member of the ErrAccess group.
	ErrReadDir = subKind(ErrAccess, "could not read directory entry")

	// ErrStat is returned when a path could not be stat'ed. The os cause is wrapped.
	ErrStat = errors.New("could not stat path")

	// ErrReadSymlink is returned when a symlink's target could not be read. The os cause is wrapped.
	ErrReadSymlink = errors.New("could not read symlink target")

	// ErrResolve is returned when the symlinks of a path could not be resolved. The os cause is wrapped.
	ErrResolve = errors.New("could not resolve path")

	// ErrCreate is returned when a file, directory, or symlink could not be created.
	// The os cause is wrapped.
	ErrCreate = errors.New("could not create path")

	// ErrRemove is returned when a path could not be removed. The os cause is wrapped.
	ErrRemove = errors.New("could not remove path")

	// ErrCopy is returned when a file's content could not be copied. The paths are
	// [source, destination]. The io cause is wrapped.
	ErrCopy = errors.New("could not copy file content")

	// ErrSetPermission is returned when a path's permission could not be changed.
	// The os cause is wrapped.
	ErrSetPermission = errors.New("could not set permission")

	// ErrWalk is returned when walking a path is aborted by an error of the walk
	// callback or by a done context. The underlying cause is wrapped.
	ErrWalk = errors.New("error walking path")

	// ErrInvalidFilter is returned when GlobOptions.Filter holds an unknown value.
	ErrInvalidFilter = errors.New("invalid glob filter option")
)
