<div align="center">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="assets/artwork/logo-white.png">
      <source media="(prefers-color-scheme: light)" srcset="assets/artwork/logo-dark.png">
      <img alt="Artwork for go-pathlib." src="assets/artwork/logo-white.png">
    </picture>
</div>

# go-pathlib

A simple one-file library for handling filesystem paths. Utilizing Golang's [path/filepath](https://pkg.go.dev/path/filepath), API-inspired by Python's [pathlib](https://docs.python.org/3/library/pathlib.html). Meant to abstract and extend the standard library and create a struct that contains a source of truth.

This library is developed and tested on Unix-based operating systems. Windows should work (in theory), please open an issue if you face any problems.

## Getting started 🚀
```shell
go get github.com/jeftadlvw/go-pathlib
```

```go
package main

import (
	"fmt"
	"github.com/jeftadlvw/go-pathlib"
)

func main() {
	p := pathlib.NewPath("path/to/your/destination")
	fmt.Println(p)
}
```

## API Documentation 📝
Repository-local documentation can be found at [docs/go-pathlib.md](docs/pathlib). It's auto-generated by `gomarkdoc` using the docstrings in the source code.

The file is updated regularly and gives a good general overview on the API. Up-to-date documentation can be found in the source code.

## Recommendations and Gotcha's 🫣
- Single-file cross-platform filepath compatibility is tricky.
- When persisting file paths, use **relative paths** and the **unix representation** for maximum portability. Also, enforce that paths must be wrapped in quotation marks (in e.g. configuration files), e.g. `"path/to/foo.bar"`. This escapes any usage of whitespaces.
- On Unix-based operating systems, the Windows path root (e.g. `C:\`) is not considered and tested as a filepath root but a regular path element. However, you still might get correct results for e.g. `Path.Root()`.

## Missing features ✨
The following features would improve the integration into other ecosystems and add some nice cherries on top:

- [ ] direct filesystem operations (creating and opening files, creating directories, delete files and directories and directory trees)
- [ ] recursive globbing using double asterisks (stable and tested without using external dependencies)
- [ ] extend globbing to not include directories
- [ ] implement "range over function" for globbing
- [ ] implement JSON marshalling interface
- [ ] function to check if a file is hidden
- [ ] integration into [go-validator](https://github.com/go-playground/validator) (custom field types and validators)
- [ ] APIs for temporary files and directories
- [ ] tested Windows support

This is a non-exhaustive list. Feel free to suggest other features and integrations!

## Attributions
The displayed Gopher in the artwork is under the [_Creative Commons 4.0 Attribution License_](https://creativecommons.org/licenses/by/4.0/) as per https://go.dev/brand#logo (last seen: 30-09-2024). This project's artwork falls under the same licence.
