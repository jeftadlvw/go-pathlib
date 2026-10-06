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

	t.Run("Alias kind matches its standard sentinel in both directions", func(t *testing.T) {
		alias := aliasKind(fs.ErrNotExist, "alias")
		member := subKind(alias, "member")

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

	t.Run("Plain subkind does not match a sibling through the cause", func(t *testing.T) {
		// Only alias kinds are matched against the cause. A cause of kind ErrOpen
		// is in the ErrAccess group, but that does not make the error an ErrReadDir.
		err := wrapErr(ErrWalk, pathErr(ErrOpen, p), p)
		require.ErrorIs(t, err, ErrOpen)
		require.ErrorIs(t, err, ErrAccess)
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
