package pathlib

import (
	"errors"
)

// ErrDisposeNil is returned when a TempPath has no dispose function set.
var ErrDisposeNil = errors.New("dispose function is nil")
