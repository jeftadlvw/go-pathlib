package pathlib

import (
	"errors"
	"strings"
)

/*
PathlibError is the root error type returned by every function in this library.
Every error the library produces is a *PathlibError, so a single
errors.As(err, new(*pathlib.PathlibError)) catches any of them, and
errors.Is(err, ErrX) matches the exported sentinels by identity.

The base type lives here in pathlib.go because every other file already depends
on it; this keeps the library's one-file rule intact (no central error file).
Domain sentinels (ErrNotExist, ErrNotDir, ...) are declared next to the code
that raises them.
*/
type PathlibError struct {
	// Kind categorizes the error. It is always set and is exposed to errors.Is
	// via Unwrap, so callers match on it by identity.
	Kind error

	// Paths are the paths this error concerns, in an order documented per
	// operation (e.g. Copy reports [source, destination]). May be empty.
	Paths []Path

	// Err is the underlying cause (e.g. a wrapped os error), or nil.
	Err error
}

// Error renders the kind, any paths, and the wrapped cause as a single message.
func (e *PathlibError) Error() string {
	var b strings.Builder

	if e.Kind != nil {
		b.WriteString(e.Kind.Error())
	} else {
		b.WriteString("pathlib error")
	}

	switch len(e.Paths) {
	case 0:
	case 1:
		b.WriteString(": ")
		b.WriteString(e.Paths[0].String())
	default:
		b.WriteString(": [")
		for i := range e.Paths {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(e.Paths[i].String())
		}
		b.WriteString("]")
	}

	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}

	return b.String()
}

// Unwrap exposes the kind and the cause as separate branches so the standard
// errors.Is / errors.As reach both by identity.
func (e *PathlibError) Unwrap() []error {
	switch {
	case e.Kind != nil && e.Err != nil:
		return []error{e.Kind, e.Err}
	case e.Err != nil:
		return []error{e.Err}
	case e.Kind != nil:
		return []error{e.Kind}
	default:
		return nil
	}
}

// pathErr builds a categorized error over zero or more paths.
func pathErr(kind error, paths ...Path) *PathlibError {
	return &PathlibError{Kind: kind, Paths: paths}
}

// wrapErr is pathErr with an underlying cause attached.
func wrapErr(kind, cause error, paths ...Path) *PathlibError {
	return &PathlibError{Kind: kind, Err: cause, Paths: paths}
}

/*
kindError is a sentinel that belongs to a broader parent sentinel. It lets a
specific error kind (e.g. ErrFileExist) be matched either precisely or by its
group (e.g. ErrExist), because errors.Is walks the parent through Unwrap.

Sentinels remain lightweight category markers; the per-call paths and wrapped
cause live on the *PathlibError instance, not on the shared sentinel value.
*/
type kindError struct {
	msg    string
	parent error
}

func (e *kindError) Error() string { return e.msg }
func (e *kindError) Unwrap() error { return e.parent }

// subKind declares a sentinel that is a member of a broader parent group, so
// that errors.Is(subKind(parent, ...), parent) reports true.
func subKind(parent error, msg string) error {
	return &kindError{msg: msg, parent: parent}
}

// Error sentinels raised by pathlib.go (lexical path operations).
var (
	// ErrEmptyPattern is returned when a match pattern is empty.
	ErrEmptyPattern = errors.New("pattern may not be empty")

	// ErrAnchorMismatch is returned when one path is Windows-anchored and the
	// other is not, or their anchors differ.
	ErrAnchorMismatch = errors.New("paths require an equal anchor when one is Windows-anchored")

	// ErrNotAbsolute is returned when an operation requires an absolute path.
	ErrNotAbsolute = errors.New("path must be absolute")

	// ErrRelImpossible is returned when one path cannot be made relative to another.
	ErrRelImpossible = errors.New("cannot make path relative to the other")
)
