// core_error.go holds the error model: the error kind type ErrorKind,
// wrappedError that gives a cause an error kind, and their descriptions for
// logs and JSON.

package pathlib

import (
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strconv"
)

// causeKey is the key of the cause in the log description of an error and of
// a cause.
const causeKey = "cause"

// codePattern is the format of one name in a code, such as not_exist.
var codePattern = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)

// ErrorKind classifies an error. Its values are error kinds, such as
// [ErrNotAbsolute], and error kinds form a tree rooted at [ErrPathlib].
//
// Every error the library returns has an error kind. [errors.Is] matches an
// error kind and every one of its ancestors, so errors.Is(err, ErrPathlib)
// catches every error of the library. [errors.As] with a *ErrorKind target
// returns the error kind of an error. The data of an error, such as its paths,
// lives in its cause, such as a [PathError].
//
// An error has the most specific error kind the library knows. An error kind
// that names a condition, such as a missing path, wins over the error kind of
// the failed operation. The error of the operating system stays in the cause,
// so errors.Is also matches standard library sentinels such as
// [fs.ErrNotExist].
//
// Each error kind has a code, such as "pathlib.exist.file", that stays stable
// when its message changes. Logs, metrics, and alerts key on it.
//
// Error kinds are package-level values and are immutable.
type ErrorKind struct {
	// parent is the error kind group the error kind belongs to, or nil for
	// the root error kind.
	parent *ErrorKind

	// code joins the names of the error kind and its ancestors with dots.
	code string

	// message describes the error.
	message string
}

// wrappedError is the only error type the package returns. It gives the cause
// of one error an error kind.
type wrappedError struct {
	// kind classifies the error. It is never nil.
	kind *ErrorKind

	// cause holds the data of the error, or is nil.
	cause error
}

// causeJSON is the JSON form of a cause.
type causeJSON struct {
	// Message is the text of the cause.
	Message string `json:"message"`

	// Details is the JSON form of the cause, if its type has one.
	Details json.RawMessage `json:"details,omitempty"`

	// Cause is the error the cause wraps, if the cause has no JSON form and an
	// error below has one.
	Cause *causeJSON `json:"cause,omitempty"`

	// Errors are the errors the cause joins, if the cause has no JSON form and
	// an error below has one.
	Errors []causeJSON `json:"errors,omitempty"`
}

// defineError declares an error kind below parent. The code of the error kind
// is the code of parent and code, joined by a dot.
func defineError(parent *ErrorKind, code, message string) *ErrorKind {
	// Error kinds are defined during package initialization, so a malformed
	// code panics before any caller runs.
	if !codePattern.MatchString(code) {
		panic("pathlib.defineError: code must match " + codePattern.String() + ", got " + strconv.Quote(code))
	}

	definedCode := code
	if parent != nil {
		definedCode = parent.code + "." + definedCode
	}

	return &ErrorKind{parent: parent, code: definedCode, message: message}
}

// wrapError returns an error of kind kind, caused by cause. A nil cause returns
// an error of kind kind alone. wrapError is the only way the package creates
// errors.
func wrapError(kind *ErrorKind, cause error) error {
	return &wrappedError{kind: kind, cause: cause}
}

// newCauseJSON describes cause by its text, and by its own JSON when its type
// has one. A cause without a JSON form, such as one of [fmt.Errorf] or
// [errors.Join], adds the error it wraps as Cause, or the errors it joins as
// Errors, when an error below has a JSON form.
func newCauseJSON(cause error) causeJSON {
	encoded := causeJSON{Message: cause.Error()}
	marshaler, ok := cause.(json.Marshaler)
	if ok {
		details, err := json.Marshal(marshaler)
		if err == nil {
			encoded.Details = details
		}

		return encoded
	}

	// The errors below are described only when one of them describes itself.
	var described json.Marshaler
	if !errors.As(cause, &described) {
		return encoded
	}

	below, joined := causesBelow(cause)
	switch {
	case joined:
		encoded.Errors = make([]causeJSON, 0, len(below))
		for _, member := range below {
			encoded.Errors = append(encoded.Errors, newCauseJSON(member))
		}
	case len(below) == 1:
		inner := newCauseJSON(below[0])
		encoded.Cause = &inner
	}

	return encoded
}

// Code returns the stable identifier of the error kind, such as
// "pathlib.exist.file".
func (k *ErrorKind) Code() string {
	return k.code
}

// Message returns the description of the error kind.
func (k *ErrorKind) Message() string {
	return k.message
}

// Error returns the message followed by the code in parentheses.
func (k *ErrorKind) Error() string {
	return k.message + " (" + k.code + ")"
}

// Unwrap returns the parent error kind, or nil for the root error kind, so
// [errors.Is] matches every ancestor of the error kind.
func (k *ErrorKind) Unwrap() error {
	if k.parent == nil {
		return nil
	}

	return k.parent
}

// Error returns the text of the error kind followed by the text of the cause.
func (e *wrappedError) Error() string {
	if e.cause == nil {
		return e.kind.Error()
	}

	return e.kind.Error() + ": " + e.cause.Error()
}

// Unwrap returns the cause, so the chain is a straight line.
func (e *wrappedError) Unwrap() error {
	return e.cause
}

// Is matches the error kind and its ancestors.
func (e *wrappedError) Is(target error) bool {
	return errors.Is(e.kind, target)
}

// As reaches the error kind and its ancestors.
func (e *wrappedError) As(target any) bool {
	return errors.As(e.kind, target)
}

// LogValue describes the error kind, the code, and the cause.
func (e *wrappedError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("kind", e.kind.message),
		slog.String("code", e.kind.code),
	}
	if e.cause != nil {
		attrs = append(attrs, slog.Attr{Key: causeKey, Value: causeLogValue(e.cause)})
	}

	return slog.GroupValue(attrs...)
}

// MarshalJSON describes the error kind, the code, and the cause.
func (e *wrappedError) MarshalJSON() ([]byte, error) {
	var cause *causeJSON
	if e.cause != nil {
		encoded := newCauseJSON(e.cause)
		cause = &encoded
	}

	return json.Marshal(struct {
		Kind  string     `json:"kind"`
		Code  string     `json:"code"`
		Cause *causeJSON `json:"cause,omitempty"`
	}{Kind: e.kind.message, Code: e.kind.code, Cause: cause})
}

// causeLogValue describes cause by its text, and by its own description when
// its type has one. A cause without a description, such as one of
// [fmt.Errorf] or [errors.Join], adds the error it wraps as cause, or the
// errors it joins as errors keyed by their index, when an error below has a
// description.
func causeLogValue(cause error) slog.Value {
	attrs := []slog.Attr{slog.String("message", cause.Error())}
	valuer, ok := cause.(slog.LogValuer)
	if ok {
		return slog.GroupValue(append(attrs, slog.Any("details", valuer))...)
	}

	// The errors below are described only when one of them describes itself.
	var described slog.LogValuer
	if !errors.As(cause, &described) {
		return slog.GroupValue(attrs...)
	}

	below, joined := causesBelow(cause)
	switch {
	case joined:
		members := make([]slog.Attr, 0, len(below))
		for i, member := range below {
			members = append(members, slog.Attr{Key: strconv.Itoa(i), Value: causeLogValue(member)})
		}
		attrs = append(attrs, slog.Attr{Key: "errors", Value: slog.GroupValue(members...)})
	case len(below) == 1:
		attrs = append(attrs, slog.Attr{Key: causeKey, Value: causeLogValue(below[0])})
	}

	return slog.GroupValue(attrs...)
}

// causesBelow returns the errors cause wraps and reports whether cause joins
// them, as [errors.Join] does. A cause that wraps a single error, as
// [fmt.Errorf] with one %w does, returns it alone. Nil errors are left out.
func causesBelow(cause error) ([]error, bool) {
	joiner, ok := cause.(interface{ Unwrap() []error })
	if ok {
		joined := joiner.Unwrap()
		members := make([]error, 0, len(joined))
		for _, member := range joined {
			if member != nil {
				members = append(members, member)
			}
		}

		return members, true
	}

	wrapper, ok := cause.(interface{ Unwrap() error })
	if ok {
		inner := wrapper.Unwrap()
		if inner != nil {
			return []error{inner}, false
		}
	}

	return nil, false
}
