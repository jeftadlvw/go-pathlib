// fs_info.go holds FileInfo, the description of a filesystem entry.

package pathlib

import (
	"io/fs"
	"os"
	"time"
)

// FileInfo implements fs.FileInfo, so it can be passed wherever one is
// accepted.
var _ fs.FileInfo = (*FileInfo)(nil)

// FileInfo describes the filesystem entry at a path, as [Path.Stat] and
// [Path.Lstat] return it. It implements [fs.FileInfo] and wraps the file info
// of the operating system.
//
// FileInfo normalizes the size, which the platforms report differently for
// entries that are no regular file. [FileInfo.Sys] returns the data of the
// operating system as it is.
//
// [os.SameFile] accepts the file info of the os package alone, so it reports
// false for a FileInfo. [FileInfo.SameFile] compares two FileInfo values.
type FileInfo struct {
	// info is the file info of the operating system.
	info fs.FileInfo
}

// newFileInfo returns the FileInfo that wraps info, the file info of the
// operating system.
func newFileInfo(info fs.FileInfo) *FileInfo {
	return &FileInfo{info: info}
}

// Name returns the base name of the path the file info was read for, as
// [os.Stat] reports it.
func (fi *FileInfo) Name() string {
	return fi.info.Name()
}

// Size returns the length in bytes of a regular file. It returns 0 for every
// other type, such as a directory, a symlink, or a device, whose size the
// platforms report differently. Unix reports the storage of a directory and
// the length of the target of a symlink, for example, and Windows reports 0.
func (fi *FileInfo) Size() int64 {
	if !fi.info.Mode().IsRegular() {
		return 0
	}

	return fi.info.Size()
}

// Mode returns the [FileMode] of the entry. On Windows, the permission bits
// tell writable (0666) and read-only (0444) entries apart alone, and a
// directory adds 0111.
func (fi *FileInfo) Mode() FileMode {
	return fi.info.Mode()
}

// ModTime returns the modification time.
func (fi *FileInfo) ModTime() time.Time {
	return fi.info.ModTime()
}

// IsDir reports whether the entry is a directory. It is short for
// Mode().IsDir().
func (fi *FileInfo) IsDir() bool {
	return fi.info.IsDir()
}

// Sys returns the data of the operating system, such as a *syscall.Stat_t on
// Unix and a *syscall.Win32FileAttributeData on Windows. Its type differs
// between platforms.
func (fi *FileInfo) Sys() any {
	return fi.info.Sys()
}

// SameFile reports whether this FileInfo and other describe the same file, as
// [os.SameFile] does. A nil other describes no file.
func (fi *FileInfo) SameFile(other *FileInfo) bool {
	if other == nil {
		return false
	}

	return os.SameFile(fi.info, other.info)
}
