package pathlib_test

import (
	"errors"
	"fmt"
	"io/fs"

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
	// PATHLIB_READ
	// missing.txt
}
