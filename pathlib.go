// Package pathlib contains every functionality for go-pathlib.
// It's a one-file library that can be used in other projects by using Go's package system
// or by placing the source code file itself into the source tree.
package pathlib

import (
	"errors"
	"os"
	"path/filepath"
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

// pathSeparator is the string representation of filepath.Separator
const pathSeparator = string(filepath.Separator)

/*
Path is a struct that represents a filesystem path.

Create a new instance using NewPath().
Other constructor functions are prefixed with 'New'.
*/
type Path struct {

	// The underlying filepath string representation. This is the source of
	// truth and other functions are relying on the assumption that this
	// value has not been changed between operations.
	path string
}

/*
NewPath is the constructor function for a new Path struct instance.
The passed path string is automatically cleaned and ready for further use.
*/
func NewPath(path string) *Path {
	return &Path{path: cleanPathString(path)}
}

/*
NewCwd returns a new Path instance pointing to the application's current working directory.

This function utilizes os.Getwd.
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

This function utilizes os.UserHomeDir.
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
IsFile returns whether this Path is an existing file.
*/
func (p *Path) IsFile() bool {
	return pathCheck(*p) == pathCheckFile
}

/*
IsDir returns whether this Path is an existing directory.
*/
func (p *Path) IsDir() bool {
	return pathCheck(*p) == pathCheckDir
}

/*
Exists returns whether this Path exists.
*/
func (p *Path) Exists() bool {
	return pathCheck(*p) != pathCheckNoExist
}

/*
Parent returns a copy of this Path in the parent directory.

This function utilizes filepath.Dir.
*/
func (p *Path) Parent() *Path {
	return NewPath(filepath.Dir(p.path))
}

/*
Parts returns all single parts of the Path.
It uses filepath.Separator to split the path string.
*/
func (p *Path) Parts() []string {
	separator := pathSeparator

	toSplit := strings.Trim(p.path, separator)
	if toSplit == "" {
		return []string{}
	}

	return strings.Split(toSplit, separator)
}

/*
Split splits this Path into its parent and base.
*/
func (p *Path) Split() (*Path, string) {
	dir, file := filepath.Split(p.path)
	return NewPath(dir), file
}

/*
Base returns the last element of this Path.

This function utilizes filepath.Base.
*/
func (p *Path) Base() string {
	return filepath.Base(p.path)
}

/*
Extension returns the last filename extension of this Path.
The prefixed dot is included.
*/
func (p *Path) Extension() string {

	base := p.Base()

	if base == "." || base == ".." || base == pathSeparator {
		return ""
	}

	base = strings.TrimLeft(base, ".")
	extension := filepath.Ext(base)

	return extension
}

/*
Extensions returns all the Path's extensions.
Prefixed dots are included.

If the file starts with a '.' (which is a common on unix
based operating systems), the first part is ignored.
*/
func (p *Path) Extensions() []string {

	base := p.Base()
	base = strings.Trim(base, ".")
	base = strings.Trim(base, pathSeparator)

	extensions := strings.Split(base, ".")
	if len(extensions) > 0 {
		extensions = extensions[1:]
	}

	for ext := range extensions {
		extensions[ext] = "." + extensions[ext]
	}

	return extensions
}

/*
Stem returns the last element of this Path without the extension.
*/
func (p *Path) Stem() string {
	base := p.Base()

	// stem definitions
	if base == "." || base == pathSeparator {
		return ""
	}

	if base == ".." {
		return ".."
	}

	// in case stem has

	return base[:len(base)-len(p.Extension())]
}

/*
MinimalStem returns the last element of this Path without all extensions.
*/
func (p *Path) MinimalStem() string {
	base := p.Base()

	if base == "." || base == pathSeparator {
		return ""
	}

	return base[:len(base)-len(strings.Join(p.Extensions(), ""))]
}

/*
Root returns the first part of the path.
On absolute paths this is the filesystem root, on relative paths all parts
up to the first non-'..' part are included.

On Unix-based operating systems, the Windows path root (e.g. 'C:\')
is not considered a filepath root. However, it will be returned as a root
because 'C:\' or 'C:/' is seen as the root of a relative path.
*/
func (p *Path) Root() string {

	if p.IsRelative() {
		parts := p.Parts()
		var rootParts []string

		for _, part := range parts {
			rootParts = append(rootParts, part)
			if part != ".." {
				break
			}
		}

		return filepath.Join(rootParts...)
	}

	pathStr := p.path

	if pathStr == pathSeparator {
		return pathStr
	}

	root := strings.SplitN(p.path, pathSeparator, 2)[0]

	if root == "" {
		return pathSeparator
	}

	return root
}

/*
IsAbsolute returns whether this Path is absolute.

On non-Windows operating systems, the Windows path root (e.g. 'C:\')
is not considered a file root but as a regular (relative) path element.
Thus, this function would return false.

This function utilizes filepath.IsAbs.
*/
func (p *Path) IsAbsolute() bool {
	return filepath.IsAbs(p.path)
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

This function utilizes filepath.Rel.
*/
func (p *Path) RelativeTo(o *Path) (*Path, error) {
	rp, err := filepath.Rel(o.path, p.path)
	return NewPath(rp), err
}

/*
Absolute returns an absolute representation of this Path.
If the Path is relative, it will be joined with the current working directory.

This function utilizes filepath.Abs.
*/
func (p *Path) Absolute() (*Path, error) {
	ap, err := filepath.Abs(p.path)
	return NewPath(ap), err
}

/*
AbsoluteTo returns an absolute representation of this Path towards another.
If the Path is relative, it will be joined with the provided Path, else this Path is returned.

Requires the other Path to be absolute.
*/
func (p *Path) AbsoluteTo(o *Path) (*Path, error) {

	// if path is already absolute, ignore it
	if p.IsAbsolute() {
		return p, nil
	}

	if o.IsRelative() {
		return nil, errors.New("other path must be absolute")
	}

	return o.Join(p), nil
}

/*
Resolve resolves all symbolic links. If this Path is relative,
the result will be relative to the current directory, unless
one of the components is an absolute symbolic link.

Resolve requires this Path to exist.

This function utilizes filepath.EvalSymlinks.
*/
func (p *Path) Resolve() (*Path, error) {
	if !p.Exists() {
		return nil, errors.New("this path does not exist")
	}

	ep, err := filepath.EvalSymlinks(p.path)
	if err != nil {
		return nil, err
	}

	return NewPath(ep), nil
}

/*
Join returns a new Path with all passed Path structs joined together.
Use JoinStrings to join strings with this Path.

This function utilizes filepath.Join.
*/
func (p *Path) Join(paths ...*Path) *Path {
	pathsStr := make([]string, len(paths))
	for i, path := range paths {
		pathsStr[i] = path.path
	}

	return NewPath(filepath.Join(append([]string{p.path}, pathsStr...)...))
}

/*
JoinStrings returns a new Path with all passed strings joined together.

This function utilizes filepath.Join.
*/
func (p *Path) JoinStrings(paths ...string) *Path {
	return NewPath(filepath.Join(append([]string{p.path}, paths...)...))
}

/*
Glob returns all paths matching the given pattern within this Path's directory.

This function utilizes filepath.Glob. It ignores IO errors.
*/
func (p *Path) Glob(pattern string) ([]*Path, error) {
	matches, err := nativeGlob(p, pattern)
	if err != nil {
		return nil, err
	}

	paths := make([]*Path, len(matches))
	for idx, match := range matches {
		paths[idx] = NewPath(match)
	}

	return paths, nil
}

/*
Contains returns whether the passed pattern exist within this Path's directory.

This function utilizes filepath.Glob.
*/
func (p *Path) Contains(pattern string) (bool, error) {
	matches, err := nativeGlob(p, pattern)
	if err != nil {
		return false, err
	}

	return len(matches) != 0, nil
}

/*
BContains returns whether the passed pattern exists within this Path's directory.
It wraps Contains and returns the boolean success value.
*/
func (p *Path) BContains(pattern string) bool {
	contains, _ := p.Contains(pattern)
	return contains
}

/*
IsCaseSensitiveFs returns whether a given path is on a case-sensitive filesystem.

Currently, this function checks the sensitivity using the path's base.
*/
func IsCaseSensitiveFs(p *Path) (bool, error) {
	// IMPORTANT:
	// It would make sense to check if this Path actually exists before
	// continuing the check. But this does not make sense in the context
	// of this function's goal.

	// TODO Check sensitivity of parent parts --> underlying fs may have mounted
	// 	multiple filesystems that are switching sensitivity.

	alt := p.Parent()
	alt = alt.JoinStrings(flipCase(p.Base()))

	// get file stat for passed file (required for later comparison in os.SameFile)
	pathInfo, err := os.Stat(p.path)
	if err != nil {
		return false, err
	}

	// if file does not exist, assume to be on case-sensitive filesystem
	altInfo, err := os.Stat(alt.path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}

		return false, err
	}

	// if both file exist, check if they are the same
	// if they are equal, then the filesystem is case-insensitive,
	// else the filesystem is case-sensitive
	return !os.SameFile(pathInfo, altInfo), nil
}

/*
Equals returns whether this and another Path are structurally the same.
It respects case sensitivity.
*/
func (p *Path) Equals(other *Path) bool {
	return p.path == other.path
}

/*
EqualsString returns whether the passed string matches this Path.

This function converts the passed string to a Path object and calls Equals.
*/
func (p *Path) EqualsString(other string) bool {
	return p.path == cleanPathString(other)
}

/*
EqualsCi returns whether this and another Path are structurally the same.
It ignores case sensitivity.
*/
func (p *Path) EqualsCi(other *Path) bool {
	return equalsStringCaseInsensitive(p.String(), other.String())
}

/*
EqualsStringCi returns whether the passed string matches this Path.
it ignores case sensitivity.
*/
func (p *Path) EqualsStringCi(other string) bool {
	return equalsStringCaseInsensitive(p.String(), other)
}

/*
EqualsFS returns whether this and another Path are the same on the filesystem.
The evaluation also considers filesystem case sensitivity.
*/
func (p *Path) EqualsFS(other *Path) bool {
	structurallyIdentical := equalsStringCaseInsensitive(p.String(), other.String())
	if !structurallyIdentical {
		return false
	}

	// if equal in lowercase, proceed to check if path is on a
	// case-sensitive filesystem or not
	caseSensitive, err := IsCaseSensitiveFs(p)
	if err != nil {
		// return false in case of an error
		return false
	}

	// if case-sensitive, compare both original strings
	if caseSensitive {
		return p.path == other.path
	}

	// if case-insensitive, return true
	return true
}

/*
ToPosix returns a string representation with forward slashes.
*/
func (p *Path) ToPosix() string {
	return filepath.ToSlash(p.String())
}

/*
WithName returns this Path but with another base.
*/
func (p *Path) WithName(name string) *Path {
	return p.Parent().JoinStrings(name)
}

/*
Copy creates a copy of this Path.

Fresh out of the oven, just for you.
*/
func (p *Path) Copy() *Path {
	return NewPath(p.path)
}

/*
String returns this Path as a string.
*/
func (p *Path) String() string {
	pathStr := p.path

	// re-add removed whitespace escape characters
	if runtime.GOOS != "windows" {
		pathStr = strings.ReplaceAll(pathStr, " ", "\\ ")
	}

	return pathStr
}

/*
UnmarshalText unmarshalls any byte array into a Path type.
Implements the encoding.TextUnmarshaler interface.
*/
func (p *Path) UnmarshalText(text []byte) error {
	*p = *NewPath(string(text))
	return nil
}

/*
MarshalText marshals this Path into a byte array.
Implements the encoding.TextMarshaler interface.
*/
func (p *Path) MarshalText() (text []byte, err error) {
	return []byte(p.String()), nil
}

/*
clean cleans up this Path.

This function utilizes filepath.Clean.
*/
func cleanPathString(p string) string {

	dirty := strings.TrimSpace(p)

	// on non-windows operating systems
	if runtime.GOOS != "windows" {
		// remove whitespace escape characters during internal representation
		dirty = strings.ReplaceAll(dirty, "\\ ", " ")

		// replace all other '\\' characters with separator
		dirty = strings.ReplaceAll(dirty, "\\", pathSeparator)
	}

	cleanPath := filepath.Clean(dirty)
	return cleanPath
}

/*
pathCheck is a lower level Path existence checker.
It returns 0 if the path does not exist, 2 if it's a file and 2 if it's a directory.
*/
func pathCheck(p Path) int {
	fileInfo, err := os.Stat(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return pathCheckNoExist
		}
	}

	if fileInfo == nil {
		return pathCheckNoExist
	}

	if fileInfo.IsDir() {
		return pathCheckDir
	}

	return pathCheckFile
}

/*
flipCase is a utility function that takes the first character
and flips it's case. The leftover characters are appended.
This results in a string which is different from the original which can be used for
e.g. case sensitivity (in)variance.
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
nativeGlob is a wrapper function for Go's filepath.Glob.
It checks if the passed Path exists and returns the raw matches or errors.

Returns an error if pattern is an empty string.

filepath.Glob ignores IO errors.
*/
func nativeGlob(p *Path, pattern string) ([]string, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, errors.New("pattern must not be empty")
	}

	if !p.Exists() {
		return nil, errors.New("this Path does not exist")
	}

	if !p.IsDir() {
		return nil, errors.New("this path is not a directory")
	}

	matches, err := filepath.Glob(filepath.Join(p.path, pattern))
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func equalsStringCaseInsensitive(first string, second string) bool {
	// lowercase the strings and compare them
	thisLowerCase := strings.ToLower(cleanPathString(first))
	otherLowerCase := strings.ToLower(cleanPathString(second))

	// if not equal in lowercase, then they are not the same path
	// this tests if the actual path strings are equal
	return thisLowerCase == otherLowerCase
}
