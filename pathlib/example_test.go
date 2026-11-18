package pathlib_test

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jeftadlvw/go-pathlib/pathlib"
)

// Every error pairs a kind with a cause. errors.Is matches the kind and its
// groups, and errors.AsType reads the kind and the cause.
func Example_errorHandling() {
	dir, dispose, err := pathlib.CreateTempDir()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = dispose() }()

	_, err = pathlib.ReadFile(dir.JoinStrings("missing.txt"))

	fmt.Println(errors.Is(err, pathlib.ErrPathlib))
	fmt.Println(errors.Is(err, pathlib.ErrNotExist))
	fmt.Println(errors.Is(err, fs.ErrNotExist))

	kind, ok := errors.AsType[*pathlib.ErrorKind](err)
	if ok {
		fmt.Println(kind.Code())
	}

	cause, ok := errors.AsType[*pathlib.PathError](err)
	if ok {
		fmt.Println(cause.Paths()[0].Base())
	}

	// Output:
	// true
	// true
	// true
	// pathlib.not_exist
	// missing.txt
}

// The caller removes a temporary file with the DisposeFunc returned next to
// it.
func ExampleCreateTempFile() {
	file, dispose, err := pathlib.CreateTempFile()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = dispose() }()

	fmt.Println(file.IsFile())

	// Output:
	// true
}

// TempPathOptions place a temporary directory in a base directory and start
// its name with a prefix.
func ExampleCreateTempDirWithOptions() {
	base, disposeBase, err := pathlib.CreateTempDir()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = disposeBase() }()

	dir, dispose, err := pathlib.CreateTempDirWithOptions(pathlib.TempPathOptions{
		BaseDir: base,
		Prefix:  "cache-",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = dispose() }()

	fmt.Println(dir.Parent().Equals(base))
	fmt.Println(strings.HasPrefix(dir.Base(), "cache-"))

	// Output:
	// true
	// true
}

// fs.WalkDir passes the slash-separated names of the file system of Path.FS,
// and NewPathFromPosix joined to the Path turns a name back into a Path.
func ExamplePath_FS() {
	root, dispose, err := pathlib.CreateTempDir()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() { _ = dispose() }()

	err = pathlib.MkDir(root.JoinStrings("dir"))
	if err != nil {
		fmt.Println(err)
		return
	}

	for name, content := range map[string]string{"a.txt": "a", "dir/b.txt": "b"} {
		_, err = pathlib.WriteString(root.Join(pathlib.NewPathFromPosix(name)), content)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		content, err := pathlib.ReadFileToString(root.Join(pathlib.NewPathFromPosix(name)))
		if err != nil {
			return err
		}

		fmt.Println(name, content)
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}

	// Output:
	// a.txt a
	// dir/b.txt b
}
