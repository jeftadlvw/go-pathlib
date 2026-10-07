// core_components.go holds the components of a path, such as its parent, its
// names, its extensions, and its anchor.

package pathlib

import (
	"path"
	"slices"
	"strings"
)

// Parent returns the parent directory of this Path. It wraps [path.Dir]. The
// parent of "/" is "/", and the parent of "." and of a single name is ".".
func (p *Path) Parent() *Path {
	return p.copyWithNewPath(path.Dir(p.path))
}

// Parts returns the names of this Path. A volume anchor, such as "C:", is the
// first part, and a UNC anchor is left out. The root "/" has no parts.
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

// Split returns the parent directory and the base name of this Path. It wraps
// [path.Split].
func (p *Path) Split() (*Path, string) {
	dir, file := path.Split(p.path)
	return p.copyWithNewPath(normalizePath(dir)), file
}

// Base returns the last name of this Path. It wraps [path.Base]. A
// drive-relative volume path without names, such as "C:", returns its anchor.
func (p *Path) Base() string {
	if p.isWindowsAnchoredPath() && p.isLocalDirectory() {
		return p.windowsAnchor
	}

	return path.Base(p.path)
}

// HasDotName reports whether the base name of this Path starts with a dot and
// is neither "." nor "..". Unix hides files with such names.
//
// The check is lexical and needs no existing path. The hidden attribute of
// Windows and the hidden flag of macOS play no role.
func (p *Path) HasDotName() bool {
	base := p.Base()
	if base == "" || base == "." || base == ".." {
		return false
	}
	return strings.HasPrefix(base, ".")
}

// HasBackslash reports whether a name of this Path contains a backslash. The
// check is lexical.
//
// Posix allows backslashes in names, and Windows reads them as separators. So
// such a path changes its structure when it is stored on Posix and used on
// Windows. A Path from [NewPathFromWindows] has none, because its backslashes
// are separators.
func (p *Path) HasBackslash() bool {
	return strings.Contains(p.path, windowsPathSeparator)
}

// Stem returns the base name of this Path without its extensions. Leading
// dots belong to the stem, so the stem of ".bashrc" is ".bashrc". The root "/"
// has the stem "".
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

// HasExtensions reports whether this Path has an extension. See
// [Path.Extension].
func (p *Path) HasExtensions() bool {
	return strings.Contains(stripLeadingDots(p.Base()), ".")
}

// ExtensionCount returns the number of extensions of this Path. See
// [Path.Extension].
func (p *Path) ExtensionCount() int {
	return strings.Count(stripLeadingDots(p.Base()), ".")
}

// Extension returns the extensions of this Path with their dots, such as
// ".tar.gz". The extensions are the part of the base name after the stem, see
// [Path.Stem].
func (p *Path) Extension() string {
	stem := p.Stem()

	// A base name without a stem, such as "/", has no extensions.
	if len(stem) == 0 {
		return ""
	}

	return p.Base()[len(stem):]
}

// ExtensionParts returns the extensions of this Path without their dots, such
// as "tar" and "gz". See [Path.Extension].
func (p *Path) ExtensionParts() []string {
	base := stripLeadingDots(p.Base())
	return strings.Split(base, ".")[1:]
}

// Anchor returns the anchor of this Path in the native form of the platform.
//
// An absolute Posix path has the anchor "/". A Windows path has its volume,
// such as "C:", or its UNC root, such as `\\host\share` on Windows and
// "//host/share" elsewhere. A relative path has the anchor "".
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

// WindowsVolume returns the volume of this Path, such as "C:", or "" for a
// path without one.
func (p *Path) WindowsVolume() string {
	if !p.isWindowsVolumeAnchoredPath() {
		return ""
	}

	return p.windowsAnchor
}

// WindowsUNCRoot returns the UNC root of this Path in the native form of the
// platform, such as `\\host\share` on Windows and "//host/share" elsewhere, or
// "" for a path without one.
func (p *Path) WindowsUNCRoot() string {
	if !p.isWindowsUNCAnchoredPath() {
		return ""
	}

	if runningOnWindows {
		return toWindowsSeparators(p.windowsAnchor)
	}

	return p.windowsAnchor
}

// stripLeadingDots returns s without its leading dots.
func stripLeadingDots(s string) string {
	return strings.TrimLeft(s, ".")
}
