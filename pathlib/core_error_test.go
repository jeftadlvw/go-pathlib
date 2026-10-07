package pathlib

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// allKinds returns every exported kind of the package.
func allKinds() []*PathlibError {
	return []*PathlibError{
		ErrPathlib,
		ErrInvalid, ErrInvalidPattern, ErrEmptyPattern, ErrBadPattern, ErrNotAbsolute,
		ErrInvalidFilter, ErrInvalidPermission, ErrInvalidMode,
		ErrRelImpossible, ErrAnchorMismatch,
		ErrNotExist, ErrParentNotExist,
		ErrExist, ErrFileExist, ErrDirExist, ErrNotEmptyDir,
		ErrPermissionDenied,
		ErrWrongType, ErrNotFile, ErrNotDir, ErrNotSymlink, ErrTypeMismatch, ErrUnsupportedType,
		ErrOperation, ErrLookup, ErrStat, ErrOpen, ErrRead, ErrReadDir, ErrReadSymlink, ErrResolve,
		ErrCreate, ErrWrite, ErrCopy, ErrRemove, ErrSetPermission,
		ErrWalk,
	}
}

func TestPathlibError(t *testing.T) {
	t.Parallel()

	t.Run("Codes are unique", func(t *testing.T) {
		t.Parallel()

		seen := map[string]bool{}
		for _, kind := range allKinds() {
			require.False(t, seen[kind.Code()], "duplicate code %q", kind.Code())
			seen[kind.Code()] = true
		}
	})

	t.Run("Every kind is below the root", func(t *testing.T) {
		t.Parallel()

		for _, kind := range allKinds() {
			require.ErrorIs(t, kind, ErrPathlib, kind.Code())
		}
	})

	t.Run("Code joins the names from the root", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "PATHLIB", ErrPathlib.Code())
		require.Equal(t, "PATHLIB.NOT_EXIST", ErrNotExist.Code())
		require.Equal(t, "PATHLIB.EXIST.FILE", ErrFileExist.Code())
		require.Equal(t, "PATHLIB.OPERATION.READ_DIR", ErrReadDir.Code())
		require.Equal(t, "PATHLIB.INVALID.PATTERN.MALFORMED", ErrBadPattern.Code())
		require.Equal(t, "PATHLIB.INVALID.MODE", ErrInvalidMode.Code())
	})

	t.Run("Error shows the message and the code", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "file already exists", ErrFileExist.Message())
		require.Equal(t, "file already exists (PATHLIB.EXIST.FILE)", ErrFileExist.Error())
	})

	t.Run("Malformed code panics", func(t *testing.T) {
		t.Parallel()

		for _, code := range []string{"", "lower", "Mixed", "A.B", "A__B", "_A", "A_", "A1"} {
			require.Panics(t, func() { _ = defineError(ErrPathlib, code, "message") }, code)
		}
	})
}

func TestRaisedError(t *testing.T) {
	t.Parallel()

	p := *NewPath("a/b")

	t.Run("Unwrap returns the cause", func(t *testing.T) {
		t.Parallel()

		err := wrapErr(ErrOpen, fs.ErrPermission, p)

		var pathErr *PathError
		require.ErrorAs(t, errors.Unwrap(err), &pathErr)
		require.Equal(t, fs.ErrPermission, errors.Unwrap(pathErr))
	})

	t.Run("Is matches the kind, its group, the root, and the cause", func(t *testing.T) {
		t.Parallel()

		err := wrapErr(ErrOpen, fs.ErrPermission, p)
		require.ErrorIs(t, err, ErrOpen)
		require.ErrorIs(t, err, ErrOperation)
		require.ErrorIs(t, err, ErrPathlib)
		require.ErrorIs(t, err, fs.ErrPermission)
		require.NotErrorIs(t, err, ErrReadDir)
	})

	t.Run("As returns the kind and the cause", func(t *testing.T) {
		t.Parallel()

		o := *NewPath("c")
		err := pathErr(ErrRelImpossible, p, o)

		var kind *PathlibError
		require.ErrorAs(t, err, &kind)
		require.Equal(t, ErrRelImpossible, kind)

		var cause *PathError
		require.ErrorAs(t, err, &cause)
		require.Equal(t, []Path{p, o}, cause.Paths())
	})

	t.Run("Paths returns a copy", func(t *testing.T) {
		t.Parallel()

		var cause *PathError
		require.ErrorAs(t, pathErr(ErrNotDir, p), &cause)
		cause.Paths()[0] = *NewPath("changed")
		require.Equal(t, []Path{p}, cause.Paths())
	})

	t.Run("A kind does not match a sibling through the cause", func(t *testing.T) {
		t.Parallel()

		// A cause of kind ErrOpen is in the ErrOperation group, but that does
		// not make the error an ErrReadDir.
		err := wrapErr(ErrWalk, pathErr(ErrOpen, p), p)
		require.ErrorIs(t, err, ErrOpen)
		require.ErrorIs(t, err, ErrOperation)
		require.NotErrorIs(t, err, ErrReadDir)
	})

	t.Run("Lookup keeps the os error as its cause", func(t *testing.T) {
		t.Parallel()

		err := wrapError(ErrLookup, fs.ErrPermission)
		require.ErrorIs(t, err, ErrLookup)
		require.Equal(t, fs.ErrPermission, errors.Unwrap(err))
	})
}

func TestRaisedError_Error(t *testing.T) {
	t.Parallel()

	p := *NewPathFromPosix("a/b")
	o := *NewPathFromPosix("c")

	cases := []struct {
		Name   string
		Err    error
		Expect string
	}{
		{"Kind without cause", wrapError(ErrNotDir, nil), "path is not a directory (PATHLIB.WRONG_TYPE.NOT_DIR)"},
		{"One path", pathErr(ErrNotDir, p), "path is not a directory (PATHLIB.WRONG_TYPE.NOT_DIR): " + p.String()},
		{"Several paths", pathErr(ErrRelImpossible, p, o), "cannot make path relative to the other (PATHLIB.REL_IMPOSSIBLE): [" + p.String() + ", " + o.String() + "]"},
		{"Path and cause", wrapErr(ErrOpen, fs.ErrPermission, p), "could not open path (PATHLIB.OPERATION.OPEN): " + p.String() + ": permission denied"},
		{"Cause without path", wrapError(ErrLookup, fs.ErrPermission), "could not look up directory (PATHLIB.OPERATION.LOOKUP): permission denied"},
		{"Permission value", permRangeErr(0o1000, p), "permission has bits outside PermissionBits (PATHLIB.INVALID.PERMISSION): " + p.String() + " (perm 01000)"},
		{"Empty pattern", patternErr(ErrEmptyPattern, "", nil, p), "pattern may not be empty (PATHLIB.INVALID.PATTERN.EMPTY): " + p.String() + ` (pattern "")`},
		{"Bad pattern", patternErr(ErrBadPattern, "a/[", path.ErrBadPattern, p), "malformed pattern (PATHLIB.INVALID.PATTERN.MALFORMED): " + p.String() + ` (pattern "a/["): syntax error in pattern`},
		{"Empty mode", permModeErr("", p), "invalid open mode (PATHLIB.INVALID.MODE): " + p.String() + ` (mode "")`},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, c.Expect, c.Err.Error())
		})
	}
}

func TestRaisedError_MarshalJSON(t *testing.T) {
	t.Parallel()

	p := *NewPathFromPosix("a/b")

	t.Run("Nested failure", func(t *testing.T) {
		t.Parallel()

		inner := wrapErr(ErrOpen, fs.ErrPermission, p)
		err := wrapErr(ErrWalk, inner, p)

		encoded, jsonErr := json.Marshal(err)
		require.NoError(t, jsonErr)

		expect := `{"kind":"error walking path","code":"PATHLIB.WALK","cause":{` +
			`"message":` + jsonString(t, errors.Unwrap(err).Error()) + `,` +
			`"details":{"paths":["a/b"],"cause":{` +
			`"message":` + jsonString(t, inner.Error()) + `,` +
			`"details":{"kind":"could not open path","code":"PATHLIB.OPERATION.OPEN","cause":{` +
			`"message":` + jsonString(t, errors.Unwrap(inner).Error()) + `,` +
			`"details":{"paths":["a/b"],"cause":{"message":"permission denied"}}}}}}}}`
		require.JSONEq(t, expect, string(encoded))
	})

	t.Run("Permission error", func(t *testing.T) {
		t.Parallel()

		encoded, jsonErr := json.Marshal(permModeErr("x", p))
		require.NoError(t, jsonErr)
		require.Contains(t, string(encoded), `"details":{"paths":["a/b"],"mode":"x"}`)

		encoded, jsonErr = json.Marshal(permRangeErr(0o1000, p))
		require.NoError(t, jsonErr)
		require.Contains(t, string(encoded), `"details":{"paths":["a/b"],"perm":512}`)
	})
}

func TestRaisedError_LogValue(t *testing.T) {
	t.Parallel()

	p := *NewPathFromPosix("a/b")
	err := wrapErr(ErrWalk, wrapErr(ErrOpen, fs.ErrPermission, p), p)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Info("", "err", err)

	out := buf.String()
	require.Contains(t, out, "err.kind=\"error walking path\" err.code=PATHLIB.WALK")
	require.Contains(t, out, "err.cause.details.paths=[a/b]")
	require.Contains(t, out, "err.cause.details.cause.details.code=PATHLIB.OPERATION.OPEN")
	require.Contains(t, out, "err.cause.details.cause.details.cause.details.cause.message=\"permission denied\"")
}

func jsonString(t *testing.T, s string) string {
	t.Helper()

	encoded, err := json.Marshal(s)
	require.NoError(t, err)
	return string(encoded)
}
