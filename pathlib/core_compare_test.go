package pathlib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_EqualsCaseSensitive(t *testing.T) {
	t.Parallel()

	cases := []TestCase[[]string, bool]{
		{Input: []string{"", ""}, Expect: true},
		{Input: []string{"", "a"}, Expect: false},
		{Input: []string{"foo", "foo"}, Expect: true},
		{Input: []string{"foo", "fsho"}, Expect: false},
		{Input: []string{"   foo", "foo"}, Expect: false},
		{Input: []string{"   foo ", "foo  "}, Expect: false},
		{Input: []string{"foo", "Foo"}, Expect: false},
		{Input: []string{"./foo", "foo"}, Expect: true},
		{Input: []string{"./foo", "\tfoo"}, Expect: false},
		{Input: []string{"./foo", "/foo"}, Expect: false},
		{Input: []string{"/foo", "/foo"}, Expect: true},
		{Input: []string{"/foo", "/foo\n"}, Expect: false},
		{Input: []string{"/foo", "/Foo"}, Expect: false},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input []string, expect bool) {
		t.Helper()

		require.Len(t, input, 2)

		basePath := NewPath(input[0])
		pathEqualsDefaultCase := basePath.Equals(NewPath(input[1]))
		pathEqualsCaseSensitive := basePath.Equals(NewPath(input[1]), CaseSensitive)
		stringEqualsCaseSensitive := basePath.EqualsString(input[1], CaseSensitive)

		require.Equal(t, expect, pathEqualsDefaultCase)
		require.Equal(t, expect, pathEqualsCaseSensitive)
		require.Equal(t, expect, stringEqualsCaseSensitive)
		require.Equal(t, pathEqualsDefaultCase, pathEqualsCaseSensitive)
	})
}

func TestPath_EqualsCaseInsensitive(t *testing.T) {
	t.Parallel()

	cases := []TestCase[[]string, bool]{
		{Input: []string{"", ""}, Expect: true},
		{Input: []string{"", "a"}, Expect: false},
		{Input: []string{"foo", "foo"}, Expect: true},
		{Input: []string{"foo", "fsho"}, Expect: false},
		{Input: []string{"   foo", "foo"}, Expect: false},
		{Input: []string{"   foo ", "foo  "}, Expect: false},
		{Input: []string{"foo", "Foo"}, Expect: true},
		{Input: []string{"./foo", "foo"}, Expect: true},
		{Input: []string{"./foo", "\tfoo"}, Expect: false},
		{Input: []string{"./foo", "/foo"}, Expect: false},
		{Input: []string{"/foo", "/foo"}, Expect: true},
		{Input: []string{"/foo", "/foo\n"}, Expect: false},
		{Input: []string{"/foo", "/Foo"}, Expect: true},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input []string, expect bool) {
		t.Helper()

		require.Len(t, input, 2)

		basePath := NewPath(input[0])
		pathEqualsDefaultCase := basePath.Equals(NewPath(input[1]))
		pathEqualsCaseSensitive := basePath.Equals(NewPath(input[1]), CaseSensitive)
		pathEqualsCaseInsensitive := basePath.Equals(NewPath(input[1]), CaseInsensitive)
		stringEqualsCaseInsensitive := basePath.EqualsString(input[1], CaseInsensitive)

		require.Equal(t, pathEqualsDefaultCase, pathEqualsCaseSensitive)
		require.Equal(t, expect, pathEqualsCaseInsensitive)
		require.Equal(t, expect, stringEqualsCaseInsensitive)
	})
}

// TestPath_EqualsStringContract pins EqualsString to its contract: it
// interprets other as a Posix path and compares via Equals, deterministically
// and regardless of the runtime OS. The previous implementation path.Clean'd
// the raw string, which never converted separators and mangled anchors.
func TestPath_EqualsStringContract(t *testing.T) {
	t.Parallel()

	bases := []string{
		`C:\foo\bar`,
		`C:\dir\README.md`,
		`C:\`,
		`\\host\share\foo`,
		`relative\dir\file`,
		`a/b`,
	}
	others := []string{
		`C:\foo\bar`, `C:/foo/bar`,
		`\\host\share\foo`, `//host/share/foo`,
		`a\b`, `a/b`,
	}

	for _, b := range bases {
		for _, o := range others {
			t.Run(fmt.Sprintf("[%s]_vs_[%s]", b, o), func(t *testing.T) {
				t.Parallel()

				p := NewPathFromWindows(b)
				// EqualsString must equal Equals(NewPathFromPosix(other)) on
				// both the case-sensitive and case-insensitive paths.
				require.Equal(t, p.Equals(NewPathFromPosix(o), CaseSensitive),
					p.EqualsString(o, CaseSensitive),
					"case-sensitive must match Equals(NewPathFromPosix(other))")
				require.Equal(t, p.Equals(NewPathFromPosix(o), CaseInsensitive),
					p.EqualsString(o, CaseInsensitive),
					"case-insensitive must match Equals(NewPathFromPosix(other))")
			})
		}
	}
}

func TestPath_EqualsWindowsSeparatorEquivalence(t *testing.T) {
	t.Parallel()

	cases := [][2]string{
		{`C:\foo\bar`, `C:/foo/bar`},
		{`\\host\share\foo`, `//host/share/foo`},
		{`C:\`, `C:/`},
	}
	for _, c := range cases {
		require.True(t,
			NewPathFromWindows(c[0]).Equals(NewPathFromWindows(c[1])),
			"%q must equal %q when both are parsed as Windows paths", c[0], c[1])
	}
}

func TestPath_EqualsStringBackslashIsPosixNameChar(t *testing.T) {
	t.Parallel()

	// EqualsString reads other as a Posix path on every platform, so a
	// backslash is a filename character: "a\b" is a single component and never
	// equals the two-component "a/b".
	require.False(t, NewPathFromPosix("a/b").EqualsString(`a\b`),
		`"a/b" (two components) must not equal "a\b" (one posix component)`)

	// A plain (non-anchored) path always equals its own ToPosix form, which is
	// the canonical string form EqualsString is meant to be paired with.
	for _, s := range []string{"foo/bar", "/foo/bar", "."} {
		p := NewPathFromPosix(s)
		require.True(t, p.EqualsString(p.ToPosix()),
			"%q must equal its own ToPosix() form", s)
	}
}

// TestPath_EqualsStringWindowsAnchorLimitation pins the documented limitation:
// because EqualsString parses other as plain Posix, a Windows anchor in the
// string is not recognized and does not round-trip. Drive roots and UNC paths
// must be compared with Equals(NewPathFromWindows(s)) instead.
func TestPath_EqualsStringWindowsAnchorLimitation(t *testing.T) {
	t.Parallel()

	for _, s := range []string{`C:\`, `\\host\share`, `\\host\share\foo`} {
		p := NewPathFromWindows(s)
		require.False(t, p.EqualsString(p.ToPosix()),
			"%q: Windows anchor is not recognized via EqualsString", s)
		// The documented workaround round-trips.
		require.True(t, p.Equals(NewPathFromWindows(p.ToWindows())),
			"%q: Equals(NewPathFromWindows(...)) is the correct comparison", s)
	}
}

func TestPath_EqualsNil(t *testing.T) {
	t.Parallel()

	var nilPath *Path
	p := NewPathFromPosix("foo")

	for _, opt := range []CompareOption{CaseSensitive, CaseInsensitive} {
		require.True(t, nilPath.Equals(nil, opt), "two nil Paths are equal")
		require.False(t, nilPath.Equals(p, opt), "nil receiver differs from a Path")
		require.False(t, p.Equals(nil, opt), "a Path differs from a nil argument")
	}

	require.False(t, nilPath.EqualsString("."), "nil never equals a string, not even \".\"")
	require.False(t, nilPath.EqualsString(""))
}

func TestPath_EqualsIsNotPointerIdentity(t *testing.T) {
	t.Parallel()

	// Two separately constructed Paths are different pointers, so == is false,
	// but they denote the same path.
	a := NewPathFromPosix("foo/bar")
	b := NewPathFromPosix("foo/bar")
	require.NotSame(t, a, b)
	require.True(t, a.Equals(b))
}
