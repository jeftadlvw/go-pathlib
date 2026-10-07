// io.go holds the opening, reading, writing, and appending of files.

package pathlib

import (
	"errors"
	"io/fs"
	"os"
)

const (
	// defaultOpenPermission is the permission of a file OpenFileWithOptions
	// creates without OpenOptions.Permission.
	defaultOpenPermission = 0644

	// defaultOpenMode is the open mode of OpenFileWithOptions without
	// OpenOptions.Mode.
	defaultOpenMode = "rw"
)

// OpenOptions configures the opening of a file. The zero value opens an
// existing file for reading and writing, truncates it, and creates no file.
type OpenOptions struct {
	// CreateIfNotExists creates a missing file.
	CreateIfNotExists bool

	// Permission is the permission of a created file. It may only contain
	// PermissionBits. Zero selects 0644.
	Permission FileMode

	// Mode is the open mode, made of "r" (read), "w" (write), and "a"
	// (append) in this order. "a" needs "w", and "w" without "a" truncates
	// the file. The supported modes are "r", "w", "rw", "wa", and "rwa". The
	// empty string selects "rw".
	Mode string
}

// DefaultOpenOptions returns the options [OpenFile] uses. They open a file for
// reading and writing, create it with permission 0644, and truncate it.
func DefaultOpenOptions() OpenOptions {
	return OpenOptions{
		CreateIfNotExists: true,
		Permission:        defaultOpenPermission,
		Mode:              defaultOpenMode,
	}
}

// OpenFile opens the file at path for reading and writing, as
// [OpenFileWithOptions] does with [DefaultOpenOptions]. A missing file is
// created with permission 0644, and an existing file is truncated. The caller
// closes the returned file.
//
// The errors of OpenFileWithOptions apply.
func OpenFile(path *Path) (*os.File, error) {
	return OpenFileWithOptions(path, DefaultOpenOptions())
}

// OpenFileWithOptions opens the file at path with options. The caller closes
// the returned file.
//
// An OpenOptions.Permission with bits outside [PermissionBits] returns
// [ErrInvalidPermission], and an unsupported OpenOptions.Mode returns
// [ErrInvalidMode]. A directory returns [ErrNotFile]. A missing file returns
// [ErrNotExist], or [ErrParentNotExist] for a missing parent directory with
// OpenOptions.CreateIfNotExists. Denied access returns [ErrPermissionDenied].
// Any other failure to open the file returns [ErrOpen], to create it
// [ErrCreate], and to check it [ErrStat].
func OpenFileWithOptions(path *Path, options OpenOptions) (*os.File, error) {
	err := checkPermission(options.Permission, path)
	if err != nil {
		return nil, err
	}

	if options.Permission == 0 {
		options.Permission = defaultOpenPermission
	}

	if len(options.Mode) == 0 {
		options.Mode = defaultOpenMode
	}

	fileOpenMode, err := openModeFlags(options.Mode, path)
	if err != nil {
		return nil, err
	}

	if options.CreateIfNotExists {
		if fileOpenMode == os.O_RDONLY && runningOnWindows {
			// On Windows, O_CREATE|O_RDONLY either fails for existing read-only
			// files (O_CREATE requires write access) or returns a writable
			// handle for new files. Separate creation from opening to guarantee
			// a read-only handle.
			err = createIfMissing(path, options.Permission)
			if err != nil {
				return nil, err
			}
		} else {
			fileOpenMode |= os.O_CREATE
		}
	}

	return openNonDir(path, fileOpenMode, options.Permission)
}

// ReadFile returns the content of the file at path. It wraps [os.ReadFile].
//
// A missing file returns [ErrNotExist], and denied access returns
// [ErrPermissionDenied]. Any other failure returns [ErrRead].
func ReadFile(path *Path) ([]byte, error) {
	bytes, err := os.ReadFile(path.String())
	if err != nil {
		return nil, osErr(ErrRead, err, *path)
	}

	return bytes, nil
}

// ReadFileToString returns the content of the file at path as a string, as
// [ReadFile] does.
//
// The errors of ReadFile apply.
func ReadFileToString(path *Path) (string, error) {
	bytes, err := ReadFile(path)
	return string(bytes), err
}

// WriteBytes writes data to the file at path and returns the number of
// written bytes, as [os.WriteFile] does. A missing file is created with
// [DefaultFileMode], and an existing file is truncated. The parent directory
// must exist.
//
// It writes as [WriteBytesWithOptions] does with FileOptions{ExistOk: true},
// and the errors of WriteBytesWithOptions apply.
func WriteBytes(path *Path, data []byte) (int, error) {
	return WriteBytesWithOptions(path, data, FileOptions{ExistOk: true})
}

// WriteString writes data to the file at path, as [WriteBytes] does.
//
// The errors of WriteBytes apply.
func WriteString(path *Path, data string) (int, error) {
	return WriteBytes(path, []byte(data))
}

// WriteBytesWithOptions writes data to the file at path with options and
// returns the number of written bytes. The parent directory must exist.
//
// A missing file is created with FileOptions.Mode. The mode applies to a
// created file alone, and an existing file keeps its permission. An existing
// file is truncated if FileOptions.ExistOk is set. Otherwise it returns
// [ErrFileExist] and is left untouched, as in [CreateFileWithOptions].
//
// A symlink to a file is written through. An existing path that is no file
// returns [ErrNotFile]. This includes a broken symlink, whose target is never
// created.
//
// A FileOptions.Mode with bits outside [PermissionBits] returns
// [ErrInvalidPermission]. A missing parent directory returns
// [ErrParentNotExist], and denied access returns [ErrPermissionDenied]. Any
// other failure to check the path returns [ErrStat], to open the file
// [ErrOpen], and to write [ErrWrite].
func WriteBytesWithOptions(path *Path, data []byte, options FileOptions) (int, error) {
	return writeBytes(path, data, os.O_TRUNC, options)
}

// WriteStringWithOptions writes data to the file at path with options, as
// [WriteBytesWithOptions] does.
//
// The errors of WriteBytesWithOptions apply.
func WriteStringWithOptions(path *Path, data string, options FileOptions) (int, error) {
	return WriteBytesWithOptions(path, []byte(data), options)
}

// AppendBytes appends data to the file at path and returns the number of
// written bytes. A missing file is created with [DefaultFileMode]. The parent
// directory must exist.
//
// A symlink to a file is written through. An existing path that is no file
// returns [ErrNotFile]. This includes a broken symlink, whose target is never
// created.
//
// A missing parent directory returns [ErrParentNotExist], and denied access
// returns [ErrPermissionDenied]. Any other failure to check the path returns
// [ErrStat], to open the file [ErrOpen], and to write [ErrWrite].
func AppendBytes(path *Path, data []byte) (int, error) {
	return writeBytes(path, data, os.O_APPEND, FileOptions{ExistOk: true})
}

// AppendString appends data to the file at path, as [AppendBytes] does.
//
// The errors of AppendBytes apply.
func AppendString(path *Path, data string) (int, error) {
	return AppendBytes(path, []byte(data))
}

// writeBytes writes data to the file at path, opened for writing with the
// additional flag, which is os.O_TRUNC or os.O_APPEND. A missing file is
// created, and options apply as [WriteBytesWithOptions] describes.
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
		// Another process may have created the path since the check. Checking
		// again returns the same error as an existing path before the call.
		if errors.Is(err, fs.ErrExist) {
			checkErr := checkWritable(path, options.ExistOk)
			if checkErr != nil {
				return 0, checkErr
			}
		}
		return 0, osCreateErr(ErrOpen, err, *path)
	}

	n, err := file.Write(data)
	if err != nil {
		_ = file.Close()
		return n, osErr(ErrWrite, err, *path)
	}

	// Closing flushes the content, so its error means the write is incomplete.
	err = file.Close()
	if err != nil {
		return n, osErr(ErrWrite, err, *path)
	}

	return n, nil
}

// openModeFlags returns the flags for [os.OpenFile] that implement mode, an
// open mode of [OpenOptions]. An unsupported mode returns
// [ErrInvalidMode].
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

// createIfMissing creates an empty file with permission perm at path if
// nothing exists there. A failed creation returns the error of osCreateErr for
// [ErrCreate].
func createIfMissing(path *Path, perm FileMode) error {
	_, err := os.Stat(path.String())
	if err == nil {
		return nil
	}

	f, err := os.OpenFile(path.String(), os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return osCreateErr(ErrCreate, err, *path)
	}

	_ = f.Close()
	return nil
}

// openNonDir opens the file at path with flag and perm as [os.OpenFile] does.
// A file that cannot be opened returns the error of osErr for [ErrOpen], or of
// osCreateErr if flag creates the file. A failed check of the opened path
// returns [ErrStat], and a directory returns [ErrNotFile].
func openNonDir(path *Path, flag int, perm FileMode) (*os.File, error) {
	file, err := os.OpenFile(path.String(), flag, perm)
	if err != nil {
		if flag&os.O_CREATE != 0 {
			return nil, osCreateErr(ErrOpen, err, *path)
		}
		return nil, osErr(ErrOpen, err, *path)
	}

	// Unix opens a directory read-only, and the library refuses it.
	stat, err := os.Stat(path.String())
	if err != nil {
		_ = file.Close()
		return nil, osErr(ErrStat, err, *path)
	}

	if stat.IsDir() {
		_ = file.Close()
		return nil, pathErr(ErrNotFile, *path)
	}

	return file, nil
}

// checkWritable checks that path can be written with the ExistOk option of
// [FileOptions] set to existOk. An existing path that is no file returns
// [ErrNotFile], and an existing file returns [ErrFileExist] unless existOk is
// set. A path that cannot be checked returns the error of [lexists].
func checkWritable(path *Path, existOk bool) error {
	// The path itself is checked, so a broken symlink is not written through.
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
		return wrapErr(ErrFileExist, fs.ErrExist, *path)
	}

	return nil
}
