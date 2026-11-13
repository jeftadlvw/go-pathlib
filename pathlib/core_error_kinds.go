// core_error_kinds.go holds the root of the error kind tree, its family groups,
// and the error kinds returned by the core group (lexical path operations).

package pathlib

// The root of the error kind tree, the groups of its families, and the error
// kinds returned by the core group.
//
// An error kind belongs to one of three families. Error kinds below
// [ErrInvalid] reject a value the caller passed. Error kinds such as
// [ErrRelImpossible] name a condition the library found. Error kinds below
// [ErrOperation] name the operation of the operating system that failed for a
// reason no condition describes.
var (
	// ErrPathlib matches every error of the package. Its cause is a
	// *[PathError], unless an error kind below it names another.
	ErrPathlib = defineError(nil, "pathlib", "pathlib error")

	// ErrInvalid is the group of the error kinds that reject a value passed to
	// the library, such as a pattern or an option. Match it with [errors.Is] to
	// catch every error kind below.
	ErrInvalid = defineError(ErrPathlib, "invalid", "invalid value")

	// ErrInvalidPattern is the group of the error kinds that reject a match
	// pattern. It is a member of the ErrInvalid group. Its cause is a
	// *[PatternError].
	ErrInvalidPattern = defineError(ErrInvalid, "pattern", "invalid pattern")

	// ErrEmptyPattern is returned when a match pattern is empty. It is a
	// member of the ErrInvalidPattern group.
	ErrEmptyPattern = defineError(ErrInvalidPattern, "empty", "pattern may not be empty")

	// ErrBadPattern is returned when a match pattern is malformed. It is a
	// member of the ErrInvalidPattern group. The underlying error of its
	// PatternError is [path.ErrBadPattern].
	ErrBadPattern = defineError(ErrInvalidPattern, "malformed", "malformed pattern")

	// ErrNotAbsolute is returned when an operation requires an absolute path.
	// It is a member of the ErrInvalid group.
	ErrNotAbsolute = defineError(ErrInvalid, "not_absolute", "path must be absolute")

	// ErrRelImpossible is returned when one path cannot be made relative to
	// another. Match it with errors.Is to catch every error kind below.
	ErrRelImpossible = defineError(ErrPathlib, "rel_impossible", "cannot make path relative to the other")

	// ErrAnchorMismatch is returned when one path is Windows-anchored and the
	// other is not, or their anchors differ. It is a member of the
	// ErrRelImpossible group.
	ErrAnchorMismatch = defineError(
		ErrRelImpossible, "anchor_mismatch", "paths require an equal anchor when one is Windows-anchored",
	)

	// ErrOperation is the group of the error kinds that name a failed operation
	// of the operating system. An error has such an error kind when no
	// condition, such as a missing path, describes it. Match it with errors.Is
	// to catch every error kind below.
	ErrOperation = defineError(ErrPathlib, "operation", "operation failed")

	// ErrLookup is returned when a well-known directory, such as the working
	// or home directory, cannot be determined. It is a member of the
	// ErrOperation group. Its cause is the error of the os package.
	ErrLookup = defineError(ErrOperation, "lookup", "could not look up directory")
)
