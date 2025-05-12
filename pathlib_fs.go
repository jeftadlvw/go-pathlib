package pathlib

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

/*
TODO windows compatibility
TODO testing
*/

/*
defaultFileMode is the default file permission mode (rw-r--r--)
*/
const defaultFileMode fs.FileMode = 0644

/*
defaultDirMode is the default directory permission mode (rwxr-xr-x)
*/
const defaultDirMode fs.FileMode = 0755

/*
FileOptions contains options for file and directory creation and deletion operations
*/
type FileOptions struct {
	// ExistOk specifies whether it's acceptable if the file/directory already exists
	ExistOk bool
	// Mode specifies the file/directory permission mode
	Mode fs.FileMode
}

/*
defaultFileOptions returns the default options for file operations.
*/
func defaultFileOptions() FileOptions {
	return FileOptions{
		ExistOk: false,
		Mode:    defaultFileMode,
	}
}

/*
defaultDirOptions returns the default options for directory operations.
*/
func defaultDirOptions() FileOptions {
	return FileOptions{
		ExistOk: false,
		Mode:    defaultDirMode,
	}
}

/*
IsFile returns whether this Path is an existing file.
*/
func (p *Path) IsFile() bool {
	return pathCheck(p) == pathCheckFile
}

/*
IsDir returns whether this Path is an existing directory.
*/
func (p *Path) IsDir() bool {
	return pathCheck(p) == pathCheckDir
}

/*
Exists returns whether this Path exists.
*/
func (p *Path) Exists() bool {
	return pathCheck(p) != pathCheckNoExist
}

/*
Resolve resolves all symbolic links and ensures an absolute path representation.

This function utilizes filepath.EvalSymlinks.
*/
func (p *Path) Resolve() (*Path, error) {
	if !p.Exists() {
		return nil, errors.New("this path does not exist")
	}

	ep, err := filepath.EvalSymlinks(p.path)
	if err != nil {
		return nil, err
	}

	return NewPath(ep).Absolute()
}

/*
Glob returns all paths matching the given pattern within this Path's directory.

This function utilizes filepath.Glob. It ignores IO errors.
*/
func (p *Path) Glob(pattern string) ([]*Path, error) {
	matches, err := nativeGlob(p, pattern, 0)
	if err != nil {
		return nil, err
	}

	paths := make([]*Path, len(matches))
	for idx, match := range matches {
		paths[idx] = NewPath(match)
	}

	return paths, nil
}

/*
HasGlobMatchE returns whether the passed pattern exist within this Path's directory.

This function utilizes filepath.Glob.
*/
func (p *Path) HasGlobMatchE(pattern string) (bool, error) {
	matches, err := nativeGlob(p, pattern, 1)
	if err != nil {
		return false, err
	}

	return len(matches) != 0, nil
}

/*
HasGlobMatch returns whether the passed pattern exists within this Path's directory.
It wraps HasGlobMatchE and returns the boolean success value or false in case of an error.
*/
func (p *Path) HasGlobMatch(pattern string) bool {
	contains, err := p.HasGlobMatchE(pattern)
	if err != nil {
		return false
	}

	return contains
}

/*
MatchesPatternFsCasingE matches this path against the provided pattern.
Returns whether the matching is successful or any occurring error.

Matching is performed based on the file system case sensitivity.

Empty patterns are not allowed.
*/
func (p *Path) MatchesPatternFsCasingE(pattern string) (bool, error) {
	// TODO implement
	panic("not implemented")
}

/*
MatchesPatternFsCasing matches this path against the provided pattern.
It wraps MatchesPatternFsCasingE and returns the boolean success return value
or false in case of an error.
*/
func (p *Path) MatchesPatternFsCasing(pattern string) bool {
	match, err := p.MatchesPatternFsCasingE(pattern)
	if err != nil {
		return false
	}

	return match
}

/*
Stat returns file info for this Path.

This function utilizes os.Stat.
*/
func (p *Path) Stat() (os.FileInfo, error) {
	return os.Stat(p.String())
}

/*
Lstat returns file info for this Path, not following symbolic links.

This function utilizes os.Lstat.
*/
func (p *Path) Lstat() (os.FileInfo, error) {
	return os.Lstat(p.String())
}

/*
IsSymlink returns whether this Path is a symbolic link.
*/
func (p *Path) IsSymlink() bool {
	return checkFileMode(p, os.ModeSymlink)
}

/*
IsBlockDevice returns whether this Path is a block device.
*/
func (p *Path) IsBlockDevice() bool {
	return checkFileMode(p, os.ModeDevice) && !checkFileMode(p, os.ModeCharDevice)
}

/*
IsCharDevice returns whether this Path is a character device.
*/
func (p *Path) IsCharDevice() bool {
	return checkFileMode(p, os.ModeDevice) && checkFileMode(p, os.ModeCharDevice)
}

/*
IsFiFoPipe returns whether this Path is a FIFO/pipe.
*/
func (p *Path) IsFiFoPipe() bool {
	return checkFileMode(p, os.ModeNamedPipe)
}

/*
IsSocket returns whether this Path is a socket.
*/
func (p *Path) IsSocket() bool {
	return checkFileMode(p, os.ModeSocket)
}

/*
EqualsFs returns whether this Path and another Path point to the same file system object.
This comparison is performed using file stats, not string comparison.
*/
func (p *Path) EqualsFs(other *Path) bool {
	equals, err := EqualsFs(p, other)
	return equals && err == nil
}

/*
IsCaseSensitiveFs returns whether this Path is on a case-sensitive filesystem.
*/
func (p *Path) IsCaseSensitiveFs() bool {
	caseSensitive, err := IsCaseSensitiveFs(p)
	return caseSensitive && err == nil
}

/*
CreateFile creates the file at the defined path with mode 0644.
If the file already exists, it will be truncated.
Parent directories must exist.
*/
func CreateFile(path *Path) error {
	_, err := CreateFileWithOptions(path, defaultFileOptions())
	return err
}

/*
CreateFileWithOptions creates the file at the defined path with given options.

If FileOptions.ExistOk is true and the file already exists, no action is taken.
Parent directories must exist.

Returns true if a new file was created, false otherwise.
*/
func CreateFileWithOptions(path *Path, options FileOptions) (bool, error) {
	if path.Exists() {
		if !path.IsFile() {
			return false, errors.New("path exists and is not a file")
		}
		if options.ExistOk {
			return false, nil
		}
		return false, errors.New("file already exists")
	}

	file, err := os.OpenFile(path.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, options.Mode)
	if err != nil {
		return false, err
	}

	err = file.Close()

	return true, err
}

/*
CreateDir creates the directory at the defined path with mode 0755.
Parent directories must exist.
*/
func CreateDir(path *Path) error {
	_, err := CreateDirWithOptions(path, defaultDirOptions())
	return err
}

/*
CreateDirWithOptions creates the directory at the defined path with given options.
If ExistOk is true and the directory already exists, no action is taken.
Parent directories must exist.
Returns true if a new directory was created, false otherwise.
*/
func CreateDirWithOptions(path *Path, options FileOptions) (bool, error) {
	if path.Exists() {
		if !path.IsDir() {
			return false, errors.New("path exists and is not a directory")
		}
		if options.ExistOk {
			return false, nil
		}
		return false, errors.New("directory already exists")
	}

	err := os.Mkdir(path.path, options.Mode)
	if err != nil {
		return false, err
	}
	return true, nil
}

/*
CreateDirAll creates the directory at the defined path with mode 0755,
creating all necessary parent directories with the same mode.
*/
func CreateDirAll(path *Path) error {
	_, err := CreateDirAllWithOptions(path, defaultDirOptions())
	return err
}

/*
CreateDirAllWithOptions creates the directory at the defined path with given options,
creating all necessary parent directories with the same mode.
If ExistOk is true and the directory already exists, no action is taken.
Returns true if directories were created, false otherwise.
*/
func CreateDirAllWithOptions(path *Path, options FileOptions) (bool, error) {
	if path.Exists() {
		if !path.IsDir() {
			return false, errors.New("path exists and is not a directory")
		}
		if options.ExistOk {
			return false, nil
		}
		return false, errors.New("directory already exists")
	}

	err := os.MkdirAll(path.path, options.Mode)
	if err != nil {
		return false, err
	}
	return true, nil
}

/*
CopyFile copies the file at the source path to the destination path.
Destination parent directories must exist.
*/
func CopyFile(src, dst *Path) error {
	if !src.IsFile() {
		return errors.New("source path is not a file")
	}

	if dst.IsDir() {
		return errors.New("destination path is a directory")
	}

	// Create parent directories if they don't exist
	if !dst.Parent().Exists() {
		return errors.New("destination parent directory does not exist")
	}

	// Get source file info for permissions
	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	// Open source file
	sourceFile, err := os.Open(src.path)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.OpenFile(dst.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy contents
	_, err = io.Copy(destFile, sourceFile)
	return err
}

/*
CopyDir copies the directory at the source path to the destination path,
recursively copying all contents. Destination parent directories must exist.
*/
func CopyDir(src, dst *Path) error {
	if !src.IsDir() {
		return errors.New("source path is not a directory")
	}

	if dst.Exists() && !dst.IsDir() {
		return errors.New("destination exists and is not a directory")
	}

	// Create parent directories if they don't exist
	if !dst.Parent().Exists() {
		return errors.New("destination parent directory does not exist")
	}

	// Get source info for permissions
	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	// Create destination directory if it doesn't exist
	if !dst.Exists() {
		if err := os.Mkdir(dst.path, srcInfo.Mode()); err != nil {
			return err
		}
	}

	// Read directory entries
	entries, err := os.ReadDir(src.path)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcEntry := src.JoinStrings(entry.Name())
		dstEntry := dst.JoinStrings(entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcEntry, dstEntry); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcEntry, dstEntry); err != nil {
				return err
			}
		}
	}

	return nil
}

/*
MoveFile moves the file at the source path to the destination path.
If the destination is on the same filesystem, this is equivalent to a rename operation.
Destination parent directories must exist.
*/
func MoveFile(src, dst *Path) error {
	if !src.IsFile() {
		return errors.New("source path is not a file")
	}

	if dst.IsDir() {
		return errors.New("destination path is a directory")
	}

	// Create parent directories if they don't exist
	if !dst.Parent().Exists() {
		return errors.New("destination parent directory does not exist")
	}

	// Try renaming first (works if on same filesystem)
	err := os.Rename(src.path, dst.path)
	if err == nil {
		return nil
	}

	// If rename fails, try copy and delete
	if err := CopyFile(src, dst); err != nil {
		return err
	}

	return os.Remove(src.path)
}

/*
MoveDir moves the directory at the source path to the destination path.
If the destination is on the same filesystem, this is equivalent to a rename operation.
Destination parent directories must exist.
*/
func MoveDir(src, dst *Path) error {
	if !src.IsDir() {
		return errors.New("source path is not a directory")
	}

	if dst.Exists() && !dst.IsDir() {
		return errors.New("destination exists and is not a directory")
	}

	// Create parent directories if they don't exist
	if !dst.Parent().Exists() {
		return errors.New("destination parent directory does not exist")
	}

	// Try renaming first (works if on same filesystem)
	err := os.Rename(src.path, dst.path)
	if err == nil {
		return nil
	}

	// If rename fails, try copy and delete
	if err := CopyDir(src, dst); err != nil {
		return err
	}

	return DeleteTree(src)
}

// TODO
// 	Renaming files/directories should take an initial path and a target name and rename the base of the path.
// 	Returns a new Path and an optional error.
//
// TODO
//	Moving files/directories should take the initial (source) path and a target path
//	and perform the move operation.

/*
RenameFile renames the file at the source path to the destination path.
This is a wrapper for MoveFile.
*/
func RenameFile(src *Path, name string) error {
	return MoveFile(src, src.Parent().JoinStrings(name))
}

/*
RenameDir renames the directory at the source path to the destination path.
This is a wrapper for MoveDir.
*/
func RenameDir(src, dst *Path) error {
	return MoveDir(src, dst)
}

/*
DeleteFile removes the file at the specified path.
Returns an error if the path is a directory.
*/
func DeleteFile(path *Path) error {
	if path.IsDir() {
		return errors.New("path is a directory, use DeleteDir or DeleteTree instead")
	}
	return os.Remove(path.path)
}

/*
DeleteDir removes the directory at the specified path.
Returns an error if the directory is not empty.
*/
func DeleteDir(path *Path) error {
	if !path.IsDir() {
		return errors.New("path is not a directory")
	}
	return os.Remove(path.path)
}

/*
DeleteTree removes the directory at the specified path and all its contents.
*/
func DeleteTree(path *Path) error {
	if !path.Exists() {
		return nil
	}
	if !path.IsDir() {
		return errors.New("path is not a directory")
	}
	return os.RemoveAll(path.path)
}

/*
SetPerm sets the permission mode for the specified path.
*/
func SetPerm(path *Path, mode fs.FileMode) error {
	return os.Chmod(path.path, mode)
}

/*
CreateSymlink creates a symbolic link at the source path pointing to the target path.
*/
func CreateSymlink(src, target *Path) error {
	_, err := CreateSymlinkWithOptions(src, target, defaultFileOptions())
	return err
}

/*
CreateSymlinkWithOptions creates a symbolic link at the source path pointing to the target path.
If ExistOk is true and the source path already exists, no action is taken.
Returns true if a new symlink was created, false otherwise.
*/
func CreateSymlinkWithOptions(src, target *Path, options FileOptions) (bool, error) {
	if src.Exists() {
		if options.ExistOk {
			return false, nil
		}
		return false, errors.New("source path already exists")
	}

	err := os.Symlink(target.path, src.path)
	if err != nil {
		return false, err
	}

	return true, nil
}

/*
EqualsFs returns whether the two paths point to the same file system object.
This comparison is performed using file stats, not string comparison.
*/
func EqualsFs(p1, p2 *Path) (bool, error) {
	if !p1.Exists() || !p2.Exists() {
		return false, errors.New("one or both paths do not exist")
	}

	stat1, err := p1.Stat()
	if err != nil {
		return false, err
	}

	stat2, err := p2.Stat()
	if err != nil {
		return false, err
	}

	return os.SameFile(stat1, stat2), nil
}

/*
IsCaseSensitiveFs checks if the filesystem at the specified path is case-sensitive.

It first tries to check for the given path, toggling the casing of the first encountered letter in the path's base,
checking if the file exists and if both file descriptors point to the same file.

If no letter exists within the path's base, a temporary file is created in the same directory with which the upper procedure is repeated.

If both attempts don't result a valid state, an error is returned.
*/
func IsCaseSensitiveFs(p *Path) (bool, error) {
	if !p.Exists() {
		return false, errors.New("path does not exist")
	}

	// TODO Implement as stated in function documentation

	// Get a directory to test in
	var testDir *Path
	if p.IsDir() {
		testDir = p
	} else {
		testDir = p.Parent()
	}

	// TODO change case of file name and check if both exist and are same stat
	//  instead of creating a new file (which may fails due to permissions)

	// Create temporary file in directory with forced lowercase prefix.
	// A forced lowercase prefix is important so we can test the  filesystem case sensitivity
	// with an uppercase variant. It is also randomly generated so it is hard to be predicted.
	//
	// We cannot use CreateTempFile from pathlib_temp.go here, because they cannot be dependencies.
	tempFile, err := os.CreateTemp(testDir.String(), strings.ToLower(generateRandomString(5, 15)))
	if err != nil {
		return false, errors.New("unable to determine case sensitivity: temporary file creation failed")
	}

	// get file name, close and defer removal
	tempFilePathStr := tempFile.Name()
	_ = tempFile.Close()
	defer os.Remove(tempFilePathStr)

	tempFilePath := NewPath(tempFilePathStr)
	caseSwitchedPath := testDir.JoinStrings(strings.ToUpper(tempFilePath.Base()))

	// if caseSwitchedPath exists and tempFilePath and caseSwitchedPath are the same file on the filesystem,
	// then the filesystem is case-insensitive and we should return false.
	return !(caseSwitchedPath.Exists() && tempFilePath.EqualsFs(caseSwitchedPath)), nil
}

/*
pathCheck is a lower level Path existence checker.
It returns 0 if the path does not exist, 2 if it's a file and 2 if it's a directory.
*/
func pathCheck(p *Path) int {
	fileInfo, err := p.Stat()
	if err != nil {
		if os.IsNotExist(err) {
			return pathCheckNoExist
		}
	}

	if fileInfo == nil {
		return pathCheckNoExist
	}

	if fileInfo.IsDir() {
		return pathCheckDir
	}

	return pathCheckFile
}

/*
nativeGlob is a wrapper function for Go's filepath.Glob.
It checks if the passed Path exists and returns the raw matches or errors.

If n == 0, all matches are returned. Else, <= n matches are returned.

Returns an error if pattern is an empty string.

filepath.Glob ignores IO errors.
*/
func nativeGlob(p *Path, pattern string, n uint) ([]string, error) {
	// TODO use os.Readdir(https://pkg.go.dev/os#File.Readdir) for more granular control on amount
	// TODO support recursive globbing "**"

	if strings.TrimSpace(pattern) == "" {
		return nil, errors.New("pattern must not be empty")
	}

	if !p.Exists() {
		return nil, errors.New("this path does not exist")
	}

	if !p.IsDir() {
		return nil, errors.New("this path is not a directory")
	}

	matches, err := filepath.Glob(filepath.Join(p.path, pattern))
	if err != nil {
		return nil, err
	}

	return matches, nil
}

/*
checkFileMode is a helper function to check file mode bits on a passed path.
*/
func checkFileMode(p *Path, modeMask os.FileMode) bool {
	info, err := p.Lstat()
	if err != nil {
		return false
	}
	return info.Mode()&modeMask != 0
}
