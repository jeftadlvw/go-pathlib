// fs_glob_filter.go holds the filter of the entry types a glob includes.

package pathlib

import (
	"strconv"
)

/*
GlobOptionFilterType selects the types of entries a glob includes. The zero value
is GlobOptionFilterAll.
*/
type GlobOptionFilterType int

const (
	// GlobOptionFilterAll includes files and directories.
	GlobOptionFilterAll GlobOptionFilterType = iota
	// GlobOptionFilterFiles includes files only.
	GlobOptionFilterFiles
	// GlobOptionFilterDirectories includes directories only.
	GlobOptionFilterDirectories
)

/*
Valid reports whether the filter is one of the defined GlobOptionFilterType
constants.
*/
func (f GlobOptionFilterType) Valid() bool {
	switch f {
	case GlobOptionFilterAll, GlobOptionFilterFiles, GlobOptionFilterDirectories:
		return true
	default:
		return false
	}
}

/*
String returns the name of the filter, such as "files". An invalid filter
returns its numeric value, as in "GlobOptionFilterType(7)".
*/
func (f GlobOptionFilterType) String() string {
	switch f {
	case GlobOptionFilterAll:
		return "all"
	case GlobOptionFilterFiles:
		return "files"
	case GlobOptionFilterDirectories:
		return "directories"
	default:
		return "GlobOptionFilterType(" + strconv.Itoa(int(f)) + ")"
	}
}
