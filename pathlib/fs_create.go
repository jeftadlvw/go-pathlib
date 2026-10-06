package pathlib

import (
	"errors"
	"io/fs"
	"os"
)

/*
SymlinkTo creates a symbolic link at the source path pointing to the target path.

This path must exist, else [ErrNotExist] is returned.

This function uses CreateSymlink, and the errors of [CreateSymlink] apply.
*/
func (p *Path) SymlinkTo(linkPath *Path) error {
	if !p.Exists() {
		return pathErr(ErrNotExist, *p)
	}

	return CreateSymlink(p, linkPath)
}

/*
ReadSymlinkTarget reads the target path for this Path.

This Path must be a symlink, else [ErrNotSymlink] is returned. A target that
cannot be read returns [ErrReadSymlink].
*/
func (p *Path) ReadSymlinkTarget() (*Path, error) {
	if !p.IsSymlink() {
		return nil, pathErr(ErrNotSymlink, *p)
	}

	target, err := os.Readlink(p.String())
	if err != nil {
		return nil, wrapErr(ErrReadSymlink, err, *p)
	}

	return NewPath(target), nil
}

/*
SetPermission sets the permission mode for the specified path.

The mode may only contain [PermissionBits]. Any other bit returns
[ErrPermissionRange]. A failure of the operating system returns
[ErrSetPermission].

Unlike on creation, the setuid and setgid bits are set reliably.
*/
func SetPermission(path *Path, mode fs.FileMode) error {
	err := checkPermission(mode, path)
	if err != nil {
		return err
	}

	err = os.Chmod(path.String(), mode)
	if err != nil {
		return wrapErr(ErrSetPermission, err, *path)
	}

	return nil
}

/*
CreateFile creates the file at the defined path with DefaultFileMode.
If the file already exists, ErrFileExist is returned and the file is left untouched.

Parent directories must exist.

Unlike os.Create, CreateFile never truncates. Use CreateFileWithOptions with
FileOptions.ExistOk to accept an existing file, or OpenFile to create or truncate it.

The errors of [CreateFileWithOptions] apply.
*/
func CreateFile(path *Path) error {
	_, err := CreateFileWithOptions(path, DefaultFileOptions())
	return err
}

/*
CreateFileWithOptions creates the file at the defined path with given options.

If FileOptions.ExistOk is true and the file already exists, no action is taken and false is returned.
Otherwise an existing file returns [ErrFileExist]. Parent directories must
exist.

An existing path that is not a file returns ErrNotFile. This includes a broken symlink,
whose target is never created.

FileOptions.Mode can never be set explicitly to 0000. This is not allowed by the operating system
and defaults to DefaultFileMode. Any bit outside PermissionBits returns ErrPermissionRange.

A path that cannot be checked returns [ErrStat], and a failed creation returns
[ErrCreate]. ErrCreate also matches [ErrNotExist] for a missing parent
directory.

Returns true if a new file was created, false otherwise.
*/
func CreateFileWithOptions(path *Path, options FileOptions) (bool, error) {
	err := checkPermission(options.Mode, path)
	if err != nil {
		return false, err
	}

	exists, err := lexists(path)
	if err != nil {
		return false, err
	}

	if exists {
		if !path.IsFile() {
			return false, pathErr(ErrNotFile, *path)
		}
		if options.ExistOk {
			return false, nil
		}
		return false, pathErr(ErrFileExist, *path)
	}

	if options.Mode == 0 {
		options.Mode = DefaultFileMode
	}

	// O_EXCL refuses a path created since the check, symlinks included, so an
	// existing file is never truncated and no symlink target is created.
	file, err := os.OpenFile(path.String(), os.O_RDWR|os.O_CREATE|os.O_EXCL, options.Mode)
	if err != nil {
		if options.ExistOk && errors.Is(err, fs.ErrExist) && path.IsFile() {
			return false, nil
		}
		return false, wrapErr(ErrCreate, err, *path)
	}

	err = file.Close()
	if err != nil {
		return true, wrapErr(ErrCreate, err, *path)
	}

	return true, nil
}

/*
MkDir creates the directory at the defined path with mode 0755.
Parent directories must exist.

The errors of [MkDirWithOptions] apply.
*/
func MkDir(path *Path) error {
	_, err := MkDirWithOptions(path, DefaultDirOptions())
	return err
}

/*
MkDirWithOptions creates the directory at the defined path with given options.
If ExistOk is true and the directory already exists, no action is taken.
Otherwise an existing directory returns [ErrDirExist].

An existing path that is not a directory returns ErrNotDir. This includes a broken symlink.

DirOptions.Mode can never be set explicitly to 0000. This is not allowed by the operating system
and defaults to DefaultDirMode. Any bit outside PermissionBits returns ErrPermissionRange.

A path that cannot be checked returns [ErrStat], and a failed creation returns
[ErrCreate]. ErrCreate also matches [ErrNotExist] for a missing parent directory
without DirOptions.CreateAll.

Returns true if a new directory was created, false otherwise.
*/
func MkDirWithOptions(path *Path, options DirOptions) (bool, error) {
	err := checkPermission(options.Mode, path)
	if err != nil {
		return false, err
	}

	exists, err := lexists(path)
	if err != nil {
		return false, err
	}

	if exists {
		if !path.IsDir() {
			return false, pathErr(ErrNotDir, *path)
		}
		if options.ExistOk {
			return false, nil
		}
		return false, pathErr(ErrDirExist, *path)
	}

	if options.Mode == 0 {
		options.Mode = DefaultDirMode
	}

	if options.CreateAll {
		err = os.MkdirAll(path.String(), options.Mode)
	} else {
		err = os.Mkdir(path.String(), options.Mode)
	}

	if err != nil {
		// Another process may have created the directory since the check.
		if options.ExistOk && errors.Is(err, fs.ErrExist) && path.IsDir() {
			return false, nil
		}
		return false, wrapErr(ErrCreate, err, *path)
	}

	return true, nil
}

/*
CreateSymlink creates a symlink at the symlinkPath that points to symlinkTarget.

symlinkTarget may be relative or absolute.

symlinkPath may not exist, but parent directory should. A broken symlink at
symlinkPath exists too and returns [ErrExist].

A missing parent directory returns [ErrParentNotExist]. A path that cannot be
checked returns [ErrStat], and a failed creation returns [ErrCreate].
*/
func CreateSymlink(symlinkTarget, symlinkPath *Path) error {
	exists, err := lexists(symlinkPath)
	if err != nil {
		return err
	}
	if exists {
		return pathErr(ErrExist, *symlinkPath)
	}

	if !symlinkPath.Parent().Exists() {
		return pathErr(ErrParentNotExist, *symlinkPath)
	}

	err = os.Symlink(symlinkTarget.String(), symlinkPath.String())
	if err != nil {
		return wrapErr(ErrCreate, err, *symlinkPath)
	}

	return nil
}
