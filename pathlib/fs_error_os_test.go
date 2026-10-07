package pathlib

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name   string
		Err    error
		Expect *PathlibError
	}{
		{"Missing path", &fs.PathError{Op: "open", Path: "a", Err: fs.ErrNotExist}, ErrNotExist},
		{"Existing path", &fs.PathError{Op: "open", Path: "a", Err: fs.ErrExist}, ErrExist},
		{"Denied access", &fs.PathError{Op: "open", Path: "a", Err: fs.ErrPermission}, ErrPermissionDenied},
		{"Any other error", &fs.PathError{Op: "read", Path: "a", Err: errSimulated}, ErrRead},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			err := osErr(ErrRead, c.Err, *NewPath("a"))

			var kind *PathlibError
			require.ErrorAs(t, err, &kind)
			require.Equal(t, c.Expect, kind, "the most specific kind")
			require.ErrorIs(t, err, c.Err, "the error of the operating system stays wrapped")
		})
	}

	t.Run("Errors of the operating system are classified", func(t *testing.T) {
		t.Parallel()

		_, err := os.Open(setupTempDir(t).JoinStrings("missing").String())
		require.ErrorIs(t, osErr(ErrOpen, err), ErrNotExist)
	})
}

func TestOsCreateErr(t *testing.T) {
	t.Parallel()

	p := *NewPath("a/b")

	cases := []struct {
		Name   string
		Err    error
		Expect *PathlibError
	}{
		{"Missing parent directory", &fs.PathError{Op: "open", Path: "a/b", Err: fs.ErrNotExist}, ErrParentNotExist},
		{"Existing path", &fs.PathError{Op: "open", Path: "a/b", Err: fs.ErrExist}, ErrExist},
		{"Denied access", &fs.PathError{Op: "open", Path: "a/b", Err: fs.ErrPermission}, ErrPermissionDenied},
		{"Any other error", &fs.PathError{Op: "open", Path: "a/b", Err: errSimulated}, ErrCreate},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			err := osCreateErr(ErrCreate, c.Err, p)

			var kind *PathlibError
			require.ErrorAs(t, err, &kind)
			require.Equal(t, c.Expect, kind, "the most specific kind")
			require.ErrorIs(t, err, c.Err, "the error of the operating system stays wrapped")
		})
	}
}
