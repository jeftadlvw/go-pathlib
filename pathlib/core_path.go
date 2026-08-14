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
	"os"
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
NewConfig returns a new Path instance pointing to the user's configuration directory.

This function wraps os.UserConfigDir.
*/
func NewConfig() (*Path, error) {
	homePath, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return NewPath(homePath), nil
}

/*
NewCache returns a new Path instance pointing to the user's cache directory.

This function wraps os.UserCacheDir.
*/
func NewCache() (*Path, error) {
	homePath, err := os.UserCacheDir()
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

On Windows this uses backslashes and prepends the anchor. On Posix it uses forward slashes.
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
