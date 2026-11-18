// fs_walk.go holds the walks over the entries of a directory, flat and
// recursive.

package pathlib

import (
	"context"
	"errors"
	"io/fs"
	"iter"
	"os"
	"slices"
	"strings"
)

// SkipDir and SkipAll are control signals a [WalkFunc] or a [WalkRFunc]
// returns. They are the signals of the standard library, so they work with its
// walk functions too.
//
//nolint:errname,gochecknoglobals // Variables alias the io/fs sentinels.
var (
	// SkipDir skips the contents of the directory it is returned for. For any
	// other entry, it skips the remaining entries of the directory that holds
	// the entry. It is [fs.SkipDir].
	SkipDir = fs.SkipDir

	// SkipAll ends the walk successfully. It is [fs.SkipAll].
	SkipAll = fs.SkipAll
)

// WalkFunc is called by [Path.Walk] for every entry, with the path of the
// entry joined to the walked directory.
//
// [SkipDir] and [SkipAll] both end the walk successfully. Any other error
// interrupts the walk, which returns it as the cause of [ErrWalkInterrupted].
type WalkFunc func(p *Path) error

// WalkRFunc is called by [Path.WalkR] for every entry, with the path of the
// entry joined to the walked directory.
//
// A directory is passed before its contents are read, with a nil
// localDirError, so [SkipDir] skips the directory without reading it. If the
// directory then cannot be opened or read, it is passed a second time with a
// non-nil localDirError, as [filepath.WalkDir] does. The walked root is only
// passed in this case.
//
// SkipDir skips the contents of a directory, and for any other entry the
// remaining entries of the directory that holds it. [SkipAll] ends the walk
// successfully. For a localDirError, nil and SkipDir skip the directory, and
// the unchanged localDirError makes WalkR return it as is. Any other error
// interrupts the walk, which returns it as the cause of [ErrWalkInterrupted].
//
// localDirError is [ErrPermissionDenied] for a directory whose access is
// denied, and [ErrNotExist] for a directory removed during the walk. Any other
// failure is [ErrOpen] or [ErrReadDir].
type WalkRFunc func(p *Path, localDirError error) error

// walkEntryFunc is a WalkRFunc that also receives the directory entry of p, as
// read from its parent directory. The entry is nil for the walked root. Its
// IsDir does not follow symlinks, so it tells whether the walk descends into
// p, without another stat.
type walkEntryFunc func(p *Path, entry fs.DirEntry, localDirError error) error

// Walk calls walkFunc for every entry of this directory, in lexical order by
// name. It does not descend into subdirectories. This Path must be a
// directory, and a symlink to a directory is followed.
//
// A missing path returns [ErrNotExist], an existing non-directory returns
// [ErrNotDir], and denied access returns [ErrPermissionDenied]. Any other
// failure to check the path returns [ErrStat], and any other failure to open
// or read the directory returns [ErrOpen] or [ErrReadDir]. An error of walkFunc
// is returned as the cause of [ErrWalkInterrupted].
//
// [Path.WalkSeq] returns the same entries as an iterator for a range loop.
func (p *Path) Walk(walkFunc WalkFunc) error {
	return p.WalkContext(context.Background(), walkFunc)
}

// WalkContext calls walkFunc for every entry of this directory, as [Path.Walk]
// does. It checks ctx before each entry, and a done ctx interrupts the walk,
// which returns ctx.Err() as the cause of [ErrWalkInterrupted].
func (p *Path) WalkContext(ctx context.Context, walkFunc WalkFunc) error {
	for entry, err := range p.WalkSeqContext(ctx) {
		if err != nil {
			return err
		}

		walkErr := walkFunc(entry)
		if walkErr == nil {
			continue
		}

		// Walk does not descend, so SkipDir and SkipAll both end it.
		if errors.Is(walkErr, SkipAll) || errors.Is(walkErr, SkipDir) {
			return nil
		}
		return wrapError(ErrWalkInterrupted, walkErr)
	}

	return nil
}

// WalkSeq returns an iterator over the entries of this directory, as
// [Path.WalkSeqContext] does without a context.
func (p *Path) WalkSeq() iter.Seq2[*Path, error] {
	return p.WalkSeqContext(context.Background())
}

// WalkSeqContext returns an iterator over the entries of this directory, in
// lexical order by name. It does not descend into subdirectories. This Path
// must be a directory, and a symlink to a directory is followed. [Path.Walk]
// and [Path.WalkContext] iterate it.
//
// The iterator yields every entry with a nil error. A failure yields a nil Path
// and the error, and ends the iteration. Breaking the loop ends it as well.
// The directory is checked and read when the iteration starts, and all of its
// entries are read at once, so an entry can be gone when it is yielded.
//
// The iterator checks ctx before each entry. A done ctx yields ctx.Err() as the
// cause of [ErrWalkInterrupted]. The errors of [Path.Walk] apply.
func (p *Path) WalkSeqContext(ctx context.Context) iter.Seq2[*Path, error] {
	return func(yield func(*Path, error) bool) {
		err := requireDir(p)
		if err != nil {
			yield(nil, err)
			return
		}

		entries, err := readDirSorted(p)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, entry := range entries {
			err := ctx.Err()
			if err != nil {
				yield(nil, wrapError(ErrWalkInterrupted, err))
				return
			}

			if !yield(p.JoinStrings(entry.Name()), nil) {
				return
			}
		}
	}
}

// WalkR calls walkFunc for every entry below this directory, recursively and
// in pre-order. Within a directory, entries are visited in lexical order by
// name. This Path must be a directory, and a symlink to a directory is
// followed. [WalkRFunc] describes how walkFunc controls the walk.
//
// Symlinks inside the tree are not followed. A symlink to a directory is a
// single entry, and its contents are not visited, as in [filepath.WalkDir].
// So symlink cycles cannot make the walk endless.
//
// A missing path returns [ErrNotExist], an existing non-directory returns
// [ErrNotDir], and denied access returns [ErrPermissionDenied]. Any other
// failure to check the path returns [ErrStat]. An error of walkFunc is returned
// as the cause of [ErrWalkInterrupted], and an unchanged localDirError is
// returned as is.
func (p *Path) WalkR(walkFunc WalkRFunc) error {
	return p.WalkRContext(context.Background(), walkFunc)
}

// WalkRContext calls walkFunc for every entry below this directory, as
// [Path.WalkR] does. It checks ctx before each directory and each entry, and a
// done ctx interrupts the walk, which returns ctx.Err() as the cause of
// [ErrWalkInterrupted].
func (p *Path) WalkRContext(ctx context.Context, walkFunc WalkRFunc) error {
	return p.walkRContext(ctx, func(entryPath *Path, _ fs.DirEntry, localDirError error) error {
		return callbackErr(walkFunc(entryPath, localDirError), localDirError)
	})
}

// walkRContext walks as [Path.WalkRContext] does, with a walkEntryFunc. It
// returns an error of walkFunc unchanged, so its caller decides on the error
// kind. Path.WalkRContext gives it ErrWalkInterrupted, and a glob returns its
// own errors as they are.
func (p *Path) walkRContext(ctx context.Context, walkFunc walkEntryFunc) error {
	err := requireDir(p)
	if err != nil {
		return err
	}

	err = walkR(ctx, p, nil, walkFunc)

	// SkipAll ends the walk successfully.
	if errors.Is(err, SkipAll) {
		return nil
	}

	return err
}

// sortDirEntries orders directory entries lexically by name, which is the order
// of os.ReadDir and filepath.WalkDir. The order of os.File.ReadDir depends on
// the filesystem.
func sortDirEntries(entries []os.DirEntry) {
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
}

// readDirSorted reads the entries of dir in lexical order, as os.ReadDir does.
//
// It opens and reads dir in two steps, so each failure keeps its own kind. A
// failed open returns the error of osErr for [ErrOpen], and a failed read the
// error of osErr for [ErrReadDir]. os.ReadDir returns one error for both steps.
func readDirSorted(dir *Path) ([]os.DirEntry, error) {
	file, err := os.Open(dir.String())
	if err != nil {
		return nil, osErr(ErrOpen, err, *dir)
	}

	// All entries are read up front, so they can be sorted.
	entries, err := file.ReadDir(-1)
	// The directory is only read, so closing it cannot lose data.
	_ = file.Close()
	if err != nil {
		return nil, osErr(ErrReadDir, err, *dir)
	}

	sortDirEntries(entries)
	return entries, nil
}

// walkR calls walkFunc for all entries below dir, recursively and in
// pre-order. A subdirectory is passed to walkFunc before it is read, so
// SkipDir avoids reading it. dirEntry is the entry of dir in its parent
// directory, or nil for the walked root.
//
// SkipDir is consumed at the level that returns it, so it never leaves the
// current call. SkipAll is passed up, so the whole walk unwinds.
func walkR(ctx context.Context, dir *Path, dirEntry fs.DirEntry, walkFunc walkEntryFunc) error {
	err := ctx.Err()
	if err != nil {
		return wrapError(ErrWalkInterrupted, err)
	}

	// The root is checked by walkRContext, and subdirectories come from
	// directory entries. A subdirectory that vanished or changed since it was
	// read fails to open or read below and is handed to walkFunc like any other
	// directory error.

	entries, err := readDirSorted(dir)
	if err != nil {
		return handleDirErr(dir, dirEntry, err, walkFunc)
	}

	for _, entry := range entries {
		err := ctx.Err()
		if err != nil {
			return wrapError(ErrWalkInterrupted, err)
		}

		entryPath := dir.JoinStrings(entry.Name())

		walkErr := walkFunc(entryPath, entry, nil)
		if walkErr != nil {
			if !errors.Is(walkErr, SkipDir) {
				return walkErr
			}

			// SkipDir skips the contents of a directory, or for any other entry
			// the remaining entries of dir.
			if entry.IsDir() {
				continue
			}
			return nil
		}

		// The entry does not follow symlinks, so a symlink is no directory
		// here.
		if entry.IsDir() {
			subErr := walkR(ctx, entryPath, entry, walkFunc)
			if subErr != nil {
				return subErr
			}
		}
	}

	return nil
}

// handleDirErr hands a failure to open or read dir to walkFunc. A nil or
// SkipDir result skips the directory. The unchanged dirErr is returned as is,
// and any other error is returned unchanged as well.
func handleDirErr(dir *Path, dirEntry fs.DirEntry, dirErr error, walkFunc walkEntryFunc) error {
	handled := walkFunc(dir, dirEntry, dirErr)
	if handled == nil || errors.Is(handled, SkipDir) {
		return nil
	}
	if errors.Is(handled, dirErr) {
		return dirErr
	}
	return handled
}

// callbackErr returns err, the result of a WalkRFunc that was passed
// localDirError, as the cause of ErrWalkInterrupted. Nil, SkipDir, SkipAll,
// and the unchanged localDirError pass as they are, so the walk handles them.
func callbackErr(err, localDirError error) error {
	switch {
	case err == nil, errors.Is(err, SkipDir), errors.Is(err, SkipAll):
		return err
	case localDirError != nil && errors.Is(err, localDirError):
		return err
	default:
		return wrapError(ErrWalkInterrupted, err)
	}
}
