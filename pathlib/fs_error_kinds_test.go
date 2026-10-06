package pathlib

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

// errCase runs one operation inside a fresh temporary root and returns its error.
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
			require.Error(t, err)
			require.ErrorIs(t, err, ErrPathlib)
			check(t, err)
		})
	}
}

func TestErrNotExist_MatchesBothSentinels(t *testing.T) {
	t.Parallel()

	cases := []errCase{
		// Raised by the operating system and wrapped with an operation kind.
		{"ReadFile", func(_ *testing.T, root *Path) error {
			_, err := ReadFile(root.JoinStrings("missing"))
			return err
		}},
		{"OpenFileWithOptions", func(_ *testing.T, root *Path) error {
			_, err := OpenFileWithOptions(root.JoinStrings("missing"), OpenOptions{Mode: "r"})
			return err
		}},
		{"Stat", func(_ *testing.T, root *Path) error {
			_, err := root.JoinStrings("missing").Stat()
			return err
		}},
		{"SetPermission", func(_ *testing.T, root *Path) error {
			return SetPermission(root.JoinStrings("missing"), 0644)
		}},
		{"MkDirWithOptions with missing parent", func(_ *testing.T, root *Path) error {
			_, err := MkDirWithOptions(root.JoinStrings("a", "b"), DirOptions{})
			return err
		}},

		// Raised by the library's own checks.
		{"Copy with missing source", func(_ *testing.T, root *Path) error {
			return Copy(root.JoinStrings("missing"), root.JoinStrings("dst"))
		}},
		{"Copy with missing destination parent", func(t *testing.T, root *Path) error {
			t.Helper()

			src := writeTempFile(t, root, "src.txt", "")
			return Copy(src, root.JoinStrings("missing", "dst.txt"))
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
		{"CreateSymlink with missing parent", func(_ *testing.T, root *Path) error {
			return CreateSymlink(root, root.JoinStrings("missing", "link"))
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
			_, _, err := CreateTempFileWithOptions(&TempPathOptions{BaseDir: root.JoinStrings("missing")})
			return err
		}},
		{"CreateTempDirWithOptions with missing BaseDir", func(_ *testing.T, root *Path) error {
			_, _, err := CreateTempDirWithOptions(&TempPathOptions{BaseDir: root.JoinStrings("missing")})
			return err
		}},

		// Missing parent directories of written files.
		{"WriteBytes with missing parent", func(_ *testing.T, root *Path) error {
			_, err := WriteBytes(root.JoinStrings("missing", "file.txt"), nil)
			return err
		}},
		{"AppendBytes with missing parent", func(_ *testing.T, root *Path) error {
			_, err := AppendBytes(root.JoinStrings("missing", "file.txt"), nil)
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

func TestErrExist_MatchesBothSentinels(t *testing.T) {
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

		// An exist error raised by the operating system needs a race against the
		// library's own checks, so it is not reproducible here. The alias mechanism
		// that matches it is covered by TestRaisedError.
	}

	runErrCases(t, cases, func(t *testing.T, err error) {
		t.Helper()

		require.ErrorIs(t, err, ErrExist)
		require.ErrorIs(t, err, fs.ErrExist)
		require.NotErrorIs(t, err, ErrNotExist)
		require.NotErrorIs(t, err, fs.ErrNotExist)
	})
}

func TestRequireDir_ExistingNonDirectory(t *testing.T) {
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

			_, _, err := CreateTempDirWithOptions(&TempPathOptions{BaseDir: writeTempFile(t, root, "file.txt", "")})
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

func TestRequireDir_ReportsKindAndPath(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	missing := root.JoinStrings("missing")

	err := requireDir(missing)
	var kind *PathlibError
	require.ErrorAs(t, err, &kind)
	require.Equal(t, ErrNotExist, kind)

	var cause *PathError
	require.ErrorAs(t, err, &cause)
	require.Equal(t, []Path{*missing}, cause.Paths())
	require.Error(t, cause.Unwrap(), "the os cause is kept")

	require.NoError(t, requireDir(root))
	require.NoError(t, requireDir(createTempSymlinkAbs(t, root, ".", "link")))
}

func TestReadFile_DirectoryIsNotErrNotExist(t *testing.T) {
	t.Parallel()

	_, err := ReadFile(setupTempDir(t))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNotExist)
	require.NotErrorIs(t, err, fs.ErrNotExist)
}
