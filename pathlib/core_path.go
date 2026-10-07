// Package pathlib handles filesystem paths through the immutable type [Path].
// A Path holds a path in a canonical Posix form and converts it to the native
// form of the platform at the boundary to the operating system.
//
// Every error the package returns pairs a kind with a cause. A kind is a
// *[PathlibError] below [ErrPathlib], such as [ErrNotAbsolute], and is matched
// with [errors.Is]. A cause holds the data of the failure, such as its paths,
// and is read with [errors.As]. It is a *[PathError], unless the kind names
// another.
package pathlib

import (
	"os"
)

// Path is a filesystem path. It holds the path in a canonical Posix form and
// is immutable.
//
// [NewPath] creates a Path from a string of the operating system.
// [NewPathFromPosix] and [NewPathFromWindows] interpret a string in one format
// on every platform.
//
// A Path implements [fmt.Stringer], [encoding.TextMarshaler], and
// [encoding.TextUnmarshaler].
type Path struct {
	// path is the path in canonical Posix form. For a Windows-anchored path it
	// holds the part after the anchor.
	path string

	// windowsAnchor is the Windows anchor with forward slashes, such as "C:"
	// for a volume path or "//host/share" for a UNC path. It is empty for
	// every other path.
	windowsAnchor string

	// windowsPathEncodings is a bitmask of the windowsPathState flags.
	windowsPathEncodings uint8
}

// NewPath returns the Path of a string the operating system handed over, such
// as the result of a system call or the output of a subprocess.
//
// It calls [NewPathFromWindows] on Windows and [NewPathFromPosix] on every
// other platform, so one string can result in different paths. A string in a
// known format, such as a serialized path, goes to the constructor of that
// format, which returns the same Path on every platform.
func NewPath(path string) *Path {
	if runningOnWindows {
		return NewPathFromWindows(path)
	}

	return NewPathFromPosix(path)
}

// NewPathFromPosix returns the Path of a Posix path string on every platform.
// A backslash is an ordinary character of a name. Such a path changes its
// structure on Windows, which [Path.HasBackslash] reports.
//
// It cleans the path with [path.Clean]. A single forward slash separates
// names, "." names and resolvable ".." names are removed, and trailing slashes
// are dropped. Names keep their whitespace and casing. The empty string
// becomes ".", and "/.." becomes "/".
func NewPathFromPosix(path string) *Path {
	return &Path{path: normalizePath(path)}
}

// NewPathFromWindows returns the Path of a Windows path string on every
// platform, such as a path written on Windows and read on Linux.
//
// Backslashes and forward slashes both separate names. A volume, such as
// "C:", or a UNC root, such as `\\host\share`, becomes the anchor of the path.
// The rest is cleaned as [NewPathFromPosix] cleans it, and ".." names never
// climb above the root of a volume.
func NewPathFromWindows(path string) *Path {
	return normalizeWindowsPath(path)
}

// NewPathFromParts returns the Path of parts, joined as [Path.JoinStrings]
// joins them.
func NewPathFromParts(parts ...string) *Path {
	return NewPathFromPosix(".").JoinStrings(parts...)
}

// NewCwd returns the Path of the current working directory. It wraps
// [os.Getwd].
//
// A directory that cannot be determined returns [ErrLookup] with the error of
// the os package as its cause.
func NewCwd() (*Path, error) {
	cwdPath, err := os.Getwd()
	if err != nil {
		return nil, wrapError(ErrLookup, err)
	}

	return NewPath(cwdPath), nil
}

// NewHome returns the Path of the home directory of the user. It wraps
// [os.UserHomeDir].
//
// A directory that cannot be determined returns [ErrLookup] with the error of
// the os package as its cause.
func NewHome() (*Path, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return nil, wrapError(ErrLookup, err)
	}

	return NewPath(homePath), nil
}

// NewConfig returns the Path of the configuration directory of the user. It
// wraps [os.UserConfigDir].
//
// A directory that cannot be determined returns [ErrLookup] with the error of
// the os package as its cause.
func NewConfig() (*Path, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return nil, wrapError(ErrLookup, err)
	}

	return NewPath(configPath), nil
}

// NewCache returns the Path of the cache directory of the user. It wraps
// [os.UserCacheDir].
//
// A directory that cannot be determined returns [ErrLookup] with the error of
// the os package as its cause.
func NewCache() (*Path, error) {
	cachePath, err := os.UserCacheDir()
	if err != nil {
		return nil, wrapError(ErrLookup, err)
	}

	return NewPath(cachePath), nil
}

// Copy returns a copy of this Path.
func (p *Path) Copy() *Path {
	return &Path{
		path:                 p.path,
		windowsAnchor:        p.windowsAnchor,
		windowsPathEncodings: p.windowsPathEncodings,
	}
}

// String returns this Path in the native form of the platform. It is the form
// of [Path.ToWindows] on Windows and the form of [Path.ToPosix] on every other
// platform.
func (p *Path) String() string {
	if runningOnWindows {
		return p.ToWindows()
	}

	return p.ToPosix()
}

// ToPosix returns this Path with forward slashes, including its Windows
// anchor, such as "C:/a".
func (p *Path) ToPosix() string {
	return p.pathWithWindowsAnchor()
}

// ToWindows returns this Path with backslashes, including its Windows anchor,
// such as `C:\a`.
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

// TrimWindowsAnchor returns a copy of this Path without its Windows anchor.
func (p *Path) TrimWindowsAnchor() *Path {
	// The path part holds no anchor and is in Posix form already, so it skips
	// the detection of Windows anchors.
	return NewPathFromPosix(p.path)
}

// MarshalText returns the Posix form of this Path. It implements
// [encoding.TextMarshaler].
func (p *Path) MarshalText() ([]byte, error) {
	return []byte(p.ToPosix()), nil
}

// UnmarshalText sets this Path to text, interpreted by [NewPathFromWindows].
// It implements [encoding.TextUnmarshaler].
//
// The Windows interpretation keeps the anchors [Path.MarshalText] writes with
// forward slashes, such as "c:/" or "//host/share".
//
// UnmarshalText changes this Path in place for decoders such as
// [encoding/json], which fill a field they own. The library never shares a
// *Path with its caller, so only Paths the caller owns change.
func (p *Path) UnmarshalText(text []byte) error {
	*p = *NewPathFromWindows(string(text))
	return nil
}

// copyWithNewPath returns a copy of this Path with the path part newPath and
// the same Windows anchor.
func (p *Path) copyWithNewPath(newPath string) *Path {
	c := p.Copy()
	c.path = newPath
	return c
}
