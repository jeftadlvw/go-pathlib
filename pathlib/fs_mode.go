// fs_mode.go holds FileMode, the type, permission, and special bits of a
// filesystem entry, and its constants.

package pathlib

import (
	"io/fs"
)

// The bits and masks of a FileMode. They are the bits of [fs.FileMode], so
// they can be combined with the constants of the io/fs and os packages.
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

	// ModeSetuid is the setuid bit. It is one of the bits of [ModeSpecial].
	ModeSetuid = fs.ModeSetuid

	// ModeSetgid is the setgid bit. It is one of the bits of ModeSpecial.
	ModeSetgid = fs.ModeSetgid

	// ModeCharDevice marks a character device. It is set together with
	// ModeDevice.
	ModeCharDevice = fs.ModeCharDevice

	// ModeSticky is the sticky bit. It is one of the bits of ModeSpecial.
	ModeSticky = fs.ModeSticky

	// ModeIrregular marks an entry of an unknown type.
	ModeIrregular = fs.ModeIrregular

	// ModeType is the mask of the type bits. A regular file has none of them.
	ModeType = fs.ModeType

	// ModePerm is the mask of the permission bits, 0o777. They are the read,
	// write, and execute bits of the owner, the group, and others.
	ModePerm = fs.ModePerm

	// ModeSpecial is the mask of the special mode bits ModeSetuid,
	// ModeSetgid, and ModeSticky.
	ModeSpecial = ModeSetuid | ModeSetgid | ModeSticky
)

// FileMode holds the mode of a filesystem entry. It is an alias of
// [fs.FileMode], so its methods, such as IsDir, IsRegular, and Perm, apply,
// and a FileMode passes to the io/fs and os packages without a conversion.
//
// A FileMode holds three groups of bits, each with a mask:
//   - The type bits, [ModeType], such as [ModeDir] or [ModeSymlink]. A regular
//     file has none of them.
//   - The permission bits, [ModePerm]. They are the read, write, and execute
//     bits of the owner, the group, and others, such as 0o755 (rwxr-xr-x).
//   - The special mode bits, [ModeSpecial]. They are [ModeSetuid],
//     [ModeSetgid], and [ModeSticky].
//
// A FileMode passed to the library, such as a CreateMode or the mode of
// [SetMode], may only contain permission bits and special mode bits. Any
// other bit returns [ErrInvalidFileMode]. This includes the type bits and the
// Unix octal notation of the special mode bits, such as 0o4755, which the os
// package silently drops. Write ModeSetuid|0o755 instead. A FileMode read
// with [Path.Stat] is passed on after masking it with ModePerm|ModeSpecial.
//
// Whether the setuid and setgid bits survive the creation of a file or
// directory depends on the operating system. macOS drops them, for example.
// SetMode sets them reliably. On Windows, the special mode bits have no
// effect, and the permission bits tell writable (0o666) and read-only (0o444)
// files apart alone.
type FileMode = fs.FileMode
