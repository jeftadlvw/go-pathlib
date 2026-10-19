package pathlib

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// errSpaced is an error whose text has surrounding and repeated whitespace.
var errSpaced = errors.New("  no  such\tfile ")

func TestPathError_Error(t *testing.T) {
	t.Parallel()

	p := *NewPathFromPosix("a/b")
	o := *NewPathFromPosix("c")

	cases := []struct {
		Name   string
		Err    error
		Expect string
	}{
		{
			"Path error on the path",
			&fs.PathError{Op: "open", Path: p.String(), Err: fs.ErrNotExist},
			p.String() + ": open: file does not exist",
		},
		{
			"Path error on another path",
			&fs.PathError{Op: "open", Path: o.String(), Err: fs.ErrNotExist},
			p.String() + ": open " + o.String() + ": file does not exist",
		},
		{
			"Whitespace of a shortened text",
			&fs.PathError{Op: " open ", Path: p.String(), Err: errSpaced},
			p.String() + ": open: no such file",
		},
		{
			"Whitespace of an unchanged text",
			errSpaced,
			p.String() + ":   no  such\tfile ",
		},
		{
			"Link error on the new path",
			&os.LinkError{Op: "symlink", Old: o.String(), New: p.String(), Err: fs.ErrExist},
			p.String() + ": symlink " + o.String() + ": file already exists",
		},
		{
			"Link error on another path",
			&os.LinkError{Op: "symlink", Old: p.String(), New: o.String(), Err: fs.ErrExist},
			p.String() + ": symlink " + p.String() + " " + o.String() + ": file already exists",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			err := &PathError{paths: []Path{p}, err: c.Err}
			require.Equal(t, c.Expect, err.Error())
		})
	}

	t.Run("Link error on both paths", func(t *testing.T) {
		t.Parallel()

		cause := &os.LinkError{Op: "rename", Old: p.String(), New: o.String(), Err: fs.ErrExist}
		err := &PathError{paths: []Path{p, o}, err: cause}
		require.Equal(t, "["+p.String()+", "+o.String()+"]: rename: file already exists", err.Error())
	})

	t.Run("Link error on both paths in another order", func(t *testing.T) {
		t.Parallel()

		cause := &os.LinkError{Op: "rename", Old: o.String(), New: p.String(), Err: fs.ErrExist}
		err := &PathError{paths: []Path{p, o}, err: cause}
		require.Equal(t, "["+p.String()+", "+o.String()+"]: "+cause.Error(), err.Error())
	})

	t.Run("Shortened text keeps the chain", func(t *testing.T) {
		t.Parallel()

		err := osErr(ErrOpen, &fs.PathError{Op: "open", Path: p.String(), Err: fs.ErrNotExist}, p)
		require.Equal(t, "path does not exist (PATHLIB.NOT_EXIST): "+p.String()+": open: file does not exist", err.Error())
		require.ErrorIs(t, err, fs.ErrNotExist)

		var osPathErr *fs.PathError
		require.ErrorAs(t, err, &osPathErr)
	})
}
