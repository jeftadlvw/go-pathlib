package pathlib

import (
	"path"
	"strings"
)

/*
CompareOption selects how casing is compared in path equality and pattern matching.

The zero value is CaseSensitive, so comparisons are strict unless CaseInsensitive
is passed explicitly.
*/
type CompareOption uint8

const (
	// CaseSensitive compares casing exactly. It is the zero value and the default.
	CaseSensitive CompareOption = iota

	// CaseInsensitive ignores casing.
	CaseInsensitive
)

/*
MatchesPatternE matches this Path's Posix representation against a pattern with support for double asterisk (**).
Returns whether the matching is successful or any occurring error.

Wraps path.Match with some custom rules for double asterisk support.
Use forward slashes as path separators.

By default, matching is case-sensitive. Pass CaseInsensitive to ignore casing.

Empty patterns cause ErrEmptyPattern, malformed patterns cause ErrBadPattern.
*/
func (p *Path) MatchesPatternE(pattern string, opts ...CompareOption) (bool, error) {
	if pattern == "" {
		return false, pathErr(ErrEmptyPattern, *p)
	}

	pathString := p.ToPosix()

	if len(opts) > 0 && opts[0] == CaseInsensitive {
		pattern = strings.ToLower(pattern)
		pathString = strings.ToLower(pathString)
	}

	match, err := matchPattern(pattern, pathString)
	if err != nil {
		return false, wrapErr(ErrBadPattern, err, *p)
	}

	return match, nil
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
Equals returns whether this and another Path match lexically.
By default, comparison is case-sensitive.

Equals is nil-safe. Two nil Paths are equal, and a nil Path differs from any
non-nil Path. Always compare Paths with Equals and never with ==, as == compares pointers.
*/
func (p *Path) Equals(other *Path, opts ...CompareOption) bool {
	if p == nil || other == nil {
		return p == other
	}

	if len(opts) > 0 && opts[0] == CaseInsensitive {
		return strings.EqualFold(p.pathWithWindowsAnchor(), other.pathWithWindowsAnchor())
	}

	return p.pathWithWindowsAnchor() == other.pathWithWindowsAnchor()
}

/*
EqualsString reports whether other denotes the same path as this Path.
By default, comparison is case-sensitive.

other is interpreted as a Posix path. Pair this with ToPosix(), not String().
String() returns the OS-native form and is a portability trap on Windows. To compare
against an OS-native string, or a Windows-formatted one, construct the operand
explicitly and use Equals(NewPath(s)) for an OS path, or Equals(NewPathFromWindows(s))
for a Windows path.

A nil Path never equals any string.
*/
func (p *Path) EqualsString(other string, opts ...CompareOption) bool {
	return p.Equals(NewPathFromPosix(other), opts...)
}
