// temp.go holds the creation of temporary files and directories.

package pathlib

import (
	"os"
)

// DisposeFunc removes a temporary path created by [CreateTempFile] or
// [CreateTempDir]. Calling it again is a no-op.
//
// It is returned next to the path, so the creator alone can remove the path,
// and the code the path is passed to cannot. A symlink that replaced the
// temporary path is removed itself, never its target.
type DisposeFunc func() error

// TempPathOptions configures the creation of a temporary path. The zero value
// creates the path in the temporary directory of the operating system,
// without a prefix.
type TempPathOptions struct {
	// BaseDir is the existing directory the path is created in. Nil selects
	// TempBaseDir. NewPath("") is ".", so it selects the working directory.
	BaseDir *Path

	// Prefix starts the name of the temporary path.
	Prefix string
}

// CreateTempFile creates an empty file in the temporary directory of the
// operating system, as [CreateTempFileWithOptions] does with the zero
// [TempPathOptions].
//
// The errors of CreateTempFileWithOptions apply.
func CreateTempFile() (*Path, DisposeFunc, error) {
	return CreateTempFileWithOptions(TempPathOptions{})
}

// CreateTempFileWithOptions creates an empty file with a unique name in the
// directory and with the prefix of options. It wraps [os.CreateTemp].
//
// The caller calls the returned [DisposeFunc] to remove the file. It is never
// nil, and a no-op if an error is returned.
//
// A missing TempPathOptions.BaseDir returns [ErrNotExist], and one that is no
// directory returns [ErrNotDir]. Denied access returns [ErrPermissionDenied],
// and any other failed creation returns [ErrCreate], both with the base
// directory as their path.
func CreateTempFileWithOptions(options TempPathOptions) (*Path, DisposeFunc, error) {
	tempBaseDir, prefix, err := options.toUsableValues()
	if err != nil {
		return nil, voidDispose, err
	}

	file, err := os.CreateTemp(tempBaseDir, prefix)
	if err != nil {
		return nil, voidDispose, osErr(ErrCreate, err, *tempDirOrDefault(tempBaseDir))
	}

	_ = file.Close()
	pathName := file.Name()

	// The DisposeFunc uses its own Path, independent of the returned one.
	dispose := func() error {
		return Remove(NewPath(pathName))
	}

	return NewPath(pathName), dispose, nil
}

// CreateTempDir creates an empty directory in the temporary directory of the
// operating system, as [CreateTempDirWithOptions] does with the zero
// [TempPathOptions].
//
// The errors of CreateTempDirWithOptions apply.
func CreateTempDir() (*Path, DisposeFunc, error) {
	return CreateTempDirWithOptions(TempPathOptions{})
}

// CreateTempDirWithOptions creates an empty directory with a unique name in
// the directory and with the prefix of options. It wraps [os.MkdirTemp].
//
// The caller calls the returned [DisposeFunc] to remove the directory with its
// whole tree. It is never nil, and a no-op if an error is returned.
//
// A missing TempPathOptions.BaseDir returns [ErrNotExist], and one that is no
// directory returns [ErrNotDir]. Denied access returns [ErrPermissionDenied],
// and any other failed creation returns [ErrCreate], both with the base
// directory as their path.
func CreateTempDirWithOptions(options TempPathOptions) (*Path, DisposeFunc, error) {
	tempBaseDir, prefix, err := options.toUsableValues()
	if err != nil {
		return nil, voidDispose, err
	}

	dirName, err := os.MkdirTemp(tempBaseDir, prefix)
	if err != nil {
		return nil, voidDispose, osErr(ErrCreate, err, *tempDirOrDefault(tempBaseDir))
	}

	// The DisposeFunc uses its own Path, independent of the returned one.
	dispose := func() error {
		return RemoveAll(NewPath(dirName))
	}

	return NewPath(dirName), dispose, nil
}

// TempBaseDir returns the temporary directory of the operating system. It
// wraps [os.TempDir].
func TempBaseDir() *Path {
	return NewPath(os.TempDir())
}

// toUsableValues returns the base directory and the prefix of the options as
// the os package takes them. The base directory is "" for the temporary
// directory of the operating system. A BaseDir that is no existing directory
// returns the error of [requireDir].
func (t TempPathOptions) toUsableValues() (string, string, error) {
	tempBaseDir := ""

	if t.BaseDir != nil {
		err := requireDir(t.BaseDir)
		if err != nil {
			return "", "", err
		}

		tempBaseDir = t.BaseDir.String()
	}

	return tempBaseDir, t.Prefix, nil
}

// voidDispose is the DisposeFunc returned when no temporary path was created.
func voidDispose() error {
	return nil
}

// tempDirOrDefault returns the directory a temporary path is created in, which
// is the temporary directory of the operating system when baseDir is empty.
func tempDirOrDefault(baseDir string) *Path {
	if baseDir == "" {
		return TempBaseDir()
	}

	return NewPath(baseDir)
}
