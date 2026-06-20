package pathlib

import (
	"io"
	"os"
)

/*
Copy copies the source path to the destination path.

If the source path is a directory, the whole directory tree is copied. All other files
and file types are copied as-is.

Copying a directory requires the target directory to be empty.

The source path must exist. Destination parent directories must exist.
*/
func Copy(src *Path, destination *Path) error {
	if !src.Exists() {
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

	// Ensure the target file does not exist
	if destination.Exists() {
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
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.OpenFile(destination.String(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy contents
	_, err = io.Copy(destinationFile, sourceFile)
	return err
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

	if dst.Exists() {
		if dst.IsFile() {
			// Ensure the target directory is not a file
			return pathErr(ErrTypeMismatch, *src, *dst)

		} else if dst.IsDir() {
			// Ensure destination directory is empty
			file, openErr := os.Open(dst.String())
			if openErr != nil {
				return openErr
			}

			defer file.Close()

			entries, readDirErr := file.ReadDir(1)
			if readDirErr != nil && readDirErr != io.EOF {
				return readDirErr
			}

			if len(entries) != 0 {
				return pathErr(ErrNotEmptyDir, *dst)
			}
		} else {
			return pathErr(ErrExist, *dst)
		}

	} else {
		// Create the destination directory if it doesn't exist
		if err := os.Mkdir(dst.String(), srcInfo.Mode()); err != nil {
			return err
		}
	}

	// Read directory entries
	entries, err := os.ReadDir(src.String())
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcEntry := src.JoinStrings(entry.Name())
		dstEntry := dst.JoinStrings(entry.Name())

		if err := Copy(srcEntry, dstEntry); err != nil {
			return err
		}
	}

	return nil
}

/*
Move moves the file or directory at the source path to the destination path.
If the destination is on the same filesystem, this is equivalent to a rename operation.

Fails if destination already exists, except if the source path is a directory, and the target path
is an empty directory.

Destination parent directories must exist.
*/
func Move(src *Path, dst *Path) error {
	if !src.Exists() {
		return pathErr(ErrNotExist, *src)
	}

	// Destination path may not exist, except if source is a directory
	// and destination is an empty directory too.
	if src.IsDir() && dst.IsEmptyDir() {
		// do nothing
	} else if dst.Exists() {
		return pathErr(ErrExist, *dst)
	}

	// Try renaming first (works if on same filesystem)
	err := os.Rename(src.String(), dst.String())
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

Nothing happens if the given path does not exist.
*/
func Remove(path *Path) error {
	if !path.Exists() {
		return nil
	}

	return os.Remove(path.String())
}

/*
RemoveAll recursively removes the directory and all its entries at the specified path.

Nothing happens if the given path does not exist.
*/
func RemoveAll(path *Path) error {
	if !path.Exists() {
		return nil
	}

	if !path.IsDir() {
		return pathErr(ErrNotDir, *path)
	}

	return os.RemoveAll(path.String())
}
