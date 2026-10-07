// fs_case.go holds the comparison of paths on the filesystem and the detection
// of case-sensitive filesystems.

package pathlib

import (
	"os"
	"strings"
	"unicode"
)

// EqualsFS reports whether this Path and other point to the same file on the
// filesystem. It compares the file info of both, following symbolic links, as
// [os.SameFile] does.
//
// A nil Path points to no file, so it never equals a Path, nil included. A
// path that cannot be checked equals no Path either.
func (p *Path) EqualsFS(other *Path) bool {
	if p == nil || other == nil {
		return false
	}

	// Stat fails for a missing path, so no separate existence check is needed.
	stat1, err := p.Stat()
	if err != nil {
		return false
	}

	stat2, err := other.Stat()
	if err != nil {
		return false
	}

	return stat1.SameFile(stat2)
}

// IsOnCaseSensitiveFS reports whether the filesystem at this Path is
// case-sensitive. This Path must exist.
//
// It toggles the casing of the first letter of the base name and checks
// whether both names point to the same file. A base name without a letter
// leads to a temporary file in the same directory, which is checked the same
// way and removed again.
//
// A missing path, a path that cannot be checked, and a failed creation of the
// temporary file all return false.
func (p *Path) IsOnCaseSensitiveFS() bool {
	if !p.Exists() {
		return false
	}

	pathBase := p.Base()

	firstLetterIndex := findFirstLetterIndex(pathBase)
	if firstLetterIndex != -1 {
		modifiedPathBase := switchCaseAtIndex(pathBase, firstLetterIndex)

		// Both casings pointing to one file means the filesystem ignores
		// casing.
		return !p.EqualsFS(p.Parent().JoinStrings(modifiedPathBase))
	}

	testDir := p.Parent()
	if p.IsDir() {
		testDir = p
	}

	// The pattern puts a letter at the start of the unique name.
	tempFile, err := os.CreateTemp(testDir.String(), "a-*")
	if err != nil {
		return false
	}

	tempFilePathStr := tempFile.Name()
	_ = tempFile.Close()
	defer func() { _ = os.Remove(tempFilePathStr) }()

	tempFilePath := NewPath(tempFilePathStr)

	modifiedTempFilePathBase := switchCaseAtIndex(tempFilePath.Base(), 0)
	return !tempFilePath.EqualsFS(tempFilePath.Parent().JoinStrings(modifiedTempFilePathBase))
}

// findFirstLetterIndex returns the byte index of the first letter of s, of any
// script, or -1 if s has no letter.
func findFirstLetterIndex(s string) int {
	return strings.IndexFunc(s, unicode.IsLetter)
}

// switchCaseAtIndex returns s with the casing of the letter at the rune index
// switched. An index out of range, a rune that is no letter, and a letter
// without upper and lower case, such as a title-case letter, return s
// unchanged.
func switchCaseAtIndex(s string, index int) string {
	// A letter can take several bytes, so the string is indexed by runes.
	runes := []rune(s)

	if index < 0 || index >= len(runes) {
		return s
	}

	r := runes[index]

	if !unicode.IsLetter(r) {
		return s
	}

	switch {
	case unicode.IsUpper(r):
		runes[index] = unicode.ToLower(r)
	case unicode.IsLower(r):
		runes[index] = unicode.ToUpper(r)
	default:
		return s
	}

	return string(runes)
}
