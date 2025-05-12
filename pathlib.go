/*
Package pathlib contains source code for go-pathlib.

It's a one-file library that can be used in other projects by using Go's package system
or by placing the source code file itself into the source tree.

pathlib.go contains lexigraphically based functions and does not interoperate with the
file system. Case sensitivity is defined explicitly. Filesystem-specific functionality is outsourced to pathlib_fs.go.

Use pathlib_fs.go, pathlib_io.go or pathlib_temp.go for more interoperability.
*/
package pathlib

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	// pathCheckNoExist indicates that the checked Path does not exist.
	pathCheckNoExist = iota

	// pathCheckFile indicates that the checked Path is a file.
	pathCheckFile

	// pathCheckDir indicates that the checked Path is a directory.
	pathCheckDir
)

const runningOnWindows = runtime.GOOS == "windows"
const notRunningOnWindows = !runningOnWindows

const osPathSeparator = string(os.PathSeparator)
const canonicalPathSeparator = "/"
const posixPathSeparator = "/"
const windowsPathSeparator = "\\"

// Posix anchor with special character combination
const windowsAnchorCanonicalEncodingPrefix = "/w\\"

// Regexes use forward slash instead of backwards slash, because we assume forward slash input
var windowsVolumeNameRegex = regexp.MustCompile("^[A-Za-z]:(/|$)")
var windowsNetworkPathRegex = regexp.MustCompile("^//[a-zA-Z0-9]+/[a-zA-Z0-9]+")

var multipleCanonicalPathSeparatorsRegex = regexp.MustCompile(`//+`)
var multipleWindowsPathSeparatorsRegex = regexp.MustCompile(`\\+`)

// PrintBackslashWarningOnPosix is a toggle for printing a warning on Posix
// if a path string contains a backslash.
var PrintBackslashWarningOnPosix = true

/*
Path is a struct that represents a filesystem path.

Create a new instance using NewPath().
Other constructor functions are prefixed with 'New'.
*/
type Path struct {

	// The underlying filepath string representation. This is the source of
	// truth. Other functions are relying on the assumption that this
	// value has not been changed between operations.
	path string
}

/*
NewPath is the constructor function for a new Path struct instance.
The passed path string is automatically cleaned and ready for further use.
*/
func NewPath(path string) *Path {
	warnForBackslashesOnPosix(path)

	return &Path{path: normalizePath(path)}
}

/*
NewPathFromWindows applies preprocessing to the passed path string to
ensure a correct internal state and behavior for Windows-styled path strings.
*/
func NewPathFromWindows(path string) *Path {
	warnForBackslashesOnPosix(path)

	return &Path{path: normalizeWindowsPath(path)}
}

/*
NewPathFromOs ensure correct internal state and behavior depending on the current
operating system.

It is meant to be used when handling file paths received by the operating system by
system calls or subprocesses.
*/
func NewPathFromOs(path string) *Path {
	if runningOnWindows {
		return NewPathFromWindows(path)
	}

	return NewPath(path)
}

/*
NewCwd returns a new Path instance pointing to the application's current working directory.

This function uses os.Getwd.
*/
func NewCwd() (*Path, error) {
	cwdPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return NewPath(cwdPath), nil
}

/*
NewHome returns a new Path instance pointing to the user's home directory.

This function uses os.UserHomeDir.
*/
func NewHome() (*Path, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return NewPath(homePath), nil
}

/*
PathFromParts combines passed parts into a new Path.
*/
func PathFromParts(parts ...string) *Path {
	return NewPath(".").JoinStrings(parts...)
}

/*
Parent returns a copy of this Path in the parent directory.

This function uses filepath.Dir.
*/
func (p *Path) Parent() *Path {
	return NewPath(filepath.Dir(p.path))
}

/*
Parts returns all single parts of the Path.
It uses filepath.Separator to split the path string.
*/
func (p *Path) Parts() []string {
	stripped := p.stripEncodings()

	if stripped == "/" {
		return []string{}
	}

	split := strings.Split(stripped, canonicalPathSeparator)

	if len(split) > 0 && split[0] == "" {
		return split[1:]
	}

	return split
}

/*
Split splits this Path into its parent and base.
*/
func (p *Path) Split() (*Path, string) {
	dir, file := filepath.Split(p.stripEncodings())
	return NewPath(dir), file
}

/*
Base returns the last element of this Path.

This function uses filepath.Base.
*/
func (p *Path) Base() string {
	return filepath.Base(p.stripEncodings())
}

/*
Stem returns the base of this Path without all extensions.
*/
func (p *Path) Stem() string {
	completeBase := p.Base()

	if completeBase == "/" {
		return ""
	}

	baseStrippedLeading := stripLeadingDots(completeBase)

	dotIndex := strings.IndexAny(baseStrippedLeading, ".")
	if dotIndex == -1 {
		return completeBase
	}

	return completeBase[:(len(completeBase)-len(baseStrippedLeading))+dotIndex]
}

/*
HasExtensions returns whether this Path has file extensions.
*/
func (p *Path) HasExtensions() bool {
	return hasDots(stripLeadingDots(p.Base()))
}

/*
ExtensionCount returns the number of extensions this Path has.
*/
func (p *Path) ExtensionCount() int {
	return dotCount(stripLeadingDots(p.Base()))
}

/*
Extension returns the complete extension of this Path.
Any prefixed dots are included.

Everything starting from the first non-leading dot in this Path's Stem()
is considered to be an extension.
*/
func (p *Path) Extension() string {
	stem := p.Stem()

	// If no stem exists, then there also are no extensions
	if len(stem) == 0 {
		return ""
	}

	return p.Base()[len(stem):]
}

/*
ExtensionParts returns all this Path's extensions.

See Extension for what is considered an extension.
*/
func (p *Path) ExtensionParts() []string {
	base := stripLeadingDots(p.Base())
	return strings.Split(base, ".")[1:]
}

/*
Anchor returns the first part of the path.

On absolute paths this is the filesystem root ("/").
For Windows paths the raw volume name is returned (e.g. "C:" or "//host/share")

Relative paths don't have a defined anchor, "" is returned.
*/
func (p *Path) Anchor() string {
	if p.hasWindowsAnchorEncodings() {
		stripped := p.stripWindowsAnchorCanonicalPrefix()
		split := strings.SplitN(stripped, canonicalPathSeparator, 2)
		return split[0]
	}

	if p.IsRelative() {
		return ""
	}

	return "/"
}

/*
MatchesPatternE matches this Path against the provided pattern.
Returns whether the matching is successful or any occurring error.

Uses path.Match. Use forward slashes as path separators.

Empty patterns are not allowed.
*/
func (p *Path) MatchesPatternE(pattern string, caseSensitive bool) (bool, error) {
	if pattern == "" {
		return false, errors.New("pattern must not be empty")
	}

	var (
		matched bool
		err     error
	)

	if caseSensitive {
		matched, err = path.Match(pattern, p.stripEncodings())
	} else {
		matched, err = path.Match(strings.ToLower(pattern), strings.ToLower(p.stripEncodings()))
	}

	if err != nil {
		return false, err
	}

	return matched, nil
}

/*
MatchesPattern matches this Path against the provided pattern.
It wraps MatchesPatternE and returns the boolean success return value
or false in case of an error.
*/
func (p *Path) MatchesPattern(pattern string, caseSensitive bool) bool {
	match, err := p.MatchesPatternE(pattern, caseSensitive)
	return match && err == nil
}

/*
IsAbsolute returns whether this Path is absolute.

This function uses filepath.IsAbs.
*/
func (p *Path) IsAbsolute() bool {
	if p.hasWindowsAnchorEncodings() {
		return true
	}

	return filepath.IsAbs(p.stripEncodings())
}

/*
IsRelative returns whether this Path is relative.

This function returns the inverse of IsAbsolute.
*/
func (p *Path) IsRelative() bool {
	return !p.IsAbsolute()
}

/*
RelativeTo returns this Path relative to another.

This function uses filepath.Rel.
*/
func (p *Path) RelativeTo(o *Path) (*Path, error) {
	rp, err := filepath.Rel(o.stripEncodings(), p.stripEncodings())
	return NewPath(rp), err
}

/*
Absolute returns an absolute representation of this Path.
If the Path is relative, it will be joined with the current working directory.
If the Path is already absolute, a copy of the Path is returned.

This function uses filepath.Abs.
*/
func (p *Path) Absolute() (*Path, error) {
	// If filepath.Abs receives an already absolute path representation,
	// it just returns the same representation.
	ap, err := filepath.Abs(p.stripEncodings())
	return NewPath(ap), err
}

/*
AbsoluteTo returns an absolute representation of this Path towards another.

If the Path is relative, it will be joined with the provided Path,
else a copy of this Path is returned.

The other path must be absolute.

Requires the other Path to be absolute.
*/
func (p *Path) AbsoluteTo(o *Path) (*Path, error) {

	// If this path is already absolute, return a copy
	if p.IsAbsolute() {
		return p.Copy(), nil
	}

	if o.IsRelative() {
		return nil, errors.New("other path must be absolute")
	}

	return o.Join(p), nil
}

/*
Join returns a new Path with all passed Path structs joined together.
Paths are not checked whether they are absolute or relative.

Use JoinStrings to join strings with this Path.

This function uses filepath.Join.
*/
func (p *Path) Join(paths ...*Path) *Path {
	pathsStr := make([]string, len(paths))
	for i, localPath := range paths {
		pathsStr[i] = localPath.stripEncodings()
	}

	return NewPath(filepath.Join(append([]string{p.stripEncodings()}, pathsStr...)...))
}

/*
JoinStrings returns a new Path with all passed strings joined together.

This function uses filepath.Join.
*/
func (p *Path) JoinStrings(paths ...string) *Path {
	for _, localPath := range paths {
		warnForBackslashesOnPosix(localPath)
	}

	return NewPath(filepath.Join(append([]string{p.stripEncodings()}, paths...)...))
}

/*
Equals returns whether this and another Path match lexically.
*/
func (p *Path) Equals(other *Path, caseSensitive bool) bool {
	if caseSensitive {
		return p.stripEncodings() == other.stripEncodings()
	}

	return strings.ToLower(p.stripEncodings()) == strings.ToLower(other.stripEncodings())
}

/*
EqualsString returns whether this and the passed string match lexically.
*/
func (p *Path) EqualsString(other string, caseSensitive bool) bool {
	otherCanonical := normalizePath(other)

	if caseSensitive {
		return p.stripEncodings() == otherCanonical
	}

	return strings.ToLower(p.stripEncodings()) == strings.ToLower(otherCanonical)
}

/*
WithName returns this Path but with another base.
*/
func (p *Path) WithName(name string) *Path {
	return p.Parent().JoinStrings(name)
}

/*
Copy creates a copy of this Path.
*/
func (p *Path) Copy() *Path {
	// Suppose the internal state is valid
	return &Path{path: p.path}
}

/*
String returns this Path as a string.
*/
func (p *Path) String() string {
	if runningOnWindows {
		return p.toWindows()
	}

	return p.ToPosix()
}

/*
ToPosix returns a string representation with forward slashes.
*/
func (p *Path) ToPosix() string {
	stripped := p.stripEncodings()
	stripped = multipleCanonicalPathSeparatorsRegex.ReplaceAllString(stripped, canonicalPathSeparator)

	if len(stripped) > 1 {
		stripped = strings.TrimRight(stripped, canonicalPathSeparator)
	}

	return stripped
}

/*
MarshalText marshals this Path's Posix representation into a byte array.
Implements the encoding.TextMarshaler interface.
*/
func (p *Path) MarshalText() (text []byte, err error) {
	return []byte(p.ToPosix()), nil
}

/*
UnmarshalText unmarshalls any byte array into a Path type using the NewPath constructor.
Implements the encoding.TextUnmarshaler interface.
*/
func (p *Path) UnmarshalText(text []byte) error {
	*p = *NewPath(string(text))
	return nil
}

func (p *Path) hasWindowsAnchorEncodings() bool {
	return strings.HasPrefix(p.path, windowsAnchorCanonicalEncodingPrefix)
}

/*
stripEncodings returns the canonical path representation with any path state
information encodings removed.
*/
func (p *Path) stripEncodings() string {
	trimmedPath, hasWindowsAnchorEncoding := strings.CutPrefix(p.path, windowsAnchorCanonicalEncodingPrefix)

	if hasWindowsAnchorEncoding {
		windowsSplit := strings.SplitN(trimmedPath, canonicalPathSeparator, 2)
		newAnchor := strings.ReplaceAll(windowsSplit[0], windowsPathSeparator, canonicalPathSeparator)

		if len(windowsSplit[1]) != 0 {
			newAnchor += canonicalPathSeparator
		}

		return newAnchor + windowsSplit[1]
	}

	return trimmedPath
}

func (p *Path) stripWindowsAnchorCanonicalPrefix() string {
	cutString, _ := strings.CutPrefix(p.path, windowsAnchorCanonicalEncodingPrefix)
	return cutString
}

func (p *Path) toWindows() string {
	if p.hasWindowsAnchorEncodings() {
		// Extract Windows anchor
		windowsSplit := strings.SplitN(strings.TrimPrefix(p.path, windowsAnchorCanonicalEncodingPrefix), canonicalPathSeparator, 2)
		windowsAnchor := windowsSplit[0]
		windowsPath := windowsSplit[1]

		if len(windowsAnchor) != 0 {
			// Replace "\" with "/" in anchor
			windowsAnchor = strings.ReplaceAll(windowsAnchor, windowsPathSeparator, canonicalPathSeparator)
		}

		if len(windowsPath) != 0 {
			// Replace "\" with "/" in anchor
			windowsPath = strings.ReplaceAll(windowsPath, windowsPathSeparator, canonicalPathSeparator)

			// Remove double separators
			windowsPath = multipleCanonicalPathSeparatorsRegex.ReplaceAllString(windowsPath, canonicalPathSeparator)
		}

		// Add path separator if anchor and path don't have a leading/ending slash
		if len(windowsAnchor) != 0 && len(windowsPath) != 0 && windowsPath[0] != '/' && windowsAnchor[len(windowsAnchor)-1] != '/' {
			windowsAnchor += windowsPathSeparator
		}

		pathString := windowsAnchor + windowsPath
		return strings.ReplaceAll(pathString, canonicalPathSeparator, windowsPathSeparator)
	}

	pathString := strings.ReplaceAll(p.stripEncodings(), canonicalPathSeparator, windowsPathSeparator)
	return multipleWindowsPathSeparatorsRegex.ReplaceAllString(pathString, windowsPathSeparator)
}

/*
normalizePath creates a canonical representation of the passed path string.

It abstracts filesystem-specific specialties so that they can be easily compared or
extracted.

This function assumes that the passed path's part separator is "/".

This function ensures:
- Parts can include whitespaces wherever they want (leading, somewhere in between and ending).
- Parts are separated by a single forward slash ("/").
- Multiple forward slashes are replaced by one single slash.
- Trailing forward slashes are removed.

Defined edge cases:
- an empty string, "." and "./" return "."
- an empty string after all filters also returns "."
- ".." returns ".."
- "/", "/..", and "/.." return "/"

The path is not lowercased, because the path might be used on a case-sensitive filesystem.
Functions that are case-insensitive must additionally lowercase this representation.

filepath.Clean is not used because it causes ambiguous behavior for creating a canonical representation.
*/
func normalizePath(p string) string {
	// Check for edge cases.
	// This also ensures that len(p) is always >= 1.
	if p == "" || p == "." || p == "./" {
		return "."
	}

	if p == ".." {
		return ".."
	}

	if p == "/" || p == "/." || p == "/.." {
		return "/"
	}

	// Start cleaning
	dirty := p
	anchor := ""

	// Remove anchor for traversal cleanup. It's readded later.
	if dirty[0] == '/' {
		anchor = canonicalPathSeparator
	}

	hasAnchor := len(anchor) != 0

	var noAnchor string
	if hasAnchor {
		noAnchor = dirty[len(anchor):]
	} else {
		noAnchor = dirty
	}

	// Clean path parts
	var pathParts []string
	splitParts := strings.Split(noAnchor, canonicalPathSeparator)

	// directoryDepth counts the depth within the directory without a possible anchor
	directoryDepth := 0

	// anchorDepth counts the depth of a possible anchor. It will always be >= 0
	// and can lexically not be incremented
	anchorDepth := 0

	for _, part := range splitParts {
		// Ignore zero-length parts or current directory references.
		// This also leads to the removal of multiple and trailing slashes.
		if len(part) == 0 || part == "." {
			continue
		}

		// Parent directory references cause the current depth to be decremented
		if part == ".." {
			if hasAnchor {
				// If the original path had an anchor, cap the minimum depth at 0.
				directoryDepth = mathMaxInt(directoryDepth-1, 0)
			} else if len(pathParts) != 0 {
				// If there are enumerated parts, decrement the directory depth
				directoryDepth--
			} else {
				// Decrement the anchor
				anchorDepth--
			}

			continue
		}

		// Strip path parts that are not relevant anymore through a previous parent directory reference
		if len(pathParts) > directoryDepth {
			pathParts = append(pathParts[:mathMaxInt(directoryDepth, 0)], part)

			// Reset directory depth as it has been applied to the parts.
			// If directory depth was negative, add them to the anchor depth.
			if directoryDepth < 0 {
				anchorDepth = anchorDepth + directoryDepth
			}

			// Set directory depth to 0
			directoryDepth = 0
		} else {
			pathParts = append(pathParts, part)
		}

		// Increment the current directory depth, because we added a new part to the path
		directoryDepth++
	}

	// Strip path parts in case of directory depth changes after a first part addition
	if len(pathParts) > directoryDepth {
		pathParts = pathParts[:mathMaxInt(directoryDepth, 0)]
	}

	// Recombine cleaned path parts
	cleanPathNoAnchor := strings.Join(pathParts, canonicalPathSeparator)

	if !hasAnchor {
		// If the original did not have an anchor and the directory depth is below zero,
		// set anchor to (directory depth * parent directory indicator)
		repeatString := ".." + canonicalPathSeparator

		finalDepth := mathAbsInt(anchorDepth)

		if directoryDepth < 0 {
			finalDepth = finalDepth + mathAbsInt(directoryDepth)
		}

		if finalDepth > 0 {
			anchor = strings.Repeat(repeatString, finalDepth)
		}
	}

	// Right-trim path separators if the cleaned path is empty and
	// the anchor is not a single path separator
	if cleanPathNoAnchor == "" && len(anchor) > 1 {
		anchor = strings.TrimRight(anchor, canonicalPathSeparator)
	}

	cleanPath := anchor + cleanPathNoAnchor

	if cleanPath == "" {
		cleanPath = "."
	}

	return cleanPath
}

/*
normalizedWindowsPath is a Windows path style-specific wrapper function for normalizePath.

It replaces all "\\" with "/".

If a Windows volume or network drive naming pattern is detected, backward slashes
until the first valid path separator are not replaced and prefixed with "/w\\", as this pattern
won't exist because of previous filters.
*/
func normalizeWindowsPath(p string) string {
	dirty := p

	dirty = strings.ReplaceAll(dirty, windowsPathSeparator, canonicalPathSeparator)

	var (
		hasWindowsAnchor        = false
		windowsAnchor           = ""
		normalizedWindowsAnchor = ""
	)

	// Check for Windows volume name anchor
	windowsAnchor = windowsVolumeNameRegex.FindString(dirty)

	if len(windowsAnchor) == 0 {
		// Check for Windows network path anchor
		windowsAnchor = windowsNetworkPathRegex.FindString(dirty)
	}

	if len(windowsAnchor) == 0 {
		// A leading path separator is also considered a valid anchor, as it points
		// to the root of the current volume.
		if strings.HasPrefix(dirty, canonicalPathSeparator) {
			windowsAnchor = canonicalPathSeparator
		}
	}

	hasWindowsAnchor = len(windowsAnchor) != 0

	if hasWindowsAnchor {
		// Normalize anchor. Replaces slashes with backslashes to prevent issues in
		// lexical parsing later on. Also prepend Windows anchor encoding prefix.
		normalizedWindowsAnchor = windowsAnchorCanonicalEncodingPrefix + strings.ReplaceAll(windowsAnchor, canonicalPathSeparator, windowsPathSeparator)
	}

	if !hasWindowsAnchor {
		// Remove leading path separators of no anchor exists
		dirty = strings.TrimLeft(dirty, canonicalPathSeparator)
	}

	// Perform regular normalization without the anchor
	dirtyNoAnchor := dirty[len(windowsAnchor):]
	cleanedNoAnchor := normalizePath(dirtyNoAnchor)

	// Prepend anchor to the cleaned path
	if hasWindowsAnchor {
		// Remove any parent directory indicators
		cleanedNoAnchor = strings.TrimLeft(cleanedNoAnchor, "."+canonicalPathSeparator)

		// normalized anchor always has appended "/"
		normalizedWindowsAnchor += canonicalPathSeparator
	}

	return normalizedWindowsAnchor + cleanedNoAnchor
}

func mathMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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

It does not use strings.Contains for overhead and complexity reasons.
*/
func hasDots(s string) bool {
	for _, v := range s {
		if v == '.' {
			return true
		}
	}

	return false
}

/*
dotCount is a simple helper function that returns the number of '.' occurrences in a string.

It does not use strings.Count for complexity reasons.
*/
func dotCount(s string) int {
	switch len(s) {
	case 0:
		return 0
	case 1:
		if s[0] == '.' {
			return 1
		}
		return 0
	}

	dotAscii := '.'
	count := 0

	for _, v := range s {
		if v == dotAscii {
			count += 1
		}
	}

	return count
}

/*
flipCase is a utility function that takes the first character and flips it's case.
The leftover characters are appended.
This results in a string which is different from the original which can be used
for e.g. case sensitivity (in)variance checks.
*/
func flipCase(s string) string {
	if s == "" {
		return s
	}
	firstChar := string(s[0])
	if strings.ToLower(firstChar) == firstChar {
		return strings.ToUpper(firstChar) + s[1:]
	}
	return strings.ToLower(firstChar) + s[1:]
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
	length := rand.Intn(maxLength-minLength+1) + minLength

	// Create a byte slice to store the random string
	result := make([]byte, length)

	// Fill the byte slice with random characters from the charset
	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}
