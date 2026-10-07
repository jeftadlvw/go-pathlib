// core_error_pattern.go holds PatternError, the cause of every failure below
// ErrInvalidPattern.

package pathlib

import (
	"encoding/json"
	"log/slog"
	"slices"
	"strconv"
)

// PatternError is the cause of every failure below [ErrInvalidPattern]. It
// holds the rejected pattern, the paths the failure concerns, and the error
// that caused it, such as [path.ErrBadPattern].
//
// Read it with [errors.As]:
//
//	var patternErr *PatternError
//	if errors.As(err, &patternErr) { use(patternErr.Pattern()) }
//
// A PatternError is immutable.
type PatternError struct {
	// paths are the paths the failure concerns, the matched path or the
	// directory a glob starts in. May be empty.
	paths []Path

	// pattern is the rejected pattern, as the caller passed it.
	pattern string

	// err is the underlying error, or nil.
	err error
}

// Paths returns a copy of the paths the failure concerns. It may be empty.
func (e *PatternError) Paths() []Path {
	return slices.Clone(e.paths)
}

// Pattern returns the rejected pattern, as the caller passed it.
func (e *PatternError) Pattern() string {
	return e.pattern
}

// Error renders the paths, the pattern, and the underlying error.
func (e *PatternError) Error() string {
	msg := "(pattern " + strconv.Quote(e.pattern) + ")"

	paths := formatPaths(e.paths)
	if paths != "" {
		msg = paths + " " + msg
	}

	if e.err != nil {
		msg += ": " + e.err.Error()
	}

	return msg
}

// Unwrap returns the underlying error, or nil.
func (e *PatternError) Unwrap() error {
	return e.err
}

// LogValue describes the paths in their Posix form, the pattern, and the
// underlying error.
func (e *PatternError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.Any("paths", posixPaths(e.paths)),
		slog.String("pattern", e.pattern),
	}
	if e.err != nil {
		attrs = append(attrs, slog.Attr{Key: causeKey, Value: causeLogValue(e.err)})
	}

	return slog.GroupValue(attrs...)
}

// MarshalJSON describes the paths in their Posix form, the pattern, and the
// underlying error.
func (e *PatternError) MarshalJSON() ([]byte, error) {
	var cause *causeJSON
	if e.err != nil {
		encoded := newCauseJSON(e.err)
		cause = &encoded
	}

	return json.Marshal(struct {
		Paths   []string   `json:"paths"`
		Pattern string     `json:"pattern"`
		Cause   *causeJSON `json:"cause,omitempty"`
	}{Paths: posixPaths(e.paths), Pattern: e.pattern, Cause: cause})
}

// patternErr returns a failure of kind for pattern, caused by cause, which may
// be nil.
func patternErr(kind *PathlibError, pattern string, cause error, paths ...Path) error {
	return wrapError(kind, &PatternError{paths: slices.Clone(paths), pattern: pattern, err: cause})
}
