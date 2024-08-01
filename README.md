# go-pathlib

A simple one-file library for handling filesystem paths. Utilizing Golang's [path/filepath](https://pkg.go.dev/path/filepath), API-inspired by Python's [pathlib](https://docs.python.org/3/library/pathlib.html).

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