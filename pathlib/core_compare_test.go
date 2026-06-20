package pathlib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_AbsoluteAndRelative(t *testing.T) {
	cases := []TestCase[*Path, bool]{
		{Input: NewPath("."), Expect: false},
		{Input: NewPath(".."), Expect: false},
		{Input: NewPath("/"), Expect: true},
		{Input: NewPath("c:"), Expect: false},
		{Input: NewPath("c:/"), Expect: onWindows(false, true)},
		{Input: NewPath("c://"), Expect: onWindows(false, true)},
		{Input: NewPath("c:\\"), Expect: onWindows(false, true)},
		{Input: NewPath("c:\\\\"), Expect: onWindows(false, true)},
		{Input: NewPath("foo/bar"), Expect: false},
		{Input: NewPath("/foo/bar.js"), Expect: true},
		{Input: NewPath("bar.js"), Expect: false},
		{Input: NewPath("../bar"), Expect: false},
		{Input: NewPath("../../bar.js"), Expect: false},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect bool) {
		require.Equal(t, expect, input.IsAbsolute())
		require.Equal(t, !expect, input.IsRelative())
	})
}

func TestPath_RelativeTo(t *testing.T) {
	// what to apply on b to get to a

	cases := []TestCase[[]*Path, *Path]{
		{Input: []*Path{NewPath("/"), NewPath("/a/b")}, Expect: NewPath("../../")},
		{Input: []*Path{NewPath("/a"), NewPath("/a/b")}, Expect: NewPath("../")},
		{Input: []*Path{NewPath("a"), NewPath("a/b")}, Expect: NewPath("../")},
		{Input: []*Path{NewPath("a/b"), NewPath("a")}, Expect: NewPath("b")},
		{Input: []*Path{NewPath("a/b/c"), NewPath("a/b/d")}, Expect: NewPath("../c")},
		{Input: []*Path{NewPath("/a"), NewPath("/b")}, Expect: NewPath("../a")},
		{Input: []*Path{NewPath("/a/c"), NewPath("/b/d")}, Expect: NewPath("../../a/c")},
		{Input: []*Path{NewPath("/a/b"), NewPath("/")}, Expect: NewPath("a/b")},
		{Input: []*Path{NewPath("/a/b"), NewPath("")}, Error: true},
		{Input: []*Path{NewPath("/a/b"), NewPath("../")}, Error: true},
		{Input: []*Path{NewPath("a/b"), NewPath("../b")}, Error: true},
		{Input: []*Path{NewPathFromPosix("a/d"), NewPathFromPosix("a/b\\ whitespace/c")}, Expect: NewPathFromPosix("../../d")},

		{Input: []*Path{NewPathFromWindows(`C:\a\b`), NewPathFromWindows(``)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:\a\b`), NewPathFromWindows(`..\`)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:\a`), NewPathFromWindows(`D:\b`)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:\a`), NewPathFromWindows(`\\server\share:\b`)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:/a/c`), NewPathFromWindows(`C:/b/d`)}, Expect: NewPathFromWindows(`..\..\a\c`)},
		{Input: []*Path{NewPathFromWindows(`\\server\share\a`), NewPathFromWindows(`\\server\share\b`)}, Expect: NewPathFromWindows(`..\a`)},
		{Input: []*Path{NewPathFromWindows(`\\server\share\a\b`), NewPathFromWindows(`\\server\share\a`)}, Expect: NewPathFromWindows(`b`)},
		{Input: []*Path{NewPathFromWindows("a/d"), NewPathFromWindows("a/b\\ whitespace/c")}, Expect: NewPathFromWindows("../../../d")},
	}

	for i := range cases {
		cases[i].Name = fmt.Sprintf("[%d]", i+1)
	}

	runForResultsE(t, cases, func(t *testing.T, input []*Path, expect *Path, expectError bool) {
		require.Equal(t, len(input), 2)

		a := input[0]
		b := input[1]
		relativePath, err := a.RelativeTo(b)

		require.Equal(t, expectError, err != nil)
		if !expectError {
			require.Equal(t, expect, relativePath)
		}
	})
}

func TestPath_RelativeFrom(t *testing.T) {
	// what to apply on a to get to b

	cases := []TestCase[[]*Path, *Path]{
		{Input: []*Path{NewPath("/"), NewPath("/a/b")}, Expect: NewPath("a/b")},
		{Input: []*Path{NewPath("/a"), NewPath("/a/b")}, Expect: NewPath("b")},
		{Input: []*Path{NewPath("a"), NewPath("a/b")}, Expect: NewPath("b")},
		{Input: []*Path{NewPath("a/b"), NewPath("a")}, Expect: NewPath("..")},
		{Input: []*Path{NewPath("a/b/c"), NewPath("a/b/d")}, Expect: NewPath("../d")},
		{Input: []*Path{NewPath("/a"), NewPath("/b")}, Expect: NewPath("../b")},
		{Input: []*Path{NewPath("/a/c"), NewPath("/b/d")}, Expect: NewPath("../../b/d")},
		{Input: []*Path{NewPath("/a/b"), NewPath("/")}, Expect: NewPath("../..")},
		{Input: []*Path{NewPath("/a/b"), NewPath("")}, Error: true},
		{Input: []*Path{NewPath("/a/b"), NewPath("../")}, Error: true},
		{Input: []*Path{NewPath("a/b"), NewPath("../b")}, Expect: NewPath("../../../b")},
		{Input: []*Path{NewPathFromPosix("a/d"), NewPathFromPosix("a/b\\ whitespace/c")}, Expect: NewPathFromPosix("../b\\ whitespace/c")},

		{Input: []*Path{NewPathFromWindows(`C:\a\b`), NewPathFromWindows(``)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:\a\b`), NewPathFromWindows(`..\`)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:\a`), NewPathFromWindows(`D:\b`)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:\a`), NewPathFromWindows(`\\server\share:\b`)}, Error: true},
		{Input: []*Path{NewPathFromWindows(`C:/a/c`), NewPathFromWindows(`C:/b/d`)}, Expect: NewPathFromWindows(`..\..\b\d`)},
		{Input: []*Path{NewPathFromWindows(`\\server\share\a`), NewPathFromWindows(`\\server\share\b`)}, Expect: NewPathFromWindows(`..\b`)},
		{Input: []*Path{NewPathFromWindows(`\\server\share\a\b`), NewPathFromWindows(`\\server\share\a`)}, Expect: NewPathFromWindows(`..`)},
		{Input: []*Path{NewPathFromWindows("a/d"), NewPathFromWindows("a/b\\ whitespace/c")}, Expect: NewPathFromWindows("../b/ whitespace/c")},
	}

	for i := range cases {
		cases[i].Name = fmt.Sprintf("[%d]", i+1)
	}

	runForResultsE(t, cases, func(t *testing.T, input []*Path, expect *Path, expectError bool) {
		require.Equal(t, len(input), 2)

		a := input[0]
		b := input[1]
		relativePath, err := a.RelativeFrom(b)

		require.Equal(t, expectError, err != nil)
		if !expectError {
			require.Equal(t, expect, relativePath)
		}
	})
}

func TestPath_Absolute(t *testing.T) {
	wdPath, err := NewCwd()
	require.NoError(t, err)

	cases := []TestCase[*Path, *Path]{
		{Input: NewPath("."), Expect: wdPath.JoinStrings(".")},
		{Input: NewPath("foo"), Expect: wdPath.JoinStrings("foo")},
		{Input: NewPath("foo/bar"), Expect: wdPath.JoinStrings("foo/bar")},
		{Input: NewPath("/"), Expect: NewPath("/")},
		{Input: NewPath("/foo"), Expect: NewPath("/foo")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect *Path) {
		absolutePath, err := input.MakeAbsolute()
		require.NoError(t, err)

		require.Equal(t, expect, absolutePath)
	})
}

func TestPath_AbsoluteFrom(t *testing.T) {
	cases := []TestCase[[]*Path, *Path]{
		{Input: []*Path{NewPath("."), NewPath(".")}, Error: true},
		{Input: []*Path{NewPath("/"), NewPath(".")}, Expect: NewPath("/")},
		{Input: []*Path{NewPath("."), NewPath("/foo")}, Expect: NewPath("/foo")},
		{Input: []*Path{NewPath("hello"), NewPath("/foo")}, Expect: NewPath("/foo/hello")},
		{Input: []*Path{NewPath("../hello"), NewPath("/foo")}, Expect: NewPath("/hello")},
		{Input: []*Path{NewPath("hello/bar"), NewPath("/foo")}, Expect: NewPath("/foo/hello/bar")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResultsE(t, cases, func(t *testing.T, input []*Path, expect *Path, expectError bool) {
		require.Equal(t, len(input), 2)

		base := input[0]
		other := input[1]
		absolutePath, err := base.AbsoluteFrom(other)
		require.Equal(t, expectError, err != nil)

		if !expectError {
			require.Equal(t, expect, absolutePath)
		}
	})
}

func TestPath_EqualsCaseSensitive(t *testing.T) {
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

// TestPath_EqualsStringContract pins EqualsString to its contract: it interprets
// other as a Posix path and compares via Equals, deterministically and regardless
// of the runtime OS. The previous implementation path.Clean'd the raw string,
// which never converted separators and mangled anchors.
func TestPath_EqualsStringContract(t *testing.T) {
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
				p := NewPathFromWindows(b)
				// EqualsString must equal Equals(NewPathFromPosix(other)) on both
				// the case-sensitive and case-insensitive paths.
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
	// EqualsString reads other as a Posix path on every platform, so a backslash
	// is a filename character: "a\b" is a single component and never equals the
	// two-component "a/b".
	require.False(t, NewPathFromPosix("a/b").EqualsString(`a\b`),
		`"a/b" (two components) must not equal "a\b" (one posix component)`)

	// A plain (non-anchored) path always equals its own ToPosix form, which is the
	// canonical string form EqualsString is meant to be paired with.
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
	for _, s := range []string{`C:\`, `\\host\share`, `\\host\share\foo`} {
		p := NewPathFromWindows(s)
		require.False(t, p.EqualsString(p.ToPosix()),
			"%q: Windows anchor is not recognized via EqualsString", s)
		// The documented workaround round-trips.
		require.True(t, p.Equals(NewPathFromWindows(p.ToWindows())),
			"%q: Equals(NewPathFromWindows(...)) is the correct comparison", s)
	}
}
