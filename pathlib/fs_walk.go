package pathlib

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
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
*/
type WalkRFunc func(p *Path, localDirError error) error

/*
Walk walks this directory and calls walkFunc for every entry (files, directories, etc.).
This path must be a directory. Symlinks are followed.

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

	file, err := os.Open(p.String())
	if err != nil {
		return wrapErr(ErrOpen, err, *p)
	}

	defer file.Close()

	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		// os.File.ReadDir returns at most n next entries for every next call.
		// So we call it with n = 1 to yield and process the directory entry by entry
		dirNames, err := file.ReadDir(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return wrapErr(ErrReadDir, err, *p)
			}
		}

		dirName := dirNames[0]
		dirNamePath := p.JoinStrings(dirName.Name())

		walkErr := walkFunc(dirNamePath)
		if walkErr != nil {
			// Walk is not recursive, so SkipDir and SkipAll both simply stop it.
			if errors.Is(walkErr, SkipAll) || errors.Is(walkErr, SkipDir) {
				break
			}
			return wrapErr(ErrWalk, walkErr, *dirNamePath)
		}
	}

	return nil
}

/*
WalkR walks this directory recursively and calls walkFunc for every entry.
This path must be a directory. Symlinks are followed.

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

	// Get file descriptor
	file, err := os.Open(currentDir.String())
	if err != nil {
		// Give walkFunc a chance to inspect and ignore the directory error. A nil
		// or SkipDir result skips this directory; anything else aborts.
		handled := walkFunc(currentDir, wrapErr(ErrOpen, err, *currentDir))
		if handled == nil || errors.Is(handled, SkipDir) {
			return nil
		}
		return handled
	}

	defer file.Close()

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

	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		// os.File.ReadDir returns at most n next entries for every next call.
		// So we call it with n = 1 to yield and process the directory entry by entry
		dirNames, err := file.ReadDir(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			handled := walkFunc(currentDir, wrapErr(ErrReadDir, err, *currentDir))
			if handled == nil || errors.Is(handled, SkipDir) {
				return nil
			}
			return handled
		}

		dirName := dirNames[0]
		dirNamePath := currentDir.JoinStrings(dirName.Name())

		// Call walkFunc for non-directory entries.
		if !dirName.IsDir() {
			walkErr := walkFunc(dirNamePath, nil)
			if walkErr != nil {
				if errors.Is(walkErr, SkipDir) {
					return nil
				}
				return walkErr
			}
		}

		// Recurse into subdirectories.
		if dirName.IsDir() {
			subErr := walkR(ctx, initialDir, dirNamePath, walkFunc)
			if subErr != nil {
				return subErr
			}
		}
	}

	return nil
}
