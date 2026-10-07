// core_relative.go holds absolute and relative paths and the conversion
// between them.

package pathlib

import (
	"path"
	"strings"
)

// IsAbsolute reports whether this Path is absolute. A UNC path is always
// absolute. It wraps [path.IsAbs].
func (p *Path) IsAbsolute() bool {
	if p.isWindowsUNCAnchoredPath() {
		return true
	}

	return path.IsAbs(p.path)
}

// IsRelative reports whether this Path is relative, which is the opposite of
// [Path.IsAbsolute].
func (p *Path) IsRelative() bool {
	return !p.IsAbsolute()
}

// RelativeTo returns this Path relative to other, so that other joined with
// the result is this Path. The computation is lexical.
//
//	this Path   other     RelativeTo
//	"/"         "/a/b"    "../.."
//	"/a"        "/b"      "../a"
//	"/a"        "/a/b"    ".."
//	"a/b/c"     "a/x/y"   "../../b/c"
//	"/a/b/c"    "/"       "a/b/c"
//	"a/b"       "../b"    error, the result depends on the working directory
//
// If one path has a Windows anchor, both need the same anchor, else
// [ErrAnchorMismatch] is returned. A result that depends on the working
// directory returns [ErrRelImpossible]. Both errors report this Path and
// other, in this order.
func (p *Path) RelativeTo(other *Path) (*Path, error) {
	if p.isWindowsAnchoredPath() || other.isWindowsAnchoredPath() {
		if p.windowsAnchor != other.windowsAnchor {
			return nil, pathErr(ErrAnchorMismatch, *p, *other)
		}
	}

	rp, ok := relPath(other.path, p.path)
	if !ok {
		return nil, pathErr(ErrRelImpossible, *p, *other)
	}

	// The result is relative and in Posix form already.
	return NewPathFromPosix(rp), nil
}

// RelativeFrom returns other relative to this Path, so that this Path joined
// with the result is other. The computation is lexical.
//
//	this Path   other     RelativeFrom
//	"/"         "/a/b"    "a/b"
//	"/a"        "/a/b"    "b"
//	"a/b/c"     "a/x/y"   "../../x/y"
//	"/a/b/c"    "/"       "../../.."
//	"../b"      "a/b"     error, the result depends on the working directory
//
// The errors of [Path.RelativeTo] apply.
func (p *Path) RelativeFrom(other *Path) (*Path, error) {
	if p.isWindowsAnchoredPath() || other.isWindowsAnchoredPath() {
		if p.windowsAnchor != other.windowsAnchor {
			return nil, pathErr(ErrAnchorMismatch, *p, *other)
		}
	}

	rp, ok := relPath(p.path, other.path)
	if !ok {
		return nil, pathErr(ErrRelImpossible, *p, *other)
	}

	// The result is relative and in Posix form already.
	return NewPathFromPosix(rp), nil
}

// MakeAbsolute returns this Path joined to the working directory if it is
// relative, or a copy of it if it is absolute.
//
// A working directory that cannot be determined returns [ErrLookup].
func (p *Path) MakeAbsolute() (*Path, error) {
	if p.IsAbsolute() {
		return p.Copy(), nil
	}

	cwd, err := NewCwd()
	if err != nil {
		return nil, err
	}

	return cwd.Join(p), nil
}

// AbsoluteFrom returns this Path joined to base if it is relative, or a copy of
// it if it is absolute.
//
// A relative path with a relative base returns [ErrNotAbsolute] with base as
// its path.
func (p *Path) AbsoluteFrom(base *Path) (*Path, error) {
	if p.IsAbsolute() {
		return p.Copy(), nil
	}

	if base.IsRelative() {
		return nil, pathErr(ErrNotAbsolute, *base)
	}

	return base.Join(p), nil
}

// relPath returns a relative path that is lexically equivalent to targPath when
// joined to basePath with an intervening separator, and whether such a path
// exists. Both paths must use forward slashes and be already normalized.
//
// This function is inspired by filepath.Rel.
func relPath(basePath, targPath string) (string, bool) {
	if basePath == targPath {
		return ".", true
	}

	base := emptyIfDot(basePath)
	targ := emptyIfDot(targPath)

	baseSlashed := len(base) > 0 && base[0] == '/'
	targSlashed := len(targ) > 0 && targ[0] == '/'
	if baseSlashed != targSlashed {
		return "", false
	}

	b0, bi, t0 := relCommonPrefix(base, targ)

	if base[b0:bi] == ".." {
		return "", false
	}

	if b0 != len(base) {
		// One ".." for every remaining element of base.
		up := ".." + strings.Repeat("/..", strings.Count(base[b0:], "/"))
		if t0 != len(targ) {
			return up + "/" + targ[t0:], true
		}
		return up, true
	}

	result := targ[t0:]
	if result == "" {
		return ".", true
	}
	return result, true
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
