// fs_mode.go holds FileMode, the type and permission bits of a filesystem
// entry, and its constants.

package pathlib

import (
	"io/fs"
)

// The bits of a FileMode. They are the bits of [fs.FileMode], so they can be
// combined with the constants of the io/fs and os packages.
const (
	// ModeDir marks a directory.
	ModeDir = fs.ModeDir

	// ModeAppend marks a file that can only be appended to.
	ModeAppend = fs.ModeAppend

	// ModeExclusive marks a file for exclusive use.
	ModeExclusive = fs.ModeExclusive

	// ModeTemporary marks a temporary file. It is used on Plan 9 alone.
	ModeTemporary = fs.ModeTemporary

	// ModeSymlink marks a symbolic link.
	ModeSymlink = fs.ModeSymlink

	// ModeDevice marks a device file.
	ModeDevice = fs.ModeDevice

	// ModeNamedPipe marks a named pipe (FIFO).
	ModeNamedPipe = fs.ModeNamedPipe

	// ModeSocket marks a Unix domain socket.
	ModeSocket = fs.ModeSocket

	// ModeSetuid is the setuid permission bit. It is one of [PermissionBits].
	ModeSetuid = fs.ModeSetuid

	// ModeSetgid is the setgid permission bit. It is one of PermissionBits.
	ModeSetgid = fs.ModeSetgid

	// ModeCharDevice marks a character device. It is set together with
	// ModeDevice.
	ModeCharDevice = fs.ModeCharDevice

	// ModeSticky is the sticky permission bit. It is one of PermissionBits.
	ModeSticky = fs.ModeSticky

	// ModeIrregular marks an entry of an unknown type.
	ModeIrregular = fs.ModeIrregular

	// ModeType is the mask of the type bits. A regular file has none of them.
	ModeType = fs.ModeType

	// ModePerm is the mask of the Unix permission bits, 0777. It is part of
	// PermissionBits.
	ModePerm = fs.ModePerm
)

// FileMode holds the type and permission bits of a filesystem entry. It is an
// alias of [fs.FileMode], so its methods, such as IsDir, IsRegular, and Perm,
// apply, and a FileMode passes to the io/fs and os packages without a
// conversion.
type FileMode = fs.FileMode
