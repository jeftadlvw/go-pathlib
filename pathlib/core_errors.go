package pathlib

import (
	"errors"
	"slices"
	"strings"
)

/*
PathlibError is the root error type returned by every function in this library.
Every error the library produces is a *PathlibError, so a single
errors.As(err, new(*pathlib.PathlibError)) catches any of them, and
errors.Is(err, ErrX) matches the exported sentinels by identity.

A PathlibError is immutable. Its fields are read through Kind, Paths, and Unwrap.

The base type lives in the core group because every other group depends on it,
so each bundle stays self-contained. Domain sentinels are declared in the
*_errors.go file of the group that raises them (core_errors.go, fs_errors.go,
io_errors.go, temp_errors.go).
*/
type PathlibError struct {
	// kind categorizes the error. It is always set and is matched by the Is
	// method, so errors.Is reaches it by identity.
	kind error

	// paths are the paths this error concerns, in an order documented per
	// operation (e.g. Copy reports [source, destination]). May be empty.
	paths []Path

	// err is the underlying cause (e.g. a wrapped os error), or nil.
	err error
}

// Kind returns the sentinel that categorizes the error. Match it with errors.Is.
func (e *PathlibError) Kind() error {
	return e.kind
}

// Paths returns a copy of the paths this error concerns, in an order documented
// per operation (e.g. Copy reports [source, destination]). It may be empty.
func (e *PathlibError) Paths() []Path {
	return slices.Clone(e.paths)
}

// Error renders the kind, any paths, and the wrapped cause as a single message.
func (e *PathlibError) Error() string {
	var b strings.Builder

	if e.kind != nil {
		b.WriteString(e.kind.Error())
	} else {
		b.WriteString("pathlib error")
	}

	switch len(e.paths) {
	case 0:
	case 1:
		b.WriteString(": ")
		b.WriteString(e.paths[0].String())
	default:
		b.WriteString(": [")
		for i := range e.paths {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(e.paths[i].String())
		}
		b.WriteString("]")
	}

	if e.err != nil {
		b.WriteString(": ")
		b.WriteString(e.err.Error())
	}

	return b.String()
}

// Unwrap returns the cause, or nil. The kind is no cause, so errors.Unwrap
// follows a linear chain.
func (e *PathlibError) Unwrap() error {
	return e.err
}

// Is reports whether target matches the kind, so errors.Is finds kinds and
// their groups as well as causes.
func (e *PathlibError) Is(target error) bool {
	return e.kind != nil && errors.Is(e.kind, target)
}

// pathErr builds a categorized error over zero or more paths.
func pathErr(kind error, paths ...Path) *PathlibError {
	return &PathlibError{kind: kind, paths: paths}
}

// wrapErr is pathErr with an underlying cause attached.
func wrapErr(kind, cause error, paths ...Path) *PathlibError {
	return &PathlibError{kind: kind, err: cause, paths: paths}
}

/*
kindError is a sentinel that belongs to a broader parent sentinel. It lets a
specific error kind (e.g. ErrFileExist) be matched either precisely or by its
group (e.g. ErrExist), because errors.Is walks the parent through Unwrap.

Sentinels remain lightweight category markers. The per-call paths and wrapped
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

// Error sentinels raised by the core group (lexical path operations).
var (
	// ErrEmptyPattern is returned when a match pattern is empty.
	ErrEmptyPattern = errors.New("pattern may not be empty")

	// ErrBadPattern is returned when a match pattern is malformed. The cause is
	// path.ErrBadPattern.
	ErrBadPattern = errors.New("malformed pattern")

	// ErrAnchorMismatch is returned when one path is Windows-anchored and the
	// other is not, or their anchors differ.
	ErrAnchorMismatch = errors.New("paths require an equal anchor when one is Windows-anchored")

	// ErrNotAbsolute is returned when an operation requires an absolute path.
	ErrNotAbsolute = errors.New("path must be absolute")

	// ErrRelImpossible is returned when one path cannot be made relative to another.
	ErrRelImpossible = errors.New("cannot make path relative to the other")

	// ErrLookup is returned when a well-known directory, such as the working or
	// home directory, cannot be determined. The os cause is wrapped.
	ErrLookup = errors.New("could not look up directory")
)
