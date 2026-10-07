// fs_remove.go holds the removal of files and directory trees.

package pathlib

import (
	"errors"
	"io/fs"
	"os"
)

// Remove removes the file or empty directory at path. It wraps [os.Remove].
// A missing path is no error. A symlink is removed itself, never its target,
// and this includes a broken symlink.
//
// Denied access returns [ErrPermissionDenied]. Any other failure to check the
// path returns [ErrStat], and any other failed removal returns [ErrRemove].
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
		return osErr(ErrRemove, err, *path)
	}

	return nil
}

// RemoveAll removes the entry at path and, for a directory, its whole tree. It
// wraps [os.RemoveAll]. A missing path is no error. A symlink is removed
// itself, never its target, and this includes a symlink to a directory and a
// broken symlink.
//
// Denied access returns [ErrPermissionDenied]. Any other failure to check the
// path returns [ErrStat], and any other failed removal returns [ErrRemove].
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
		return osErr(ErrRemove, err, *path)
	}

	return nil
}
