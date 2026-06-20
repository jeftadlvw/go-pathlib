package pathlib

import (
	"path"
	"slices"
	"strings"
)

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
HasDotName reports whether this Path's base name follows the Unix dotfile
convention (begins with a dot but is not "." or "..").

This is a purely lexical, conventional check that does not touch the filesystem
and does not require the path to exist. It does not reflect Windows hidden-file
attributes or macOS hidden flags.
*/
func (p *Path) HasDotName() bool {
	base := p.Base()
	if base == "" || base == "." || base == ".." {
		return false
	}
	return strings.HasPrefix(base, ".")
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
