package pathlib

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

/*
Copy copies the source path to the destination path.

If the source path is a directory, the whole directory tree is copied. All other files
and file types are copied as-is. A symlink is copied as a symlink, and this includes
a broken symlink.

Copying a directory requires the target directory to be empty.

The source path must exist. Destination parent directories must exist.

A missing source returns [ErrNotExist], and a missing destination parent returns
[ErrParentNotExist]. An existing destination returns a kind below [ErrExist], or
[ErrNotEmptyDir] for a directory that is not empty. A destination of an
incompatible file type returns [ErrTypeMismatch], and a source of a file type
that cannot be copied returns [ErrCopyType]. A failure of the operating system
returns [ErrStat], a kind below [ErrAccess], [ErrCreate], [ErrCopy], or
[ErrReadSymlink].

A symlink with a relative target is copied with a target rebased to the
destination. Rebasing resolves relative paths against the working directory and
returns [ErrLookup] if it cannot be determined. On Windows, a source and
destination on different volumes return [ErrAnchorMismatch].
*/
func Copy(src *Path, destination *Path) error {
	srcExists, err := lexists(src)
	if err != nil {
		return err
	}
	if !srcExists {
		return wrapErr(ErrNotExist, fs.ErrNotExist, *src)
	}

	// Ensure parent directories exist
	if !destination.Parent().Exists() {
		return wrapErr(ErrParentNotExist, fs.ErrNotExist, *destination)
	}

	switch {
	case src.IsSymlink(): // A symlink is a special file, thus checking that first
		return copySymlink(src, destination)
	case src.IsFile():
		return copyFile(src, destination)
	case src.IsDir():
		return copyDir(src, destination)
	default:
		return pathErr(ErrCopyType, *src)
	}
}

/*
Move moves the file or directory at the source path to the destination path.
If the destination is on the same filesystem, this is equivalent to a rename operation.

Fails if destination already exists, except if the source path is a directory, and the target path
is an empty directory. A broken symlink at the destination exists too.

A symlink is moved as a symlink, and this includes a broken symlink.

Destination parent directories must exist.

An existing destination returns [ErrExist]. Moving across filesystems copies and
removes the source, so the errors of [Copy] and [RemoveAll] apply.
*/
func Move(src *Path, dst *Path) error {
	srcExists, err := lexists(src)
	if err != nil {
		return err
	}
	if !srcExists {
		return wrapErr(ErrNotExist, fs.ErrNotExist, *src)
	}

	// Destination path may not exist, except if source is a directory
	// and destination is an empty directory too.
	if !src.IsDir() || !dst.IsEmptyDir() {
		dstExists, err := lexists(dst)
		if err != nil {
			return err
		}
		if dstExists {
			return wrapErr(ErrExist, fs.ErrExist, *dst)
		}
	}

	// Try renaming first (works if on same filesystem)
	err = os.Rename(src.String(), dst.String())
	if err == nil {
		return nil
	}

	// If rename fails, try copy and delete
	err = Copy(src, dst)
	if err != nil {
		return err
	}

	return RemoveAll(src)
}

/*
Rename renames the file at the source path to the destination path.
This is a convenience wrapper for Move, and the errors of [Move] apply.
*/
func Rename(src *Path, name string) error {
	return Move(src, src.Parent().JoinStrings(name))
}

/*
Remove removes the file at the specified path
or removes an empty directory.

Nothing happens if the given path does not exist. A symlink is removed itself,
never its target, and this includes a broken symlink. If the path cannot be
checked, e.g. for missing permissions, [ErrStat] is returned. A failed removal
returns [ErrRemove].
*/
func Remove(path *Path) error {
	exists, err := lexists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// The path may have vanished since the check, which is the wanted outcome.
	err = os.Remove(path.String())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return wrapErr(ErrRemove, err, *path)
	}

	return nil
}

/*
RemoveAll removes the path at the specified path and, if it is a directory, all its
entries, like os.RemoveAll.

Nothing happens if the given path does not exist. A symlink is removed itself, never
its target, and this includes a symlink to a directory and a broken symlink. If the
path cannot be checked, e.g. for missing permissions, [ErrStat] is returned. A
failed removal returns [ErrRemove].
*/
func RemoveAll(path *Path) error {
	exists, err := lexists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	err = os.RemoveAll(path.String())
	if err != nil {
		return wrapErr(ErrRemove, err, *path)
	}

	return nil
}

/*
copyFile copies a file.

Assumes source and destination parent directories exist.
*/
func copyFile(source *Path, destination *Path) error {
	// Get source file info for permissions
	srcInfo, err := source.Stat()
	if err != nil {
		return err
	}

	err = requireCopyFileDestinationFree(source, destination)
	if err != nil {
		return err
	}

	// Open source file
	sourceFile, err := os.Open(source.String())
	if err != nil {
		return wrapErr(ErrOpen, err, *source)
	}
	// The source is only read, so closing it cannot lose data.
	defer func() { _ = sourceFile.Close() }()

	// Create the destination file. O_EXCL refuses a path created since the check,
	// symlinks included, so nothing is overwritten or written through.
	destinationFile, err := os.OpenFile(destination.String(), os.O_RDWR|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
	if err != nil {
		return wrapErr(ErrCreate, err, *destination)
	}

	// Copy contents
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		_ = destinationFile.Close()
		return wrapErr(ErrCopy, err, *source, *destination)
	}

	// Closing flushes the content, so its error means the copy is incomplete.
	err = destinationFile.Close()
	if err != nil {
		return wrapErr(ErrCopy, err, *source, *destination)
	}

	return nil
}

/*
copySymlink copies a symlink.

Assumes source and destination parent directories exist.
*/
func copySymlink(source *Path, destination *Path) error {
	// All checks should be already done by called functions

	originalSymlinkTarget, err := source.ReadSymlinkTarget()
	if err != nil {
		return err
	}

	// The original target might be relative to source. If so, make it relative to destination.
	if originalSymlinkTarget.IsRelative() {
		originalSymlinkTarget, err = rebaseSymlinkTarget(originalSymlinkTarget, source, destination)
		if err != nil {
			return err
		}
	}

	return CreateSymlink(originalSymlinkTarget, destination)
}

/*
rebaseSymlinkTarget returns the relative target of the source symlink as seen
from the destination symlink, so both point to the same path.

A relative source or destination is relative to the working directory, as the
operating system resolves it. Both parents are made absolute first, so they
share a base.
*/
func rebaseSymlinkTarget(target *Path, source *Path, destination *Path) (*Path, error) {
	sourceParent, err := source.Parent().MakeAbsolute()
	if err != nil {
		return nil, err
	}

	destinationParent, err := destination.Parent().MakeAbsolute()
	if err != nil {
		return nil, err
	}

	absoluteTarget, err := target.AbsoluteFrom(sourceParent)
	if err != nil {
		return nil, err
	}

	return absoluteTarget.RelativeTo(destinationParent)
}

/*
copyDir copies a directory recursively.

Assumes source and destination parent directories exist.
*/
func copyDir(src *Path, dst *Path) error {
	err := prepareCopyDirDestination(src, dst)
	if err != nil {
		return err
	}

	// Read directory entries
	entries, err := os.ReadDir(src.String())
	if err != nil {
		return wrapErr(ErrReadDir, err, *src)
	}

	// Copy each entry
	for _, entry := range entries {
		srcEntry := src.JoinStrings(entry.Name())
		dstEntry := dst.JoinStrings(entry.Name())

		err := Copy(srcEntry, dstEntry)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
prepareCopyDirDestination makes dst ready to receive the entries of the source
directory src. A missing dst is created with the mode of src. An existing dst
must be an empty directory.
*/
func prepareCopyDirDestination(src *Path, dst *Path) error {
	// Get source file info for permissions
	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	dstExists, err := lexists(dst)
	if err != nil {
		return err
	}

	if !dstExists {
		err = os.Mkdir(dst.String(), srcInfo.Mode())
		if err != nil {
			return wrapErr(ErrCreate, err, *dst)
		}
		return nil
	}

	switch {
	case dst.IsFile():
		return pathErr(ErrTypeMismatch, *src, *dst)
	case dst.IsDir():
		return requireEmptyDir(dst)
	default:
		return wrapErr(ErrExist, fs.ErrExist, *dst)
	}
}

/*
requireEmptyDir returns [ErrNotEmptyDir] if the directory dir has entries.
*/
func requireEmptyDir(dir *Path) error {
	file, err := os.Open(dir.String())
	if err != nil {
		return wrapErr(ErrOpen, err, *dir)
	}

	// The directory is only read, so closing it cannot lose data.
	defer func() { _ = file.Close() }()

	entries, err := file.ReadDir(1)
	if err != nil && !errors.Is(err, io.EOF) {
		return wrapErr(ErrReadDir, err, *dir)
	}

	if len(entries) != 0 {
		return pathErr(ErrNotEmptyDir, *dir)
	}

	return nil
}

/*
requireCopyFileDestinationFree returns an error if anything exists at the
destination of a file copied from source. An existing file returns
[ErrFileExist], a directory returns [ErrTypeMismatch], and any other entry
returns [ErrExist]. A path that cannot be checked returns [ErrStat].
*/
func requireCopyFileDestinationFree(source *Path, destination *Path) error {
	// A broken symlink exists too, and writing through it would create its target.
	exists, err := lexists(destination)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	switch {
	case destination.IsFile():
		return wrapErr(ErrFileExist, fs.ErrExist, *destination)
	case destination.IsDir():
		return pathErr(ErrTypeMismatch, *source, *destination)
	default:
		return wrapErr(ErrExist, fs.ErrExist, *destination)
	}
}
