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
*/
func Copy(src *Path, destination *Path) error {
	srcExists, err := lexists(src)
	if err != nil {
		return err
	}
	if !srcExists {
		return pathErr(ErrNotExist, *src)
	}

	// Ensure parent directories exist
	if !destination.Parent().Exists() {
		return pathErr(ErrParentNotExist, *destination)
	}

	if src.IsSymlink() { // A symlink is a special file, thus checking that first
		return copySymlink(src, destination)
	} else if src.IsFile() {
		return copyFile(src, destination)
	} else if src.IsDir() {
		return copyDir(src, destination)
	} else {
		return pathErr(ErrCopyType, *src)
	}
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

	// Ensure the target file does not exist. A broken symlink exists too, and
	// writing through it would create its target.
	destinationExists, err := lexists(destination)
	if err != nil {
		return err
	}

	if destinationExists {
		if destination.IsFile() {
			return pathErr(ErrFileExist, *destination)
		} else if destination.IsDir() {
			return pathErr(ErrTypeMismatch, *source, *destination)
		} else {
			return pathErr(ErrExist, *destination)
		}
	}

	// Open source file
	sourceFile, err := os.Open(source.String())
	if err != nil {
		return wrapErr(ErrOpen, err, *source)
	}
	defer sourceFile.Close()

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
		originalSymlinkTarget, err = originalSymlinkTarget.AbsoluteFrom(source.Parent())
		if err != nil {
			return err
		}

		originalSymlinkTarget, err = originalSymlinkTarget.RelativeTo(destination.Parent())
		if err != nil {
			return err
		}
	}

	return CreateSymlink(originalSymlinkTarget, destination)
}

/*
copyDir copies a directory recursively.

Assumes source and destination parent directories exist.
*/
func copyDir(src *Path, dst *Path) error {
	// Get source file info for permissions
	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	dstExists, err := lexists(dst)
	if err != nil {
		return err
	}

	if dstExists {
		if dst.IsFile() {
			// Ensure the target directory is not a file
			return pathErr(ErrTypeMismatch, *src, *dst)

		} else if dst.IsDir() {
			// Ensure destination directory is empty
			file, openErr := os.Open(dst.String())
			if openErr != nil {
				return wrapErr(ErrOpen, openErr, *dst)
			}

			defer file.Close()

			entries, readDirErr := file.ReadDir(1)
			if readDirErr != nil && readDirErr != io.EOF {
				return wrapErr(ErrReadDir, readDirErr, *dst)
			}

			if len(entries) != 0 {
				return pathErr(ErrNotEmptyDir, *dst)
			}
		} else {
			return pathErr(ErrExist, *dst)
		}

	} else {
		// Create the destination directory if it doesn't exist
		err := os.Mkdir(dst.String(), srcInfo.Mode())
		if err != nil {
			return wrapErr(ErrCreate, err, *dst)
		}
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
Move moves the file or directory at the source path to the destination path.
If the destination is on the same filesystem, this is equivalent to a rename operation.

Fails if destination already exists, except if the source path is a directory, and the target path
is an empty directory. A broken symlink at the destination exists too.

A symlink is moved as a symlink, and this includes a broken symlink.

Destination parent directories must exist.
*/
func Move(src *Path, dst *Path) error {
	srcExists, err := lexists(src)
	if err != nil {
		return err
	}
	if !srcExists {
		return pathErr(ErrNotExist, *src)
	}

	// Destination path may not exist, except if source is a directory
	// and destination is an empty directory too.
	if !(src.IsDir() && dst.IsEmptyDir()) {
		dstExists, err := lexists(dst)
		if err != nil {
			return err
		}
		if dstExists {
			return pathErr(ErrExist, *dst)
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

	if src.IsDir() {
		return RemoveAll(src)
	} else {
		return Remove(src)
	}
}

/*
Rename renames the file at the source path to the destination path.
This is a convenience wrapper for Move.
*/
func Rename(src *Path, name string) error {
	return Move(src, src.Parent().JoinStrings(name))
}

/*
Remove removes the file at the specified path
or removes an empty directory.

Nothing happens if the given path does not exist. A symlink is removed itself,
never its target, and this includes a broken symlink. If the path cannot be
checked, e.g. for missing permissions, ErrStat is returned.
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
RemoveAll recursively removes the directory and all its entries at the specified path.

Nothing happens if the given path does not exist.

Unlike os.RemoveAll, the path must be a directory. Any other existing path returns
ErrNotDir and is left in place, and this includes a broken symlink. Use Remove for
files and symlinks. If the path is a symlink to a directory, only the symlink is removed.
If the path cannot be checked, e.g. for missing permissions, ErrStat is returned.
*/
func RemoveAll(path *Path) error {
	exists, err := lexists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if !path.IsDir() {
		return pathErr(ErrNotDir, *path)
	}

	err = os.RemoveAll(path.String())
	if err != nil {
		return wrapErr(ErrRemove, err, *path)
	}

	return nil
}
