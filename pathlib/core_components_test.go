package pathlib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_Parent(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: "."},
		{Input: NewPath(".."), Expect: "."},
		{Input: NewPath("/"), Expect: "/"},
		{Input: NewPath("foo/bar"), Expect: "foo"},
		{Input: NewPath("foo/bar.js"), Expect: "foo"},
		{Input: NewPath("/foo/bar.js"), Expect: "/foo"},
		{Input: NewPath("bar.js"), Expect: "."},
		{Input: NewPath("../bar.js"), Expect: ".."},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.Parent().path)
	})
}

func TestPath_Parts(t *testing.T) {
	cases := []TestCase[*Path, []string]{
		{Input: NewPath("."), Expect: []string{"."}},
		{Input: NewPath(".."), Expect: []string{".."}},
		{Input: NewPath("/"), Expect: []string{}},
		{Input: NewPath("foo/bar"), Expect: []string{"foo", "bar"}},
		{Input: NewPath("foo/bar.js"), Expect: []string{"foo", "bar.js"}},
		{Input: NewPath("/foo/bar.js"), Expect: []string{"foo", "bar.js"}},
		{Input: NewPath("bar.js"), Expect: []string{"bar.js"}},
		{Input: NewPath("../bar.js"), Expect: []string{"..", "bar.js"}},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect []string) {
		require.Equal(t, expect, input.Parts())
	})
}

func TestPath_Base(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: "."},
		{Input: NewPath(".."), Expect: ".."},
		{Input: NewPath("/"), Expect: "/"},
		{Input: NewPath("foo/bar"), Expect: "bar"},
		{Input: NewPath("foo/bar.js"), Expect: "bar.js"},
		{Input: NewPath("/foo/bar.js"), Expect: "bar.js"},
		{Input: NewPath("bar.js"), Expect: "bar.js"},
		{Input: NewPath("../bar.js"), Expect: "bar.js"},
		{Input: NewPath(".bar.js"), Expect: ".bar.js"},
		{Input: NewPath("..bar.js"), Expect: "..bar.js"},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.Base())
	})
}

func TestPath_Split(t *testing.T) {
	cases := []TestCase[*Path, []string]{
		{Input: NewPath("."), Expect: []string{".", "."}},
		{Input: NewPath(".."), Expect: []string{".", ".."}},
		{Input: NewPath("/"), Expect: []string{"/", ""}},
		{Input: NewPath("foo/bar"), Expect: []string{"foo", "bar"}},
		{Input: NewPath("foo/bar.js"), Expect: []string{"foo", "bar.js"}},
		{Input: NewPath("/foo/bar.js"), Expect: []string{"/foo", "bar.js"}},
		{Input: NewPath("bar.js"), Expect: []string{".", "bar.js"}},
		{Input: NewPath("../bar.js"), Expect: []string{"..", "bar.js"}},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect []string) {
		require.Equal(t, len(expect), 2)

		inputPartParent, inputPartBase := input.Split()

		require.Equal(t, expect[0], inputPartParent.path, "Parent")
		require.Equal(t, expect[1], inputPartBase, "Base")
	})
}

func TestPath_Stem(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: "."},
		{Input: NewPath(".."), Expect: ".."},
		{Input: NewPath("/"), Expect: ""},
		{Input: NewPath("foo/bar"), Expect: "bar"},
		{Input: NewPath("foo/bar.js"), Expect: "bar"},
		{Input: NewPath("/foo/bar.js"), Expect: "bar"},
		{Input: NewPath("/foo/baz.kf/bar.js"), Expect: "bar"},
		{Input: NewPath("bar.js"), Expect: "bar"},
		{Input: NewPath("bar.js.foo"), Expect: "bar"},
		{Input: NewPath("bar.js..foo"), Expect: "bar"},
		{Input: NewPath("../bar.js"), Expect: "bar"},
		{Input: NewPath("../bar.js.foo"), Expect: "bar"},
		{Input: NewPath(".bar.js"), Expect: ".bar"},
		{Input: NewPath("..bar.js"), Expect: "..bar"},
		{Input: NewPath("..bar.js."), Expect: "..bar"},
		{Input: NewPath("..bar.js.foo"), Expect: "..bar"},
		{Input: NewPath("..bar.js.."), Expect: "..bar"},
		{Input: NewPath("..bar.js..a"), Expect: "..bar"},
		{Input: NewPath("..bar.js..a.b"), Expect: "..bar"},
		{Input: NewPath("..bar"), Expect: "..bar"},
		{Input: NewPath("...bar."), Expect: "...bar"},
		{Input: NewPath("...bar."), Expect: "...bar"},
		{Input: NewPath("...bar"), Expect: "...bar"},
		{Input: NewPath("...bar.."), Expect: "...bar"},
		{Input: NewPath("...bar..a"), Expect: "...bar"},
		{Input: NewPath("...bar..a.b"), Expect: "...bar"},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.Stem())
	})
}

func TestPath_HasExtensions(t *testing.T) {
	cases := []TestCase[*Path, bool]{
		{Input: NewPath("."), Expect: false},
		{Input: NewPath(".."), Expect: false},
		{Input: NewPath("/"), Expect: false},
		{Input: NewPath("foo/bar"), Expect: false},
		{Input: NewPath("foo/bar.js"), Expect: true},
		{Input: NewPath("/foo/bar.js"), Expect: true},
		{Input: NewPath("bar.js"), Expect: true},
		{Input: NewPath("../bar.js"), Expect: true},
		{Input: NewPath("../bar.js.foo"), Expect: true},
		{Input: NewPath(".bar.js"), Expect: true},
		{Input: NewPath("..bar.js"), Expect: true},
		{Input: NewPath("..bar.js."), Expect: true},
		{Input: NewPath("..bar.js.."), Expect: true},
		{Input: NewPath("..bar.js..a"), Expect: true},
		{Input: NewPath("..bar.js..a.b"), Expect: true},
		{Input: NewPath("..bar"), Expect: false},
		{Input: NewPath("...bar"), Expect: false},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect bool) {
		require.Equal(t, expect, input.HasExtensions())
	})
}

func TestPath_Extension(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: ""},
		{Input: NewPath(".."), Expect: ""},
		{Input: NewPath("/"), Expect: ""},
		{Input: NewPath("foo/bar"), Expect: ""},
		{Input: NewPath("foo/bar.js"), Expect: ".js"},
		{Input: NewPath("/foo/bar.js"), Expect: ".js"},
		{Input: NewPath("bar.js"), Expect: ".js"},
		{Input: NewPath("../bar.js"), Expect: ".js"},
		{Input: NewPath("../bar.js.foo"), Expect: ".js.foo"},
		{Input: NewPath(".bar.js"), Expect: ".js"},
		{Input: NewPath("..bar.js"), Expect: ".js"},
		{Input: NewPath("..bar.js."), Expect: ".js."},
		{Input: NewPath("..bar.js.."), Expect: ".js.."},
		{Input: NewPath("..bar.js..a"), Expect: ".js..a"},
		{Input: NewPath("..bar.js..a.b"), Expect: ".js..a.b"},
		{Input: NewPath("..bar"), Expect: ""},
		{Input: NewPath("...bar"), Expect: ""},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.Extension())
	})
}

func TestPath_ExtensionParts(t *testing.T) {
	cases := []TestCase[*Path, []string]{
		{Input: NewPath("."), Expect: []string{}},
		{Input: NewPath(".."), Expect: []string{}},
		{Input: NewPath("/"), Expect: []string{}},
		{Input: NewPath("foo/bar"), Expect: []string{}},
		{Input: NewPath("foo/bar.js"), Expect: []string{"js"}},
		{Input: NewPath("/foo/bar.js"), Expect: []string{"js"}},
		{Input: NewPath("bar.js"), Expect: []string{"js"}},
		{Input: NewPath("../bar.js"), Expect: []string{"js"}},
		{Input: NewPath("../bar.js.foo"), Expect: []string{"js", "foo"}},
		{Input: NewPath("../bar.js.foo."), Expect: []string{"js", "foo", ""}},
		{Input: NewPath("../bar.js.foo.."), Expect: []string{"js", "foo", "", ""}},
		{Input: NewPath("../bar.js.foo..a"), Expect: []string{"js", "foo", "", "a"}},
		{Input: NewPath("../bar.js.foo..a.b"), Expect: []string{"js", "foo", "", "a", "b"}},
		{Input: NewPath(".bar.js"), Expect: []string{"js"}},
		{Input: NewPath("..bar.js"), Expect: []string{"js"}},
		{Input: NewPath("..bar"), Expect: []string{}},
		{Input: NewPath("...bar"), Expect: []string{}},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect []string) {
		require.Equal(t, expect, input.ExtensionParts())

		// if the expected parts match, ExtensionCount should also be correct.
		require.Equal(t, len(expect), input.ExtensionCount())
	})
}

func TestPath_Anchor(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: ""},
		{Input: NewPath(".."), Expect: ""},
		{Input: NewPath("/"), Expect: "/"},
		{Input: NewPath("c:/"), Expect: onWindows("", "c:")},
		{Input: NewPath("c:/foo"), Expect: onWindows("", "c:")},
		{Input: NewPath("c://"), Expect: onWindows("", "c:")},
		{Input: NewPath("c:\\"), Expect: onWindows("", "c:")},
		{Input: NewPath("c:\\foo"), Expect: onWindows("", "c:")},
		{Input: NewPath("c:\\\\"), Expect: onWindows("", "c:")},
		{Input: NewPath("//host/share"), Expect: onWindows("/", platformNativeUNC("//host/share"))},
		{Input: NewPath("//host/share/"), Expect: onWindows("/", platformNativeUNC("//host/share"))},
		{Input: NewPath("//host/share/foo"), Expect: onWindows("/", platformNativeUNC("//host/share"))},
		{Input: NewPath("\\\\host\\share"), Expect: onWindows("", platformNativeUNC("//host/share"))},
		{Input: NewPath("\\\\host\\share\\"), Expect: onWindows("", platformNativeUNC("//host/share"))},
		{Input: NewPath("\\\\host\\share\\foo"), Expect: onWindows("", platformNativeUNC("//host/share"))},
		{Input: NewPathFromWindows("\\\\host\\share"), Expect: platformNativeUNC("//host/share")},
		{Input: NewPathFromWindows("\\\\host\\share\\"), Expect: platformNativeUNC("//host/share")},
		{Input: NewPathFromWindows("\\\\host\\share\\foo"), Expect: platformNativeUNC("//host/share")},
		{Input: NewPathFromWindows("foo/bar"), Expect: ""},
		{Input: NewPath("foo/bar.js"), Expect: ""},
		{Input: NewPath("/foo/bar.js"), Expect: "/"},
		{Input: NewPath("bar.js"), Expect: ""},
		{Input: NewPath("../bar"), Expect: ""},
		{Input: NewPath("../../bar.js"), Expect: ""},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("%d-[%s]", i, testCase.Input.path)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.Anchor())
	})
}

func TestPath_WindowsVolume(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: ""},
		{Input: NewPath("/"), Expect: ""},
		{Input: NewPath("/foo/bar"), Expect: ""},
		{Input: NewPath("foo/bar"), Expect: ""},
		{Input: NewPathFromWindows("C:/foo"), Expect: "C:"},
		{Input: NewPathFromWindows("c:/"), Expect: "c:"},
		{Input: NewPathFromWindows("D:\\bar"), Expect: "D:"},
		{Input: NewPathFromWindows("c:"), Expect: "c:"},
		{Input: NewPathFromWindows("\\\\host\\share"), Expect: ""},
		{Input: NewPathFromWindows("\\\\host\\share\\foo"), Expect: ""},
		{Input: NewPathFromWindows("//host/share"), Expect: ""},
		{Input: NewPathFromWindows("//host/share/foo"), Expect: ""},
		{Input: NewPathFromWindows("foo/bar"), Expect: ""},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("%d-[%s]", i, testCase.Input.path)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.WindowsVolume())
	})
}

func TestPath_WindowsUncRoot(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: ""},
		{Input: NewPath("/"), Expect: ""},
		{Input: NewPath("/foo/bar"), Expect: ""},
		{Input: NewPath("foo/bar"), Expect: ""},
		{Input: NewPathFromWindows("C:/foo"), Expect: ""},
		{Input: NewPathFromWindows("c:/"), Expect: ""},
		{Input: NewPathFromWindows("\\\\host\\share"), Expect: platformNativeUNC("//host/share")},
		{Input: NewPathFromWindows("\\\\host\\share\\"), Expect: platformNativeUNC("//host/share")},
		{Input: NewPathFromWindows("\\\\host\\share\\foo"), Expect: platformNativeUNC("//host/share")},
		{Input: NewPathFromWindows("//server/vol"), Expect: platformNativeUNC("//server/vol")},
		{Input: NewPathFromWindows("//server/vol/bar"), Expect: platformNativeUNC("//server/vol")},
		{Input: NewPathFromWindows("foo/bar"), Expect: ""},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("%d-[%s]", i, testCase.Input.path)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.WindowsUncRoot())
	})
}

func TestPath_HasDotName(t *testing.T) {
	cases := map[string]bool{
		".config":      true,
		".a":           true,
		"/home/u/.ssh": true,
		"a/b/.hidden":  true,
		"visible.txt":  false,
		"a/b/c":        false,
		".":            false,
		"..":           false,
		"":             false,
	}

	for input, expected := range cases {
		require.Equal(t, expected, NewPath(input).HasDotName(), "HasDotName(%q)", input)
	}
}
