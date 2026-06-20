package pathlib

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path"
	"strings"
)

// matchPattern is the internal implementation that handles ** expansion.
func matchPattern(pattern, name string) (bool, error) {
	// Validate pattern for bad syntax (check each segment)
	if err := validatePattern(pattern); err != nil {
		return false, err
	}

	// If no **, use standard path.Match
	if !strings.Contains(pattern, "**") {
		return path.Match(pattern, name)
	}

	return matchWithDoubleAsterisk(pattern, name)
}

// validatePattern checks if the pattern has valid syntax.
func validatePattern(pattern string) error {
	// Split by ** and validate each segment with path.Match
	segments := strings.Split(pattern, "**")
	for _, seg := range segments {
		// Remove leading/trailing slashes for validation
		seg = strings.Trim(seg, "/")
		if seg == "" {
			continue
		}
		// Use path.Match to validate syntax (match against empty string just to check pattern validity)
		_, err := path.Match(seg, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// matchWithDoubleAsterisk handles patterns containing **.
func matchWithDoubleAsterisk(pattern, name string) (bool, error) {
	// Split pattern by **
	parts := strings.Split(pattern, "**")

	// Handle edge cases
	if len(parts) == 1 {
		// No ** found (shouldn't reach here, but safety check)
		return path.Match(pattern, name)
	}

	// For pattern like "**", it matches everything
	if pattern == "**" {
		return true, nil
	}

	// Process the pattern parts
	return matchParts(parts, name)
}

// matchParts matches the name against pattern parts split by **.
func matchParts(parts []string, name string) (bool, error) {
	// First part must match the beginning of name (if not empty)
	firstPart := parts[0]
	if firstPart != "" {
		// First part doesn't start with **, so it must match from the beginning
		firstPart = strings.TrimSuffix(firstPart, "/")
		if !matchesPrefix(name, firstPart) {
			return false, nil
		}
		// Calculate how much of name was consumed
		prefixLen := findPrefixMatchLength(name, firstPart)
		if prefixLen == -1 {
			return false, nil
		}
		name = name[prefixLen:]
		name = strings.TrimPrefix(name, "/")
	}

	// Last part must match the end of name (if not empty)
	lastPart := parts[len(parts)-1]
	if lastPart != "" {
		lastPart = strings.TrimPrefix(lastPart, "/")
		if !matchesSuffix(name, lastPart) {
			return false, nil
		}
		// Calculate how much of name remains
		suffixLen := findSuffixMatchLength(name, lastPart)
		if suffixLen == -1 {
			return false, nil
		}
		name = name[:len(name)-suffixLen]
		name = strings.TrimSuffix(name, "/")
	}

	// Middle parts must appear in order somewhere in name
	for i := 1; i < len(parts)-1; i++ {
		middlePart := strings.Trim(parts[i], "/")
		if middlePart == "" {
			continue
		}

		idx := findPatternInPath(name, middlePart)
		if idx == -1 {
			return false, nil
		}
		// Move past this match
		matchLen := findMatchLengthAt(name, idx, middlePart)
		name = name[idx+matchLen:]
		name = strings.TrimPrefix(name, "/")
	}

	return true, nil
}

// matchesPrefix checks if name starts with a pattern prefix.
func matchesPrefix(name, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return false
	}

	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[i])
		if err != nil || !matched {
			return false
		}
	}
	return true
}

// findPrefixMatchLength returns the length of name consumed by matching the pattern prefix.
func findPrefixMatchLength(name, pattern string) int {
	if pattern == "" {
		return 0
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return -1
	}

	length := 0
	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[i])
		if err != nil || !matched {
			return -1
		}
		if i > 0 {
			length++ // for the /
		}
		length += len(nameParts[i])
	}
	return length
}

// matchesSuffix checks if name ends with a pattern suffix.
func matchesSuffix(name, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return false
	}

	offset := len(nameParts) - len(patternParts)
	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[offset+i])
		if err != nil || !matched {
			return false
		}
	}
	return true
}

// findSuffixMatchLength returns the length of name consumed by matching the pattern suffix.
func findSuffixMatchLength(name, pattern string) int {
	if pattern == "" {
		return 0
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return -1
	}

	offset := len(nameParts) - len(patternParts)
	length := 0
	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[offset+i])
		if err != nil || !matched {
			return -1
		}
		if i > 0 {
			length++ // for the /
		}
		length += len(nameParts[offset+i])
	}
	return length
}

// findPatternInPath finds where a pattern segment matches within the path.
// Returns the byte index or -1 if not found.
func findPatternInPath(name, pattern string) int {
	if pattern == "" {
		return 0
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return -1
	}

	// Try to find pattern parts as a contiguous sequence in name parts
	for startIdx := 0; startIdx <= len(nameParts)-len(patternParts); startIdx++ {
		allMatch := true
		for i, pp := range patternParts {
			matched, err := path.Match(pp, nameParts[startIdx+i])
			if err != nil || !matched {
				allMatch = false
				break
			}
		}
		if allMatch {
			// Calculate byte position
			pos := 0
			for i := 0; i < startIdx; i++ {
				if i > 0 {
					pos++
				}
				pos += len(nameParts[i])
			}
			if startIdx > 0 {
				pos++ // trailing /
			}
			return pos
		}
	}
	return -1
}

// findMatchLengthAt returns the length of the match starting at the given position.
func findMatchLengthAt(name string, startIdx int, pattern string) int {
	remaining := name[startIdx:]
	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(remaining, "/")

	length := 0
	for i := 0; i < len(patternParts) && i < len(nameParts); i++ {
		if i > 0 {
			length++
		}
		length += len(nameParts[i])
	}
	return length
}

func mathAbsInt(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func stripLeadingDots(s string) string {
	return strings.TrimLeft(s, ".")
}

/*
hasDots is a simple helper function that returns whether the given
string contains a '.' character.
*/
func hasDots(s string) bool {
	return strings.Contains(s, ".")
}

/*
dotCount is a simple helper function that returns the number of '.' occurrences in a string.
*/
func dotCount(s string) int {
	return strings.Count(s, ".")
}

/*
warnForBackslashesOnPosix prints a warning if we're running on in a non-Windows environment
and the given string contains backslashes.

This is because backslashes are allowed as a path part in Posix path strings.
However, on Windows they are a path separator. When persisting a path containing backslashes
from a Posix environment using Path.String or Path.ToPosix will cause path traversal errors
when the persisted path is read and used on Windows environments.

Printing the warning can be disabled by setting PrintBackslashWarningOnPosix to false.
*/
func warnForBackslashesOnPosix(p string) {
	if notRunningOnWindows && PrintBackslashWarningOnPosix && strings.Contains(p, "\\") {
		_, _ = os.Stderr.WriteString("Warning: Usage of backslashes in path string on Posix-like environments. This will break the path part structure if used on Windows: " + fmt.Sprintf(`"%s"`, p) + ".\n")
	}
}

/*
charset contains all numbers from 0 to 9 and all letters of the latin alphabet in lower and upper case.
*/
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

/*
generateRandomString generates a random string with a random length.

This is a utility function used by tests and extensions.
*/
func generateRandomString(minLength, maxLength int) string {
	// Generate a random length between minLength and maxLength
	length := rand.IntN(maxLength-minLength+1) + minLength

	// Create a byte slice to store the random string
	result := make([]byte, length)

	// Fill the byte slice with random characters from the charset
	for i := 0; i < length; i++ {
		result[i] = charset[rand.IntN(len(charset))]
	}

	return string(result)
}
