package pathlib

import (
	"io/fs"
	"os"
)

/*
SymlinkTo creates a symbolic link at the source path pointing to the target path.

This path must exist.

This function uses CreateSymlink.
*/
func (p *Path) SymlinkTo(linkPath *Path) error {
	if !p.Exists() {
		return pathErr(ErrNotExist, *p)
	}

	return CreateSymlink(p, linkPath)
}

/*
ReadSymlinkTarget reads the target path for this Path.

This Path must be a symlink.
*/
func (p *Path) ReadSymlinkTarget() (*Path, error) {
	if !p.IsSymlink() {
		return nil, pathErr(ErrNotSymlink, *p)
	}

	target, err := os.Readlink(p.String())
	if err != nil {
		return nil, err
	}

	return NewPath(target), nil
}

/*
SetPermission sets the permission mode for the specified path.
*/
func SetPermission(path *Path, mode fs.FileMode) error {
	return os.Chmod(path.String(), mode)
}

/*
CreateFile creates the file at the defined path with mode 0644.
If the file already exists, it will be truncated.

Parent directories must exist.
*/
func CreateFile(path *Path) error {
	_, err := CreateFileWithOptions(path, DefaultFileOptions())
	return err
}

/*
CreateFileWithOptions creates the file at the defined path with given options.

If FileOptions.ExistOk is true and the file already exists, no action is taken and false is returned.
Parent directories must exist.

FileOptions.Mode can never be set explicitly to 0000. This is not allowed by the operating system
and defaults to DefaultFileMode.

Returns true if a new file was created, false otherwise.
*/
func CreateFileWithOptions(path *Path, options FileOptions) (bool, error) {
	if path.Exists() {
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

	file, err := os.OpenFile(path.String(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, options.Mode)
	if err != nil {
		return false, err
	}

	err = file.Close()

	return true, err
}

/*
MkDir creates the directory at the defined path with mode 0755.
Parent directories must exist.
*/
func MkDir(path *Path) error {
	_, err := MkDirWithOptions(path, DefaultDirOptions())
	return err
}

/*
MkDirWithOptions creates the directory at the defined path with given options.
If ExistOk is true and the directory already exists, no action is taken.

DirOptions.Mode can never be set explicitly to 0000. This is not allowed by the operating system
and defaults to DefaultDirMode.

Returns true if a new directory was created, false otherwise.
*/
func MkDirWithOptions(path *Path, options DirOptions) (bool, error) {
	if path.Exists() {
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

	var err error
	if options.CreateAll {
		err = os.MkdirAll(path.String(), options.Mode)
	} else {
		err = os.Mkdir(path.String(), options.Mode)
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

/*
CreateSymlink creates a symlink at the symlinkPath that points to symlinkTarget.

symlinkTarget may be relative or absolute.

symlinkPath may not exist, but parent directory should.
*/
func CreateSymlink(symlinkTarget, symlinkPath *Path) error {
	if symlinkPath.Exists() {
		return pathErr(ErrExist, *symlinkPath)
	}

	if !symlinkPath.Parent().Exists() {
		return pathErr(ErrParentNotExist, *symlinkPath)
	}

	return os.Symlink(symlinkTarget.String(), symlinkPath.String())
}
