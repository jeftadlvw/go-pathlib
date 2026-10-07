// fs_error_kinds.go holds the kinds returned by the fs group (filesystem
// operations).

package pathlib

// Kinds returned by the fs group. Their cause is a *[PathError], unless a kind
// names another, and the error of the operating system is kept as the
// underlying error of the PathError.
//
// A failed call to the operating system returns the most specific kind the
// library knows. An error matching [fs.ErrNotExist], [fs.ErrExist], or
// [fs.ErrPermission] returns ErrNotExist, ErrExist, or ErrPermissionDenied. A
// missing parent directory of a created path returns ErrParentNotExist. Any
// other error returns the kind of the operation, which is a member of the
// [ErrOperation] group.
var (
	// ErrInvalidFilter is returned when [GlobOptions].Filter holds an unknown
	// value. It is a member of the [ErrInvalid] group.
	ErrInvalidFilter = defineError(ErrInvalid, "FILTER", "invalid glob filter option")

	// ErrInvalidPermission is returned when a permission has bits outside
	// [PermissionBits]. It is a member of the ErrInvalid group. Its cause is a
	// *[PermissionError].
	ErrInvalidPermission = defineError(ErrInvalid, "PERMISSION", "permission has bits outside PermissionBits")

	// ErrNotExist is returned when a required path does not exist, whether a
	// check of the library or the operating system found it. Match it with
	// [errors.Is] to catch every kind below. The underlying error matches
	// fs.ErrNotExist. [os.IsNotExist] does not unwrap errors and never
	// matches.
	ErrNotExist = defineError(ErrPathlib, "NOT_EXIST", "path does not exist")

	// ErrParentNotExist is returned when a required parent directory is
	// missing. It is a member of the ErrNotExist group.
	ErrParentNotExist = defineError(ErrNotExist, "PARENT", "parent directory does not exist")

	// ErrExist is returned when a path already exists and would be
	// overwritten, whether a check of the library or the operating system
	// found it. Match it with errors.Is to catch every kind below. The
	// underlying error matches fs.ErrExist. [os.IsExist] does not unwrap
	// errors and never matches.
	ErrExist = defineError(ErrPathlib, "EXIST", "path already exists")

	// ErrFileExist is returned when the conflicting path is a file. It is a
	// member of the ErrExist group.
	ErrFileExist = defineError(ErrExist, "FILE", "file already exists")

	// ErrDirExist is returned when the conflicting path is a directory. It is
	// a member of the ErrExist group.
	ErrDirExist = defineError(ErrExist, "DIR", "directory already exists")

	// ErrNotEmptyDir is returned when the conflicting path is a directory with
	// entries, where an empty directory is accepted. It is a member of the
	// ErrExist group.
	ErrNotEmptyDir = defineError(ErrExist, "NOT_EMPTY_DIR", "directory is not empty")

	// ErrPermissionDenied is returned when the operating system denies access
	// to a path. The underlying error matches fs.ErrPermission.
	ErrPermissionDenied = defineError(ErrPathlib, "PERMISSION_DENIED", "permission denied")

	// ErrWrongType is the group of the kinds that report a path whose type the
	// operation cannot handle. Match it with errors.Is to catch every kind
	// below.
	ErrWrongType = defineError(ErrPathlib, "WRONG_TYPE", "path has the wrong type")

	// ErrNotFile is returned when a path is expected to be a file but is not,
	// such as a directory. It is a member of the ErrWrongType group.
	ErrNotFile = defineError(ErrWrongType, "NOT_FILE", "path is not a file")

	// ErrNotDir is returned when a path is expected to be a directory but is
	// not. It is a member of the ErrWrongType group.
	ErrNotDir = defineError(ErrWrongType, "NOT_DIR", "path is not a directory")

	// ErrNotSymlink is returned when a path is expected to be a symlink but is
	// not. It is a member of the ErrWrongType group.
	ErrNotSymlink = defineError(ErrWrongType, "NOT_SYMLINK", "path is not a symlink")

	// ErrTypeMismatch is returned when the source and destination of a copy or
	// move have incompatible types. The paths are [source, destination]. It is
	// a member of the ErrWrongType group.
	ErrTypeMismatch = defineError(ErrWrongType, "MISMATCH", "source and destination types are incompatible")

	// ErrUnsupportedType is returned when the type of a path, such as a
	// device, is not supported by the operation. It is a member of the
	// ErrWrongType group.
	ErrUnsupportedType = defineError(ErrWrongType, "UNSUPPORTED", "file type is not supported")

	// ErrStat is returned when a path could not be stat'ed. It is a member of
	// the [ErrOperation] group.
	ErrStat = defineError(ErrOperation, "STAT", "could not stat path")

	// ErrOpen is returned when a path could not be opened. It is a member of
	// the ErrOperation group.
	ErrOpen = defineError(ErrOperation, "OPEN", "could not open path")

	// ErrReadDir is returned when the entries of a directory could not be
	// read. It is a member of the ErrOperation group.
	ErrReadDir = defineError(ErrOperation, "READ_DIR", "could not read directory entry")

	// ErrReadSymlink is returned when the target of a symlink could not be
	// read. It is a member of the ErrOperation group.
	ErrReadSymlink = defineError(ErrOperation, "READ_SYMLINK", "could not read symlink target")

	// ErrResolve is returned when the symlinks of a path could not be
	// resolved. It is a member of the ErrOperation group.
	ErrResolve = defineError(ErrOperation, "RESOLVE", "could not resolve path")

	// ErrCreate is returned when a file, directory, or symlink could not be
	// created. It is a member of the ErrOperation group.
	ErrCreate = defineError(ErrOperation, "CREATE", "could not create path")

	// ErrCopy is returned when the content of a file could not be copied. The
	// paths are [source, destination], and the underlying error is the error
	// of the io package. It is a member of the ErrOperation group.
	ErrCopy = defineError(ErrOperation, "COPY", "could not copy file content")

	// ErrRemove is returned when a path could not be removed. It is a member
	// of the ErrOperation group.
	ErrRemove = defineError(ErrOperation, "REMOVE", "could not remove path")

	// ErrSetPermission is returned when the permission of a path could not be
	// changed. It is a member of the ErrOperation group.
	ErrSetPermission = defineError(ErrOperation, "SET_PERMISSION", "could not set permission")

	// ErrWalk is returned when a walk is stopped by an error of the walk
	// callback or by a done context. The underlying error is the error of the
	// callback or of the context. An error of the callback can hold another
	// failure of the library, which errors.Is and [errors.As] then search as
	// well.
	ErrWalk = defineError(ErrPathlib, "WALK", "error walking path")
)
