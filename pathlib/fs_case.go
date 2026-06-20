package pathlib

import (
	"os"
	"strings"
	"unicode"
)

/*
EqualsFs returns whether this Path and another Path point to the same file system inode.
This comparison is performed using file stats, not string comparison.

Symlinks are resolved.
*/
func (p *Path) EqualsFs(other *Path) bool {
	// No need to call Exists for both paths, as Stat() will return an error
	// if the path does not exist.

	stat1, err := p.Stat()
	if err != nil {
		return false
	}

	stat2, err := other.Stat()
	if err != nil {
		return false
	}

	return os.SameFile(stat1, stat2)
}

/*
IsOnCaseSensitiveFs checks if the filesystem at this Path is case-sensitive.

It first tries to check for the given path, toggling the casing of the first encountered letter in the path's base,
checking if the file exists and if both file descriptors point to the same file.

If no letter exists within the path's base, a temporary file is created in the same directory for which
the upper procedure is repeated.

If both attempts result in an invalid state, false is returned.
*/
func (p *Path) IsOnCaseSensitiveFs() bool {
	if !p.Exists() {
		return false
	}

	// Check for case sensitivity by inverting a letter in a copy of this Path's base
	// and check if both paths are EqualsFs()
	pathBase := p.Base()

	firstLetterIndex := findFirstLetterIndex(pathBase)
	hasLetter := firstLetterIndex != -1

	if hasLetter {
		modifiedPathBase := switchCaseAtIndex(pathBase, firstLetterIndex)
		equalOnFilesystem := p.EqualsFs(p.Parent().JoinStrings(modifiedPathBase))

		// If both paths are equal on the filesystem, the unmodified and modified paths both point to the same file.
		// This means the filesystem is NOT case-sensitive.
		return !equalOnFilesystem
	}

	// If the base does not include a letter, create a temporary file and repeat the check.

	// Get a directory to test in
	var testDir *Path
	if p.IsDir() {
		testDir = p
	} else {
		testDir = p.Parent()
	}

	// os.CreateTemp ensures a unique filename
	// The passed pattern is to ensure a single letter at a defined index
	tempFile, err := os.CreateTemp(testDir.String(), "a-*")
	if err != nil {
		return false
	}

	// Get file name, close and defer removal
	tempFilePathStr := tempFile.Name()
	_ = tempFile.Close()
	defer os.Remove(tempFilePathStr)

	tempFilePath := NewPath(tempFilePathStr)

	modifiedTempFilePathBase := switchCaseAtIndex(tempFilePath.Base(), firstLetterIndex)
	return !tempFilePath.EqualsFs(tempFilePath.Parent().JoinStrings(modifiedTempFilePathBase))
}

// findFirstLetterIndex returns the index of the first letter within a string.
// It correctly identifies letters from various scripts using unicode.IsLetter.
// If no letter is found, it returns -1.
func findFirstLetterIndex(s string) int {
	// strings.IndexFunc finds the first index where the provided function returns true.
	// unicode.IsLetter is the function that checks if a rune is a letter
	// in any language or script.
	return strings.IndexFunc(s, unicode.IsLetter)
}

// switchCaseAtIndex switches the case of the letter at the specified index in the string.
// If the index is out of bounds or the character at the index is not a letter,
// the original string is returned.
func switchCaseAtIndex(s string, index int) string {
	// Convert the string to a slice of runes. This is important because Go
	// strings are UTF-8 encoded, and a character (rune) might take up
	// more than one byte. Working with runes ensures we handle characters correctly.
	runes := []rune(s)

	// Check if the index is valid (within the bounds of the rune slice)
	if index < 0 || index >= len(runes) {
		// Index out of bounds, return the original string unchanged
		return s
	}

	// Get the rune at the specified index
	r := runes[index]

	// Check if the character is a letter
	if !unicode.IsLetter(r) {
		// Not a letter, return the original string unchanged
		return s
	}

	// Switch the case of the letter
	if unicode.IsUpper(r) {
		// If it's an uppercase letter, convert it to lowercase
		runes[index] = unicode.ToLower(r)
	} else if unicode.IsLower(r) {
		// If it's a lowercase letter, convert it to uppercase
		runes[index] = unicode.ToUpper(r)
	} else {
		// It's a letter, but neither upper nor lower (e.g., titlecase).
		// In this function's scope, we only handle upper/lower switching.
		// Return the original string as no standard case switch occurred.
		return s
	}

	// Convert the modified slice of runes back into a string
	return string(runes)
}
