package pathlib

import (
	"errors"
	"fmt"
	"os"
)

const defaultOpenPermission = 0644 // rw-r--r--
const defaultOpenMode = "rw"

// Error sentinels raised by pathlib_io.go (file open/read/write operations).
var (
	// ErrPermission is the broad group for an invalid permission or open-mode
	// configuration. Match it with errors.Is to catch both members below.
	ErrPermission = errors.New("invalid file permission or mode")

	// ErrPermissionRange is raised when a permission value is out of bounds.
	// It is a member of the ErrPermission group and is carried by a *PermissionError.
	ErrPermissionRange = subKind(ErrPermission, "permission out of bounds (0..0o777)")

	// ErrUnsupportedMode is raised when an open-mode string is not supported.
	// It is a member of the ErrPermission group and is carried by a *PermissionError.
	ErrUnsupportedMode = subKind(ErrPermission, "unsupported open mode")

	// ErrStat is returned when a path could not be stat'ed; wraps the os cause.
	ErrStat = errors.New("could not stat path")

	// ErrIsDir is returned when a file operation targets a directory.
	ErrIsDir = errors.New("path is a directory")
)

/*
PermissionError is a *PathlibError that additionally carries the rejected
permission or open-mode value. It is returned for the ErrPermission group, so it
is caught both by its own type and by the PathlibError catch-all:

	var pe *PermissionError
	if errors.As(err, &pe) { use(pe.Perm, pe.Mode) }

	var base *PathlibError
	if errors.As(err, &base) { use(base.Paths) }
*/
type PermissionError struct {
	*PathlibError

	// Perm is the rejected permission value (0 when not applicable).
	Perm os.FileMode

	// Mode is the rejected open-mode string (empty when not applicable).
	Mode string
}

// Error appends the rejected value to the base message.
func (e *PermissionError) Error() string {
	msg := e.PathlibError.Error()
	switch {
	case e.Mode != "":
		msg += fmt.Sprintf(" (mode %q)", e.Mode)
	case e.Perm != 0:
		msg += fmt.Sprintf(" (perm %#o)", e.Perm)
	}
	return msg
}

// As lets errors.As(err, **PathlibError) reach the embedded base error, keeping
// the "every error is a *PathlibError" catch-all intact for this typed superset.
func (e *PermissionError) As(target any) bool {
	if t, ok := target.(**PathlibError); ok {
		*t = e.PathlibError
		return true
	}
	return false
}

// permRangeErr builds a PermissionError for an out-of-bounds permission value.
func permRangeErr(perm os.FileMode, paths ...Path) *PermissionError {
	return &PermissionError{PathlibError: pathErr(ErrPermissionRange, paths...), Perm: perm}
}

// permModeErr builds a PermissionError for an unsupported open-mode string.
func permModeErr(mode string, paths ...Path) *PermissionError {
	return &PermissionError{PathlibError: pathErr(ErrUnsupportedMode, paths...), Mode: mode}
}

/*
OpenOptions is a configuration struct for opening files.
*/
type OpenOptions struct {
	// Create the file if it does not exist.
	CreateIfNotExists bool

	// Permissions for file creation.
	Permission os.FileMode

	// Open mode. Loosely defined as a string that may only contain "r" (read), "w" (write) and "a" (append).
	// Enforced by functions that receive this struct.
	Mode string
}

func defaultOpenOptions() OpenOptions {
	return OpenOptions{
		true,
		defaultOpenPermission,
		defaultOpenMode,
	}
}

/*
OpenFile opens a file for reading and writing.
If the file does not exist, it is created with 0644 permissions. If the file already
exists, it is truncated.

It's the caller's responsibility to close the returned os.File.
*/
func OpenFile(path *Path) (*os.File, error) {
	return OpenFileWithOptions(path, defaultOpenOptions())
}

/*
OpenFileWithOptions opens a file with passed extended configuration.

If OpenOptions.Permission is 0, the value defaults to 0644.
If OpenOptions.Mode is an empty string, it defaults to "rw"

The order for OpenOptions.Mode is enforced as follows: "r" (read), "w" (write), "a" (append) must be used
in exactly this order. "a" can only be used if "w" is used.

It's the caller's responsibility to close the returned os.File.
*/
func OpenFileWithOptions(path *Path, opts OpenOptions) (*os.File, error) {

	if opts.Permission > 0777 { // opts.Permission is uint32, so it can never be < 0
		return nil, permRangeErr(opts.Permission, *path)
	}

	// set the default permission value
	if opts.Permission == 0 {
		opts.Permission = defaultOpenPermission
	}

	// set the default open mode
	if len(opts.Mode) == 0 {
		opts.Mode = defaultOpenMode
	}

	// determine file open mode from the mode string
	var fileOpenMode int
	switch opts.Mode {
	case "r":
		fileOpenMode = os.O_RDONLY
	case "w":
		fileOpenMode = os.O_WRONLY | os.O_TRUNC
	case "rw":
		fileOpenMode = os.O_RDWR | os.O_TRUNC
	case "wa":
		fileOpenMode = os.O_WRONLY | os.O_APPEND
	case "rwa":
		fileOpenMode = os.O_RDWR | os.O_APPEND
	default:
		return nil, permModeErr(opts.Mode, *path)
	}

	if opts.CreateIfNotExists {
		if fileOpenMode == os.O_RDONLY && runningOnWindows {
			// On Windows, O_CREATE|O_RDONLY either fails for existing read-only files
			// (O_CREATE requires write access) or returns a writable handle for new files.
			// Separate creation from opening to guarantee a read-only handle.
			_, err := os.Stat(path.String())
			pathExists := err == nil

			if !pathExists {
				f, createErr := os.OpenFile(path.String(), os.O_CREATE|os.O_WRONLY, opts.Permission)

				if createErr != nil {
					return nil, createErr
				}

				_ = f.Close()
			}
		} else {
			fileOpenMode = fileOpenMode | os.O_CREATE
		}
	}

	file, err := os.OpenFile(path.String(), fileOpenMode, opts.Permission)
	if err != nil {
		return nil, err
	}

	// Fun fact: Unix-based operating systems support opening a file descriptor
	// on directory paths in readonly mode. Fuck that inconsistency and return an error
	// if the path is a directory.
	stat, err := os.Stat(path.String())

	if err != nil {
		_ = file.Close()
		return nil, wrapErr(ErrStat, err, *path)
	}

	if stat.IsDir() {
		_ = file.Close()
		return nil, pathErr(ErrIsDir, *path)
	}

	return file, nil
}

/*
ReadFile reads the passed file and returns read bytes.
*/
func ReadFile(path *Path) ([]byte, error) {
	return os.ReadFile(path.String())
}

/*
ReadFileToString reads the passed file and returns its content as a string.
*/
func ReadFileToString(path *Path) (string, error) {
	bytes, err := os.ReadFile(path.String())
	return string(bytes), err
}

/*
WriteBytes writes raw byte data to the defined file.

The file is not created. Preexisting content is truncated.
*/
func WriteBytes(path *Path, data []byte) (int, error) {
	return writeBytes(path, data, "w")
}

/*
WriteString writes a string to the defined file.

The file is not created. Preexisting content is truncated.
*/
func WriteString(path *Path, data string) (int, error) {
	return WriteBytes(path, []byte(data))
}

/*
AppendBytes appends byte data to the defined file.

The file is not created.
*/
func AppendBytes(path *Path, data []byte) (int, error) {
	return writeBytes(path, data, "wa")
}

/*
AppendString appends a string to the defined file.

The file is not created.
*/
func AppendString(path *Path, data string) (int, error) {
	return AppendBytes(path, []byte(data))
}

/*
writeBytes is an internal function that writes data to a file, opened in a specific mode.
*/
func writeBytes(path *Path, data []byte, mode string) (int, error) {
	file, err := OpenFileWithOptions(path, OpenOptions{
		CreateIfNotExists: false,
		Mode:              mode,
	})

	if err != nil {
		return 0, err
	}

	defer file.Close()

	return file.Write(data)
}
