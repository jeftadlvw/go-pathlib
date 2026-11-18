// fs_dirfs.go holds the view of the directory tree at a path as an fs.FS.

package pathlib

import (
	"io/fs"
	"os"
)

// FS returns [os.DirFS] of this Path, for APIs that take an [fs.FS]. The file
// system belongs to the standard library, not to pathlib. It takes
// slash-separated names relative to this Path, and its errors are those of
// [io/fs] without error kinds, matched with sentinels such as [fs.ErrNotExist].
// [NewPathFromPosix] turns a name back into a Path below this Path.
//
// FS does not check this Path. The file system follows symlinks out of it, so
// it is no boundary for untrusted names.
func (p *Path) FS() fs.FS {
	return os.DirFS(p.String())
}
