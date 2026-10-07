// io_error_open_mode.go holds OpenModeError, the cause of the failures of
// ErrInvalidOpenMode.

package pathlib

import (
	"encoding/json"
	"log/slog"
	"slices"
)

// OpenModeError is the cause of the failures of [ErrInvalidOpenMode]. It holds
// the rejected [OpenMode] and the paths the failure concerns.
//
// Read it with [errors.As]:
//
//	var openModeErr *OpenModeError
//	if errors.As(err, &openModeErr) { use(openModeErr.OpenMode()) }
//
// An OpenModeError is immutable.
type OpenModeError struct {
	// paths are the paths the failure concerns. May be empty.
	paths []Path

	// openMode is the rejected open mode, as the caller passed it.
	openMode OpenMode
}

// Paths returns a copy of the paths the failure concerns. It may be empty.
func (e *OpenModeError) Paths() []Path {
	return slices.Clone(e.paths)
}

// OpenMode returns the rejected open mode, as the caller passed it.
func (e *OpenModeError) OpenMode() OpenMode {
	return e.openMode
}

// Error renders the paths and the rejected open mode.
func (e *OpenModeError) Error() string {
	msg := "(open mode " + e.openMode.String() + ")"

	paths := formatPaths(e.paths)
	if paths == "" {
		return msg
	}

	return paths + " " + msg
}

// LogValue describes the paths in their Posix form and the rejected open
// mode.
func (e *OpenModeError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("paths", posixPaths(e.paths)),
		slog.String("openMode", e.openMode.String()),
	)
}

// MarshalJSON describes the paths in their Posix form and the rejected open
// mode.
func (e *OpenModeError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Paths    []string `json:"paths"`
		OpenMode string   `json:"openMode"`
	}{Paths: posixPaths(e.paths), OpenMode: e.openMode.String()})
}

// openModeErr returns ErrInvalidOpenMode for openMode, which is no OpenMode
// constant.
func openModeErr(openMode OpenMode, paths ...Path) error {
	return wrapError(ErrInvalidOpenMode, &OpenModeError{paths: slices.Clone(paths), openMode: openMode})
}
