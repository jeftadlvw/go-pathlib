// fs_error_file_mode.go holds FileModeError, the cause of the failures of
// ErrInvalidFileMode.

package pathlib

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
)

// FileModeError is the cause of the failures of [ErrInvalidFileMode]. It holds
// the rejected [FileMode] and the paths the failure concerns. The file mode is
// the value passed where the library accepts one, such as
// FileOptions.CreateMode or the mode of [SetMode].
//
// Read it with [errors.As]:
//
//	var fileModeErr *FileModeError
//	if errors.As(err, &fileModeErr) { use(fileModeErr.FileMode()) }
//
// A FileModeError is immutable.
type FileModeError struct {
	// paths are the paths the failure concerns. May be empty.
	paths []Path

	// fileMode is the rejected file mode, as the caller passed it.
	fileMode FileMode
}

// Paths returns a copy of the paths the failure concerns. It may be empty.
func (e *FileModeError) Paths() []Path {
	return slices.Clone(e.paths)
}

// FileMode returns the rejected file mode, as the caller passed it, including
// the bits outside [ModePerm] and [ModeSpecial].
func (e *FileModeError) FileMode() FileMode {
	return e.fileMode
}

// Error renders the paths and the rejected file mode.
func (e *FileModeError) Error() string {
	msg := fmt.Sprintf("(file mode %#o)", e.fileMode)

	paths := formatPaths(e.paths)
	if paths == "" {
		return msg
	}

	return paths + " " + msg
}

// LogValue describes the paths in their Posix form and the rejected file mode.
func (e *FileModeError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("paths", posixPaths(e.paths)),
		slog.Uint64("fileMode", uint64(e.fileMode)),
	)
}

// MarshalJSON describes the paths in their Posix form and the rejected file
// mode.
func (e *FileModeError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Paths    []string `json:"paths"`
		FileMode uint32   `json:"fileMode"`
	}{Paths: posixPaths(e.paths), FileMode: uint32(e.fileMode)})
}

// fileModeErr returns ErrInvalidFileMode for fileMode, which has bits outside
// ModePerm and ModeSpecial.
func fileModeErr(fileMode FileMode, paths ...Path) error {
	return wrapError(ErrInvalidFileMode, &FileModeError{paths: slices.Clone(paths), fileMode: fileMode})
}
