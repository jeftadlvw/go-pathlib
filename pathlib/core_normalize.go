package pathlib

import (
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
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
	match = windowsNetworkPathRegex.FindString(dirty)
	if match != "" {
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
