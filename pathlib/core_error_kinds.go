// core_error_kinds.go holds the root of the kind tree and the kinds raised by
// the core group (lexical path operations).

package pathlib

/*
The kind tree is declared by the group that raises its kinds, so each bundle
stays self-contained. The root and the core kinds live here. The kinds of the
fs and io groups live in fs_error_kinds.go and io_error_kinds.go.
*/
var (
	// ErrPathlib matches every error of the package. Its cause is a
	// *[PathError], unless a kind below it names another.
	ErrPathlib = defineError(nil, "PATHLIB", "pathlib error")

	// ErrEmptyPattern is returned when a match pattern is empty. The pattern is
	// the only path.
	ErrEmptyPattern = defineError(ErrPathlib, "EMPTY_PATTERN", "pattern may not be empty")

	// ErrBadPattern is returned when a match pattern is malformed. The pattern is
	// the only path, and the underlying error is [path.ErrBadPattern].
	ErrBadPattern = defineError(ErrPathlib, "BAD_PATTERN", "malformed pattern")

	// ErrAnchorMismatch is returned when one path is Windows-anchored and the
	// other is not, or their anchors differ.
	ErrAnchorMismatch = defineError(
		ErrPathlib, "ANCHOR_MISMATCH", "paths require an equal anchor when one is Windows-anchored",
	)

	// ErrNotAbsolute is returned when an operation requires an absolute path.
	ErrNotAbsolute = defineError(ErrPathlib, "NOT_ABSOLUTE", "path must be absolute")

	// ErrRelImpossible is returned when one path cannot be made relative to
	// another.
	ErrRelImpossible = defineError(ErrPathlib, "REL_IMPOSSIBLE", "cannot make path relative to the other")

	// ErrLookup is returned when a well-known directory, such as the working or
	// home directory, cannot be determined. Its cause is the error of the os
	// package.
	ErrLookup = defineError(ErrPathlib, "LOOKUP", "could not look up directory")
)
