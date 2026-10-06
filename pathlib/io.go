package pathlib

import (
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

	// Permissions for file creation. It may only contain PermissionBits.
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

It's the caller's responsibility to close the returned os.File.

The errors of [OpenFileWithOptions] apply.
*/
func OpenFile(path *Path) (*os.File, error) {
	return OpenFileWithOptions(path, defaultOpenOptions())
}

/*
OpenFileWithOptions opens a file with passed extended configuration.

If OpenOptions.Permission is 0, the value defaults to 0644. Any bit outside
PermissionBits returns ErrPermissionRange.
If OpenOptions.Mode is an empty string, it defaults to "rw"

The order for OpenOptions.Mode is enforced as follows: "r" (read), "w" (write), "a" (append) must be used
in exactly this order. "a" can only be used if "w" is used.

It's the caller's responsibility to close the returned os.File.

An unsupported mode returns [ErrUnsupportedMode]. A file that cannot be opened
returns [ErrOpen], which also matches [ErrNotExist] for a missing file. A
directory returns [ErrIsDir]. A failed creation returns [ErrCreate], and a
failed check of the opened path returns [ErrStat].
*/
func OpenFileWithOptions(path *Path, opts OpenOptions) (*os.File, error) {
	err := checkPermission(opts.Permission, path)
	if err != nil {
		return nil, err
	}

	// set the default permission value
	if opts.Permission == 0 {
		opts.Permission = defaultOpenPermission
	}

	// set the default open mode
	if len(opts.Mode) == 0 {
		opts.Mode = defaultOpenMode
	}

	fileOpenMode, err := openModeFlags(opts.Mode, path)
	if err != nil {
		return nil, err
	}

	if opts.CreateIfNotExists {
		if fileOpenMode == os.O_RDONLY && runningOnWindows {
			// On Windows, O_CREATE|O_RDONLY either fails for existing read-only files
			// (O_CREATE requires write access) or returns a writable handle for new files.
			// Separate creation from opening to guarantee a read-only handle.
			err = createIfMissing(path, opts.Permission)
			if err != nil {
				return nil, err
			}
		} else {
			fileOpenMode |= os.O_CREATE
		}
	}

	return openNonDir(path, fileOpenMode, opts.Permission)
}

/*
ReadFile reads the passed file and returns read bytes.

A file that cannot be read returns [ErrRead], which also matches [ErrNotExist]
for a missing file.
*/
func ReadFile(path *Path) ([]byte, error) {
	bytes, err := os.ReadFile(path.String())
	if err != nil {
		return nil, wrapErr(ErrRead, err, *path)
	}

	return bytes, nil
}

/*
ReadFileToString reads the passed file and returns its content as a string.

The errors of [ReadFile] apply.
*/
func ReadFileToString(path *Path) (string, error) {
	bytes, err := ReadFile(path)
	return string(bytes), err
}

/*
WriteBytes writes raw byte data to the defined file, like os.WriteFile.

A missing file is created with DefaultFileMode. Preexisting content is truncated.
Parent directories must exist.

This function uses WriteBytesWithOptions. The same behaviors apply.
*/
func WriteBytes(path *Path, data []byte) (int, error) {
	return WriteBytesWithOptions(path, data, FileOptions{ExistOk: true})
}

/*
WriteString writes a string to the defined file, like os.WriteFile.

A missing file is created with DefaultFileMode. Preexisting content is truncated.
Parent directories must exist.

This function uses WriteBytesWithOptions. The same behaviors apply.
*/
func WriteString(path *Path, data string) (int, error) {
	return WriteBytes(path, []byte(data))
}

/*
WriteBytesWithOptions writes raw byte data to the defined file with given options.

A missing file is created with FileOptions.Mode, which defaults to DefaultFileMode
if it is 0. Any bit outside PermissionBits returns ErrPermissionRange. The mode only
applies to a created file, an existing file keeps its permissions. Parent directories
must exist.

If FileOptions.ExistOk is true, preexisting content is truncated. Otherwise an
existing file returns ErrFileExist and is left untouched, as in CreateFileWithOptions.

A symlink to a file is written through. An existing path that is not a file returns
ErrNotFile. This includes a broken symlink, whose target is never created.

A path that cannot be checked returns [ErrStat]. A file that cannot be opened
returns [ErrOpen], which also matches [ErrNotExist] for a missing parent
directory. A failed write returns [ErrWrite].

Returns the number of bytes written.
*/
func WriteBytesWithOptions(path *Path, data []byte, options FileOptions) (int, error) {
	return writeBytes(path, data, os.O_TRUNC, options)
}

/*
WriteStringWithOptions writes a string to the defined file with given options.

This function uses WriteBytesWithOptions. The same behaviors apply.
*/
func WriteStringWithOptions(path *Path, data string, options FileOptions) (int, error) {
	return WriteBytesWithOptions(path, []byte(data), options)
}

/*
AppendBytes appends byte data to the defined file.

A missing file is created with DefaultFileMode. Parent directories must exist.

A symlink to a file is written through. An existing path that is not a file returns
ErrNotFile. This includes a broken symlink, whose target is never created.

A path that cannot be checked returns [ErrStat]. A file that cannot be opened
returns [ErrOpen], which also matches [ErrNotExist] for a missing parent
directory. A failed write returns [ErrWrite].
*/
func AppendBytes(path *Path, data []byte) (int, error) {
	return writeBytes(path, data, os.O_APPEND, FileOptions{ExistOk: true})
}

/*
AppendString appends a string to the defined file.

This function uses AppendBytes. The same behaviors apply.
*/
func AppendString(path *Path, data string) (int, error) {
	return AppendBytes(path, []byte(data))
}

/*
writeBytes is an internal function that writes data to a file, opened for writing
with the additional flag (os.O_TRUNC or os.O_APPEND). The file is created if it
does not exist, and options apply as described in WriteBytesWithOptions.
*/
func writeBytes(path *Path, data []byte, flag int, options FileOptions) (int, error) {
	err := checkPermission(options.Mode, path)
	if err != nil {
		return 0, err
	}

	err = checkWritable(path, options.ExistOk)
	if err != nil {
		return 0, err
	}

	if options.Mode == 0 {
		options.Mode = DefaultFileMode()
	}

	flag |= os.O_WRONLY | os.O_CREATE
	if !options.ExistOk {
		// O_EXCL refuses a path created since the check, symlinks included.
		flag |= os.O_EXCL
	}

	file, err := os.OpenFile(path.String(), flag, options.Mode)
	if err != nil {
		return 0, wrapErr(ErrOpen, err, *path)
	}

	n, err := file.Write(data)
	if err != nil {
		_ = file.Close()
		return n, wrapErr(ErrWrite, err, *path)
	}

	// Closing flushes the content, so its error means the write is incomplete.
	err = file.Close()
	if err != nil {
		return n, wrapErr(ErrWrite, err, *path)
	}

	return n, nil
}

/*
openModeFlags returns the flags for [os.OpenFile] that implement the open mode
of [OpenOptions]. An unsupported mode returns [ErrUnsupportedMode].
*/
func openModeFlags(mode string, path *Path) (int, error) {
	switch mode {
	case "r":
		return os.O_RDONLY, nil
	case "w":
		return os.O_WRONLY | os.O_TRUNC, nil
	case "rw":
		return os.O_RDWR | os.O_TRUNC, nil
	case "wa":
		return os.O_WRONLY | os.O_APPEND, nil
	case "rwa":
		return os.O_RDWR | os.O_APPEND, nil
	default:
		return 0, permModeErr(mode, *path)
	}
}

/*
createIfMissing creates an empty file with permission perm at path if nothing
exists there. A failed creation returns [ErrCreate].
*/
func createIfMissing(path *Path, perm os.FileMode) error {
	_, err := os.Stat(path.String())
	if err == nil {
		return nil
	}

	f, err := os.OpenFile(path.String(), os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return wrapErr(ErrCreate, err, *path)
	}

	_ = f.Close()
	return nil
}

/*
openNonDir opens the file at path with flag and perm as [os.OpenFile] does. A
file that cannot be opened returns [ErrOpen], a failed check of the opened path
returns [ErrStat], and a directory returns [ErrIsDir].
*/
func openNonDir(path *Path, flag int, perm os.FileMode) (*os.File, error) {
	file, err := os.OpenFile(path.String(), flag, perm)
	if err != nil {
		return nil, wrapErr(ErrOpen, err, *path)
	}

	// Fun fact: Unix-based operating systems support opening a file descriptor
	// on directory paths in readonly mode. We don't do that and return an error
	// if the path is a directory.
	stat, err := os.Stat(path.String())

	if err != nil {
		_ = file.Close()
		return nil, wrapErr(ErrStat, err, *path)
	}

	if stat.IsDir() {
		_ = file.Close()
		return nil, pathErr(ErrIsDir, *path)
	}

	return file, nil
}

/*
checkWritable checks that path can be written with the ExistOk option of
[FileOptions] set to existOk. An existing path that is no file returns
[ErrNotFile], and an existing file returns [ErrFileExist] unless existOk is set.
A path that cannot be checked returns [ErrStat].
*/
func checkWritable(path *Path, existOk bool) error {
	// Check the path itself, so a broken symlink is not written through.
	exists, err := lexists(path)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	if !path.IsFile() {
		return pathErr(ErrNotFile, *path)
	}

	if !existOk {
		return pathErr(ErrFileExist, *path)
	}

	return nil
}
