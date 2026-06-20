package pathlib

import (
	"path"
)

/*
Join returns a new Path with all passed Path structs joined together.
Paths are not checked whether they are absolute or relative.

Use JoinStrings to join strings with this Path.

This function uses path.Join.
*/
func (p *Path) Join(paths ...*Path) *Path {
	pathsStr := make([]string, len(paths))
	for i, localPath := range paths {
		pathsStr[i] = localPath.path
	}

	return p.copyWithNewPath(path.Join(append([]string{p.path}, pathsStr...)...))
}

/*
JoinStrings returns a new Path with all passed strings joined together.

Each segment is interpreted as a Posix string (the library's canonical string
form). To join a Windows-formatted or OS-native string, parse it first and use Join, e.g.
p.Join(NewPathFromWindows(s)) or p.Join(NewPath(s)).
*/
func (p *Path) JoinStrings(paths ...string) *Path {
	for _, localPath := range paths {
		warnForBackslashesOnPosix(localPath)
	}

	return p.copyWithNewPath(path.Join(append([]string{p.path}, paths...)...))
}

/*
WithName returns this Path but with another base.

name is interpreted as a Posix string, like JoinStrings: a backslash is an ordinary
filename character, not a separator, and passing one on Posix logs a warning.
*/
func (p *Path) WithName(name string) *Path {
	return p.Parent().JoinStrings(name)
}
