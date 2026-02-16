package pathlib

import (
	"errors"
	"os"
)

/*
Disposable interface shows that a struct has resources that must be disposed manually.
*/
type Disposable interface {
	// Dispose cleans up struct resources.
	Dispose() error
}

/*
TempPath is a container for a temporary path.
*/
type TempPath struct {
	Path

	// dispose is an internal function that removes the temporary path.
	dispose func() error
}

func (p *TempPath) Dispose() error {
	if p.dispose == nil {
		return errors.New("dispose function is nil")
	}

	return p.dispose()
}

/*
TempPathOptions is a struct that contains options for more control over the creation
of a temporary path.
*/
type TempPathOptions struct {
	/*
		BaseDir is the base directory for the temporary path.

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
		if !t.BaseDir.IsDir() {
			return "", "", errors.New("BaseDir in option struct is not a directory")
		}

		tempBaseDir = t.BaseDir.String()
	}

	return tempBaseDir, t.Prefix, nil
}

/*
CreateTempFile creates a new temporary file.

It's the caller's responsibility to call TempPath.Dispose().

Example:

	tempFile, err := CreateTempFile()
	if err != nil {
		// handle error
	}
	defer func() { _ = tempFile.Dispose() }()
*/
func CreateTempFile() (*TempPath, error) {
	return CreateTempFileWithOptions(nil)
}

/*
CreateTempFileWithOptions creates a temporary file with further options.

It's the caller's responsibility to call TempPath.Dispose().

Example:

	options := &TempPathOptions{
		BaseDir: NewPath("TEMP"),
		Prefix: "foo"
	}

	tempFile, err := CreateTempFileWithOptions(options)
	if err != nil {
		// handle error
	}
	defer func() { _ = tempFile.Dispose() }()
*/
func CreateTempFileWithOptions(options *TempPathOptions) (*TempPath, error) {
	tempBaseDir, prefix, err := options.toUsableValues()
	if err != nil {
		return nil, err
	}

	file, err := os.CreateTemp(tempBaseDir, prefix)
	if err != nil {
		return nil, err
	}

	_ = file.Close()
	pathName := file.Name()

	tempFilePath := *NewPath(pathName)

	return &TempPath{
		Path: tempFilePath,
		dispose: func() error {
			return os.Remove(pathName)
		},
	}, nil
}

/*
CreateTempDir creates a new temporary directory.
It's the caller's responsibility to call TempPath.Dispose().

Example:

	tempDir, err := CreateTempDir()
	if err != nil {
		// handle error
	}
	defer func() { _ = tempDir.Dispose() }()
*/
func CreateTempDir() (*TempPath, error) {
	return CreateTempDirWithOptions(nil)
}

/*
CreateTempDirWithOptions creates a temporary directory with further options.

It's the caller's responsibility to call TempPath.Dispose().

Example:

	options := &TempPathOptions{
		BaseDir: NewPath("TEMP"),
		Prefix: "foo"
	}

	tempDir, err := CreateTempDirWithOptions(options)
	if err != nil {
		// handle error
	}
	defer func() { _ = tempDir.Dispose() }()
*/
func CreateTempDirWithOptions(options *TempPathOptions) (*TempPath, error) {
	tempBaseDir, prefix, err := options.toUsableValues()
	if err != nil {
		return nil, err
	}

	dirName, err := os.MkdirTemp(tempBaseDir, prefix)
	if err != nil {
		return nil, err
	}

	tempDirPath := *NewPath(dirName)

	return &TempPath{
		Path: tempDirPath,
		dispose: func() error {
			return os.RemoveAll(dirName)
		},
	}, nil
}

func TempBaseDir() *Path {
	return NewPath(os.TempDir())
}
