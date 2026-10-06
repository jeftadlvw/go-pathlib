package pathlib

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// allKinds returns every exported kind of the package.
func allKinds() []*PathlibError {
	return []*PathlibError{
		ErrPathlib,
		ErrEmptyPattern, ErrBadPattern, ErrAnchorMismatch, ErrNotAbsolute, ErrRelImpossible, ErrLookup,
		ErrNotExist, ErrParentNotExist, ErrExist, ErrFileExist, ErrDirExist,
		ErrNotFile, ErrNotDir, ErrNotSymlink, ErrNotEmptyDir, ErrCopyType, ErrTypeMismatch,
		ErrAccess, ErrOpen, ErrReadDir, ErrStat, ErrReadSymlink, ErrResolve, ErrCreate, ErrRemove,
		ErrCopy, ErrSetPermission, ErrWalk, ErrInvalidFilter,
		ErrPermission, ErrPermissionRange, ErrUnsupportedMode, ErrRead, ErrWrite, ErrIsDir,
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
		require.Equal(t, "PATHLIB_NOT_EXIST", ErrNotExist.Code())
		require.Equal(t, "PATHLIB_EXIST_FILE", ErrFileExist.Code())
		require.Equal(t, "PATHLIB_ACCESS_READ_DIR", ErrReadDir.Code())
		require.Equal(t, "PATHLIB_PERMISSION_UNSUPPORTED_MODE", ErrUnsupportedMode.Code())
	})

	t.Run("Error shows the message and the code", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "file already exists", ErrFileExist.Message())
		require.Equal(t, "file already exists [PATHLIB_EXIST_FILE]", ErrFileExist.Error())
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
		require.ErrorIs(t, err, ErrAccess)
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

	t.Run("Alias kind matches its standard sentinel in both directions", func(t *testing.T) {
		t.Parallel()

		alias := aliasError(ErrPathlib, fs.ErrNotExist, "ALIAS", "alias")
		member := defineError(alias, "MEMBER", "member")

		// An error of the alias kind, or of a member, matches the standard sentinel.
		require.ErrorIs(t, pathErr(alias, p), fs.ErrNotExist)
		require.ErrorIs(t, pathErr(member, p), fs.ErrNotExist)
		require.ErrorIs(t, pathErr(member, p), alias)

		// An error whose cause matches the standard sentinel matches the alias kind.
		err := wrapErr(ErrOpen, fs.ErrNotExist, p)
		require.ErrorIs(t, err, alias)
		require.ErrorIs(t, wrapErr(ErrWalk, err, p), alias)

		// A matching cause does not make an error a specific member of the group.
		require.NotErrorIs(t, err, member)
		require.NotErrorIs(t, wrapErr(ErrOpen, fs.ErrPermission, p), alias)
		require.NotErrorIs(t, pathErr(ErrOpen, p), alias)
	})

	t.Run("Plain kind does not match a sibling through the cause", func(t *testing.T) {
		t.Parallel()

		// Only alias kinds are matched against the cause. A cause of kind ErrOpen
		// is in the ErrAccess group, but that does not make the error an ErrReadDir.
		err := wrapErr(ErrWalk, pathErr(ErrOpen, p), p)
		require.ErrorIs(t, err, ErrOpen)
		require.ErrorIs(t, err, ErrAccess)
		require.NotErrorIs(t, err, ErrReadDir)
	})

	t.Run("Lookup keeps the os error as its cause", func(t *testing.T) {
		t.Parallel()

		err := raiseError(ErrLookup, fs.ErrPermission)
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
		{"Kind without cause", raiseError(ErrNotDir, nil), "path is not a directory [PATHLIB_NOT_DIR]"},
		{"One path", pathErr(ErrNotDir, p), "path is not a directory [PATHLIB_NOT_DIR]: " + p.String()},
		{"Several paths", pathErr(ErrRelImpossible, p, o), "cannot make path relative to the other [PATHLIB_REL_IMPOSSIBLE]: [" + p.String() + ", " + o.String() + "]"},
		{"Path and cause", wrapErr(ErrOpen, fs.ErrPermission, p), "could not open path [PATHLIB_ACCESS_OPEN]: " + p.String() + ": permission denied"},
		{"Cause without path", raiseError(ErrLookup, fs.ErrPermission), "could not look up directory [PATHLIB_LOOKUP]: permission denied"},
		{"Permission value", permRangeErr(0o1000, p), "permission has bits outside PermissionBits [PATHLIB_PERMISSION_RANGE]: " + p.String() + " (perm 01000)"},
		{"Empty mode", permModeErr("", p), "unsupported open mode [PATHLIB_PERMISSION_UNSUPPORTED_MODE]: " + p.String() + ` (mode "")`},
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

		expect := `{"kind":"error walking path","code":"PATHLIB_WALK","cause":{` +
			`"message":` + jsonString(t, errors.Unwrap(err).Error()) + `,` +
			`"details":{"paths":["a/b"],"cause":{` +
			`"message":` + jsonString(t, inner.Error()) + `,` +
			`"details":{"kind":"could not open path","code":"PATHLIB_ACCESS_OPEN","cause":{` +
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
	require.Contains(t, out, "err.kind=\"error walking path\" err.code=PATHLIB_WALK")
	require.Contains(t, out, "err.cause.details.paths=[a/b]")
	require.Contains(t, out, "err.cause.details.cause.details.code=PATHLIB_ACCESS_OPEN")
	require.Contains(t, out, "err.cause.details.cause.details.cause.details.cause.message=\"permission denied\"")
}

func TestPermissionError(t *testing.T) {
	t.Parallel()

	p := *NewPath("a/b")

	t.Run("Kind and cause", func(t *testing.T) {
		t.Parallel()

		err := permModeErr("x", p)
		require.ErrorIs(t, err, ErrUnsupportedMode)
		require.ErrorIs(t, err, ErrPermission)
		require.ErrorIs(t, err, ErrPathlib)
		require.ErrorAs(t, err, new(*PathlibError))
		require.NotErrorAs(t, err, new(*PathError))
	})

	t.Run("Returns the rejected value", func(t *testing.T) {
		t.Parallel()

		var modeErr *PermissionError
		require.ErrorAs(t, permModeErr("x", p), &modeErr)
		require.Equal(t, "x", modeErr.Mode())
		require.Zero(t, modeErr.Perm())
		require.Equal(t, []Path{p}, modeErr.Paths())

		var permErr *PermissionError
		require.ErrorAs(t, permRangeErr(0o1000, p), &permErr)
		require.Equal(t, fs.FileMode(0o1000), permErr.Perm())
		require.Empty(t, permErr.Mode())
	})
}

func jsonString(t *testing.T, s string) string {
	t.Helper()

	encoded, err := json.Marshal(s)
	require.NoError(t, err)
	return string(encoded)
}
