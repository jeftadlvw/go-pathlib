// io_open_mode.go holds OpenMode, the mode a file is opened in.

package pathlib

import (
	"os"
	"strconv"
)

// The modes a file is opened in. Truncation and appending are part of the
// name, so an open mode that empties a file says so.
const (
	// OpenRead opens the file for reading alone. It is the zero value.
	OpenRead OpenMode = iota

	// OpenWrite opens the file for writing alone. The content is kept, and
	// writing starts at the beginning of the file.
	OpenWrite

	// OpenWriteTruncate opens the file for writing alone and truncates it.
	OpenWriteTruncate

	// OpenReadWrite opens the file for reading and writing. The content is
	// kept, and writing starts at the beginning of the file.
	OpenReadWrite

	// OpenReadWriteTruncate opens the file for reading and writing and
	// truncates it.
	OpenReadWriteTruncate

	// OpenAppend opens the file for writing alone. Every write appends to the
	// end of the file.
	OpenAppend

	// OpenReadAppend opens the file for reading and writing. Every write
	// appends to the end of the file.
	OpenReadAppend
)

// OpenMode is the mode [OpenFileWithOptions] opens a file in. It sets whether
// the file is read, written, or both, and whether writing truncates the file
// or appends to it. The zero value is OpenRead.
type OpenMode int

// Valid reports whether the open mode is one of the OpenMode constants.
func (m OpenMode) Valid() bool {
	_, ok := m.flags()
	return ok
}

// String returns the name of the open mode, such as "OpenReadWrite", or
// "OpenMode(9)" for an invalid open mode.
func (m OpenMode) String() string {
	switch m {
	case OpenRead:
		return "OpenRead"
	case OpenWrite:
		return "OpenWrite"
	case OpenWriteTruncate:
		return "OpenWriteTruncate"
	case OpenReadWrite:
		return "OpenReadWrite"
	case OpenReadWriteTruncate:
		return "OpenReadWriteTruncate"
	case OpenAppend:
		return "OpenAppend"
	case OpenReadAppend:
		return "OpenReadAppend"
	default:
		return "OpenMode(" + strconv.Itoa(int(m)) + ")"
	}
}

// flags returns the flags for [os.OpenFile] that implement the open mode. An
// invalid open mode returns false.
func (m OpenMode) flags() (int, bool) {
	switch m {
	case OpenRead:
		return os.O_RDONLY, true
	case OpenWrite:
		return os.O_WRONLY, true
	case OpenWriteTruncate:
		return os.O_WRONLY | os.O_TRUNC, true
	case OpenReadWrite:
		return os.O_RDWR, true
	case OpenReadWriteTruncate:
		return os.O_RDWR | os.O_TRUNC, true
	case OpenAppend:
		return os.O_WRONLY | os.O_APPEND, true
	case OpenReadAppend:
		return os.O_RDWR | os.O_APPEND, true
	default:
		return 0, false
	}
}
