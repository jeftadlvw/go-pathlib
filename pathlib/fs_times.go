// fs_times.go holds the change of the access and modification times of a
// path.

package pathlib

import (
	"os"
	"time"
)

// SetTimes sets the access time of path to accessTime and its modification
// time to modTime. It wraps [os.Chtimes] and follows symlinks. The path may be
// any existing entry, such as a file or a directory.
//
// A zero [time.Time] leaves the corresponding time unchanged. The filesystem
// may round the times to its precision.
//
// A missing path returns [ErrNotExist], and denied access returns
// [ErrPermissionDenied]. Any other failure returns [ErrSetTimes].
func SetTimes(path *Path, accessTime, modTime time.Time) error {
	err := os.Chtimes(path.String(), accessTime, modTime)
	if err != nil {
		return osErr(ErrSetTimes, err, *path)
	}

	return nil
}
