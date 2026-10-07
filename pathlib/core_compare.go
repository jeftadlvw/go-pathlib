// core_compare.go holds the lexical comparison of paths.

package pathlib

import (
	"strings"
)

// Equals reports whether this Path and other are lexically equal. The
// comparison is case-sensitive, unless opts holds [CaseInsensitive].
//
// Equals is nil-safe. Two nil Paths are equal, and a nil Path differs from
// every other Path. Paths are compared with Equals, because == compares
// pointers.
func (p *Path) Equals(other *Path, opts ...CompareOption) bool {
	if p == nil || other == nil {
		return p == other
	}

	if ignoresCase(opts) {
		return strings.EqualFold(p.pathWithWindowsAnchor(), other.pathWithWindowsAnchor())
	}

	return p.pathWithWindowsAnchor() == other.pathWithWindowsAnchor()
}

// EqualsString reports whether this Path equals other, interpreted as a Posix
// path by [NewPathFromPosix]. The options of [Path.Equals] apply.
//
// A Posix string pairs with [Path.ToPosix]. The result of [Path.String] is
// native and holds backslashes on Windows. A native or Windows string is
// compared as Equals(NewPath(s)) or Equals(NewPathFromWindows(s)).
//
// A nil Path never equals a string.
func (p *Path) EqualsString(other string, opts ...CompareOption) bool {
	return p.Equals(NewPathFromPosix(other), opts...)
}
