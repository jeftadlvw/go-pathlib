package pathlib

import (
	"os"
)

/*
DisposeFunc removes a temporary path created by CreateTempFile or CreateTempDir.

It is returned separately from the path, so code the path is passed to cannot
remove it. Only the creator, who holds the DisposeFunc, can.

A symlink that replaced the temporary path is removed itself, never its target.
Calling a DisposeFunc again is a no-op.
*/
type DisposeFunc func() error

// voidDispose is returned when no temporary path was created.
func voidDispose() error {
	return nil
}

/*
TempPathOptions is a struct that contains options for more control over the creation
of a temporary path.
*/
type TempPathOptions struct {
	/*
		BaseDir is the base directory for the temporary path. It must be an existing directory.
		A missing path returns ErrNotExist, an existing non-directory returns ErrNotDir.

		If set to NewPath(""), the current working directory is used, because NewPath("") translates to NewPath(".").
		Keep nil for default os temp directory.
	*/
	BaseDir *Path

	/*
		Prefix is the prefix added to the temporary path element.
	*/
	Prefix string
}

func (t *TempPathOptions) toUsableValues() (string, string, error) {
	if t == nil {
		return "", "", nil
	}

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

/*
CreateTempFile creates a new temporary file.

It's the caller's responsibility to call the returned DisposeFunc. It is never
nil, and a no-op if an error is returned.

Example:

	tempFile, dispose, err := CreateTempFile()
	if err != nil {
		// handle error
	}
	defer func() { _ = dispose() }()

The errors of [CreateTempFileWithOptions] apply.
*/
func CreateTempFile() (*Path, DisposeFunc, error) {
	return CreateTempFileWithOptions(nil)
}

/*
CreateTempFileWithOptions creates a temporary file with further options.

It's the caller's responsibility to call the returned DisposeFunc. It is never
nil, and a no-op if an error is returned.

Example:

	options := &TempPathOptions{
		BaseDir: NewPath("TEMP"),
		Prefix: "foo"
	}

	tempFile, dispose, err := CreateTempFileWithOptions(options)
	if err != nil {
		// handle error
	}
	defer func() { _ = dispose() }()

A TempPathOptions.BaseDir that is missing returns [ErrNotExist], and one that is
not a directory returns [ErrNotDir]. A failed creation returns [ErrCreate] with
the base directory as its path.
*/
func CreateTempFileWithOptions(options *TempPathOptions) (*Path, DisposeFunc, error) {
	tempBaseDir, prefix, err := options.toUsableValues()
	if err != nil {
		return nil, voidDispose, err
	}

	file, err := os.CreateTemp(tempBaseDir, prefix)
	if err != nil {
		return nil, voidDispose, wrapErr(ErrCreate, err, *tempDirOrDefault(tempBaseDir))
	}

	_ = file.Close()
	pathName := file.Name()

	// The DisposeFunc uses its own Path, independent of the returned one.
	dispose := func() error {
		return Remove(NewPath(pathName))
	}

	return NewPath(pathName), dispose, nil
}

/*
CreateTempDir creates a new temporary directory.

It's the caller's responsibility to call the returned DisposeFunc. It is never
nil, and a no-op if an error is returned.

Example:

	tempDir, dispose, err := CreateTempDir()
	if err != nil {
		// handle error
	}
	defer func() { _ = dispose() }()

The errors of [CreateTempDirWithOptions] apply.
*/
func CreateTempDir() (*Path, DisposeFunc, error) {
	return CreateTempDirWithOptions(nil)
}

/*
CreateTempDirWithOptions creates a temporary directory with further options.

It's the caller's responsibility to call the returned DisposeFunc. It is never
nil, and a no-op if an error is returned.

Example:

	options := &TempPathOptions{
		BaseDir: NewPath("TEMP"),
		Prefix: "foo"
	}

	tempDir, dispose, err := CreateTempDirWithOptions(options)
	if err != nil {
		// handle error
	}
	defer func() { _ = dispose() }()

A TempPathOptions.BaseDir that is missing returns [ErrNotExist], and one that is
not a directory returns [ErrNotDir]. A failed creation returns [ErrCreate] with
the base directory as its path.
*/
func CreateTempDirWithOptions(options *TempPathOptions) (*Path, DisposeFunc, error) {
	tempBaseDir, prefix, err := options.toUsableValues()
	if err != nil {
		return nil, voidDispose, err
	}

	dirName, err := os.MkdirTemp(tempBaseDir, prefix)
	if err != nil {
		return nil, voidDispose, wrapErr(ErrCreate, err, *tempDirOrDefault(tempBaseDir))
	}

	// The DisposeFunc uses its own Path, independent of the returned one.
	dispose := func() error {
		return RemoveAll(NewPath(dirName))
	}

	return NewPath(dirName), dispose, nil
}

func TempBaseDir() *Path {
	return NewPath(os.TempDir())
}

// tempDirOrDefault returns the directory a temporary path is created in, which
// is the OS temp directory when baseDir is empty.
func tempDirOrDefault(baseDir string) *Path {
	if baseDir == "" {
		return TempBaseDir()
	}

	return NewPath(baseDir)
}
