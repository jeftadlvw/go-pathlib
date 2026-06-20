package pathlib

import (
	"errors"
	"fmt"
	"os"
)

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

	// ErrStat is returned when a path could not be stat'ed. The os cause is wrapped.
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
	t, ok := target.(**PathlibError)
	if ok {
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
