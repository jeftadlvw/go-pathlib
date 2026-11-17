// fs_dirfs.go holds the view of the directory tree at a path as an fs.FS.

package pathlib

import (
	"io/fs"
	"os"
)

// FS returns the directory tree at this Path as an [fs.FS]. It wraps
// [os.DirFS], and the file system implements the interfaces os.DirFS
// implements, such as [fs.ReadFileFS], [fs.StatFS], and [fs.ReadLinkFS]. So
// [fs.ReadLink] reads the target of a symlink, and [fs.Lstat] describes the
// symlink itself.
//
// The file system takes names in the form [fs.ValidPath] requires. A name is
// slash-separated and relative to this Path, such as "a/b.txt". It has no
// leading slash and no ".." names. [Path.RelativeTo] and [Path.ToPosix] turn a
// Path below this Path into such a name.
//
// FS does not check this Path. A missing path or a non-directory is reported
// when the file system is used. A relative Path is resolved against the
// working directory at every access.
//
// The file system follows symlinks, including those that lead outside this
// Path. It is no boundary for untrusted names.
func (p *Path) FS() fs.FS {
	return os.DirFS(p.String())
}
