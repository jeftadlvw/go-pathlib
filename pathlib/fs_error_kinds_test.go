package pathlib

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

// errCase runs one operation inside a fresh temporary root and returns its
// error.
type errCase struct {
	Name string
	Run  func(t *testing.T, root *Path) error
}

func runErrCases(t *testing.T, cases []errCase, check func(t *testing.T, err error)) {
	t.Helper()

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			err := c.Run(t, setupTempDir(t))
			require.ErrorIs(t, err, ErrPathlib)
			check(t, err)
		})
	}
}

func TestErrNotExist_LibraryChecks(t *testing.T) {
	t.Parallel()

	cases := []errCase{
		// Raised by the library's own checks.
		{"Copy with missing source", func(_ *testing.T, root *Path) error {
			return Copy(root.JoinStrings("missing"), root.JoinStrings("dst"))
		}},
		{"Move with missing source", func(_ *testing.T, root *Path) error {
			return Move(root.JoinStrings("missing"), root.JoinStrings("dst"))
		}},
		{"Resolve", func(_ *testing.T, root *Path) error {
			_, err := root.JoinStrings("missing").Resolve()
			return err
		}},
		{"SymlinkTo with missing target", func(_ *testing.T, root *Path) error {
			return root.JoinStrings("missing").SymlinkTo(root.JoinStrings("link"))
		}},

		// Missing directories.
		{"Walk", func(_ *testing.T, root *Path) error {
			return root.JoinStrings("missing").Walk(func(*Path) error { return nil })
		}},
		{"Walk with broken symlink root", func(t *testing.T, root *Path) error {
			t.Helper()

			link := createTempSymlinkAbs(t, root, "missing", "link")
			return link.Walk(func(*Path) error { return nil })
		}},
		{"WalkR", func(_ *testing.T, root *Path) error {
			return root.JoinStrings("missing").WalkR(func(*Path, error) error { return nil })
		}},
		{"Glob", func(_ *testing.T, root *Path) error {
			_, err := root.JoinStrings("missing").Glob("*")
			return err
		}},
		{"List", func(_ *testing.T, root *Path) error {
			_, err := root.JoinStrings("missing").List(DefaultListOptions())
			return err
		}},
		{"ListFiles", func(_ *testing.T, root *Path) error {
			_, err := root.JoinStrings("missing").ListFiles(false)
			return err
		}},
		{"ListDirs", func(_ *testing.T, root *Path) error {
			_, err := root.JoinStrings("missing").ListDirs(false)
			return err
		}},
		{"CreateTempFileWithOptions with missing BaseDir", func(_ *testing.T, root *Path) error {
			_, _, err := CreateTempFileWithOptions(TempPathOptions{BaseDir: root.JoinStrings("missing")})
			return err
		}},
		{"CreateTempDirWithOptions with missing BaseDir", func(_ *testing.T, root *Path) error {
			_, _, err := CreateTempDirWithOptions(TempPathOptions{BaseDir: root.JoinStrings("missing")})
			return err
		}},
	}

	runErrCases(t, cases, func(t *testing.T, err error) {
		t.Helper()

		require.ErrorIs(t, err, ErrNotExist)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.NotErrorIs(t, err, ErrExist)
		require.NotErrorIs(t, err, fs.ErrExist)
		require.NotErrorIs(t, err, ErrNotDir)
	})
}

func TestErrNotExist_OperatingSystem(t *testing.T) {
	t.Parallel()

	// Raised by the operating system and classified as ErrNotExist.
	cases := []errCase{
		{"ReadFile", func(_ *testing.T, root *Path) error {
			_, err := ReadFile(root.JoinStrings("missing"))
			return err
		}},
		{"OpenFileWithOptions", func(_ *testing.T, root *Path) error {
			_, err := OpenFileWithOptions(root.JoinStrings("missing"), OpenOptions{OpenMode: OpenRead})
			return err
		}},
		{"Stat", func(_ *testing.T, root *Path) error {
			_, err := root.JoinStrings("missing").Stat()
			return err
		}},
		{"SetMode", func(_ *testing.T, root *Path) error {
			return SetMode(root.JoinStrings("missing"), 0644)
		}},
	}

	runErrCases(t, cases, func(t *testing.T, err error) {
		t.Helper()

		require.ErrorIs(t, err, ErrNotExist)
		require.NotErrorIs(t, err, ErrParentNotExist)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.NotErrorIs(t, err, ErrExist)
		require.NotErrorIs(t, err, fs.ErrExist)
	})
}

func TestErrParentNotExist(t *testing.T) {
	t.Parallel()

	// The library's own checks and the operating system both find the missing
	// parent directory of a created path.
	cases := []errCase{
		{"Copy", func(t *testing.T, root *Path) error {
			t.Helper()

			src := writeTempFile(t, root, "src.txt", "")
			return Copy(src, root.JoinStrings("missing", "dst.txt"))
		}},
		{"CreateSymlink", func(_ *testing.T, root *Path) error {
			return CreateSymlink(root, root.JoinStrings("missing", "link"))
		}},
		{"CreateFile", func(_ *testing.T, root *Path) error {
			return CreateFile(root.JoinStrings("missing", "file.txt"))
		}},
		{"MkDir", func(_ *testing.T, root *Path) error {
			return MkDir(root.JoinStrings("missing", "dir"))
		}},
		{"OpenFile", func(_ *testing.T, root *Path) error {
			_, err := OpenFile(root.JoinStrings("missing", "file.txt"))
			return err
		}},
		{"OpenFileWithOptions read-only", func(_ *testing.T, root *Path) error {
			_, err := OpenFileWithOptions(
				root.JoinStrings("missing", "file.txt"), OpenOptions{CreateIfNotExists: true, OpenMode: OpenRead},
			)
			return err
		}},
		{"WriteBytes", func(_ *testing.T, root *Path) error {
			_, err := WriteBytes(root.JoinStrings("missing", "file.txt"), nil)
			return err
		}},
		{"WriteBytesWithOptions without ExistOk", func(_ *testing.T, root *Path) error {
			_, err := WriteBytesWithOptions(root.JoinStrings("missing", "file.txt"), nil, FileOptions{})
			return err
		}},
		{"AppendBytes", func(_ *testing.T, root *Path) error {
			_, err := AppendBytes(root.JoinStrings("missing", "file.txt"), nil)
			return err
		}},
	}

	runErrCases(t, cases, func(t *testing.T, err error) {
		t.Helper()

		require.ErrorIs(t, err, ErrParentNotExist)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.NotErrorIs(t, err, ErrExist)
	})
}

func TestErrExist_LibraryChecks(t *testing.T) {
	t.Parallel()

	cases := []errCase{
		// Raised by the library's own checks.
		{"CreateFile", func(t *testing.T, root *Path) error {
			t.Helper()

			return CreateFile(writeTempFile(t, root, "file.txt", ""))
		}},
		{"MkDir", func(t *testing.T, root *Path) error {
			t.Helper()

			return MkDir(createTempDir(t, root, "dir"))
		}},
		{"CreateSymlink", func(t *testing.T, root *Path) error {
			t.Helper()

			return CreateSymlink(root, writeTempFile(t, root, "file.txt", ""))
		}},
		{"Copy onto existing file", func(t *testing.T, root *Path) error {
			t.Helper()

			src := writeTempFile(t, root, "src.txt", "")
			return Copy(src, writeTempFile(t, root, "dst.txt", ""))
		}},
		{"Move onto existing file", func(t *testing.T, root *Path) error {
			t.Helper()

			src := writeTempFile(t, root, "src.txt", "")
			return Move(src, writeTempFile(t, root, "dst.txt", ""))
		}},
		{"CreateSymlink over broken symlink", func(t *testing.T, root *Path) error {
			t.Helper()

			link := createTempSymlinkAbs(t, root, "missing", "link")
			return CreateSymlink(root, link)
		}},
		{"WriteBytesWithOptions without ExistOk", func(t *testing.T, root *Path) error {
			t.Helper()

			_, err := WriteBytesWithOptions(writeTempFile(t, root, "file.txt", ""), nil, FileOptions{})
			return err
		}},
		{"Copy onto broken symlink", func(t *testing.T, root *Path) error {
			t.Helper()

			src := writeTempFile(t, root, "src.txt", "")
			return Copy(src, createTempSymlinkAbs(t, root, "missing", "link"))
		}},
		{"Move onto broken symlink", func(t *testing.T, root *Path) error {
			t.Helper()

			src := writeTempFile(t, root, "src.txt", "")
			return Move(src, createTempSymlinkAbs(t, root, "missing", "link"))
		}},

		// An exist error raised by the operating system needs a race against
		// the library's own checks, so it is not reproducible here.
	}

	runErrCases(t, cases, func(t *testing.T, err error) {
		t.Helper()

		require.ErrorIs(t, err, ErrExist)
		require.ErrorIs(t, err, fs.ErrExist)
		require.NotErrorIs(t, err, ErrNotExist)
		require.NotErrorIs(t, err, fs.ErrNotExist)
	})
}

func TestErrNotDir_NonDirectoryRoot(t *testing.T) {
	t.Parallel()

	cases := []errCase{
		{"Walk", func(t *testing.T, root *Path) error {
			t.Helper()

			return writeTempFile(t, root, "file.txt", "").Walk(func(*Path) error { return nil })
		}},
		{"WalkR", func(t *testing.T, root *Path) error {
			t.Helper()

			return writeTempFile(t, root, "file.txt", "").WalkR(func(*Path, error) error { return nil })
		}},
		{"Glob", func(t *testing.T, root *Path) error {
			t.Helper()

			_, err := writeTempFile(t, root, "file.txt", "").Glob("*")
			return err
		}},
		{"List", func(t *testing.T, root *Path) error {
			t.Helper()

			_, err := writeTempFile(t, root, "file.txt", "").List(DefaultListOptions())
			return err
		}},
		{"CreateTempDirWithOptions", func(t *testing.T, root *Path) error {
			t.Helper()

			_, _, err := CreateTempDirWithOptions(TempPathOptions{BaseDir: writeTempFile(t, root, "file.txt", "")})
			return err
		}},
	}

	runErrCases(t, cases, func(t *testing.T, err error) {
		t.Helper()

		require.ErrorIs(t, err, ErrNotDir)
		require.NotErrorIs(t, err, ErrNotExist)
		require.NotErrorIs(t, err, fs.ErrNotExist)
	})
}

func TestReadFile_DirectoryIsNotErrNotExist(t *testing.T) {
	t.Parallel()

	_, err := ReadFile(setupTempDir(t))
	require.ErrorIs(t, err, ErrPathlib)
	require.NotErrorIs(t, err, ErrNotExist)
	require.NotErrorIs(t, err, fs.ErrNotExist)
}
