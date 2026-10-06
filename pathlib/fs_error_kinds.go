// fs_error_kinds.go holds the kinds raised by the fs group (filesystem
// operations).

package pathlib

/*
Kinds raised by the fs group. Their cause is a *[PathError], and the error of the
operating system is kept as the underlying error of the PathError.

A failed call to the operating system returns the most specific kind the library
knows. An error matching [fs.ErrNotExist], [fs.ErrExist], or [fs.ErrPermission]
returns ErrNotExist, ErrExist, or ErrPermissionDenied. Any other error returns
the kind of the operation, such as [ErrRead].
*/
var (
	// ErrNotExist is returned when a required path does not exist, whether a
	// check of the library or the operating system found it. Match it with
	// [errors.Is] to catch every kind below. The underlying error matches
	// fs.ErrNotExist. [os.IsNotExist] does not unwrap errors and never matches.
	ErrNotExist = defineError(ErrPathlib, "NOT_EXIST", "path does not exist")

	// ErrParentNotExist is raised when a required parent directory is missing.
	// It is a member of the ErrNotExist group.
	ErrParentNotExist = defineError(ErrNotExist, "PARENT", "parent directory does not exist")

	// ErrExist is returned when a path already exists and would be overwritten,
	// whether a check of the library or the operating system found it. Match it
	// with errors.Is to catch every kind below. The underlying error matches
	// fs.ErrExist. [os.IsExist] does not unwrap errors and never matches.
	ErrExist = defineError(ErrPathlib, "EXIST", "path already exists")

	// ErrFileExist is raised when the conflicting path is a file.
	// It is a member of the ErrExist group.
	ErrFileExist = defineError(ErrExist, "FILE", "file already exists")

	// ErrDirExist is raised when the conflicting path is a directory.
	// It is a member of the ErrExist group.
	ErrDirExist = defineError(ErrExist, "DIR", "directory already exists")

	// ErrPermissionDenied is returned when the operating system denies access
	// to a path. The underlying error matches fs.ErrPermission.
	ErrPermissionDenied = defineError(ErrPathlib, "PERMISSION_DENIED", "permission denied")

	// ErrNotFile is returned when a path is expected to be a regular file but
	// is not.
	ErrNotFile = defineError(ErrPathlib, "NOT_FILE", "path is not a file")

	// ErrNotDir is returned when a path is expected to be a directory but is not.
	ErrNotDir = defineError(ErrPathlib, "NOT_DIR", "path is not a directory")

	// ErrNotSymlink is returned when a path is expected to be a symlink but is
	// not.
	ErrNotSymlink = defineError(ErrPathlib, "NOT_SYMLINK", "path is not a symlink")

	// ErrNotEmptyDir is returned when a directory is expected to be empty but
	// is not.
	ErrNotEmptyDir = defineError(ErrPathlib, "NOT_EMPTY_DIR", "directory is not empty")

	// ErrCopyType is returned when a path's file type cannot be copied.
	ErrCopyType = defineError(ErrPathlib, "COPY_TYPE", "no copy operation defined for this file type")

	// ErrTypeMismatch is returned when the source and destination of a copy or
	// move have incompatible file types. The paths are [source, destination].
	ErrTypeMismatch = defineError(ErrPathlib, "TYPE_MISMATCH", "source and destination types are incompatible")

	// ErrAccess is the broad group for "a path could not be accessed", covering
	// both the open and the read-dir failures below. Match it with errors.Is to
	// catch every access failure without distinguishing the exact cause.
	ErrAccess = defineError(ErrPathlib, "ACCESS", "could not access path")

	// ErrOpen is returned when a path could not be opened. It is a member of the
	// ErrAccess group.
	ErrOpen = defineError(ErrAccess, "OPEN", "could not open path")

	// ErrReadDir is returned when a directory entry could not be read. It is a
	// member of the ErrAccess group.
	ErrReadDir = defineError(ErrAccess, "READ_DIR", "could not read directory entry")

	// ErrStat is returned when a path could not be stat'ed.
	ErrStat = defineError(ErrPathlib, "STAT", "could not stat path")

	// ErrReadSymlink is returned when a symlink's target could not be read.
	ErrReadSymlink = defineError(ErrPathlib, "READ_SYMLINK", "could not read symlink target")

	// ErrResolve is returned when the symlinks of a path could not be resolved.
	ErrResolve = defineError(ErrPathlib, "RESOLVE", "could not resolve path")

	// ErrCreate is returned when a file, directory, or symlink could not be
	// created.
	ErrCreate = defineError(ErrPathlib, "CREATE", "could not create path")

	// ErrRemove is returned when a path could not be removed.
	ErrRemove = defineError(ErrPathlib, "REMOVE", "could not remove path")

	// ErrCopy is returned when a file's content could not be copied. The paths are
	// [source, destination], and the underlying error is the error of the io
	// package.
	ErrCopy = defineError(ErrPathlib, "COPY", "could not copy file content")

	// ErrSetPermission is returned when a path's permission could not be changed.
	ErrSetPermission = defineError(ErrPathlib, "SET_PERMISSION", "could not set permission")

	// ErrWalk is returned when walking a path is aborted by an error of the walk
	// callback or by a done context. The underlying error is the error of the
	// callback or of the context. An error of the callback can hold another
	// failure of the library, which errors.Is and [errors.As] then search as well.
	ErrWalk = defineError(ErrPathlib, "WALK", "error walking path")

	// ErrInvalidFilter is returned when [GlobOptions].Filter holds an unknown
	// value.
	ErrInvalidFilter = defineError(ErrPathlib, "INVALID_FILTER", "invalid glob filter option")
)
