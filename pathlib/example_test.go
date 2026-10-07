package pathlib_test

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jeftadlvw/go-pathlib/pathlib"
)

// Every error pairs a kind with a cause. errors.Is matches the kind and its
// groups, and errors.As reads the kind and the cause.
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

	var kind *pathlib.PathlibError
	if errors.As(err, &kind) {
		fmt.Println(kind.Code())
	}

	var cause *pathlib.PathError
	if errors.As(err, &cause) {
		fmt.Println(cause.Paths()[0].Base())
	}

	// Output:
	// true
	// true
	// true
	// PATHLIB.NOT_EXIST
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
