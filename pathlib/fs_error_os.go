// fs_error_os.go holds osErr and osCreateErr, which return the error of a call
// to the operating system with the most specific error kind the library knows.

package pathlib

import (
	"errors"
	"io/fs"
)

// osErr returns an error of the operation that failed with err, an error of
// the operating system. A condition err matches becomes the error kind of the
// error. Any other error keeps operation as the error kind. err stays the
// underlying error of the PathError.
func osErr(operation *ErrorKind, err error, paths ...Path) error {
	return wrapErr(osKind(operation, err), err, paths...)
}

// osCreateErr returns an error of the operation that failed with err while
// creating path. A missing path is its missing parent directory, so an err
// matching fs.ErrNotExist becomes ErrParentNotExist. Any other error is
// returned as osErr returns it.
func osCreateErr(operation *ErrorKind, err error, path Path) error {
	if errors.Is(err, fs.ErrNotExist) {
		return wrapErr(ErrParentNotExist, err, path)
	}

	return osErr(operation, err, path)
}

// osKind returns ErrNotExist, ErrExist, or ErrPermissionDenied when err matches
// fs.ErrNotExist, fs.ErrExist, or fs.ErrPermission, and operation otherwise.
func osKind(operation *ErrorKind, err error) *ErrorKind {
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
