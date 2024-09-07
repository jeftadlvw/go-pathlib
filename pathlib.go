package pathlib

import (
	"errors"
	"os"
	"path/filepath"
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

/*
Path is a struct that represents a filesystem path.
Enables stability in filepath handling.

Create a new instance using NewPath.
Other constructor functions are available too and are prefixed with 'New'.
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
	p := &Path{path: path}
	p.clean()

	return p
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
	return strings.Split(p.path, string(filepath.Separator))
}

/*
Split splits this Path into a directory and final part component.
*/
func (p *Path) Split() (*Path, *Path) {
	dir, file := filepath.Split(p.path)
	return NewPath(dir), NewPath(file)
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

This function utilizes filepath.Ext.
*/
func (p *Path) Extension() string {
	return filepath.Ext(p.path)
}

/*
Extensions returns all the Path's extensions.
Prefixed dots are included.

If the file starts with a '.' (which is a common on unix
based operating systems), the first part is ignored.
*/
func (p *Path) Extensions() []string {
	extensions := strings.Split(p.Base(), ".")
	for ext := range extensions {
		extensions[ext] = "." + extensions[ext]
	}

	return extensions
}

/*
Stem returns the last element of this Path without the extension.
*/
func (p *Path) Stem() string {
	return strings.TrimSuffix(p.Base(), p.Extension())
}

/*
MinimalStem returns the last element of this Path without all extensions.
*/
func (p *Path) MinimalStem() string {
	return strings.SplitN(p.Base(), string(filepath.Separator), 2)[0]
}

/*
Root returns the first part of the path.
On absolute paths this is the filesystem root, on relative paths all parts
up to the first non-'..' part are included
*/
func (p *Path) Root() string {

	if p.IsAbsolute() {
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

	return strings.SplitN(p.String(), string(filepath.Separator), 2)[0]
}

/*
RelativeTo returns the relative path from another Path to this Path.

This function utilizes filepath.Rel.
*/
func (p *Path) RelativeTo(o Path) (*Path, error) {
	rp, err := filepath.Rel(p.path, o.path)
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
AbsoluteTo returns an absolute representation of this Path.
If the Path is relative, it will be joined with the provided Path.

This function utilizes filepath.Abs.
*/
func (p *Path) AbsoluteTo(o *Path) *Path {

	// if path is already absolute, ignore it
	if p.IsAbsolute() {
		return p
	}

	return o.Join(p)
}

/*
IsAbsolute returns whether this Path is absolute.

This function utilizes filepath.IsAbs.
*/
func (p *Path) IsAbsolute() bool {
	return filepath.IsAbs(p.path)
}

/*
Resolve resolves all symbolic links.

This function utilizes filepath.EvalSymlinks.
*/
func (p *Path) Resolve(o *Path) (*Path, error) {
	rp, err := filepath.EvalSymlinks(o.path)
	if err != nil {
		return nil, err
	}

	return NewPath(rp), nil
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
Contains returns whether the passed pattern exist within this Path's directory.

This function utilizes filepath.Glob.
*/
func (p *Path) Contains(pattern string) (bool, error) {

	if !p.IsDir() {
		return false, errors.New("path does not exist or is not a directory")
	}

	matches, err := filepath.Glob(filepath.Join(p.path, pattern))
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
*/
func IsCaseSensitiveFs(p *Path) (bool, error) {
	alt := p.Parent()
	alt = alt.JoinStrings(flipCase(p.Base()))

	// get file stat for passed file
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
	return !os.SameFile(pathInfo, altInfo), nil
}

/*
Equals returns whether this and another Path are the same.
The evaluation also considers filesystem case sensitivity.
*/
func (p *Path) Equals(other *Path) bool {
	// lowercase the strings and compare them
	thisLowerCase := strings.ToLower(p.path)
	otherLowerCase := strings.ToLower(other.path)

	// if not equal in lowercase, then they are not the same path
	if thisLowerCase != otherLowerCase {
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
EqualsString returns whether the passed string matches this Path.

This function converts the passed string to a Path object and calls Equals.
*/
func (p *Path) EqualsString(other string) bool {
	return p.Equals(NewPath(other))
}

/*
ToPosix returns a string representation with forward slashes.
*/
func (p *Path) ToPosix() string {
	return filepath.ToSlash(p.path)
}

/*
WithName returns this Path but with another base.
*/
func (p *Path) WithName(name string) *Path {
	return p.Parent().JoinStrings(name)
}

/*
String returns this Path as a string.
*/
func (p *Path) String() string {
	return p.path
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
MarshalText marshalls this Path into a byte array.
Implements the encoding.TextMarshaler interface.
*/
func (p *Path) MarshalText() (text []byte, err error) {
	return []byte(p.String()), nil
}

/*
clean cleans up this Path.

This function utilizes filepath.Clean.
*/
func (p *Path) clean() {
	p.path = filepath.Clean(p.path)
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
