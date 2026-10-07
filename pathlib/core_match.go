// core_match.go holds the matching of paths against patterns with support for
// "**".

package pathlib

import (
	"path"
	"strings"
)

// MatchesPatternE reports whether the Posix form of this Path matches pattern.
// It wraps [path.Match] and adds "**", which matches any number of names.
// Patterns separate names with forward slashes. The match is case-sensitive,
// unless opts holds [CaseInsensitive].
//
// An empty pattern returns [ErrEmptyPattern], and a malformed pattern returns
// [ErrBadPattern].
func (p *Path) MatchesPatternE(pattern string, opts ...CompareOption) (bool, error) {
	if pattern == "" {
		return false, patternErr(ErrEmptyPattern, pattern, nil, *p)
	}

	matchedPattern := pattern
	pathString := p.ToPosix()

	if ignoresCase(opts) {
		matchedPattern = strings.ToLower(pattern)
		pathString = strings.ToLower(pathString)
	}

	match, err := matchPattern(matchedPattern, pathString)
	if err != nil {
		return false, patternErr(ErrBadPattern, pattern, err, *p)
	}

	return match, nil
}

// MatchesPattern reports whether this Path matches pattern, as
// [Path.MatchesPatternE] does. It returns false on an error.
func (p *Path) MatchesPattern(pattern string, opts ...CompareOption) bool {
	match, err := p.MatchesPatternE(pattern, opts...)
	return match && err == nil
}

// matchPattern reports whether name matches pattern, which may contain "**".
// A malformed pattern returns the error of [path.Match].
func matchPattern(pattern, name string) (bool, error) {
	err := validatePattern(pattern)
	if err != nil {
		return false, err
	}

	if !strings.Contains(pattern, "**") {
		return path.Match(pattern, name)
	}

	return matchWithDoubleAsterisk(pattern, name)
}

// validatePattern returns the error of [path.Match] for a malformed segment of
// pattern. The segments are the parts between "**".
func validatePattern(pattern string) error {
	segments := strings.Split(pattern, "**")
	for _, seg := range segments {
		seg = strings.Trim(seg, "/")
		if seg == "" {
			continue
		}

		// path.Match checks the syntax of the whole pattern, whatever the name.
		_, err := path.Match(seg, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// matchWithDoubleAsterisk reports whether name matches pattern, which contains
// "**".
func matchWithDoubleAsterisk(pattern, name string) (bool, error) {
	if pattern == "**" {
		return true, nil
	}

	return matchParts(strings.Split(pattern, "**"), name)
}

// matchParts reports whether name matches the parts of a pattern split at
// "**". The first part matches the start of name, the last part matches its
// end, and the parts between match in order in between.
func matchParts(parts []string, name string) (bool, error) {
	firstPart := parts[0]
	if firstPart != "" {
		firstPart = strings.TrimSuffix(firstPart, "/")
		if !matchesPrefix(name, firstPart) {
			return false, nil
		}
		prefixLen := findPrefixMatchLength(name, firstPart)
		if prefixLen == -1 {
			return false, nil
		}
		name = name[prefixLen:]
		name = strings.TrimPrefix(name, "/")
	}

	lastPart := parts[len(parts)-1]
	if lastPart != "" {
		lastPart = strings.TrimPrefix(lastPart, "/")
		if !matchesSuffix(name, lastPart) {
			return false, nil
		}
		suffixLen := findSuffixMatchLength(name, lastPart)
		if suffixLen == -1 {
			return false, nil
		}
		name = name[:len(name)-suffixLen]
		name = strings.TrimSuffix(name, "/")
	}

	for i := 1; i < len(parts)-1; i++ {
		middlePart := strings.Trim(parts[i], "/")
		if middlePart == "" {
			continue
		}

		idx := findPatternInPath(name, middlePart)
		if idx == -1 {
			return false, nil
		}
		matchLen := findMatchLengthAt(name, idx, middlePart)
		name = name[idx+matchLen:]
		name = strings.TrimPrefix(name, "/")
	}

	return true, nil
}

// matchesPrefix reports whether name starts with names matching pattern.
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

// findPrefixMatchLength returns the length of the start of name that matches
// pattern, or -1 if it does not match.
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

// matchesSuffix reports whether name ends with names matching pattern.
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

// findSuffixMatchLength returns the length of the end of name that matches
// pattern, or -1 if it does not match.
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

// findPatternInPath returns the byte index in name of the first run of names
// matching pattern, or -1 if there is none.
func findPatternInPath(name, pattern string) int {
	if pattern == "" {
		return 0
	}

	patternParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	if len(nameParts) < len(patternParts) {
		return -1
	}

	for startIdx := 0; startIdx <= len(nameParts)-len(patternParts); startIdx++ {
		if !partsMatchAt(nameParts, patternParts, startIdx) {
			continue
		}
		if startIdx == 0 {
			return 0
		}
		// The match starts after the preceding parts and their trailing /
		return len(strings.Join(nameParts[:startIdx], "/")) + 1
	}
	return -1
}

// partsMatchAt reports whether each pattern part matches the name part at the
// same offset from startIdx.
func partsMatchAt(nameParts, patternParts []string, startIdx int) bool {
	for i, pp := range patternParts {
		matched, err := path.Match(pp, nameParts[startIdx+i])
		if err != nil || !matched {
			return false
		}
	}
	return true
}

// findMatchLengthAt returns the length of the names of name that start at
// startIdx and match pattern, which has matched there.
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
