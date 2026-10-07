// fs_error_permission.go holds PermissionError, the cause of the failures of
// ErrInvalidPermission and ErrInvalidMode.

package pathlib

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
)

// PermissionError is the cause of the failures of [ErrInvalidPermission] and
// [ErrInvalidMode]. It holds the rejected permission or open-mode value and the
// paths the failure concerns.
//
// Read it with [errors.As]:
//
//	var permErr *PermissionError
//	if errors.As(err, &permErr) { use(permErr.Perm(), permErr.Mode()) }
//
// A PermissionError is immutable.
type PermissionError struct {
	// paths are the paths the failure concerns. May be empty.
	paths []Path

	// perm is the rejected permission value (0 when not applicable).
	perm FileMode

	// mode is the rejected open-mode string (empty when not applicable).
	mode string

	// isMode reports whether the rejected value is the open mode. An empty
	// mode is rejected as well, so mode alone cannot tell.
	isMode bool
}

// Paths returns a copy of the paths the failure concerns. It may be empty.
func (e *PermissionError) Paths() []Path {
	return slices.Clone(e.paths)
}

// Perm returns the rejected permission value, or 0 when not applicable.
func (e *PermissionError) Perm() FileMode {
	return e.perm
}

// Mode returns the rejected open-mode string, or "" when not applicable.
func (e *PermissionError) Mode() string {
	return e.mode
}

// Error renders the paths and the rejected value.
func (e *PermissionError) Error() string {
	value := fmt.Sprintf("(perm %#o)", e.perm)
	if e.isMode {
		value = fmt.Sprintf("(mode %q)", e.mode)
	}

	paths := formatPaths(e.paths)
	if paths == "" {
		return value
	}

	return paths + " " + value
}

// LogValue describes the paths in their Posix form and the rejected value.
func (e *PermissionError) LogValue() slog.Value {
	value := slog.Uint64("perm", uint64(e.perm))
	if e.isMode {
		value = slog.String("mode", e.mode)
	}

	return slog.GroupValue(slog.Any("paths", posixPaths(e.paths)), value)
}

// MarshalJSON describes the paths in their Posix form and the rejected value.
func (e *PermissionError) MarshalJSON() ([]byte, error) {
	if e.isMode {
		return json.Marshal(struct {
			Paths []string `json:"paths"`
			Mode  string   `json:"mode"`
		}{Paths: posixPaths(e.paths), Mode: e.mode})
	}

	return json.Marshal(struct {
		Paths []string `json:"paths"`
		Perm  uint32   `json:"perm"`
	}{Paths: posixPaths(e.paths), Perm: uint32(e.perm)})
}

// permRangeErr returns ErrInvalidPermission for a permission with bits
// outside PermissionBits.
func permRangeErr(perm FileMode, paths ...Path) error {
	return wrapError(ErrInvalidPermission, &PermissionError{paths: slices.Clone(paths), perm: perm})
}

// permModeErr returns ErrInvalidMode for an unsupported open-mode string.
func permModeErr(mode string, paths ...Path) error {
	return wrapError(ErrInvalidMode, &PermissionError{paths: slices.Clone(paths), mode: mode, isMode: true})
}
