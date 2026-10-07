// fs_remove.go holds the removal of files and directory trees.

package pathlib

import (
	"errors"
	"io/fs"
	"os"
)

// RemoveOptions configures the removal of a path. The zero value removes a
// file or an empty directory and refuses a missing path.
type RemoveOptions struct {
	// MissingOk accepts a missing path.
	MissingOk bool

	// Recursive removes a directory with its whole tree.
	Recursive bool
}

// DefaultRemoveOptions returns the options [Remove] uses. They remove a file
// or an empty directory and refuse a missing path.
func DefaultRemoveOptions() RemoveOptions {
	return RemoveOptions{
		MissingOk: false,
		Recursive: false,
	}
}

// Remove removes the file or empty directory at path, as [os.Remove] does. A
// missing path returns [ErrNotExist].
//
// It removes as [RemoveWithOptions] does with [DefaultRemoveOptions], and the
// errors of RemoveWithOptions apply.
func Remove(path *Path) error {
	return RemoveWithOptions(path, DefaultRemoveOptions())
}

// RemoveAll removes the entry at path and, for a directory, its whole tree, as
// [os.RemoveAll] does. A missing path is no error.
//
// It removes as [RemoveWithOptions] does with RemoveOptions.MissingOk and
// RemoveOptions.Recursive set, and the errors of RemoveWithOptions apply.
func RemoveAll(path *Path) error {
	return RemoveWithOptions(path, RemoveOptions{MissingOk: true, Recursive: true})
}

// RemoveWithOptions removes the entry at path with options. It wraps
// [os.Remove], or [os.RemoveAll] with RemoveOptions.Recursive. A symlink is
// removed itself, never its target, and this includes a symlink to a
// directory and a broken symlink.
//
// A missing path returns [ErrNotExist], unless RemoveOptions.MissingOk
// accepts it. A broken symlink exists. A directory with entries returns
// [ErrNotEmptyDir], unless RemoveOptions.Recursive removes its whole tree.
//
// Denied access returns [ErrPermissionDenied]. Any other failure to check the
// path returns [ErrStat], and any other failed removal returns [ErrRemove].
func RemoveWithOptions(path *Path, options RemoveOptions) error {
	exists, err := lexists(path)
	if err != nil {
		return err
	}
	if !exists {
		return missingRemoveTarget(path, options.MissingOk)
	}

	if options.Recursive {
		err = os.RemoveAll(path.String())
	} else {
		err = os.Remove(path.String())
	}

	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrNotExist):
		// The path may have vanished since the check.
		return missingRemoveTarget(path, options.MissingOk)
	case errors.Is(err, fs.ErrExist) && !options.Recursive:
		// Removing a directory with entries fails with ENOTEMPTY, EEXIST, or
		// ERROR_DIR_NOT_EMPTY, and each matches fs.ErrExist.
		return wrapErr(ErrNotEmptyDir, err, *path)
	default:
		return osErr(ErrRemove, err, *path)
	}
}

// missingRemoveTarget returns the result of removing path, which does not
// exist. It is nil if missingOk is set, and [ErrNotExist] otherwise.
func missingRemoveTarget(path *Path, missingOk bool) error {
	if missingOk {
		return nil
	}

	return wrapErr(ErrNotExist, fs.ErrNotExist, *path)
}
