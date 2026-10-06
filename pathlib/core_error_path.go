// core_error_path.go holds PathError, the cause of every failure that concerns
// paths.

package pathlib

import (
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
)

/*
PathError is the cause of every failure below [ErrPathlib], except for those
below [ErrPermission] and [ErrLookup]. It holds the paths the failure concerns and
the error that caused it, such as an error of the os package.

Read it with [errors.As]:

	var pathErr *PathError
	if errors.As(err, &pathErr) { use(pathErr.Paths()) }

A PathError is immutable.
*/
type PathError struct {
	// paths are the paths the failure concerns, in an order documented per
	// operation (e.g. RelativeTo reports [this, other]). May be empty.
	paths []Path

	// err is the underlying error, or nil.
	err error
}

// Paths returns a copy of the paths the failure concerns, in an order
// documented per operation (e.g. [Path.RelativeTo] reports [this, other]). It
// may be empty.
func (e *PathError) Paths() []Path {
	return slices.Clone(e.paths)
}

// Error renders the paths and the underlying error.
func (e *PathError) Error() string {
	msg := formatPaths(e.paths)
	if e.err == nil {
		return msg
	}

	if msg == "" {
		return e.err.Error()
	}

	return msg + ": " + e.err.Error()
}

// Unwrap returns the underlying error, or nil.
func (e *PathError) Unwrap() error {
	return e.err
}

// LogValue describes the paths in their Posix form, and the underlying error.
func (e *PathError) LogValue() slog.Value {
	attrs := []slog.Attr{slog.Any("paths", posixPaths(e.paths))}
	if e.err != nil {
		attrs = append(attrs, slog.Attr{Key: "cause", Value: causeLogValue(e.err)})
	}

	return slog.GroupValue(attrs...)
}

// MarshalJSON describes the paths in their Posix form, and the underlying
// error.
func (e *PathError) MarshalJSON() ([]byte, error) {
	var cause *causeJSON
	if e.err != nil {
		encoded := newCauseJSON(e.err)
		cause = &encoded
	}

	return json.Marshal(struct {
		Paths []string   `json:"paths"`
		Cause *causeJSON `json:"cause,omitempty"`
	}{Paths: posixPaths(e.paths), Cause: cause})
}

// formatPaths renders no path as "", one path as itself, and several paths as
// a bracketed list.
func formatPaths(paths []Path) string {
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return paths[0].String()
	}

	var b strings.Builder
	b.WriteString("[")
	for i := range paths {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(paths[i].String())
	}
	b.WriteString("]")

	return b.String()
}

// posixPaths returns paths in their Posix form, which is the form a Path
// serializes to.
func posixPaths(paths []Path) []string {
	posix := make([]string, len(paths))
	for i := range paths {
		posix[i] = paths[i].ToPosix()
	}

	return posix
}

// pathErr raises a failure of kind that concerns paths.
func pathErr(kind *PathlibError, paths ...Path) error {
	return raiseError(kind, &PathError{paths: paths})
}

// wrapErr raises a failure of kind that concerns paths and is caused by cause.
func wrapErr(kind *PathlibError, cause error, paths ...Path) error {
	return raiseError(kind, &PathError{paths: paths, err: cause})
}
