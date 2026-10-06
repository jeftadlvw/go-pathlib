// io_error_kinds.go holds the kinds raised by the io group (file open, read,
// and write operations).

package pathlib

// Kinds raised by the io group. Their cause is a *[PathError], unless a kind
// names another.
var (
	// ErrInvalidPermission is the broad group for an invalid permission or open-mode
	// value passed to the library. Match it with [errors.Is] to catch both
	// members below. Its cause is a *[PermissionError].
	ErrInvalidPermission = defineError(ErrPathlib, "INVALID_PERMISSION", "invalid file permission or mode")

	// ErrPermissionRange is raised when a permission value has bits outside
	// [PermissionBits]. It is a member of the ErrInvalidPermission group.
	ErrPermissionRange = defineError(ErrInvalidPermission, "RANGE", "permission has bits outside PermissionBits")

	// ErrUnsupportedMode is raised when an open-mode string is not supported.
	// It is a member of the ErrInvalidPermission group.
	ErrUnsupportedMode = defineError(ErrInvalidPermission, "UNSUPPORTED_MODE", "unsupported open mode")

	// ErrRead is returned when a file could not be read.
	ErrRead = defineError(ErrPathlib, "READ", "could not read file")

	// ErrWrite is returned when a file could not be written.
	ErrWrite = defineError(ErrPathlib, "WRITE", "could not write file")

	// ErrIsDir is returned when a file operation targets a directory.
	ErrIsDir = defineError(ErrPathlib, "IS_DIR", "path is a directory")
)
