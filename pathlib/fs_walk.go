package pathlib

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
)

/*
SkipDir and SkipAll are control signals returned from a WalkFunc or WalkRFunc.

They are aliases of io/fs.SkipDir and io/fs.SkipAll, so they interoperate with
the standard library's walk sentinels.
*/
var (

	/*
	 	SkipDir skips the remaining entries of the current directory. If it's returned
	  	for a directory entry, it skips that directory's contents. If it's returned for a file, it
	   	skips the remaining entries of the containing directory.

	    Aliases io/fs.SkipDir.
	*/
	SkipDir = fs.SkipDir

	/*
	 	SkipAll stops the entire walk. Any other non-nil error aborts the walk and is returned
	  	to the caller.

	   Aliases io/fs.SkipAll.
	*/
	SkipAll = fs.SkipAll
)

/*
WalkFunc is called by Walk for every entry, receiving a Path joined with the
walked directory.

Return SkipDir or SkipAll to stop walking. Return any other non-nil error to abort
and have Walk return it. The error will be wrapped as ErrWalk.
*/
type WalkFunc func(p *Path) error

/*
WalkRFunc is called by WalkR for every entry.

Return SkipDir to skip the rest of the current directory or a directory's content.
Return SkipAll to stop the entire walk, or return any other non-nil error to abort
and have WalkR return it. The error will be wrapped as ErrWalk, except for an
unchanged localDirError, which is returned as is.

A directory is passed to walkFunc before its contents are read, with a nil
localDirError, so returning SkipDir skips the directory without reading it.

If a directory then cannot be opened or read, walkFunc is called a second time
for it, with a non-nil localDirError. This matches filepath.WalkDir. The walked
root itself is only passed to walkFunc in this case. Users may inspect the error
and act upon it by e.g. ignoring it (return nil), bubbling it (return error) or
returning a different error instead (e.g. SkipDir).

localDirError wraps ErrOpen (open failure) or ErrReadDir (read failure). These errors
can be matched either precisely, or by their shared ErrAccess group.
*/
type WalkRFunc func(p *Path, localDirError error) error

/*
Walk walks this directory and calls walkFunc for every entry (files, directories, etc.).
This path must be a directory. If this Path is a symlink to a directory, it is followed.

A missing path returns [ErrNotExist], and an existing non-directory returns
[ErrNotDir]. A path that cannot be checked returns [ErrStat], and a directory that
cannot be opened or read returns a kind below [ErrAccess]. An error of walkFunc is
returned as the cause of [ErrWalk].

Entries are visited in lexical order by name, making the traversal deterministic.

walkFunc receives a path joined with this Path.
*/
func (p *Path) Walk(walkFunc WalkFunc) error {
	return p.WalkContext(context.Background(), walkFunc)
}

/*
WalkContext is Walk with support for cancellation through ctx. The walk stops as
soon as ctx is done and returns ctx.Err() wrapped as ErrWalk, with cancellation
checked before each entry.
*/
func (p *Path) WalkContext(ctx context.Context, walkFunc WalkFunc) error {
	err := requireDir(p)
	if err != nil {
		return err
	}

	// Open and read are kept as separate steps so failures can be reported with
	// distinct sentinels (ErrOpen vs ErrReadDir). See sortDirEntries for why this
	// is preferred over os.ReadDir.
	file, err := os.Open(p.String())
	if err != nil {
		return wrapErr(ErrOpen, err, *p)
	}

	// Read all entries up front so they can be sorted into a deterministic order.
	entries, err := file.ReadDir(-1)
	file.Close()
	if err != nil {
		return wrapErr(ErrReadDir, err, *p)
	}

	sortDirEntries(entries)

	for _, entry := range entries {
		err := ctx.Err()
		if err != nil {
			return wrapErr(ErrWalk, err, *p)
		}

		entryPath := p.JoinStrings(entry.Name())

		walkErr := walkFunc(entryPath)
		if walkErr != nil {
			// Walk is not recursive, so SkipDir and SkipAll both simply stop it.
			if errors.Is(walkErr, SkipAll) || errors.Is(walkErr, SkipDir) {
				break
			}
			return wrapErr(ErrWalk, walkErr, *entryPath)
		}
	}

	return nil
}

/*
WalkR walks this directory recursively and calls walkFunc for every entry.
This path must be a directory. If this Path is a symlink to a directory, it is followed.

A missing path returns [ErrNotExist], and an existing non-directory returns
[ErrNotDir]. A path that cannot be checked returns [ErrStat]. An error of walkFunc
is returned as the cause of [ErrWalk], and an unchanged localDirError, a kind below
[ErrAccess], is returned as is.

Symlinks inside the tree are not followed. A symlink to a directory is passed to
walkFunc as a single entry, and its contents are not visited. This matches
filepath.WalkDir and avoids endless walks through symlink cycles.

Within each directory, entries are visited in lexical order by name, making the
traversal deterministic.

walkFunc receives paths that are already joined with this Path. See WalkRFunc for
when it is called and how its return value controls the walk.
*/
func (p *Path) WalkR(walkFunc WalkRFunc) error {
	return p.WalkRContext(context.Background(), walkFunc)
}

/*
WalkRContext is WalkR with support for cancellation through ctx. The walk stops
as soon as ctx is done and returns ctx.Err() wrapped as ErrWalk, with cancellation
checked before each directory and each entry.
*/
func (p *Path) WalkRContext(ctx context.Context, walkFunc WalkRFunc) error {
	return p.walkRContext(ctx, func(entryPath *Path, _ fs.DirEntry, localDirError error) error {
		return walkFunc(entryPath, localDirError)
	})
}

/*
walkEntryFunc is the internal callback of walkR. It is a WalkRFunc that also
receives the directory entry of p, as read from its parent directory. The entry
is nil for the walked root. Its IsDir does not follow symlinks, so it tells
whether walkR descends into p, without another stat.
*/
type walkEntryFunc func(p *Path, entry fs.DirEntry, localDirError error) error

/*
walkRContext is WalkRContext with a walkEntryFunc.
*/
func (p *Path) walkRContext(ctx context.Context, walkFunc walkEntryFunc) error {
	err := requireDir(p)
	if err != nil {
		return err
	}

	err = walkR(ctx, p, nil, walkFunc)

	// SkipAll is a successful early termination, not a failure.
	if errors.Is(err, SkipAll) {
		return nil
	}

	return err
}

// sortDirEntries orders directory entries lexically by name. This gives Walk and
// WalkR a deterministic, platform-independent traversal order that matches the
// standard library's filepath.WalkDir (and os.ReadDir), instead of the raw,
// filesystem-dependent order returned by os.File.ReadDir.
//
// Walk and WalkR deliberately open the directory and call file.ReadDir(-1)
// themselves, then sort with this helper, rather than calling os.ReadDir (which
// would open, read and sort in one step). The reason is error reporting, as the
// open step and the read step are wrapped with distinct sentinels (ErrOpen and
// ErrReadDir, both members of the ErrAccess group) that are handed to the walk
// callback as localDirError. os.ReadDir collapses both into a single, unwrappable
// error, so callers could no longer tell an open failure from a read failure.
// Keeping the two steps separate is the only reason this helper exists instead of
// a plain os.ReadDir call.
func sortDirEntries(entries []os.DirEntry) {
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
}

/*
walkR calls walkFunc for all entries below dir, recursively and in pre-order. A
subdirectory is passed to walkFunc before it is read, so SkipDir avoids reading it.
dirEntry is the entry of dir in its parent directory, or nil for the walked root.

SkipDir is consumed at the level that raises it, so it never escapes the current
function call frame. SkipAll is propagated up, so the whole walk unwinds.
*/
func walkR(ctx context.Context, dir *Path, dirEntry fs.DirEntry, walkFunc walkEntryFunc) error {
	err := ctx.Err()
	if err != nil {
		return wrapErr(ErrWalk, err, *dir)
	}

	// The root is checked by walkRContext, and subdirectories come from directory
	// entries. A subdirectory that vanished or changed since it was read fails to
	// open or read below and is handed to walkFunc like any other directory error.

	// Open and read are kept as separate steps so an open failure and a read
	// failure can be reported with distinct sentinels (ErrOpen vs ErrReadDir);
	// see sortDirEntries for why this is preferred over os.ReadDir.
	file, err := os.Open(dir.String())
	if err != nil {
		// Give walkFunc a chance to inspect and ignore the directory error. A nil
		// or SkipDir result skips this directory. Anything else aborts.
		return handleDirErr(dir, dirEntry, wrapErr(ErrOpen, err, *dir), walkFunc)
	}

	// Read all entries up front so they can be sorted into a deterministic order.
	// A read failure is surfaced through walkFunc the same way an open failure is.
	entries, readErr := file.ReadDir(-1)
	file.Close()
	if readErr != nil {
		return handleDirErr(dir, dirEntry, wrapErr(ErrReadDir, readErr, *dir), walkFunc)
	}

	sortDirEntries(entries)

	for _, entry := range entries {
		err := ctx.Err()
		if err != nil {
			return wrapErr(ErrWalk, err, *dir)
		}

		entryPath := dir.JoinStrings(entry.Name())

		walkErr := walkFunc(entryPath, entry, nil)
		if walkErr != nil {
			if !errors.Is(walkErr, SkipDir) {
				return callbackErr(walkErr, entryPath)
			}

			// SkipDir skips the contents of a directory, or for any other entry
			// the remaining entries of dir.
			if entry.IsDir() {
				continue
			}
			return nil
		}

		// Recurse into subdirectories. Symlinks are no directories here.
		if entry.IsDir() {
			subErr := walkR(ctx, entryPath, entry, walkFunc)
			if subErr != nil {
				return subErr
			}
		}
	}

	return nil
}

/*
handleDirErr hands a failure to open or read dir to walkFunc. A nil or SkipDir
result skips the directory. The unchanged dirErr is returned as is, and any
other error is wrapped as ErrWalk.
*/
func handleDirErr(dir *Path, dirEntry fs.DirEntry, dirErr error, walkFunc walkEntryFunc) error {
	handled := walkFunc(dir, dirEntry, dirErr)
	if handled == nil || errors.Is(handled, SkipDir) {
		return nil
	}
	if errors.Is(handled, dirErr) {
		return dirErr
	}
	return callbackErr(handled, dir)
}

// callbackErr wraps an error returned by a WalkRFunc for p as ErrWalk. SkipAll
// passes through unchanged, so WalkRContext can end the walk successfully.
func callbackErr(err error, p *Path) error {
	if errors.Is(err, SkipAll) {
		return err
	}
	return wrapErr(ErrWalk, err, *p)
}
