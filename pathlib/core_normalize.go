// core_normalize.go holds the normalization of path strings into the canonical
// form of a Path, and the platform constants it relies on.

package pathlib

import (
	"path"
	"regexp"
	"runtime"
	"strings"
)

const (
	// runningOnWindows reports whether the program runs on Windows.
	runningOnWindows = runtime.GOOS == "windows"

	// canonicalPathSeparator separates the names of a canonical path.
	canonicalPathSeparator = "/"

	// windowsPathSeparator separates the names of a native Windows path.
	windowsPathSeparator = "\\"
)

// Flags of the windowsPathEncodings bitmask of a Path.
const (
	// windowsPathStateVolume marks a path anchored to a volume, such as `C:\a`.
	windowsPathStateVolume uint8 = 1 << iota

	// windowsPathStateUNC marks a UNC path, such as `\\host\share\a`.
	windowsPathStateUNC
)

// The patterns match paths with forward slashes, because normalization
// replaces backslashes first.
var (
	// windowsVolumeNameRegex matches a volume anchor, such as "C:" or "C:/".
	windowsVolumeNameRegex = regexp.MustCompile("^[A-Za-z]:(/|$)")

	// windowsNetworkPathRegex matches a UNC anchor, such as "//host/share".
	windowsNetworkPathRegex = regexp.MustCompile("^//[a-zA-Z0-9]+/[a-zA-Z0-9]+")

	// multipleWindowsPathSeparatorsRegex matches a run of backslashes.
	multipleWindowsPathSeparatorsRegex = regexp.MustCompile(`\\{2,}`)
)

// pathWithWindowsAnchor returns the path part of this Path, prefixed with its
// Windows anchor.
func (p *Path) pathWithWindowsAnchor() string {
	if p.isWindowsAnchoredPath() {
		return p.windowsAnchor + p.path
	}

	return p.path
}

// isWindowsAnchoredPath reports whether this Path has a Windows anchor.
func (p *Path) isWindowsAnchoredPath() bool {
	return p.windowsPathEncodings != 0
}

// isWindowsVolumeAnchoredPath reports whether this Path is anchored to a
// Windows volume, such as `C:\a`.
func (p *Path) isWindowsVolumeAnchoredPath() bool {
	return p.windowsPathEncodings&windowsPathStateVolume != 0
}

// isWindowsUNCAnchoredPath reports whether this Path is a Windows UNC path,
// such as `\\host\share\a`.
func (p *Path) isWindowsUNCAnchoredPath() bool {
	return p.windowsPathEncodings&windowsPathStateUNC != 0
}

// isLocalDirectory reports whether the path part of this Path is ".".
func (p *Path) isLocalDirectory() bool {
	return p.path == "."
}

// normalizePath returns the canonical form of p, a path with forward slashes.
// It wraps [path.Clean]. See [NewPathFromPosix] for the rules.
func normalizePath(p string) string {
	return path.Clean(p)
}

// normalizeWindowsPath returns the Path of p, a Windows path string. It splits
// off a volume anchor, such as `C:\`, or a UNC anchor, such as
// `\\host\share`, and normalizes the rest.
func normalizeWindowsPath(p string) *Path {
	// Backslashes are replaced first, so every later step sees forward
	// slashes alone.
	dirty := toCanonicalSeparators(p)

	// Volume anchor: "C:/" (rooted) or "C:" (drive-relative).
	match := windowsVolumeNameRegex.FindString(dirty)
	if match != "" {
		return normalizeWindowsVolumePath(dirty, match)
	}

	// UNC anchor: "//host/share".
	match = windowsNetworkPathRegex.FindString(dirty)
	if match != "" {
		return normalizeWindowsUNCPath(dirty, match)
	}

	return &Path{path: normalizePath(dirty)}
}

// normalizeWindowsVolumePath normalizes dirty, a path with canonical separators
// that starts with the volume anchor match.
func normalizeWindowsVolumePath(dirty, match string) *Path {
	anchor := match[:2] // "C:" (letter + colon)
	hasRoot := strings.HasSuffix(match, canonicalPathSeparator)

	// Strip any leading slashes that normalizePath may have returned.
	cleanedRest := strings.TrimLeft(normalizePath(dirty[len(match):]), canonicalPathSeparator)
	cleanedRest = stripDriveRootTraversal(cleanedRest)

	pathField := cleanedRest
	if hasRoot {
		pathField = canonicalPathSeparator + cleanedRest
	}

	return &Path{
		path:                 pathField,
		windowsAnchor:        anchor,
		windowsPathEncodings: windowsPathStateVolume,
	}
}

// stripDriveRootTraversal removes the ".." traversals of rest that cannot go
// above the drive root. A remaining "." becomes empty.
func stripDriveRootTraversal(rest string) string {
	for strings.HasPrefix(rest, "../") {
		rest = rest[len("../"):]
	}

	if rest == ".." || rest == "." {
		return ""
	}

	return rest
}

// normalizeWindowsUNCPath normalizes dirty, a path with canonical separators
// that starts with the UNC anchor match.
func normalizeWindowsUNCPath(dirty, match string) *Path {
	// An empty rest cleans to ".", which is the root of the share.
	cleanedRest := normalizePath(dirty[len(match):])
	if cleanedRest == "." {
		cleanedRest = canonicalPathSeparator
	}

	return &Path{
		path:                 cleanedRest,
		windowsAnchor:        match,
		windowsPathEncodings: windowsPathStateUNC,
	}
}

// toWindowsSeparators returns s with every forward slash replaced by a
// backslash.
func toWindowsSeparators(s string) string {
	return strings.ReplaceAll(s, canonicalPathSeparator, windowsPathSeparator)
}

// toCanonicalSeparators returns s with every backslash replaced by a forward
// slash.
func toCanonicalSeparators(s string) string {
	return strings.ReplaceAll(s, windowsPathSeparator, canonicalPathSeparator)
}
