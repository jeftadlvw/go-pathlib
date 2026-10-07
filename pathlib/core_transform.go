// core_transform.go holds the derivation of paths from a path, by joining names
// and by replacing them.

package pathlib

import (
	"path"
)

// Join returns this Path joined with paths. It wraps [path.Join] and joins
// absolute paths like relative ones. This Path keeps its Windows anchor, and
// the anchors of paths are dropped.
func (p *Path) Join(paths ...*Path) *Path {
	pathsStr := make([]string, len(paths))
	for i, localPath := range paths {
		pathsStr[i] = localPath.path
	}

	return p.copyWithNewPath(path.Join(append([]string{p.path}, pathsStr...)...))
}

// JoinStrings returns this Path joined with paths, each interpreted as a Posix
// path. A Windows or native string is parsed first and passed to [Path.Join],
// as in p.Join(NewPathFromWindows(s)).
func (p *Path) JoinStrings(paths ...string) *Path {
	return p.copyWithNewPath(path.Join(append([]string{p.path}, paths...)...))
}

// WithName returns this Path with its base name replaced by name. The name is
// interpreted as a Posix path, as [Path.JoinStrings] interprets it.
func (p *Path) WithName(name string) *Path {
	return p.Parent().JoinStrings(name)
}
