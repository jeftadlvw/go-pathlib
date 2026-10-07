// core_compare_option.go holds CompareOption, which selects how comparisons
// treat casing.

package pathlib

import (
	"strconv"
)

// CompareOption selects how path equality and pattern matching treat casing.
// The zero value is CaseSensitive.
type CompareOption int

const (
	// CaseSensitive compares casing exactly. It is the default.
	CaseSensitive CompareOption = iota

	// CaseInsensitive ignores casing.
	CaseInsensitive
)

// Valid reports whether the option is one of the defined CompareOption
// constants.
func (o CompareOption) Valid() bool {
	switch o {
	case CaseSensitive, CaseInsensitive:
		return true
	default:
		return false
	}
}

// String returns the name of the option, such as "case-sensitive". An invalid
// option returns its numeric value, as in "CompareOption(7)".
func (o CompareOption) String() string {
	switch o {
	case CaseSensitive:
		return "case-sensitive"
	case CaseInsensitive:
		return "case-insensitive"
	default:
		return "CompareOption(" + strconv.Itoa(int(o)) + ")"
	}
}

// ignoresCase reports whether opts, the variadic options of a comparison,
// select CaseInsensitive. Only the first option counts.
func ignoresCase(opts []CompareOption) bool {
	return len(opts) > 0 && opts[0] == CaseInsensitive
}
