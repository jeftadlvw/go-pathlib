package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileModeError(t *testing.T) {
	t.Parallel()

	p := *NewPath("a/b")

	t.Run("Kind and cause", func(t *testing.T) {
		t.Parallel()

		err := fileModeErr(0o1000, p)
		require.ErrorIs(t, err, ErrInvalidFileMode)
		require.ErrorIs(t, err, ErrInvalid)
		require.NotErrorIs(t, err, ErrInvalidOpenMode)
		require.NotErrorIs(t, err, ErrPermissionDenied)
		require.ErrorIs(t, err, ErrPathlib)
		require.ErrorAs(t, err, new(*PathlibError))
		require.NotErrorAs(t, err, new(*PathError))
		require.NotErrorAs(t, err, new(*OpenModeError))
	})

	t.Run("Returns the rejected value", func(t *testing.T) {
		t.Parallel()

		var cause *FileModeError
		require.ErrorAs(t, fileModeErr(0o1000, p), &cause)
		require.Equal(t, FileMode(0o1000), cause.FileMode())
		require.Equal(t, []Path{p}, cause.Paths())
	})
}
