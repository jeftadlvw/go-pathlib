package pathlib

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathlibError_Chain(t *testing.T) {
	p := *NewPath("a/b")

	t.Run("Unwrap returns the cause", func(t *testing.T) {
		err := wrapErr(ErrOpen, fs.ErrPermission, p)
		require.Equal(t, fs.ErrPermission, errors.Unwrap(err))
	})

	t.Run("Unwrap returns nil without a cause", func(t *testing.T) {
		require.Nil(t, errors.Unwrap(pathErr(ErrNotDir, p)))
	})

	t.Run("Is matches the kind, its group, and the cause", func(t *testing.T) {
		err := wrapErr(ErrOpen, fs.ErrPermission, p)
		require.ErrorIs(t, err, ErrOpen)
		require.ErrorIs(t, err, ErrAccess)
		require.ErrorIs(t, err, fs.ErrPermission)
		require.NotErrorIs(t, err, ErrReadDir)
	})

	t.Run("Kind and Paths return the fields", func(t *testing.T) {
		o := *NewPath("c")
		err := pathErr(ErrRelImpossible, p, o)
		require.Equal(t, ErrRelImpossible, err.Kind())
		require.Equal(t, []Path{p, o}, err.Paths())
	})

	t.Run("Paths returns a copy", func(t *testing.T) {
		err := pathErr(ErrNotDir, p)
		err.Paths()[0] = *NewPath("changed")
		require.Equal(t, []Path{p}, err.Paths())
	})

	t.Run("PermissionError keeps the kind and the catch-all", func(t *testing.T) {
		var err error = permModeErr("x", p)
		require.ErrorIs(t, err, ErrUnsupportedMode)
		require.ErrorIs(t, err, ErrPermission)
		require.ErrorAs(t, err, new(*PathlibError))
		require.ErrorAs(t, err, new(*PermissionError))
	})

	t.Run("PermissionError returns the rejected value", func(t *testing.T) {
		modeErr := permModeErr("x", p)
		require.Equal(t, "x", modeErr.Mode())
		require.Zero(t, modeErr.Perm())
		require.Equal(t, ErrUnsupportedMode, modeErr.Kind())
		require.Equal(t, []Path{p}, modeErr.Paths())

		permErr := permRangeErr(0o1000, p)
		require.Equal(t, fs.FileMode(0o1000), permErr.Perm())
		require.Empty(t, permErr.Mode())
	})
}
