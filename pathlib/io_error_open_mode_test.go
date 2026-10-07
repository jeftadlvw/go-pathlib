package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenModeError(t *testing.T) {
	t.Parallel()

	p := *NewPath("a/b")

	t.Run("Kind and cause", func(t *testing.T) {
		t.Parallel()

		err := openModeErr(OpenMode(99), p)
		require.ErrorIs(t, err, ErrInvalidOpenMode)
		require.ErrorIs(t, err, ErrInvalid)
		require.NotErrorIs(t, err, ErrInvalidFileMode)
		require.ErrorIs(t, err, ErrPathlib)
		require.ErrorAs(t, err, new(*PathlibError))
		require.NotErrorAs(t, err, new(*PathError))
		require.NotErrorAs(t, err, new(*FileModeError))
	})

	t.Run("Returns the rejected value", func(t *testing.T) {
		t.Parallel()

		var cause *OpenModeError
		require.ErrorAs(t, openModeErr(OpenMode(99), p), &cause)
		require.Equal(t, OpenMode(99), cause.OpenMode())
		require.Equal(t, []Path{p}, cause.Paths())
	})
}
