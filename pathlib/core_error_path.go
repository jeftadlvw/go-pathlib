// core_error_path.go holds PathError, the cause of every failure that concerns
// paths.

package pathlib

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"strings"
)

// PathError is the cause of every failure below [ErrPathlib], unless the
// documentation of its kind names another cause. It holds the paths the failure
// concerns and the error that caused it, such as an error of the os package.
//
// Read it with [errors.As]:
//
//	var pathErr *PathError
//	if errors.As(err, &pathErr) { use(pathErr.Paths()) }
//
// A PathError is immutable.
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

// Error renders the paths and the underlying error. An underlying error of the
// os package that repeats the paths renders without them, so the text names
// every path once.
func (e *PathError) Error() string {
	msg := formatPaths(e.paths)
	if e.err == nil {
		return msg
	}

	cause := e.causeText()
	if msg == "" {
		return cause
	}

	return msg + ": " + cause
}

// Unwrap returns the underlying error, or nil.
func (e *PathError) Unwrap() error {
	return e.err
}

// LogValue describes the paths in their Posix form, and the underlying error.
func (e *PathError) LogValue() slog.Value {
	attrs := []slog.Attr{slog.Any("paths", posixPaths(e.paths))}
	if e.err != nil {
		attrs = append(attrs, slog.Attr{Key: causeKey, Value: causeLogValue(e.err)})
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

// causeText renders the underlying error without the paths it repeats. An
// [fs.PathError] on the single path renders as its operation and reason, such
// as "open: no such file or directory". An [os.LinkError] on the two paths
// renders the same way, and one whose new path is the single path keeps its
// old path, such as "symlink a: file exists". The operation and the reason of
// such a text are trimmed, and runs of whitespace in them become one space.
// The old path stays as it is, since a name may hold any whitespace. Any other
// error renders as its own text.
func (e *PathError) causeText() string {
	// The underlying error itself is checked, not its chain, because a wrapped
	// error such as a nested failure renders its own paths.
	switch err := e.err.(type) { //nolint:errorlint // The chain must not be searched.
	case *fs.PathError:
		if len(e.paths) == 1 && err.Path == e.paths[0].String() {
			return compactText(err.Op) + ": " + compactText(err.Err.Error())
		}
	case *os.LinkError:
		if len(e.paths) == 2 && err.Old == e.paths[0].String() && err.New == e.paths[1].String() {
			return compactText(err.Op) + ": " + compactText(err.Err.Error())
		}
		if len(e.paths) == 1 && err.New == e.paths[0].String() {
			return compactText(err.Op) + " " + err.Old + ": " + compactText(err.Err.Error())
		}
	}

	return e.err.Error()
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

// compactText trims text and turns every run of whitespace into one space.
func compactText(text string) string {
	return strings.Join(strings.Fields(text), " ")
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

// pathErr returns a failure of kind that concerns paths.
func pathErr(kind *PathlibError, paths ...Path) error {
	return wrapError(kind, &PathError{paths: slices.Clone(paths)})
}

// wrapErr returns a failure of kind that concerns paths and is caused by cause.
func wrapErr(kind *PathlibError, cause error, paths ...Path) error {
	return wrapError(kind, &PathError{paths: slices.Clone(paths), err: cause})
}
