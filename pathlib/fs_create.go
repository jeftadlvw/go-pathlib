// fs_create.go holds the creation of files, directories, and symbolic links.

package pathlib

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

// FileOptions configures the creation of a file. The zero value creates a new
// file with [DefaultFileMode] and refuses an existing one.
type FileOptions struct {
	// ExistOk accepts an existing file. CreateFileWithOptions leaves it
	// untouched, and WriteBytesWithOptions truncates it.
	ExistOk bool

	// UpdateTimes sets the access and modification times of an existing file
	// that ExistOk accepts to the current time, as the touch command does.
	// Without ExistOk it has no effect. WriteBytesWithOptions ignores it,
	// because writing sets the modification time.
	UpdateTimes bool

	// Mode is the permission of a created file. It may only contain
	// PermissionBits. Zero selects DefaultFileMode, because the operating
	// system refuses a file without permissions.
	Mode FileMode
}

// DirOptions configures the creation of a directory. The zero value creates a
// new directory with [DefaultDirMode] in an existing parent directory and
// refuses an existing one.
type DirOptions struct {
	// ExistOk accepts an existing directory.
	ExistOk bool

	// UpdateTimes sets the access and modification times of an existing
	// directory that ExistOk accepts to the current time, as the touch command
	// does. Without ExistOk it has no effect.
	UpdateTimes bool

	// Mode is the permission of a created directory. It may only contain
	// PermissionBits. Zero selects DefaultDirMode, because the operating
	// system refuses a directory without permissions.
	Mode FileMode

	// CreateAll creates missing parent directories as well.
	CreateAll bool
}

// DefaultFileOptions returns the options [CreateFile] uses. They create a new
// file with [DefaultFileMode].
func DefaultFileOptions() FileOptions {
	return FileOptions{
		ExistOk:     false,
		UpdateTimes: false,
		Mode:        DefaultFileMode(),
	}
}

// DefaultDirOptions returns the options [MkDir] uses. They create a new
// directory with [DefaultDirMode] in an existing parent directory.
func DefaultDirOptions() DirOptions {
	return DirOptions{
		ExistOk:     false,
		UpdateTimes: false,
		Mode:        DefaultDirMode(),
		CreateAll:   false,
	}
}

// CreateFile creates an empty file at path with [DefaultFileMode]. The parent
// directory must exist.
//
// CreateFile never truncates. An existing file returns [ErrFileExist] and is
// left untouched. [CreateFileWithOptions] accepts an existing file with
// FileOptions.ExistOk. With FileOptions.UpdateTimes as well, it sets the times
// of the file to the current time, as a touch command does. [OpenFile]
// creates or truncates a file, as [os.Create] does.
//
// The errors of CreateFileWithOptions apply.
func CreateFile(path *Path) error {
	_, err := CreateFileWithOptions(path, DefaultFileOptions())
	return err
}

// CreateFileWithOptions creates an empty file at path with options and reports
// whether it created one. The parent directory must exist.
//
// An existing file returns [ErrFileExist], unless FileOptions.ExistOk accepts
// it. Then false is returned, and the file is left untouched. With
// FileOptions.UpdateTimes, its access and modification times are set to the
// current time, as [SetTimes] does. An existing path that is no file returns
// [ErrNotFile]. This includes a broken symlink, whose target is never created.
//
// A FileOptions.Mode with bits outside [PermissionBits] returns
// [ErrInvalidPermission]. A missing parent directory returns
// [ErrParentNotExist], and denied access returns [ErrPermissionDenied]. Any
// other failure to check the path returns [ErrStat], any other failed
// creation returns [ErrCreate], and any other failed update of the times
// returns [ErrSetTimes].
func CreateFileWithOptions(path *Path, options FileOptions) (bool, error) {
	err := checkPermission(options.Mode, path)
	if err != nil {
		return false, err
	}

	check := func() (bool, error) {
		return checkCreateTarget(path, path.IsFile, options.ExistOk, options.UpdateTimes, ErrNotFile, ErrFileExist)
	}

	exists, err := check()
	if err != nil || exists {
		return false, err
	}

	if options.Mode == 0 {
		options.Mode = DefaultFileMode()
	}

	// O_EXCL refuses a path created since the check, symlinks included, so an
	// existing file is never truncated and no symlink target is created.
	file, err := os.OpenFile(path.String(), os.O_RDWR|os.O_CREATE|os.O_EXCL, options.Mode)
	if err != nil {
		// Another process may have created the path since the check. Checking
		// again returns the same error as an existing path before the call.
		if errors.Is(err, fs.ErrExist) {
			exists, checkErr := check()
			if checkErr != nil || exists {
				return false, checkErr
			}
		}
		return false, osCreateErr(ErrCreate, err, *path)
	}

	err = file.Close()
	if err != nil {
		return true, osErr(ErrCreate, err, *path)
	}

	return true, nil
}

// MkDir creates a directory at path with [DefaultDirMode]. The parent
// directory must exist.
//
// The errors of [MkDirWithOptions] apply.
func MkDir(path *Path) error {
	_, err := MkDirWithOptions(path, DefaultDirOptions())
	return err
}

// MkDirWithOptions creates a directory at path with options and reports
// whether it created one. The parent directory must exist, unless
// DirOptions.CreateAll creates it.
//
// An existing directory returns [ErrDirExist], unless DirOptions.ExistOk
// accepts it. Then false is returned, and the directory is left untouched.
// With DirOptions.UpdateTimes, its access and modification times are set to
// the current time, as [SetTimes] does. An existing path that is no directory
// returns [ErrNotDir]. This includes a broken symlink.
//
// A DirOptions.Mode with bits outside [PermissionBits] returns
// [ErrInvalidPermission]. A missing parent directory returns
// [ErrParentNotExist], and denied access returns [ErrPermissionDenied]. Any
// other failure to check the path returns [ErrStat], any other failed
// creation returns [ErrCreate], and any other failed update of the times
// returns [ErrSetTimes].
func MkDirWithOptions(path *Path, options DirOptions) (bool, error) {
	err := checkPermission(options.Mode, path)
	if err != nil {
		return false, err
	}

	check := func() (bool, error) {
		return checkCreateTarget(path, path.IsDir, options.ExistOk, options.UpdateTimes, ErrNotDir, ErrDirExist)
	}

	exists, err := check()
	if err != nil || exists {
		return false, err
	}

	if options.Mode == 0 {
		options.Mode = DefaultDirMode()
	}

	if options.CreateAll {
		err = os.MkdirAll(path.String(), options.Mode)
	} else {
		err = os.Mkdir(path.String(), options.Mode)
	}

	if err != nil {
		// Another process may have created the path since the check. Checking
		// again returns the same error as an existing path before the call.
		if errors.Is(err, fs.ErrExist) {
			exists, checkErr := check()
			if checkErr != nil || exists {
				return false, checkErr
			}
		}
		return false, osCreateErr(ErrCreate, err, *path)
	}

	return true, nil
}

// CreateSymlink creates a symbolic link at symlinkPath that points to
// symlinkTarget, which may be relative or absolute. The parent directory of
// symlinkPath must exist. It wraps [os.Symlink].
//
// An existing symlinkPath returns [ErrExist], and this includes a broken
// symlink. A missing parent directory returns [ErrParentNotExist], and denied
// access returns [ErrPermissionDenied]. Any other failure to check the path
// returns [ErrStat], and any other failed creation returns [ErrCreate].
func CreateSymlink(symlinkTarget, symlinkPath *Path) error {
	exists, err := lexists(symlinkPath)
	if err != nil {
		return err
	}
	if exists {
		return wrapErr(ErrExist, fs.ErrExist, *symlinkPath)
	}

	if !symlinkPath.Parent().Exists() {
		return wrapErr(ErrParentNotExist, fs.ErrNotExist, *symlinkPath)
	}

	err = os.Symlink(symlinkTarget.String(), symlinkPath.String())
	if err != nil {
		return osCreateErr(ErrCreate, err, *symlinkPath)
	}

	return nil
}

// SymlinkTo creates a symbolic link at linkPath that points to this Path, as
// [CreateSymlink] does.
//
// If this Path does not exist, [ErrNotExist] is returned. The errors of
// CreateSymlink apply.
func (p *Path) SymlinkTo(linkPath *Path) error {
	if !p.Exists() {
		return wrapErr(ErrNotExist, fs.ErrNotExist, *p)
	}

	return CreateSymlink(p, linkPath)
}

// checkCreateTarget checks the path an entry is created at and reports whether
// the path exists. isKind reports whether the existing path has the kind of
// the created entry.
//
// An existing path of another kind returns notKind, and an existing path of
// the kind returns exist unless existOk is set. An accepted existing entry has
// its access and modification times set to the current time if updateTimes is
// set. A path that cannot be checked returns the error of [lexists], and times
// that cannot be set return the errors of [SetTimes].
func checkCreateTarget(
	path *Path, isKind func() bool, existOk, updateTimes bool, notKind, exist *PathlibError,
) (bool, error) {
	exists, err := lexists(path)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	if !isKind() {
		return true, pathErr(notKind, *path)
	}

	if !existOk {
		return true, wrapErr(exist, fs.ErrExist, *path)
	}

	if updateTimes {
		now := time.Now()
		return true, SetTimes(path, now, now)
	}

	return true, nil
}
