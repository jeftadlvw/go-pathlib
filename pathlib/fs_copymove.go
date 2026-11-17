// fs_copymove.go holds the copying, moving, and renaming of files and
// directory trees.

package pathlib

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

// Copy copies the entry at source to destination. A directory is copied with
// its whole tree, and every other entry as it is. A symlink is copied as a
// symlink, and this includes a broken symlink. A relative symlink target is
// rebased, so the copy points to the same path.
//
// The parent directory of destination must exist. A copied directory needs a
// missing or empty destination directory. A copied file is closed without
// (*os.File).Sync, so a crash can lose its content.
//
// A missing source returns [ErrNotExist], and a missing parent directory of
// destination returns [ErrParentNotExist]. An existing destination returns a
// kind below [ErrExist], or [ErrNotEmptyDir] for a directory with entries. A
// destination of an incompatible type returns [ErrTypeMismatch], and a source
// of a type that cannot be copied returns [ErrUnsupportedType]. Denied access
// returns [ErrPermissionDenied]. Any other failure of the operating system
// returns [ErrStat], [ErrOpen], [ErrReadDir], [ErrCreate], [ErrCopy], or
// [ErrReadSymlink].
//
// Rebasing a symlink target resolves relative paths against the working
// directory and returns [ErrLookup] if it cannot be determined. On Windows, a
// source and destination on different volumes return [ErrAnchorMismatch].
func Copy(source, destination *Path) error {
	sourceExists, err := lexists(source)
	if err != nil {
		return err
	}
	if !sourceExists {
		return wrapErr(ErrNotExist, fs.ErrNotExist, *source)
	}

	if !destination.Parent().Exists() {
		return wrapErr(ErrParentNotExist, fs.ErrNotExist, *destination)
	}

	switch {
	// IsFile and IsDir follow symlinks, so symlinks are checked first.
	case source.IsSymlink():
		return copySymlink(source, destination)
	case source.IsFile():
		return copyFile(source, destination)
	case source.IsDir():
		return copyDir(source, destination)
	default:
		return pathErr(ErrUnsupportedType, *source)
	}
}

// Move moves the entry at source to destination. A symlink is moved as a
// symlink, this includes a broken symlink.
//
// Move never replaces an entry at destination, also not one created after its
// check. A directory is renamed with [os.Rename], which replaces at most an
// empty directory. A file gets its new name as a hard link with [os.Link], and
// a symlink is created anew with the same target. Both fail for an existing
// destination, and source is removed afterwards. Where this fails for another
// reason, such as across filesystems, the entry is copied and removed. A
// symlink that cannot be created, such as on Windows without the privilege, is
// renamed instead.
//
// The parent directory of destination must exist. destination must not
// exist, unless source is a directory and destination an empty directory. A
// broken symlink at destination exists too.
//
// A missing source returns [ErrNotExist], and an existing destination returns
// [ErrExist]. Denied access returns [ErrPermissionDenied], and any other
// failure to check a path returns [ErrStat]. A failed read of a symlink
// returns [ErrReadSymlink]. The errors of [Copy], [Remove], and [RemoveAll]
// apply.
func Move(source, destination *Path) error {
	sourceInfo, err := os.Lstat(source.String())
	if err != nil {
		return osErr(ErrStat, err, *source)
	}

	if !sourceInfo.IsDir() || !destination.IsEmptyDir() {
		destinationExists, err := lexists(destination)
		if err != nil {
			return err
		}
		if destinationExists {
			return wrapErr(ErrExist, fs.ErrExist, *destination)
		}
	}

	moved, err := moveInPlace(source, destination, sourceInfo)
	if moved || err != nil {
		return err
	}

	err = Copy(source, destination)
	if err != nil {
		return err
	}

	return RemoveAll(source)
}

// Rename moves the entry at source to name in the same directory, as [Move]
// does. The errors of Move apply.
func Rename(source *Path, name string) error {
	return Move(source, source.Parent().JoinStrings(name))
}

// CopyFromFS copies the file system fsys into the directory destination. It
// wraps [os.CopyFS]. fsys can be any [fs.FS], such as an [embed.FS], the
// [zip.Reader] of an archive, or the file system of [Path.FS].
//
// The parent directory of destination must exist. destination must be missing
// or an empty directory, as for a directory copied by [Copy], and a symlink to
// a directory is followed. Each name of fsys becomes a path below destination
// with [filepath.Localize], so a name that cannot be represented on the
// platform fails the copy.
//
// Files are created with mode 0o666 and the execute bits of their source, and
// directories with mode 0o777, before the umask. Modification times are not
// kept. A symlink is copied as a symlink with its target unchanged, also a
// target that leads outside destination, if fsys implements [fs.ReadLinkFS].
// Otherwise a symlink fails the copy. The copy stops at the first error and
// leaves the entries copied until then. A copied file is closed without
// (*os.File).Sync, so a crash can lose its content.
//
// A missing parent directory of destination returns [ErrParentNotExist]. An
// existing non-directory returns [ErrNotDir], a directory with entries returns
// [ErrNotEmptyDir], and any other existing entry returns [ErrExist]. Denied
// access returns [ErrPermissionDenied]. Any other failure to check destination
// returns [ErrStat], [ErrOpen], or [ErrReadDir], and any other failure of the
// copy returns [ErrNotExist], [ErrExist], or [ErrCopy]. The underlying error
// of a failed copy names the entry it concerns.
func CopyFromFS(fsys fs.FS, destination *Path) error {
	if !destination.Parent().Exists() {
		return wrapErr(ErrParentNotExist, fs.ErrNotExist, *destination)
	}

	err := requireCopyFromFSDestination(destination)
	if err != nil {
		return err
	}

	err = os.CopyFS(destination.String(), fsys)
	if err != nil {
		return osErr(ErrCopy, err, *destination)
	}

	return nil
}

// moveInPlace gives the entry at source the name destination without copying
// it. A directory is renamed, and every other entry is moved by moveEntry. It
// reports false with a nil error where the entry has to be copied, such as
// across filesystems.
func moveInPlace(source, destination *Path, sourceInfo fs.FileInfo) (bool, error) {
	if sourceInfo.IsDir() {
		err := os.Rename(source.String(), destination.String())
		return err == nil, nil
	}

	return moveEntry(source, destination, sourceInfo)
}

// moveEntry gives the entry at source, which is no directory, the name
// destination without replacing an entry there, and removes source. A symlink
// is created anew with its target, and every other entry is hard linked, since
// a hard link of a symlink follows it on some systems. An existing destination
// returns [ErrExist]. moveEntry reports false with a nil error where the
// filesystem cannot give the name, such as across filesystems, so the caller
// copies the entry. A symlink that cannot be created is renamed instead.
func moveEntry(source, destination *Path, sourceInfo fs.FileInfo) (bool, error) {
	var err error
	isSymlink := sourceInfo.Mode()&fs.ModeSymlink != 0
	if isSymlink {
		target, readErr := os.Readlink(source.String())
		if readErr != nil {
			return false, osErr(ErrReadSymlink, readErr, *source)
		}
		err = os.Symlink(target, destination.String())
	} else {
		err = os.Link(source.String(), destination.String())
	}

	switch {
	case err == nil:
		err = removeMovedSource(source, destination, sourceInfo)
		return err == nil, err
	case errors.Is(err, fs.ErrExist):
		return false, osErr(ErrCreate, err, *destination)
	case isSymlink:
		// Windows creates symlinks with a privilege alone. A rename needs none,
		// but it replaces an entry created at destination since the check.
		err = os.Rename(source.String(), destination.String())
		return err == nil, nil
	default:
		return false, nil
	}
}

// removeMovedSource removes source after moveEntry gave its entry the name
// destination. If source cannot be removed, the new name is removed again, so
// the move changes nothing.
func removeMovedSource(source, destination *Path, sourceInfo fs.FileInfo) error {
	err := Remove(source)
	if err != nil {
		_ = os.Remove(destination.String())
		return err
	}

	// Windows clears the read-only attribute of a file to remove it, and a
	// hard link shares the attribute, so it is set again.
	readOnly := sourceInfo.Mode().Perm() == windowsReadOnlyFileMode
	if runningOnWindows && sourceInfo.Mode().IsRegular() && readOnly {
		return SetMode(destination, windowsReadOnlyFileMode)
	}

	return nil
}

// copyFile copies the file at source to destination, whose parent directory
// exists.
func copyFile(source, destination *Path) error {
	// The copy takes the permission of the source.
	sourceInfo, err := source.Stat()
	if err != nil {
		return err
	}

	err = requireCopyFileDestinationFree(source, destination)
	if err != nil {
		return err
	}

	sourceFile, err := os.Open(source.String())
	if err != nil {
		return osErr(ErrOpen, err, *source)
	}
	// The source is only read, so closing it cannot lose data.
	defer func() { _ = sourceFile.Close() }()

	// O_EXCL refuses a path created since the check, symlinks included, so
	// nothing is overwritten or written through.
	destinationFile, err := os.OpenFile(destination.String(), os.O_RDWR|os.O_CREATE|os.O_EXCL, sourceInfo.Mode())
	if err != nil {
		// Checking again returns the same error as an existing destination
		// before the call.
		if errors.Is(err, fs.ErrExist) {
			checkErr := requireCopyFileDestinationFree(source, destination)
			if checkErr != nil {
				return checkErr
			}
		}
		return osCreateErr(ErrCreate, err, *destination)
	}

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		_ = destinationFile.Close()
		return osErr(ErrCopy, err, *source, *destination)
	}

	// Closing reports a write error the operating system deferred, such as on a
	// network filesystem, so its error means the copy may be incomplete. It
	// does not write the content to the disk. Only Sync does.
	err = destinationFile.Close()
	if err != nil {
		return osErr(ErrCopy, err, *source, *destination)
	}

	return nil
}

// copySymlink copies the symlink at source to destination, whose parent
// directory exists. A relative target is rebased to destination.
func copySymlink(source, destination *Path) error {
	target, err := source.ReadSymlinkTarget()
	if err != nil {
		return err
	}

	if target.IsRelative() {
		target, err = rebaseSymlinkTarget(target, source, destination)
		if err != nil {
			return err
		}
	}

	return CreateSymlink(target, destination)
}

// rebaseSymlinkTarget returns the relative target of the source symlink as seen
// from the destination symlink, so both point to the same path.
//
// A relative source or destination is relative to the working directory, as the
// operating system resolves it. Both parents are made absolute first, so they
// share a base.
func rebaseSymlinkTarget(target, source, destination *Path) (*Path, error) {
	sourceParent, err := source.Parent().MakeAbsolute()
	if err != nil {
		return nil, err
	}

	destinationParent, err := destination.Parent().MakeAbsolute()
	if err != nil {
		return nil, err
	}

	absoluteTarget, err := target.AbsoluteFrom(sourceParent)
	if err != nil {
		return nil, err
	}

	return absoluteTarget.RelativeTo(destinationParent)
}

// copyDir copies the directory at source with its whole tree to destination,
// whose parent directory exists.
func copyDir(source, destination *Path) error {
	err := prepareCopyDirDestination(source, destination)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(source.String())
	if err != nil {
		return osErr(ErrReadDir, err, *source)
	}

	for _, entry := range entries {
		err := Copy(source.JoinStrings(entry.Name()), destination.JoinStrings(entry.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

// prepareCopyDirDestination makes destination ready to receive the entries of
// the directory at source. A missing destination is created with the
// permission of source. An existing destination must be an empty directory.
func prepareCopyDirDestination(source, destination *Path) error {
	sourceInfo, err := source.Stat()
	if err != nil {
		return err
	}

	destinationExists, err := lexists(destination)
	if err != nil {
		return err
	}

	if !destinationExists {
		err = os.Mkdir(destination.String(), sourceInfo.Mode())
		if err != nil {
			return osCreateErr(ErrCreate, err, *destination)
		}
		return nil
	}

	switch {
	case destination.IsFile():
		return pathErr(ErrTypeMismatch, *source, *destination)
	case destination.IsDir():
		return requireEmptyDir(destination)
	default:
		return wrapErr(ErrExist, fs.ErrExist, *destination)
	}
}

// requireEmptyDir returns [ErrNotEmptyDir] if the directory dir has entries.
// The underlying error is fs.ErrExist, as for every kind below [ErrExist].
func requireEmptyDir(dir *Path) error {
	file, err := os.Open(dir.String())
	if err != nil {
		return osErr(ErrOpen, err, *dir)
	}

	// The directory is only read, so closing it cannot lose data.
	defer func() { _ = file.Close() }()

	entries, err := file.ReadDir(1)
	if err != nil && !errors.Is(err, io.EOF) {
		return osErr(ErrReadDir, err, *dir)
	}

	if len(entries) != 0 {
		return wrapErr(ErrNotEmptyDir, fs.ErrExist, *dir)
	}

	return nil
}

// requireCopyFileDestinationFree returns an error if anything exists at the
// destination of a file copied from source. An existing file returns
// [ErrFileExist], a directory returns [ErrTypeMismatch], and any other entry
// returns [ErrExist]. A path that cannot be checked returns the error of
// [lexists].
func requireCopyFileDestinationFree(source, destination *Path) error {
	// A broken symlink exists too, and writing through it would create its
	// target.
	exists, err := lexists(destination)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	switch {
	case destination.IsFile():
		return wrapErr(ErrFileExist, fs.ErrExist, *destination)
	case destination.IsDir():
		return pathErr(ErrTypeMismatch, *source, *destination)
	default:
		return wrapErr(ErrExist, fs.ErrExist, *destination)
	}
}

// requireCopyFromFSDestination returns an error unless destination is missing
// or an empty directory, which [CopyFromFS] can copy into. A symlink to a
// directory is followed, as [os.CopyFS] follows it. An existing non-directory
// returns [ErrNotDir], a directory with entries returns [ErrNotEmptyDir], and
// any other entry, such as a broken symlink, returns [ErrExist].
func requireCopyFromFSDestination(destination *Path) error {
	exists, err := lexists(destination)
	if err != nil {
		return err
	}

	switch {
	case !exists:
		return nil
	case destination.IsDir():
		return requireEmptyDir(destination)
	case destination.IsFile():
		return pathErr(ErrNotDir, *destination)
	default:
		return wrapErr(ErrExist, fs.ErrExist, *destination)
	}
}
