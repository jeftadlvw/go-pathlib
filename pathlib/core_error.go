// core_error.go holds the error model: the kind type PathlibError, the failure
// type raisedError that pairs a kind with its cause, and their descriptions for
// logs and JSON.

package pathlib

import (
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strconv"
)

// causeKey is the key of the cause in the log description of a failure and of
// a cause.
const causeKey = "cause"

// codePattern is the format of one name in a code, such as NOT_EXIST.
var codePattern = regexp.MustCompile(`^[A-Z]+(_[A-Z]+)*$`)

/*
PathlibError classifies a failure. Its values are kinds, such as [ErrNotExist],
and kinds form a tree rooted at [ErrPathlib].

Every error the library returns has a kind. [errors.Is] matches a kind and every
one of its ancestors, so errors.Is(err, ErrPathlib) catches every error of the
library. [errors.As] with a *PathlibError target returns the kind of a failure.
The data of a failure, such as its paths, lives in its cause. See [PathError],
[PatternError], and [PermissionError].

A failure has the most specific kind the library knows. A missing path is
[ErrNotExist], whether a check of the library or the operating system found it.
The kind of an operation, such as [ErrRead], describes the failures no other
kind describes. The error of the operating system stays in the cause, so
errors.Is also matches standard library sentinels such as [fs.ErrNotExist].

Each kind has a code, such as "PATHLIB.EXIST.FILE", that stays stable when its
message changes. Logs, metrics, and alerts key on it.

Kinds are package-level values and are immutable.
*/
//nolint:revive // The name is part of the published API.
type PathlibError struct {
	// parent is the group the kind belongs to, or nil for the root.
	parent *PathlibError

	// code joins the names of the kind and its ancestors with dots.
	code string

	// message describes the failure.
	message string
}

// raisedError is the only error type the package returns. It pairs a kind with
// the cause of one failure.
type raisedError struct {
	// kind classifies the failure. It is never nil.
	kind *PathlibError

	// cause holds the data of the failure, or is nil.
	cause error
}

// causeJSON is the JSON form of a cause.
type causeJSON struct {
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

// defineError declares a kind below parent. The code of the kind is the code of
// parent and code, joined by a dot.
func defineError(parent *PathlibError, code, message string) *PathlibError {
	// Kinds are defined during package initialization, so a malformed code
	// panics before any caller runs.
	if !codePattern.MatchString(code) {
		panic("pathlib.defineError: code must match " + codePattern.String() + ", got " + strconv.Quote(code))
	}

	definedCode := code
	if parent != nil {
		definedCode = parent.code + "." + definedCode
	}

	return &PathlibError{parent: parent, code: definedCode, message: message}
}

// raiseError returns a failure of kind caused by cause. It is the only way the
// package creates errors.
func raiseError(kind *PathlibError, cause error) error {
	return &raisedError{kind: kind, cause: cause}
}

// Code returns the stable identifier of the kind, such as "PATHLIB.EXIST.FILE".
func (k *PathlibError) Code() string {
	return k.code
}

// Message returns the description of the kind.
func (k *PathlibError) Message() string {
	return k.message
}

// Error returns the message followed by the code in round brackets.
func (k *PathlibError) Error() string {
	return k.message + " (" + k.code + ")"
}

// Unwrap returns the parent kind, or nil for the root, so [errors.Is] matches
// every ancestor of the kind.
func (k *PathlibError) Unwrap() error {
	if k.parent == nil {
		return nil
	}

	return k.parent
}

// Error returns the text of the kind followed by the text of the cause.
func (e *raisedError) Error() string {
	if e.cause == nil {
		return e.kind.Error()
	}

	return e.kind.Error() + ": " + e.cause.Error()
}

// Unwrap returns the cause, so the chain is a straight line.
func (e *raisedError) Unwrap() error {
	return e.cause
}

// Is matches the kind and its ancestors.
func (e *raisedError) Is(target error) bool {
	return errors.Is(e.kind, target)
}

// As reaches the kind and its ancestors.
func (e *raisedError) As(target any) bool {
	return errors.As(e.kind, target)
}

// LogValue describes the kind, the code, and the cause.
func (e *raisedError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("kind", e.kind.message),
		slog.String("code", e.kind.code),
	}
	if e.cause != nil {
		attrs = append(attrs, slog.Attr{Key: causeKey, Value: causeLogValue(e.cause)})
	}

	return slog.GroupValue(attrs...)
}

// MarshalJSON describes the kind, the code, and the cause.
func (e *raisedError) MarshalJSON() ([]byte, error) {
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
// its type has one.
func causeLogValue(cause error) slog.Value {
	attrs := []slog.Attr{slog.String("message", cause.Error())}
	valuer, ok := cause.(slog.LogValuer)
	if ok {
		attrs = append(attrs, slog.Any("details", valuer))
	}

	return slog.GroupValue(attrs...)
}

// newCauseJSON describes cause by its text, and by its own JSON when its type
// has one.
func newCauseJSON(cause error) causeJSON {
	encoded := causeJSON{Message: cause.Error()}
	marshaler, ok := cause.(json.Marshaler)
	if ok {
		details, err := json.Marshal(marshaler)
		if err == nil {
			encoded.Details = details
		}
	}

	return encoded
}
