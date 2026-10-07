// fs_stat.go holds the inspection of the filesystem entry at a path: whether
// it exists, its type, its file info, and the target of a symbolic link.

package pathlib

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// IsFile reports whether this Path exists and is no directory, such as a
// regular file or a device. A symlink is judged by its target, and
// [Path.IsSymlink] reports the symlink itself.
func (p *Path) IsFile() bool {
	info, err := p.Stat()
	return err == nil && !info.IsDir()
}

// IsDir reports whether this Path is an existing directory. A symlink is
// judged by its target, and [Path.IsSymlink] reports the symlink itself.
func (p *Path) IsDir() bool {
	info, err := p.Stat()
	return err == nil && info.IsDir()
}

// IsEmptyDir reports whether this Path is an existing directory without
// entries. A symlink is judged by its target.
func (p *Path) IsEmptyDir() bool {
	return p.IsDir() && !p.HasGlobMatch("*")
}

// Exists reports whether this Path exists. A symlink is judged by its target,
// so a broken symlink does not exist, and [Path.LExists] checks the symlink
// itself. A path that cannot be checked, such as for missing permissions, does
// not exist either.
func (p *Path) Exists() bool {
	_, err := p.Stat()
	return err == nil
}

// LExists reports whether this Path exists, without following symlinks. A
// symlink exists, whether its target exists or not. A path that cannot be
// checked, such as for missing permissions, does not exist.
func (p *Path) LExists() bool {
	exists, err := lexists(p)
	return exists && err == nil
}

// Resolve returns the absolute path of this Path with every symbolic link
// resolved. It wraps [filepath.EvalSymlinks] and [Path.MakeAbsolute].
//
// A missing path returns [ErrNotExist], and denied access returns
// [ErrPermissionDenied]. Any other failed resolution returns [ErrResolve]. The
// errors of Path.MakeAbsolute apply.
func (p *Path) Resolve() (*Path, error) {
	if !p.Exists() {
		return nil, wrapErr(ErrNotExist, fs.ErrNotExist, *p)
	}

	ep, err := filepath.EvalSymlinks(p.String())
	if err != nil {
		return nil, osErr(ErrResolve, err, *p)
	}

	return NewPath(ep).MakeAbsolute()
}

// Stat returns the file info of this Path. It wraps [os.Stat] and follows
// symbolic links.
//
// A missing path returns [ErrNotExist], and denied access returns
// [ErrPermissionDenied]. Any other failure returns [ErrStat].
func (p *Path) Stat() (os.FileInfo, error) {
	info, err := os.Stat(p.String())
	if err != nil {
		return nil, osErr(ErrStat, err, *p)
	}

	return info, nil
}

// Lstat returns the file info of this Path without following symbolic links.
// It wraps [os.Lstat].
//
// A missing path returns [ErrNotExist], and denied access returns
// [ErrPermissionDenied]. Any other failure returns [ErrStat].
func (p *Path) Lstat() (os.FileInfo, error) {
	info, err := os.Lstat(p.String())
	if err != nil {
		return nil, osErr(ErrStat, err, *p)
	}

	return info, nil
}

// IsSymlink reports whether this Path is a symbolic link.
func (p *Path) IsSymlink() bool {
	return checkFileMode(p, os.ModeSymlink)
}

// IsBlockDevice reports whether this Path is a block device.
func (p *Path) IsBlockDevice() bool {
	return checkFileMode(p, os.ModeDevice) && !checkFileMode(p, os.ModeCharDevice)
}

// IsCharDevice reports whether this Path is a character device.
func (p *Path) IsCharDevice() bool {
	return checkFileMode(p, os.ModeDevice) && checkFileMode(p, os.ModeCharDevice)
}

// IsFIFO reports whether this Path is a named pipe (FIFO).
func (p *Path) IsFIFO() bool {
	return checkFileMode(p, os.ModeNamedPipe)
}

// IsSocket reports whether this Path is a Unix domain socket.
func (p *Path) IsSocket() bool {
	return checkFileMode(p, os.ModeSocket)
}

// ReadSymlinkTarget returns the target of the symbolic link at this Path. It
// wraps [os.Readlink].
//
// A path that is no symlink returns [ErrNotSymlink]. Denied access returns
// [ErrPermissionDenied], and any other failure to read the target returns
// [ErrReadSymlink].
func (p *Path) ReadSymlinkTarget() (*Path, error) {
	if !p.IsSymlink() {
		return nil, pathErr(ErrNotSymlink, *p)
	}

	target, err := os.Readlink(p.String())
	if err != nil {
		return nil, osErr(ErrReadSymlink, err, *p)
	}

	return NewPath(target), nil
}

// requireDir returns nil if p is an existing directory, following symbolic
// links.
//
// An existing non-directory returns [ErrNotDir]. A failed stat returns the
// error of osErr for [ErrStat], such as [ErrNotExist] for a missing path.
func requireDir(p *Path) error {
	info, err := os.Stat(p.String())
	if err != nil {
		return osErr(ErrStat, err, *p)
	}

	if !info.IsDir() {
		return pathErr(ErrNotDir, *p)
	}

	return nil
}

// lexists reports whether p exists, without following symbolic links. It
// tells a missing path apart from one that cannot be checked.
//
// A missing path, including one below a non-directory, returns false and no
// error. Any other failure returns the error of osErr for [ErrStat], such as
// [ErrPermissionDenied] for missing permissions.
func lexists(p *Path) (bool, error) {
	_, err := os.Lstat(p.String())
	if err == nil {
		return true, nil
	}

	// Posix reports a path below a file as ENOTDIR, Windows as not existing.
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return false, nil
	}

	return false, osErr(ErrStat, err, *p)
}

// checkFileMode reports whether the mode of p, without following symbolic
// links, has a bit of modeMask. A path that cannot be checked has none.
func checkFileMode(p *Path, modeMask os.FileMode) bool {
	info, err := p.Lstat()
	if err != nil {
		return false
	}
	return info.Mode()&modeMask != 0
}
