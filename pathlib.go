/*
Package pathlib contains source code for go-pathlib.

It's a one-file library that can be used in other projects by using Go's package system
or by placing the source code file itself into the source tree.

pathlib.go contains lexicographically based functions and does not interoperate with the
file system. Case sensitivity is defined explicitly. Filesystem-specific functionality is outsourced to pathlib_fs.go.

Use pathlib_fs.go, pathlib_io.go or pathlib_temp.go for more interoperability.
*/
package pathlib

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

type CompareOption bool

const (
	CaseSensitive   CompareOption = true
	CaseInsensitive CompareOption = false
)

const (
	// pathCheckNoExistOrUnreadable indicates that the checked Path does not exist.
	pathCheckNoExistOrUnreadable = iota

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

// Regexes use forward slash instead of backwards slash, because we assume forward slash input
var windowsVolumeNameRegex = regexp.MustCompile("^[A-Za-z]:(/|$)")
var windowsNetworkPathRegex = regexp.MustCompile("^//[a-zA-Z0-9]+/[a-zA-Z0-9]+")

var multipleCanonicalPathSeparatorsRegex = regexp.MustCompile(`//+`)
var multipleWindowsPathSeparatorsRegex = regexp.MustCompile(`\\+`)

// PrintBackslashWarningOnPosix is a toggle for printing a warning on Posix
// if a path string contains a backslash.
var PrintBackslashWarningOnPosix = true

// Windows path state bitmask values for windowsPathEncodings.
const (
	// windowsPathStateVolume indicates a Windows path anchored to a drive volume (e.g. "C:\foo").
	windowsPathStateVolume uint8 = 1 << iota

	// windowsPathStateUNC indicates a Windows UNC path (e.g. "\\server\share\foo").
	windowsPathStateUNC
)

/*
Path is a struct that represents a filesystem path.

Create a new instance using NewPath().
Other constructor functions are prefixed with 'New'.

Implements the fmt.Stringer interface.
*/
type Path struct {

	// The underlying filepath string representation in canonical (unix-style) form.
	// For Windows paths this holds only the portion after the anchor.
	path string

	// windowsAnchor holds the Windows-specific anchor in canonical (forward-slash) form.
	// "C:"           for volume paths  (windowsPathStateVolume)
	// "//host/share" for UNC paths     (windowsPathStateUNC)
	// ""             for all others
	windowsAnchor string

	// windowsPathEncodings is a bitmask of windowsPathState* flags.
	windowsPathEncodings uint8
}

/*
NewPath is the constructor function for a new Path struct instance.

The passed path string is automatically cleaned and ready for further use using the following rules:
- Parts can include whitespaces wherever they want (leading, somewhere in between and ending).
- Parts are separated by a single forward slash ("/").
- Multiple forward slashes are replaced by one single slash.
- Trailing forward slashes are removed.

Defined edge cases:
- an empty string, "." and "./" results into "."
- if all rules result into an empty string, the path also result into "."
- ".." stays ".."
- "/", "/.", and "/.." result into "/"

The path is not lowercased, because the path might be used on a case-sensitive filesystem.
Functions that are case-insensitive must additionally lowercase this representation.
*/
func NewPath(path string) *Path {
	warnForBackslashesOnPosix(path)

	return &Path{path: normalizePath(path)}
}

/*
NewPathFromWindows applies preprocessing to the passed path string to
ensure a correct internal state and behavior for Windows-styled path strings.

The same normalization rules as NewPath apply, with additional handling for
Windows volume names (e.g. "C:") and UNC paths (e.g. "\\\\host\\share").
*/
func NewPathFromWindows(path string) *Path {
	warnForBackslashesOnPosix(path)
	return normalizeWindowsPath(path)
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

This function uses path.Dir.
*/
func (p *Path) Parent() *Path {
	return NewPath(path.Dir(p.path))
}

/*
Parts returns all single parts of the Path.
*/
func (p *Path) Parts() []string {
	localPath := p.path

	if localPath == "/" {
		return []string{}
	}

	split := strings.Split(localPath, canonicalPathSeparator)

	if len(split) > 0 && split[0] == "" {
		split = split[1:]
	}

	if p.isWindowsVolumeAnchoredPath() {
		split = slices.Insert(split, 0, p.windowsAnchor)
	}

	return split
}

/*
Split splits this Path into its parent and base.
*/
func (p *Path) Split() (*Path, string) {
	dir, file := path.Split(p.pathWithWindowsAnchor())
	return NewPath(dir), file
}

/*
Base returns the last element of this Path.

This function uses path.Base.
*/
func (p *Path) Base() string {
	if p.isWindowsAnchoredPath() && p.isLocalDirectory() {
		return p.windowsAnchor
	}

	return path.Base(p.path)
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
	if p.isWindowsAnchoredPath() {
		return strings.ReplaceAll(p.windowsAnchor, canonicalPathSeparator, windowsPathSeparator)
	}

	if p.IsRelative() {
		return ""
	}

	return "/"
}

/*
MatchesPatternE matches this Path's Posix representation against a pattern with support for double asterisk (**).
Returns whether the matching is successful or any occurring error.

Wraps path.Match with some custom rules for double asterisk support.
Use forward slashes as path separators.

By default, matching is case-sensitive. Pass CaseInsensitive to ignore casing.

Empty patterns cause an error.
*/
func (p *Path) MatchesPatternE(pattern string, opts ...CompareOption) (bool, error) {
	if pattern == "" {
		return false, errors.New("pattern may not be empty")
	}

	caseSensitive := true
	if len(opts) > 0 {
		caseSensitive = bool(opts[0])
	}

	pathString := p.ToPosix()

	if !caseSensitive {
		pattern = strings.ToLower(pattern)
		pathString = strings.ToLower(pathString)
	}

	return matchPattern(pattern, pathString)
}

/*
MatchesPattern matches this Path against the provided pattern.

It wraps MatchesPatternE and returns the boolean success return value or false in case of an error.
*/
func (p *Path) MatchesPattern(pattern string, opts ...CompareOption) bool {
	match, err := p.MatchesPatternE(pattern, opts...)
	return match && err == nil
}

/*
IsAbsolute returns whether this Path is absolute.

This function uses path.IsAbs.
*/
func (p *Path) IsAbsolute() bool {
	if p.isWindowsUNCAnchoredPath() {
		return true
	}

	return path.IsAbs(p.path)
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

On Windows, paths cannot be made relative across different anchors.
This function does not enforce it and selects this Path's anchor.
*/
func (p *Path) RelativeTo(o *Path) (*Path, error) {
	rp, err := relPath(o.path, p.path)

	if err != nil {
		return nil, err
	}

	newPath := p.Copy()
	newPath.path = rp

	return newPath, nil
}

/*
Absolute returns an absolute representation of this Path.
If the Path is relative, it will be joined with the current working directory.
If the Path is already absolute, a copy of the Path is returned.
*/
func (p *Path) Absolute() (*Path, error) {
	// If already absolute, return a copy
	if p.IsAbsolute() {
		return p.Copy(), nil
	}

	cwd, err := NewCwd()
	if err != nil {
		return nil, err
	}

	return cwd.Join(p), nil
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

This function uses path.Join.
*/
func (p *Path) Join(paths ...*Path) *Path {
	pathsStr := make([]string, len(paths))
	for i, localPath := range paths {
		pathsStr[i] = localPath.path
	}

	newPath := p.Copy()
	newPath.path = path.Join(append([]string{p.path}, pathsStr...)...)

	return newPath
}

/*
JoinStrings returns a new Path with all passed strings joined together.
*/
func (p *Path) JoinStrings(paths ...string) *Path {
	for _, localPath := range paths {
		warnForBackslashesOnPosix(localPath)
	}

	newPath := p.Copy()
	newPath.path = path.Join(append([]string{p.path}, paths...)...)

	return newPath
}

/*
Equals returns whether this and another Path match lexically.
By default, comparison is case-sensitive.
*/
func (p *Path) Equals(other *Path, opts ...CompareOption) bool {
	caseSensitive := CaseSensitive
	if len(opts) > 0 {
		caseSensitive = opts[0]
	}

	if caseSensitive {
		return p.pathWithWindowsAnchor() == other.pathWithWindowsAnchor()
	}

	return strings.EqualFold(p.pathWithWindowsAnchor(), other.pathWithWindowsAnchor())
}

/*
EqualsString returns whether this and the passed string match lexically.
By default, comparison is case-sensitive.
*/
func (p *Path) EqualsString(other string, opts ...CompareOption) bool {
	caseSensitive := true
	if len(opts) > 0 {
		caseSensitive = bool(opts[0])
	}

	otherCanonical := normalizePath(other)

	if caseSensitive {
		return p.pathWithWindowsAnchor() == otherCanonical
	}

	return strings.EqualFold(p.pathWithWindowsAnchor(), otherCanonical)
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
	return &Path{
		path:                 p.path,
		windowsAnchor:        p.windowsAnchor,
		windowsPathEncodings: p.windowsPathEncodings,
	}
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
	pathStr := p.pathWithWindowsAnchor()
	pathStr = multipleCanonicalPathSeparatorsRegex.ReplaceAllString(pathStr, canonicalPathSeparator)

	if len(pathStr) > 1 {
		pathStr = strings.TrimRight(pathStr, canonicalPathSeparator)
	}

	return pathStr
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

func (p *Path) pathWithWindowsAnchor() string {
	if p.isWindowsAnchoredPath() {
		return p.windowsAnchor + p.path
	}

	return p.path
}

// isWindowsAnchoredPath reports whether the path was constructed from a Windows-style path string.
func (p *Path) isWindowsAnchoredPath() bool {
	return p.windowsPathEncodings != 0
}

// isWindows reports whether the path is a Windows UNC path (e.g. "\\server\share").
func (p *Path) isWindowsVolumeAnchoredPath() bool {
	return p.windowsPathEncodings&windowsPathStateVolume != 0
}

// isWindowsUNCAnchoredPath reports whether the path is a Windows UNC path (e.g. "\\server\share").
func (p *Path) isWindowsUNCAnchoredPath() bool {
	return p.windowsPathEncodings&windowsPathStateUNC != 0
}

func (p *Path) toWindows() string {
	if p.isWindowsUNCAnchoredPath() {
		anchorWindows := strings.ReplaceAll(p.windowsAnchor, canonicalPathSeparator, windowsPathSeparator)
		if p.path == canonicalPathSeparator {
			return anchorWindows
		}
		return anchorWindows + strings.ReplaceAll(p.path, canonicalPathSeparator, windowsPathSeparator)
	}

	fullPath := p.windowsAnchor + p.path
	pathString := strings.ReplaceAll(fullPath, canonicalPathSeparator, windowsPathSeparator)
	return multipleWindowsPathSeparatorsRegex.ReplaceAllString(pathString, windowsPathSeparator)
}

/*
normalizePath creates a canonical representation of the passed path string.

It abstracts filesystem-specific specialties so that they can be easily compared or
extracted.

This function assumes that the passed path's part separator is "/" and uses path.Clean.

This function ensures:
- Parts can include whitespaces wherever they want (leading, somewhere in between and ending).
- Parts are separated by a single forward slash ("/").
- Multiple forward slashes are replaced by one single slash.
- Trailing forward slashes are removed.

Defined edge cases:
- an empty string, "." and "./" return "."
- an empty string after all filters also returns "."
- ".." returns ".."
- "/", "/.", and "/.." return "/"

The path is not lowercased, because the path might be used on a case-sensitive filesystem.
Functions that are case-insensitive must additionally lowercase this representation.
*/
func normalizePath(p string) string {
	return path.Clean(p)
}

// normalizeWindowsPath processes a Windows-style path string into a normalized Path.
// It handles volume paths (e.g. "C:\foo"), UNC paths (e.g. "\\host\share\foo"),
// and relative/absolute paths without a Windows-specific anchor.
func normalizeWindowsPath(p string) *Path {
	dirty := strings.ReplaceAll(p, windowsPathSeparator, canonicalPathSeparator)

	// Volume anchor: "C:\" (rooted) or "C:" (drive-relative).
	if match := windowsVolumeNameRegex.FindString(dirty); match != "" {
		anchor := match[:2]        // "C:" (letter + colon)
		hasRoot := len(match) == 3 // true when match includes the trailing "/"

		rest := dirty[len(match):]
		cleanedRest := normalizePath(rest)

		// Strip any leading slashes that normalizePath may have returned.
		cleanedRest = strings.TrimLeft(cleanedRest, canonicalPathSeparator)

		// Remove ".." traversals that cannot go above the drive root.
		for {
			if cleanedRest == ".." {
				cleanedRest = ""
				break
			}
			if strings.HasPrefix(cleanedRest, "../") {
				cleanedRest = cleanedRest[3:]
				continue
			}
			if cleanedRest == "." {
				cleanedRest = ""
				break
			}
			break
		}

		var pathField string
		if hasRoot {
			if cleanedRest == "" {
				pathField = canonicalPathSeparator
			} else {
				pathField = canonicalPathSeparator + cleanedRest
			}
		} else {
			pathField = cleanedRest
		}

		return &Path{
			path:                 pathField,
			windowsAnchor:        anchor,
			windowsPathEncodings: windowsPathStateVolume,
		}
	}

	// UNC anchor: "//host/share".
	if match := windowsNetworkPathRegex.FindString(dirty); match != "" {
		rest := dirty[len(match):]
		if rest == "" || rest == canonicalPathSeparator {
			return &Path{
				path:                 canonicalPathSeparator,
				windowsAnchor:        match,
				windowsPathEncodings: windowsPathStateUNC,
			}
		}
		cleanedRest := normalizePath(rest)
		if cleanedRest == "." {
			cleanedRest = canonicalPathSeparator
		}
		return &Path{
			path:                 cleanedRest,
			windowsAnchor:        match,
			windowsPathEncodings: windowsPathStateUNC,
		}
	}

	// No Windows-specific anchor: treat as a regular path.
	return &Path{
		path:                 normalizePath(dirty),
		windowsPathEncodings: 0,
	}
}

// isLocalDirectory returns whether the current path is "."
func (p *Path) isLocalDirectory() bool {
	return p.path == "."
}

// relPath returns a relative path that is lexically equivalent to targPath when
// joined to basePath with an intervening separator. Both paths must use forward
// slashes and be already normalized.
//
// This function is inspired by filepath.Rel.
func relPath(basePath, targPath string) (string, error) {
	if basePath == targPath {
		return ".", nil
	}

	base := basePath
	targ := targPath

	if base == "." {
		base = ""
	}
	if targ == "." {
		targ = ""
	}

	baseSlashed := len(base) > 0 && base[0] == '/'
	targSlashed := len(targ) > 0 && targ[0] == '/'
	if baseSlashed != targSlashed {
		return "", errors.New("Rel: can't make " + targPath + " relative to " + basePath)
	}

	bl := len(base)
	tl := len(targ)
	var b0, bi, t0, ti int
	for {
		for bi < bl && base[bi] != '/' {
			bi++
		}
		for ti < tl && targ[ti] != '/' {
			ti++
		}
		if base[b0:bi] != targ[t0:ti] {
			break
		}
		if bi < bl {
			bi++
		}
		if ti < tl {
			ti++
		}
		b0 = bi
		t0 = ti
	}

	if base[b0:bi] == ".." {
		return "", errors.New("Rel: can't make " + targPath + " relative to " + basePath)
	}

	if b0 != bl {
		seps := strings.Count(base[b0:bl], "/")
		size := 2 + seps*3
		if tl != t0 {
			size += 1 + tl - t0
		}
		buf := make([]byte, size)
		n := copy(buf, "..")
		for i := 0; i < seps; i++ {
			buf[n] = '/'
			copy(buf[n+1:], "..")
			n += 3
		}
		if t0 != tl {
			buf[n] = '/'
			copy(buf[n+1:], targ[t0:])
		}
		return string(buf), nil
	}

	result := targ[t0:]
	if result == "" {
		return ".", nil
	}
	return result, nil
}

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
