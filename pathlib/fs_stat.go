package pathlib

import (
	"os"
	"path/filepath"
)

/*
IsFile returns whether this Path is an existing file.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) IsFile() bool {
	return pathCheck(p) == pathCheckFile
}

/*
IsDir returns whether this Path is an existing directory.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) IsDir() bool {
	return pathCheck(p) == pathCheckDir
}

/*
IsEmptyDir returns whether this Path is an empty existing directory.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) IsEmptyDir() bool {
	return p.IsDir() && !p.HasGlobMatch("*")
}

/*
Exists returns whether this Path exists.

If this Path is a symlink, the target is used. Use IsSymlink to check if this Path is a symlink.
*/
func (p *Path) Exists() bool {
	return pathCheck(p) != pathCheckNoExistOrUnreadable
}

/*
Resolve resolves all symbolic links and ensures an absolute path representation.

This function uses filepath.EvalSymlinks and MakeAbsolute.
*/
func (p *Path) Resolve() (*Path, error) {
	if !p.Exists() {
		return nil, pathErr(ErrNotExist, *p)
	}

	ep, err := filepath.EvalSymlinks(p.String())
	if err != nil {
		return nil, err
	}

	return NewPath(ep).MakeAbsolute()
}

/*
Stat returns file info for this Path.

This function uses os.Stat.
*/
func (p *Path) Stat() (os.FileInfo, error) {
	return os.Stat(p.String())
}

/*
Lstat returns file info for this Path, not following symbolic links.

This function uses os.Lstat.
*/
func (p *Path) Lstat() (os.FileInfo, error) {
	return os.Lstat(p.String())
}

/*
IsSymlink returns whether this Path is a symbolic link.
*/
func (p *Path) IsSymlink() bool {
	// Symlinks must be checked with Stat instead of Lstat
	return checkFileMode(p, os.ModeSymlink)
}

/*
IsBlockDevice returns whether this Path is a block device.
*/
func (p *Path) IsBlockDevice() bool {
	return checkFileMode(p, os.ModeDevice) && !checkFileMode(p, os.ModeCharDevice)
}

/*
IsCharDevice returns whether this Path is a character device.
*/
func (p *Path) IsCharDevice() bool {
	return checkFileMode(p, os.ModeDevice) && checkFileMode(p, os.ModeCharDevice)
}

/*
IsFiFoPipe returns whether this Path is a FIFO/pipe.
*/
func (p *Path) IsFiFoPipe() bool {
	return checkFileMode(p, os.ModeNamedPipe)
}

/*
IsSocket returns whether this Path is a socket.
*/
func (p *Path) IsSocket() bool {
	return checkFileMode(p, os.ModeSocket)
}

/*
pathCheck is a lower level Path existence checker.
It returns 0 if the path does not exist, 1 if it's a file and 2 if it's a directory.
*/
func pathCheck(p *Path) int {
	fileInfo, err := p.Stat()
	if fileInfo == nil || err != nil {
		return pathCheckNoExistOrUnreadable
	}

	if fileInfo.IsDir() {
		return pathCheckDir
	}

	return pathCheckFile
}

/*
checkFileMode is a helper function to check file mode bits on a passed path.

It uses Lstat to get the mode.
*/
func checkFileMode(p *Path, modeMask os.FileMode) bool {
	info, err := p.Lstat()
	if err != nil {
		return false
	}
	return info.Mode()&modeMask != 0
}

const (
	// pathCheckNoExistOrUnreadable indicates that the checked Path does not exist.
	pathCheckNoExistOrUnreadable = iota

	// pathCheckFile indicates that the checked Path is a file.
	pathCheckFile

	// pathCheckDir indicates that the checked Path is a directory.
	pathCheckDir
)
