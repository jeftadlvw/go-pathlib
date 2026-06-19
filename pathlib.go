/*
Package pathlib contains source code for go-pathlib.

It's a one-file library that can be used in other projects by using Go's package system
or by placing the source code file itself into the source tree.

pathlib.go contains lexicographically based functions and does not interoperate with the
file system. Case sensitivity is defined explicitly. Filesystem-specific functionality is outsourced to pathlib_fs.go.

Use pathlib_fs.go, pathlib_io.go or pathlib_temp.go for more interoperability.
*/
package pathlib

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

type CompareOption bool

const (
	CaseSensitive   CompareOption = true
	CaseInsensitive CompareOption = false
)

const (
	// pathCheckNoExistOrUnreadable indicates that the checked Path does not exist.
	pathCheckNoExistOrUnreadable = iota

	// pathCheckFile indicates that the checked Path is a file.
	pathCheckFile

	// pathCheckDir indicates that the checked Path is a directory.
	pathCheckDir
)

const runningOnWindows = runtime.GOOS == "windows"
const notRunningOnWindows = !runningOnWindows

const osPathSeparator = string(os.PathSeparator)
const canonicalPathSeparator = "/"
const posixPathSeparator = "/"
const windowsPathSeparator = "\\"

// Regexes use forward slash instead of backwards slash, because we assume forward slash input
var windowsVolumeNameRegex = regexp.MustCompile("^[A-Za-z]:(/|$)")
var windowsNetworkPathRegex = regexp.MustCompile("^//[a-zA-Z0-9]+/[a-zA-Z0-9]+")

var multipleWindowsPathSeparatorsRegex = regexp.MustCompile(`\\{2,}`)

// PrintBackslashWarningOnPosix is a toggle for printing a warning on Posix
// if a path string contains a backslash.
var PrintBackslashWarningOnPosix = true

// Windows path state bitmask values for windowsPathEncodings.
const (
	// windowsPathStateVolume indicates a Windows path anchored to a drive volume (e.g. "C:\foo").
	windowsPathStateVolume uint8 = 1 << iota

	// windowsPathStateUNC indicates a Windows UNC path (e.g. "\\server\share\foo").
	windowsPathStateUNC
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

/*
Path is a struct that represents a filesystem path.

Create a new instance using NewPath() for paths coming from the operating
system, or NewPathFromPosix() / NewPathFromWindows() to interpret a string in a
specific format regardless of the runtime OS.
Other constructor functions are prefixed with 'New'.

Implements the fmt.Stringer interface.
*/
type Path struct {

	// The underlying filepath string representation in canonical (unix-style) form.
	// For Windows paths this holds only the portion after the anchor.
	path string

	// windowsAnchor holds the Windows-specific anchor in canonical (forward-slash) form.
	// "C:"           for volume paths  (windowsPathStateVolume)
	// "//host/share" for UNC paths     (windowsPathStateUNC)
	// ""             for all others
	windowsAnchor string

	// windowsPathEncodings is a bitmask of windowsPathState* flags.
	windowsPathEncodings uint8
}

/*
NewPath ensures correct internal state and behavior depending on the current
operating system. It is OS-adaptive: the same input may produce a different Path
on Windows than on Posix, so its result is platform-dependent by design.

It is meant to be used when handling file paths received by the operating system by
system calls or subprocesses.

It branches to either NewPathFromPosix or NewPathFromWindows. When the input
format is known ahead of time (serialization, cross-platform handling, tests),
prefer those format-explicit constructors so the result is deterministic across
platforms.

Rule of thumb: reach for NewPath only for strings handed to you by the operating
system. If you already hold a path in a known format (including the library's own
canonical posix form), use the format-explicit constructor instead.
*/
func NewPath(path string) *Path {
	if runningOnWindows {
		return NewPathFromWindows(path)
	}

	return NewPathFromPosix(path)
}

/*
NewPathFromPosix interprets the passed string as a Posix path, independent of the
runtime OS. A backslash is treated as an ordinary filename character, not a
separator. On Posix, a warning is printed to flag the cross-platform ambiguity.

The passed path string is automatically cleaned and ready for further use using the following rules:
  - Parts can include whitespaces wherever they want (leading, somewhere in between and ending).
  - Parts are separated by a single forward slash ("/").
  - Multiple forward slashes are replaced by one single slash.
  - Trailing forward slashes are removed.

Defined edge cases:
  - an empty string, "." and "./" results into "."
  - if all rules result into an empty string, the path also result into "."
  - ".." stays ".."
  - "/", "/.", and "/.." result into "/"

The path is not lowercased, because the path might be used on a case-sensitive filesystem.
Functions that are case-insensitive must additionally lowercase this representation.
*/
func NewPathFromPosix(path string) *Path {
	warnForBackslashesOnPosix(path)

	return &Path{path: normalizePath(path)}
}

/*
NewPathFromWindows interprets the passed string as a Windows path, independent of
the runtime OS. Use it to parse Windows-formatted strings on any platform (e.g.
when reading serialized paths on a Posix server).

It is effectively a superset of NewPathFromPosix. Backslashes are converted to the
canonical separator and Windows volume names (e.g. "C:") and UNC anchors (e.g.
"\\\\host\\share") are split off, after which the same normalization rules as
NewPathFromPosix apply to the remainder.
*/
func NewPathFromWindows(path string) *Path {
	warnForBackslashesOnPosix(path)
	return normalizeWindowsPath(path)
}

/*
NewCwd returns a new Path instance pointing to the application's current working directory.

This function uses os.Getwd.
*/
func NewCwd() (*Path, error) {
	cwdPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return NewPath(cwdPath), nil
}

/*
NewHome returns a new Path instance pointing to the user's home directory.

This function uses os.UserHomeDir.
*/
func NewHome() (*Path, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return NewPath(homePath), nil
}

/*
PathFromParts combines passed parts into a new Path.
*/
func PathFromParts(parts ...string) *Path {
	return NewPathFromPosix(".").JoinStrings(parts...)
}

/*
Parent returns a copy of this Path in the parent directory.

This function uses path.Dir.
*/
func (p *Path) Parent() *Path {
	return p.copyWithNewPath(path.Dir(p.path))
}

/*
Parts returns all single parts of the Path.
*/
func (p *Path) Parts() []string {
	localPath := p.path

	if localPath == "/" {
		return []string{}
	}

	split := strings.Split(localPath, canonicalPathSeparator)

	if len(split) > 0 && split[0] == "" {
		split = split[1:]
	}

	if p.isWindowsVolumeAnchoredPath() {
		split = slices.Insert(split, 0, p.windowsAnchor)
	}

	return split
}

/*
Split splits this Path into its parent and base.
*/
func (p *Path) Split() (*Path, string) {
	dir, file := path.Split(p.path)
	return p.copyWithNewPath(normalizePath(dir)), file
}

/*
Base returns the last element of this Path.

This function uses path.Base.
*/
func (p *Path) Base() string {
	if p.isWindowsAnchoredPath() && p.isLocalDirectory() {
		return p.windowsAnchor
	}

	return path.Base(p.path)
}

/*
Stem returns the base of this Path without all extensions.
*/
func (p *Path) Stem() string {
	completeBase := p.Base()

	if completeBase == "/" {
		return ""
	}

	baseStrippedLeading := stripLeadingDots(completeBase)

	dotIndex := strings.IndexAny(baseStrippedLeading, ".")
	if dotIndex == -1 {
		return completeBase
	}

	return completeBase[:(len(completeBase)-len(baseStrippedLeading))+dotIndex]
}

/*
HasExtensions returns whether this Path has file extensions.
*/
func (p *Path) HasExtensions() bool {
	return hasDots(stripLeadingDots(p.Base()))
}

/*
ExtensionCount returns the number of extensions this Path has.
*/
func (p *Path) ExtensionCount() int {
	return dotCount(stripLeadingDots(p.Base()))
}

/*
Extension returns the complete extension of this Path.
Any prefixed dots are included.

Everything starting from the first non-leading dot in this Path's Stem()
is considered to be an extension.
*/
func (p *Path) Extension() string {
	stem := p.Stem()

	// If no stem exists, then there also are no extensions
	if len(stem) == 0 {
		return ""
	}

	return p.Base()[len(stem):]
}

/*
ExtensionParts returns all this Path's extensions.

See Extension for what is considered an extension.
*/
func (p *Path) ExtensionParts() []string {
	base := stripLeadingDots(p.Base())
	return strings.Split(base, ".")[1:]
}

/*
Anchor returns the first part of the path in platform-native form.

On absolute paths this is the filesystem root ("/").
For Windows paths the volume name or UNC root is returned
(e.g. "C:" or "//host/share" on Posix, "C:" or "\\host\share" on Windows).

Relative paths don't have a defined anchor, "" is returned.
*/
func (p *Path) Anchor() string {
	if p.isWindowsAnchoredPath() {
		if runningOnWindows {
			return toWindowsSeparators(p.windowsAnchor)
		}
		return p.windowsAnchor
	}

	if p.IsRelative() {
		return ""
	}

	return "/"
}

/*
WindowsVolume returns the drive letter anchor of a Windows volume path
(e.g. "C:") in platform-native form.

Returns "" if this is not a volume-anchored path.
*/
func (p *Path) WindowsVolume() string {
	if !p.isWindowsVolumeAnchoredPath() {
		return ""
	}

	return p.windowsAnchor
}

/*
WindowsUncRoot returns the UNC root of a Windows network path
(e.g. "//host/share" on Posix, "\\host\share" on Windows) in platform-native form.

Returns "" if this is not a UNC-anchored path.
*/
func (p *Path) WindowsUncRoot() string {
	if !p.isWindowsUNCAnchoredPath() {
		return ""
	}

	if runningOnWindows {
		return toWindowsSeparators(p.windowsAnchor)
	}

	return p.windowsAnchor
}

/*
MatchesPatternE matches this Path's Posix representation against a pattern with support for double asterisk (**).
Returns whether the matching is successful or any occurring error.

Wraps path.Match with some custom rules for double asterisk support.
Use forward slashes as path separators.

By default, matching is case-sensitive. Pass CaseInsensitive to ignore casing.

Empty patterns cause an error.
*/
func (p *Path) MatchesPatternE(pattern string, opts ...CompareOption) (bool, error) {
	if pattern == "" {
		return false, pathErr(ErrEmptyPattern, *p)
	}

	caseSensitive := true
	if len(opts) > 0 {
		caseSensitive = bool(opts[0])
	}

	pathString := p.ToPosix()

	if !caseSensitive {
		pattern = strings.ToLower(pattern)
		pathString = strings.ToLower(pathString)
	}

	return matchPattern(pattern, pathString)
}

/*
MatchesPattern matches this Path against the provided pattern.

It wraps MatchesPatternE and returns the boolean success return value or false in case of an error.
*/
func (p *Path) MatchesPattern(pattern string, opts ...CompareOption) bool {
	match, err := p.MatchesPatternE(pattern, opts...)
	return match && err == nil
}

/*
IsAbsolute returns whether this Path is absolute.

This function uses path.IsAbs.
*/
func (p *Path) IsAbsolute() bool {
	if p.isWindowsUNCAnchoredPath() {
		return true
	}

	return path.IsAbs(p.path)
}

/*
IsRelative returns whether this Path is relative.

This function returns the inverse of IsAbsolute.
*/
func (p *Path) IsRelative() bool {
	return !p.IsAbsolute()
}

/*
RelativeTo returns this Path relative to another, so that
joining the other Path with the returned Path results in this Path:

	relativeTo := a.RelativeTo(b)	// what to apply on b to get to a
	assert b.Join(relativeTo) == a

The returned Path is always relative to this Path.

The operation is lexically. An error is returned if the other Path
can't be made relative to this Path or if knowing the current working directory
would be necessary to compute it.

  - / RelativeTo /a/b → ../..
  - /a RelativeTo /b → ../a
  - /a RelativeTo /a/b → ..
  - a/b RelativeTo ../b → error, other path escapes working directory
  - a/b/c RelativeTo a/x/y → ../../b/c
  - /a/b/c RelativeTo / → a/b/c

If one path has a Windows anchor, the other also needs one. Else an error is returned.
If the Windows anchor for both paths do not match, an error is returned.
*/
func (p *Path) RelativeTo(o *Path) (*Path, error) {
	if p.isWindowsAnchoredPath() || o.isWindowsAnchoredPath() {
		if p.windowsAnchor != o.windowsAnchor {
			return nil, pathErr(ErrAnchorMismatch, *p, *o)
		}
	}

	rp, err := relPath(o.path, p.path)

	if err != nil {
		return nil, pathErr(ErrRelImpossible, *p, *o)
	}

	// Use posix here as rp is relative and already posix style.
	return NewPathFromPosix(rp), nil
}

/*
RelativeFrom returns this Path relative from another, so that
joining this Path with the returned Path results in the other:

	relativeFrom := a.RelativeFrom(b)	// what to apply on a to get to b
	assert a.Join(relativeFrom) == b

The returned Path is always relative from this Path.

The operation is lexically. An error is returned if the other Path
can't be made relative from this Path or if knowing the current working directory
would be necessary to compute it.

  - /a/b RelativeFrom / → a/b
  - /a/b RelativeFrom /a → b
  - ../b RelativeFrom a/b → error, other path escapes working directory
  - a/x/y RelativeFrom a/b/c → ../../x/y
  - /a/b/c RelativeFrom / → ../../..

If one path has a Windows anchor, the other also needs one. Else an error is returned.
If the Windows anchor for both paths do not match, an error is returned.
*/
func (p *Path) RelativeFrom(o *Path) (*Path, error) {
	if p.isWindowsAnchoredPath() || o.isWindowsAnchoredPath() {
		if p.windowsAnchor != o.windowsAnchor {
			return nil, pathErr(ErrAnchorMismatch, *p, *o)
		}
	}

	rp, err := relPath(p.path, o.path)

	if err != nil {
		return nil, pathErr(ErrRelImpossible, *p, *o)
	}

	// Use posix here as rp will be relative and already posix style
	return NewPathFromPosix(rp), nil
}

/*
MakeAbsolute returns an absolute representation of this Path.
If the Path is relative, it will be joined with the current working directory.
If the Path is already absolute, a copy of the Path is returned.
*/
func (p *Path) MakeAbsolute() (*Path, error) {
	// If already absolute, return a copy
	if p.IsAbsolute() {
		return p.Copy(), nil
	}

	cwd, err := NewCwd()
	if err != nil {
		return nil, err
	}

	return cwd.Join(p), nil
}

/*
AbsoluteFrom returns an absolute representation of this Path towards another.

If the Path is relative, it will be joined with the provided Path,
else a copy of this Path is returned.

The other path must be absolute.

Requires the other Path to be absolute.
*/
func (p *Path) AbsoluteFrom(o *Path) (*Path, error) {

	// If this path is already absolute, return a copy
	if p.IsAbsolute() {
		return p.Copy(), nil
	}

	if o.IsRelative() {
		return nil, pathErr(ErrNotAbsolute, *o)
	}

	return o.Join(p), nil
}

/*
Join returns a new Path with all passed Path structs joined together.
Paths are not checked whether they are absolute or relative.

Use JoinStrings to join strings with this Path.

This function uses path.Join.
*/
func (p *Path) Join(paths ...*Path) *Path {
	pathsStr := make([]string, len(paths))
	for i, localPath := range paths {
		pathsStr[i] = localPath.path
	}

	return p.copyWithNewPath(path.Join(append([]string{p.path}, pathsStr...)...))
}

/*
JoinStrings returns a new Path with all passed strings joined together.

Each segment is interpreted as a Posix string (the library's canonical string
form). To join a Windows-formatted or OS-native string, parse it first and use Join, e.g.
p.Join(NewPathFromWindows(s)) or p.Join(NewPath(s)).
*/
func (p *Path) JoinStrings(paths ...string) *Path {
	for _, localPath := range paths {
		warnForBackslashesOnPosix(localPath)
	}

	return p.copyWithNewPath(path.Join(append([]string{p.path}, paths...)...))
}

/*
Equals returns whether this and another Path match lexically.
By default, comparison is case-sensitive.
*/
func (p *Path) Equals(other *Path, opts ...CompareOption) bool {
	caseSensitive := CaseSensitive
	if len(opts) > 0 {
		caseSensitive = opts[0]
	}

	if caseSensitive {
		return p.pathWithWindowsAnchor() == other.pathWithWindowsAnchor()
	}

	return strings.EqualFold(p.pathWithWindowsAnchor(), other.pathWithWindowsAnchor())
}

/*
EqualsString reports whether other denotes the same path as this Path.
By default, comparison is case-sensitive.

other is interpreted as a Posix path. Pair this with ToPosix(), not String().
String() returns the OS-native form and is a portability trap on Windows. To compare
against an OS-native string, or a Windows-formatted one, construct the operand
explicitly and use Equals(NewPath(s)) for an OS path, or Equals(NewPathFromWindows(s))
for a Windows path.
*/
func (p *Path) EqualsString(other string, opts ...CompareOption) bool {
	return p.Equals(NewPathFromPosix(other), opts...)
}

/*
WithName returns this Path but with another base.

name is interpreted as a Posix string, like JoinStrings: a backslash is an ordinary
filename character, not a separator, and passing one on Posix logs a warning.
*/
func (p *Path) WithName(name string) *Path {
	return p.Parent().JoinStrings(name)
}

/*
Copy creates a copy of this Path.
*/
func (p *Path) Copy() *Path {
	return &Path{
		path:                 p.path,
		windowsAnchor:        p.windowsAnchor,
		windowsPathEncodings: p.windowsPathEncodings,
	}
}

// copyWithNewPath creates a copy of this Path with a different path portion,
// preserving the Windows anchor fields.
func (p *Path) copyWithNewPath(newPath string) *Path {
	c := p.Copy()
	c.path = newPath
	return c
}

/*
String returns this Path in platform-native form.

On Windows this uses backslashes and prepends the anchor; on Posix it uses forward slashes.
Use ToPosix or ToWindows for an explicit representation.
*/
func (p *Path) String() string {
	if runningOnWindows {
		return p.ToWindows()
	}

	return p.ToPosix()
}

/*
ToPosix returns a string representation with forward slashes.
*/
func (p *Path) ToPosix() string {
	return p.pathWithWindowsAnchor()
}

/*
ToWindows returns a string representation with backward slashes.
*/
func (p *Path) ToWindows() string {
	pathWindows := multipleWindowsPathSeparatorsRegex.ReplaceAllString(
		toWindowsSeparators(p.path), windowsPathSeparator,
	)

	if p.isWindowsUNCAnchoredPath() {
		anchorWindows := toWindowsSeparators(p.windowsAnchor)
		if p.path == canonicalPathSeparator {
			return anchorWindows
		}
		return anchorWindows + pathWindows
	}

	if p.isWindowsVolumeAnchoredPath() {
		return p.windowsAnchor + pathWindows
	}

	return pathWindows
}

/*
TrimWindowsAnchor returns a copy of this Path with stripped
Windows anchor encoding information.
*/
func (p *Path) TrimWindowsAnchor() *Path {
	// p.path is already canonical posix (the anchor is held separately), so
	// interpret it as posix rather than re-running OS-dependent detection.
	return NewPathFromPosix(p.path)
}

/*
MarshalText marshals this Path's Posix representation into a byte array.
Implements the encoding.TextMarshaler interface.
*/
func (p *Path) MarshalText() (text []byte, err error) {
	return []byte(p.ToPosix()), nil
}

/*
UnmarshalText unmarshalls any byte array into a Path type.
Implements the encoding.TextUnmarshaler interface.

Uses NewPathFromWindows to ensure Windows anchors survive a marshal/unmarshal round-trip,
since MarshalText serializes them in forward-slash form (e.g. "c:/" or "//host/share").
*/
func (p *Path) UnmarshalText(text []byte) error {
	*p = *NewPathFromWindows(string(text))
	return nil
}

func (p *Path) pathWithWindowsAnchor() string {
	if p.isWindowsAnchoredPath() {
		return p.windowsAnchor + p.path
	}

	return p.path
}

// isWindowsAnchoredPath reports whether the path was constructed from a Windows-style path string.
func (p *Path) isWindowsAnchoredPath() bool {
	return p.windowsPathEncodings != 0
}

// isWindows reports whether the path is a Windows UNC path (e.g. "\\server\share").
func (p *Path) isWindowsVolumeAnchoredPath() bool {
	return p.windowsPathEncodings&windowsPathStateVolume != 0
}

// isWindowsUNCAnchoredPath reports whether the path is a Windows UNC path (e.g. "\\server\share").
func (p *Path) isWindowsUNCAnchoredPath() bool {
	return p.windowsPathEncodings&windowsPathStateUNC != 0
}

func toWindowsSeparators(s string) string {
	return strings.ReplaceAll(s, canonicalPathSeparator, windowsPathSeparator)
}

func toCanonicalSeparators(s string) string {
	return strings.ReplaceAll(s, windowsPathSeparator, canonicalPathSeparator)
}

/*
normalizePath creates a canonical representation of the passed path string.

It abstracts filesystem-specific specialties so that they can be easily compared or
extracted.

This function assumes that the passed path's part separator is "/" and uses path.Clean.

This function ensures:
- Parts can include whitespaces wherever they want (leading, somewhere in between and ending).
- Parts are separated by a single forward slash ("/").
- Multiple forward slashes are replaced by one single slash.
- Trailing forward slashes are removed.

Defined edge cases:
- an empty string, "." and "./" return "."
- an empty string after all filters also returns "."
- ".." returns ".."
- "/", "/.", and "/.." return "/"

The path is not lowercased, because the path might be used on a case-sensitive filesystem.
Functions that are case-insensitive must additionally lowercase this representation.
*/
func normalizePath(p string) string {
	return path.Clean(p)
}

// normalizeWindowsPath processes a Windows-style path string into a normalized Path.
//
// It handles volume paths (e.g. "C:\foo"), UNC paths (e.g. "\\host\share\foo"),
// and relative/absolute paths without a Windows-specific anchor.
//
// Windows path separator backslashes are replaced with the canonical forward slash separator
// before any other comparison operation.
func normalizeWindowsPath(p string) *Path {
	dirty := toCanonicalSeparators(p)

	// Volume anchor: "C:\" (rooted) or "C:" (drive-relative).
	if match := windowsVolumeNameRegex.FindString(dirty); match != "" {
		anchor := match[:2]        // "C:" (letter + colon)
		hasRoot := len(match) == 3 // true when match includes the trailing "/"

		rest := dirty[len(match):]
		cleanedRest := normalizePath(rest)

		// Strip any leading slashes that normalizePath may have returned.
		cleanedRest = strings.TrimLeft(cleanedRest, canonicalPathSeparator)

		// Remove ".." traversals that cannot go above the drive root.
		for {
			if cleanedRest == ".." {
				cleanedRest = ""
				break
			}
			if strings.HasPrefix(cleanedRest, "../") {
				cleanedRest = cleanedRest[3:]
				continue
			}
			if cleanedRest == "." {
				cleanedRest = ""
				break
			}
			break
		}

		var pathField string
		if hasRoot {
			if cleanedRest == "" {
				pathField = canonicalPathSeparator
			} else {
				pathField = canonicalPathSeparator + cleanedRest
			}
		} else {
			pathField = cleanedRest
		}

		return &Path{
			path:                 pathField,
			windowsAnchor:        anchor,
			windowsPathEncodings: windowsPathStateVolume,
		}
	}

	// UNC anchor: "//host/share".
	if match := windowsNetworkPathRegex.FindString(dirty); match != "" {
		rest := dirty[len(match):]
		if rest == "" || rest == canonicalPathSeparator {
			return &Path{
				path:                 canonicalPathSeparator,
				windowsAnchor:        match,
				windowsPathEncodings: windowsPathStateUNC,
			}
		}
		cleanedRest := normalizePath(rest)
		if cleanedRest == "." {
			cleanedRest = canonicalPathSeparator
		}
		return &Path{
			path:                 cleanedRest,
			windowsAnchor:        match,
			windowsPathEncodings: windowsPathStateUNC,
		}
	}

	// No Windows-specific anchor: treat as a regular path.
	return &Path{
		path:                 normalizePath(dirty),
		windowsPathEncodings: 0,
	}
}

// isLocalDirectory returns whether the current path is "."
func (p *Path) isLocalDirectory() bool {
	return p.path == "."
}

// relPath returns a relative path that is lexically equivalent to targPath when
// joined to basePath with an intervening separator. Both paths must use forward
// slashes and be already normalized.
//
// This function is inspired by filepath.Rel.
func relPath(basePath, targPath string) (string, error) {
	if basePath == targPath {
		return ".", nil
	}

	base := basePath
	targ := targPath

	if base == "." {
		base = ""
	}
	if targ == "." {
		targ = ""
	}

	baseSlashed := len(base) > 0 && base[0] == '/'
	targSlashed := len(targ) > 0 && targ[0] == '/'
	if baseSlashed != targSlashed {
		return "", ErrRelImpossible
	}

	bl := len(base)
	tl := len(targ)
	var b0, bi, t0, ti int
	for {
		for bi < bl && base[bi] != '/' {
			bi++
		}
		for ti < tl && targ[ti] != '/' {
			ti++
		}
		if base[b0:bi] != targ[t0:ti] {
			break
		}
		if bi < bl {
			bi++
		}
		if ti < tl {
			ti++
		}
		b0 = bi
		t0 = ti
	}

	if base[b0:bi] == ".." {
		return "", ErrRelImpossible
	}

	if b0 != bl {
		seps := strings.Count(base[b0:bl], "/")
		size := 2 + seps*3
		if tl != t0 {
			size += 1 + tl - t0
		}
		buf := make([]byte, size)
		n := copy(buf, "..")
		for i := 0; i < seps; i++ {
			buf[n] = '/'
			copy(buf[n+1:], "..")
			n += 3
		}
		if t0 != tl {
			buf[n] = '/'
			copy(buf[n+1:], targ[t0:])
		}
		return string(buf), nil
	}

	result := targ[t0:]
	if result == "" {
		return ".", nil
	}
	return result, nil
}

// matchPattern is the internal implementation that handles ** expansion.
func matchPattern(pattern, name string) (bool, error) {
	// Validate pattern for bad syntax (check each segment)
	if err := validatePattern(pattern); err != nil {
		return false, err
	}

	// If no **, use standard path.Match
	if !strings.Contains(pattern, "**") {
		return path.Match(pattern, name)
	}

	return matchWithDoubleAsterisk(pattern, name)
}

// validatePattern checks if the pattern has valid syntax.
func validatePattern(pattern string) error {
	// Split by ** and validate each segment with path.Match
	segments := strings.Split(pattern, "**")
	for _, seg := range segments {
		// Remove leading/trailing slashes for validation
		seg = strings.Trim(seg, "/")
		if seg == "" {
			continue
		}
		// Use path.Match to validate syntax (match against empty string just to check pattern validity)
		_, err := path.Match(seg, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// matchWithDoubleAsterisk handles patterns containing **.
func matchWithDoubleAsterisk(pattern, name string) (bool, error) {
	// Split pattern by **
	parts := strings.Split(pattern, "**")

	// Handle edge cases
	if len(parts) == 1 {
		// No ** found (shouldn't reach here, but safety check)
		return path.Match(pattern, name)
	}

	// For pattern like "**", it matches everything
	if pattern == "**" {
		return true, nil
	}

	// Process the pattern parts
	return matchParts(parts, name)
}

// matchParts matches the name against pattern parts split by **.
func matchParts(parts []string, name string) (bool, error) {
	// First part must match the beginning of name (if not empty)
	firstPart := parts[0]
	if firstPart != "" {
		// First part doesn't start with **, so it must match from the beginning
		firstPart = strings.TrimSuffix(firstPart, "/")
		if !matchesPrefix(name, firstPart) {
			return false, nil
		}
		// Calculate how much of name was consumed
		prefixLen := findPrefixMatchLength(name, firstPart)
		if prefixLen == -1 {
			return false, nil
		}
		name = name[prefixLen:]
		name = strings.TrimPrefix(name, "/")
	}

	// Last part must match the end of name (if not empty)
	lastPart := parts[len(parts)-1]
	if lastPart != "" {
		lastPart = strings.TrimPrefix(lastPart, "/")
		if !matchesSuffix(name, lastPart) {
			return false, nil
		}
		// Calculate how much of name remains
		suffixLen := findSuffixMatchLength(name, lastPart)
		if suffixLen == -1 {
			return false, nil
		}
		name = name[:len(name)-suffixLen]
		name = strings.TrimSuffix(name, "/")
	}

	// Middle parts must appear in order somewhere in name
	for i := 1; i < len(parts)-1; i++ {
		middlePart := strings.Trim(parts[i], "/")
		if middlePart == "" {
			continue
		}

		idx := findPatternInPath(name, middlePart)
		if idx == -1 {
			return false, nil
		}
		// Move past this match
		matchLen := findMatchLengthAt(name, idx, middlePart)
		name = name[idx+matchLen:]
		name = strings.TrimPrefix(name, "/")
	}

	return true, nil
}

// matchesPrefix checks if name starts with a pattern prefix.
func matchesPrefix(name, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return false
	}

	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[i])
		if err != nil || !matched {
			return false
		}
	}
	return true
}

// findPrefixMatchLength returns the length of name consumed by matching the pattern prefix.
func findPrefixMatchLength(name, pattern string) int {
	if pattern == "" {
		return 0
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return -1
	}

	length := 0
	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[i])
		if err != nil || !matched {
			return -1
		}
		if i > 0 {
			length++ // for the /
		}
		length += len(nameParts[i])
	}
	return length
}

// matchesSuffix checks if name ends with a pattern suffix.
func matchesSuffix(name, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return false
	}

	offset := len(nameParts) - len(patternParts)
	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[offset+i])
		if err != nil || !matched {
			return false
		}
	}
	return true
}

// findSuffixMatchLength returns the length of name consumed by matching the pattern suffix.
func findSuffixMatchLength(name, pattern string) int {
	if pattern == "" {
		return 0
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return -1
	}

	offset := len(nameParts) - len(patternParts)
	length := 0
	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[offset+i])
		if err != nil || !matched {
			return -1
		}
		if i > 0 {
			length++ // for the /
		}
		length += len(nameParts[offset+i])
	}
	return length
}

// findPatternInPath finds where a pattern segment matches within the path.
// Returns the byte index or -1 if not found.
func findPatternInPath(name, pattern string) int {
	if pattern == "" {
		return 0
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return -1
	}

	// Try to find pattern parts as a contiguous sequence in name parts
	for startIdx := 0; startIdx <= len(nameParts)-len(patternParts); startIdx++ {
		allMatch := true
		for i, pp := range patternParts {
			matched, err := path.Match(pp, nameParts[startIdx+i])
			if err != nil || !matched {
				allMatch = false
				break
			}
		}
		if allMatch {
			// Calculate byte position
			pos := 0
			for i := 0; i < startIdx; i++ {
				if i > 0 {
					pos++
				}
				pos += len(nameParts[i])
			}
			if startIdx > 0 {
				pos++ // trailing /
			}
			return pos
		}
	}
	return -1
}

// findMatchLengthAt returns the length of the match starting at the given position.
func findMatchLengthAt(name string, startIdx int, pattern string) int {
	remaining := name[startIdx:]
	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(remaining, "/")

	length := 0
	for i := 0; i < len(patternParts) && i < len(nameParts); i++ {
		if i > 0 {
			length++
		}
		length += len(nameParts[i])
	}
	return length
}

func mathAbsInt(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func stripLeadingDots(s string) string {
	return strings.TrimLeft(s, ".")
}

/*
hasDots is a simple helper function that returns whether the given
string contains a '.' character.
*/
func hasDots(s string) bool {
	return strings.Contains(s, ".")
}

/*
dotCount is a simple helper function that returns the number of '.' occurrences in a string.
*/
func dotCount(s string) int {
	return strings.Count(s, ".")
}

/*
warnForBackslashesOnPosix prints a warning if we're running on in a non-Windows environment
and the given string contains backslashes.

This is because backslashes are allowed as a path part in Posix path strings.
However, on Windows they are a path separator. When persisting a path containing backslashes
from a Posix environment using Path.String or Path.ToPosix will cause path traversal errors
when the persisted path is read and used on Windows environments.

Printing the warning can be disabled by setting PrintBackslashWarningOnPosix to false.
*/
func warnForBackslashesOnPosix(p string) {
	if notRunningOnWindows && PrintBackslashWarningOnPosix && strings.Contains(p, "\\") {
		_, _ = os.Stderr.WriteString("Warning: Usage of backslashes in path string on Posix-like environments. This will break the path part structure if used on Windows: " + fmt.Sprintf(`"%s"`, p) + ".\n")
	}
}

/*
charset contains all numbers from 0 to 9 and all letters of the latin alphabet in lower and upper case.
*/
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

/*
generateRandomString generates a random string with a random length.

This is a utility function used by tests and extensions.
*/
func generateRandomString(minLength, maxLength int) string {
	// Generate a random length between minLength and maxLength
	length := rand.IntN(maxLength-minLength+1) + minLength

	// Create a byte slice to store the random string
	result := make([]byte, length)

	// Fill the byte slice with random characters from the charset
	for i := 0; i < length; i++ {
		result[i] = charset[rand.IntN(len(charset))]
	}

	return string(result)
}
