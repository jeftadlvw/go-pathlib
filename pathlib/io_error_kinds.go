// io_error_kinds.go holds the error kinds returned by the io group (file open,
// read, and write operations).

package pathlib

// Error kinds returned by the io group. Their cause is a *[PathError], unless
// an error kind names another.
var (
	// ErrInvalidOpenMode is returned when the open mode of [OpenOptions] is
	// no [OpenMode] constant. It is a member of the [ErrInvalid] group. Its
	// cause is an *[OpenModeError].
	ErrInvalidOpenMode = defineError(ErrInvalid, "open_mode", "invalid open mode")

	// ErrRead is returned when a file could not be read. It is a member of the
	// [ErrOperation] group.
	ErrRead = defineError(ErrOperation, "read", "could not read file")

	// ErrWrite is returned when a file could not be written. It is a member of
	// the ErrOperation group.
	ErrWrite = defineError(ErrOperation, "write", "could not write file")
)
