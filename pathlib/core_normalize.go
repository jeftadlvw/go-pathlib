package pathlib

import (
	"path"
	"regexp"
	"runtime"
	"strings"
)

const runningOnWindows = runtime.GOOS == "windows"
const notRunningOnWindows = !runningOnWindows

const canonicalPathSeparator = "/"
const windowsPathSeparator = "\\"

// Regexes use forward slash instead of backwards slash, because we assume forward slash input.
var windowsVolumeNameRegex = regexp.MustCompile("^[A-Za-z]:(/|$)")
var windowsNetworkPathRegex = regexp.MustCompile("^//[a-zA-Z0-9]+/[a-zA-Z0-9]+")

var multipleWindowsPathSeparatorsRegex = regexp.MustCompile(`\\{2,}`)

// Windows path state bitmask values for windowsPathEncodings.
const (
	// windowsPathStateVolume indicates a Windows path anchored to a drive volume (e.g. "C:\foo").
	windowsPathStateVolume uint8 = 1 << iota

	// windowsPathStateUNC indicates a Windows UNC path (e.g. "\\server\share\foo").
	windowsPathStateUNC
)

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
	match := windowsVolumeNameRegex.FindString(dirty)
	if match != "" {
		return normalizeWindowsVolumePath(dirty, match)
	}

	// UNC anchor: "//host/share".
	match = windowsNetworkPathRegex.FindString(dirty)
	if match != "" {
		return normalizeWindowsUNCPath(dirty, match)
	}

	// No Windows-specific anchor: treat as a regular path.
	return &Path{
		path:                 normalizePath(dirty),
		windowsPathEncodings: 0,
	}
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

// isLocalDirectory returns whether the current path is ".".
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

	base := emptyIfDot(basePath)
	targ := emptyIfDot(targPath)

	baseSlashed := len(base) > 0 && base[0] == '/'
	targSlashed := len(targ) > 0 && targ[0] == '/'
	if baseSlashed != targSlashed {
		return "", ErrRelImpossible
	}

	b0, bi, t0 := relCommonPrefix(base, targ)

	if base[b0:bi] == ".." {
		return "", ErrRelImpossible
	}

	if b0 != len(base) {
		// One ".." for every remaining element of base.
		up := ".." + strings.Repeat("/..", strings.Count(base[b0:], "/"))
		if t0 != len(targ) {
			return up + "/" + targ[t0:], nil
		}
		return up, nil
	}

	result := targ[t0:]
	if result == "" {
		return ".", nil
	}
	return result, nil
}

// relCommonPrefix walks the elements base and targ have in common. It returns
// the start b0 and end bi of the first element of base that differs, and the
// start t0 of the first element of targ that differs.
func relCommonPrefix(base, targ string) (int, int, int) {
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
			return b0, bi, t0
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
}

// emptyIfDot returns an empty string for ".", and s otherwise.
func emptyIfDot(s string) string {
	if s == "." {
		return ""
	}
	return s
}
