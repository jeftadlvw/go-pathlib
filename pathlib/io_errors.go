package pathlib

import (
	"errors"
	"fmt"
	"os"
)

// Error sentinels raised by the io group (file open/read/write operations).
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

	// ErrRead is returned when a file could not be read. The os cause is wrapped.
	ErrRead = errors.New("could not read file")

	// ErrWrite is returned when a file could not be written. The os cause is wrapped.
	ErrWrite = errors.New("could not write file")

	// ErrIsDir is returned when a file operation targets a directory.
	ErrIsDir = errors.New("path is a directory")
)

/*
PermissionError is a *PathlibError that additionally carries the rejected
permission or open-mode value. It is returned for the ErrPermission group, so it
is caught both by its own type and by the PathlibError catch-all:

	var pe *PermissionError
	if errors.As(err, &pe) { use(pe.Perm(), pe.Mode()) }

	var base *PathlibError
	if errors.As(err, &base) { use(base.Paths()) }

A PermissionError is immutable, like PathlibError.
*/
type PermissionError struct {
	// base holds the kind, the paths, and the cause.
	base *PathlibError

	// perm is the rejected permission value (0 when not applicable).
	perm os.FileMode

	// mode is the rejected open-mode string (empty when not applicable).
	mode string
}

// Perm returns the rejected permission value, or 0 when not applicable.
func (e *PermissionError) Perm() os.FileMode {
	return e.perm
}

// Mode returns the rejected open-mode string, or "" when not applicable.
func (e *PermissionError) Mode() string {
	return e.mode
}

// Kind returns the sentinel that categorizes the error, as PathlibError.Kind does.
func (e *PermissionError) Kind() error {
	return e.base.Kind()
}

// Paths returns a copy of the paths this error concerns, as PathlibError.Paths does.
func (e *PermissionError) Paths() []Path {
	return e.base.Paths()
}

// Error appends the rejected value to the base message.
func (e *PermissionError) Error() string {
	msg := e.base.Error()
	switch {
	case e.mode != "":
		msg += fmt.Sprintf(" (mode %q)", e.mode)
	case e.perm != 0:
		msg += fmt.Sprintf(" (perm %#o)", e.perm)
	}
	return msg
}

// Unwrap returns the cause, or nil, as PathlibError.Unwrap does.
func (e *PermissionError) Unwrap() error {
	return e.base.Unwrap()
}

// Is reports whether target matches the kind, as PathlibError.Is does.
func (e *PermissionError) Is(target error) bool {
	return e.base.Is(target)
}

// As lets errors.As(err, **PathlibError) reach the base error, keeping the
// "every error is a *PathlibError" catch-all intact for this typed superset.
func (e *PermissionError) As(target any) bool {
	t, ok := target.(**PathlibError)
	if ok {
		*t = e.base
		return true
	}
	return false
}

// permRangeErr builds a PermissionError for an out-of-bounds permission value.
func permRangeErr(perm os.FileMode, paths ...Path) *PermissionError {
	return &PermissionError{base: pathErr(ErrPermissionRange, paths...), perm: perm}
}

// permModeErr builds a PermissionError for an unsupported open-mode string.
func permModeErr(mode string, paths ...Path) *PermissionError {
	return &PermissionError{base: pathErr(ErrUnsupportedMode, paths...), mode: mode}
}
