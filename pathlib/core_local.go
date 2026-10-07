// core_local.go holds the check whether a path stays inside the directory it
// is relative to, and the names Windows reserves for devices.

package pathlib

import (
	"strings"
)

// IsLocal reports whether this Path stays inside the directory it is relative
// to. It applies the rules [filepath.IsLocal] applies on Windows, on every
// platform, so a path from an untrusted source, such as the name of an entry
// in an archive, can be checked once for every platform. "." is local.
//
// A local Path is relative and has no Windows anchor. It does not start with
// a backslash, and its ".." names never lead above its start. A backslash in a
// name separates names, as on Windows, so `a\..\..` is not local. No name
// contains a colon, and no name is a device name Windows reserves, such as
// "NUL", "com1", or "CONIN$". A reserved name with an extension, such as
// "nul.txt", counts as reserved, because Windows 10 and older reserve it.
//
// The check is lexical. A symlink inside the path can still lead outside the
// directory.
func (p *Path) IsLocal() bool {
	if p.isWindowsAnchoredPath() || p.IsAbsolute() {
		return false
	}

	// Windows reads a leading backslash as the root of the current drive.
	if strings.HasPrefix(p.path, windowsPathSeparator) {
		return false
	}

	// Windows reads a backslash as a separator, so names are split at both.
	names := strings.FieldsFunc(p.path, func(r rune) bool {
		return r == '/' || r == '\\'
	})

	return areLocalNames(names)
}

// areLocalNames reports whether names, the names of a relative path, stay
// inside the directory the path is relative to. No name may contain a colon
// or be a device name Windows reserves, and the ".." names may never lead
// above the start.
func areLocalNames(names []string) bool {
	depth := 0
	for _, name := range names {
		if strings.Contains(name, ":") || isWindowsReservedName(name) {
			return false
		}

		switch name {
		case ".":
		case "..":
			depth--
			if depth < 0 {
				return false
			}
		default:
			depth++
		}
	}

	return true
}

// isWindowsReservedName reports whether Windows reserves name for a device.
// The reserved base names are CON, PRN, AUX, NUL, COM1 to COM9, LPT1 to LPT9,
// CONIN$, and CONOUT$, in any casing. COM and LPT also take the superscript
// digits 1, 2, and 3. The base name ends at the first dot or colon, and
// trailing spaces are ignored, so "nul.txt" and "NUL " are reserved as well.
func isWindowsReservedName(name string) bool {
	base := name
	cut := strings.IndexAny(base, ".:")
	if cut >= 0 {
		base = base[:cut]
	}

	base = strings.TrimRight(base, " ")

	return isWindowsReservedBaseName(strings.ToUpper(base))
}

// isWindowsReservedBaseName reports whether upper, a base name in upper case,
// is a device name Windows reserves.
func isWindowsReservedBaseName(upper string) bool {
	switch upper {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}

	if len(upper) < len("COM1") {
		return false
	}

	prefix, suffix := upper[:len("COM")], upper[len("COM"):]
	if prefix != "COM" && prefix != "LPT" {
		return false
	}

	switch suffix {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9", "¹", "²", "³":
		return true
	default:
		return false
	}
}
