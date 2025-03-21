package pathlib

import (
	"errors"
	"fmt"
	"os"
)

const defaultOpenPermission = 0644 // rw-r--r--
const defaultOpenMode = "rw"

/*
OpenOptions is a configuration struct for opening files.
*/
type OpenOptions struct {
	// Create the file if it does not exist.
	CreateIfNotExists bool

	// Permissions for file creation.
	Permission os.FileMode

	// Open mode. Loosely defined as a string that may only contain "r" (read), "w" (write) and "a" (append).
	// Enforced by functions that receive this struct.
	Mode string
}

func defaultOpenOptions() OpenOptions {
	return OpenOptions{
		true,
		defaultOpenPermission,
		defaultOpenMode,
	}
}

/*
OpenFile opens a file for reading and writing.
If the file does not exist, it is created with 0644 permissions. If the file already
exists, it is truncated.

It's the callers responsibility to close the os.File.
*/
func OpenFile(path *Path) (*os.File, error) {
	return OpenFileWithOptions(path, defaultOpenOptions())
}

/*
OpenFileWithOptions opens a file with passed extended configuration.

If OpenOptions.Permission is 0, the value defaults to 0644.
If OpenOptions.Mode is an empty string, it defaults to "rw"

The order for OpenOptions.Mode is enforced as follows: "r" (read), "w" (write), "a" (append) must be used
in exactly this order. "a" can only be used if "w" is used.

It's the callers responsibility to close the os.File.
*/
func OpenFileWithOptions(path *Path, opts OpenOptions) (*os.File, error) {

	if opts.Permission < 0 || opts.Permission > 0777 {
		return nil, errors.New("permission out of bounds. min: 0, max: Oo777")
	}

	// set default permission value
	if opts.Permission == 0 {
		opts.Permission = defaultOpenPermission
	}

	// set default open mode
	if len(opts.Mode) == 0 {
		opts.Mode = defaultOpenMode
	}

	// determine file open mode from the mode string
	var fileOpenMode int
	switch opts.Mode {
	case "r":
		fileOpenMode = os.O_RDONLY
	case "w":
		fileOpenMode = os.O_WRONLY | os.O_TRUNC
	case "rw":
		fileOpenMode = os.O_RDWR | os.O_TRUNC
	case "wa":
		fileOpenMode = os.O_WRONLY | os.O_APPEND
	case "rwa":
		fileOpenMode = os.O_RDWR | os.O_APPEND
	default:
		return nil, fmt.Errorf("unsupported mode: %s", opts.Mode)
	}

	if opts.CreateIfNotExists {
		fileOpenMode = fileOpenMode | os.O_CREATE
	}

	file, err := os.OpenFile(path.String(), fileOpenMode, opts.Permission)
	if err != nil {
		return nil, err
	}

	// Fun fact: unix based operating systems support open a file descriptor
	// on directory paths in readonly mode. Fuck that inconsistency and return an error
	// if the path is a directory.
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("could not read file stat to determine if path is a directory: %w", err)
	}

	if stat.IsDir() {
		_ = file.Close()
		return nil, fmt.Errorf("cannot open file: is a directory")
	}

	return file, nil
}

/*
ReadFile reads the passed file and returns read bytes.
*/
func ReadFile(path *Path) ([]byte, error) {
	return os.ReadFile(path.String())
}

/*
ReadFileToString reads the passed file and returns its content as a string.
*/
func ReadFileToString(path *Path) (string, error) {
	bytes, err := os.ReadFile(path.String())
	return string(bytes), err
}

/*
WriteBytes writes raw byte data to the defined file.

The file is not created. Preexisting content is truncated.
*/
func WriteBytes(path *Path, data []byte) (int, error) {
	return writeBytes(path, data, "w")
}

/*
WriteString writes a string data to the defined file.

The file is not created. Preexisting content is truncated.
*/
func WriteString(path *Path, data string) (int, error) {
	return WriteBytes(path, []byte(data))
}

/*
AppendBytes appends byte data to the defined file.

The file is not created.
*/
func AppendBytes(path *Path, data []byte) (int, error) {
	return writeBytes(path, data, "wa")
}

/*
AppendString appends a string to the defined file.

The file is not created.
*/
func AppendString(path *Path, data string) (int, error) {
	return AppendBytes(path, []byte(data))
}

/*
writeBytes is an internal function that writes data to a file, opened in a specific mode.
*/
func writeBytes(path *Path, data []byte, mode string) (int, error) {
	file, err := OpenFileWithOptions(path, OpenOptions{
		CreateIfNotExists: false,
		Mode:              mode,
	})

	if err != nil {
		return 0, err
	}

	defer file.Close()

	return file.Write(data)
}
