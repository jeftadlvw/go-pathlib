package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPermissionError(t *testing.T) {
	t.Parallel()

	p := *NewPath("a/b")

	t.Run("Kind and cause", func(t *testing.T) {
		t.Parallel()

		err := permModeErr("x", p)
		require.ErrorIs(t, err, ErrInvalidMode)
		require.ErrorIs(t, err, ErrInvalid)
		require.NotErrorIs(t, err, ErrInvalidPermission)
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
		require.Equal(t, FileMode(0o1000), permErr.Perm())
		require.Empty(t, permErr.Mode())
	})
}
