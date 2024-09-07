# go-pathlib

A simple one-file library for handling filesystem paths. Utilizing Golang's [path/filepath](https://pkg.go.dev/path/filepath), API-inspired by Python's [pathlib](https://docs.python.org/3/library/pathlib.html). Meant to abstract and extend the standard library and create a struct that contains a source of truth.

Also, let's be honest... double backslash path separators are the stupidest thing we've ever seen. But hey, we support 'em.

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

> [!NOTE]
> ToDo

## Missing features
The following features would improve the integration into other ecosystems and add some nice cherries on top:

- [ ] direct filesystem operations (creating and opening files, creating directories, delete files and directories and directory trees)
- [ ] implement JSON marshalling interface
- [ ] integration into [go-validator](https://github.com/go-playground/validator) (custom field types and validators)

This is a non-exhaustive list. Feel free to suggest other features and integrations!
