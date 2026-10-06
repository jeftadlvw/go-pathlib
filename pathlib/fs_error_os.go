// fs_error_os.go holds osErr, which raises the failure of a call to the
// operating system with the most specific kind the library knows.

package pathlib

import (
	"errors"
	"io/fs"
)

// osErr raises a failure of the operation that failed with err, an error of
// the operating system. A condition err matches becomes the kind of the
// failure. Any other error keeps operation as the kind. err stays the
// underlying error of the PathError.
func osErr(operation *PathlibError, err error, paths ...Path) error {
	return wrapErr(osKind(operation, err), err, paths...)
}

// osKind returns ErrNotExist, ErrExist, or ErrPermissionDenied when err matches
// fs.ErrNotExist, fs.ErrExist, or fs.ErrPermission, and operation otherwise.
func osKind(operation *PathlibError, err error) *PathlibError {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ErrNotExist
	case errors.Is(err, fs.ErrExist):
		return ErrExist
	case errors.Is(err, fs.ErrPermission):
		return ErrPermissionDenied
	default:
		return operation
	}
}
