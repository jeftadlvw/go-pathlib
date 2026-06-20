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
and have WalkR return it.

If the current directory could not be opened or read, localDirError is non-nil.
Users may inspect it and act upon it by e.g. ignoring it (return nil), bubbling
it (return error) or returning a different error instead (e.g. SkipDir).

localDirError wraps ErrOpen (open failure) or ErrReadDir (read failure). These errors
can be matched either precisely, or by their shared ErrDirAccess group.
*/
type WalkRFunc func(p *Path, localDirError error) error

/*
Walk walks this directory and calls walkFunc for every entry (files, directories, etc.).
This path must be a directory. Symlinks are followed.

Entries are visited in lexical order by name, making the traversal deterministic.

walkFunc receives a path joined with this Path.
*/
func (p *Path) Walk(walkFunc WalkFunc) error {
	return p.WalkContext(context.Background(), walkFunc)
}

/*
WalkContext is Walk with support for cancellation through ctx. The walk stops and
returns ctx.Err() as soon as ctx is done, with cancellation checked before each entry.
*/
func (p *Path) WalkContext(ctx context.Context, walkFunc WalkFunc) error {
	if !p.IsDir() {
		return pathErr(ErrNotDir, *p)
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
			return err
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
This path must be a directory. Symlinks are followed.

Within each directory, entries are visited in lexical order by name, making the
traversal deterministic.

walkFunc receives paths that are already joined with this Path.

walkFunc takes the current entry, a function to abort walking the entire tree and
a function to abort walking the current branch.
*/
func (p *Path) WalkR(walkFunc WalkRFunc) error {
	return p.WalkRContext(context.Background(), walkFunc)
}

/*
WalkRContext is WalkR with support for cancellation through ctx. The walk stops
and returns ctx.Err() as soon as ctx is done, with cancellation checked before
each directory and each entry.
*/
func (p *Path) WalkRContext(ctx context.Context, walkFunc WalkRFunc) error {
	err := walkR(ctx, p, nil, walkFunc)

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
// ErrReadDir, both members of the ErrDirAccess group) that are handed to the walk
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
walkR calls walkFunc recursively for all path entries in a given root Path.

currentDir must be nil on the initial function call. SkipDir is consumed at the
level that raises it, so it never escapes the current function call frame. SkipAll
is propagated up, so the whole walk unwinds.
*/
func walkR(ctx context.Context, initialDir *Path, currentDir *Path, walkFunc WalkRFunc) error {
	// Set currentDir to initialDir if currentDir is nil (initial call)
	if currentDir == nil {
		currentDir = initialDir
	}

	err := ctx.Err()
	if err != nil {
		return err
	}

	if !currentDir.IsDir() {
		return pathErr(ErrNotDir, *currentDir)
	}

	// Open and read are kept as separate steps so an open failure and a read
	// failure can be reported with distinct sentinels (ErrOpen vs ErrReadDir);
	// see sortDirEntries for why this is preferred over os.ReadDir.
	file, err := os.Open(currentDir.String())
	if err != nil {
		// Give walkFunc a chance to inspect and ignore the directory error. A nil
		// or SkipDir result skips this directory. Anything else aborts.
		handled := walkFunc(currentDir, wrapErr(ErrOpen, err, *currentDir))
		if handled == nil || errors.Is(handled, SkipDir) {
			return nil
		}
		return handled
	}

	// Read all entries up front so they can be sorted into a deterministic order.
	// A read failure is surfaced through walkFunc the same way an open failure is.
	entries, readErr := file.ReadDir(-1)
	file.Close()
	if readErr != nil {
		handled := walkFunc(currentDir, wrapErr(ErrReadDir, readErr, *currentDir))
		if handled == nil || errors.Is(handled, SkipDir) {
			return nil
		}
		return handled
	}

	sortDirEntries(entries)

	// Call walkFunc for the directory itself (except the initial root).
	if !currentDir.Equals(initialDir, CaseInsensitive) {
		walkErr := walkFunc(currentDir, nil)
		if walkErr != nil {
			if errors.Is(walkErr, SkipDir) {
				return nil
			}
			return walkErr
		}
	}

	for _, entry := range entries {
		err := ctx.Err()
		if err != nil {
			return err
		}

		entryPath := currentDir.JoinStrings(entry.Name())

		// Call walkFunc for non-directory entries.
		if !entry.IsDir() {
			walkErr := walkFunc(entryPath, nil)
			if walkErr != nil {
				if errors.Is(walkErr, SkipDir) {
					return nil
				}
				return walkErr
			}
			continue
		}

		// Recurse into subdirectories.
		subErr := walkR(ctx, initialDir, entryPath, walkFunc)
		if subErr != nil {
			return subErr
		}
	}

	return nil
}
