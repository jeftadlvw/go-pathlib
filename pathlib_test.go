package pathlib

import (
	"encoding"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

type TestInput[I any] struct {
	Name  string
	Input I
}

type TestExpect[T any] struct {
	Expect T
	Error  bool
}

type TestCase[I any, E any] struct {
	Name   string
	Input  I
	Expect E
	Error  bool
}

func TestPathInterfaceImplementations(t *testing.T) {
	path := NewPath("foo")

	t.Run("TextMarshaler", func(t *testing.T) {
		_, ok := interface{}(path).(encoding.TextMarshaler)
		require.True(t, ok)
	})

	t.Run("TextUnMarshaler", func(t *testing.T) {
		_, ok := interface{}(path).(encoding.TextUnmarshaler)
		require.True(t, ok)
	})
}

func TestPathConversions(t *testing.T) {
	cases := []TestCase[string, *Path]{
		{Input: "", Expect: NewPath(".")},
		{Input: ".", Expect: NewPath(".")},
		{Input: "..", Expect: NewPath("../")},
		{Input: "/", Expect: NewPath("/")},
		{Input: "//", Expect: NewPath("/")},
		{Input: "./", Expect: NewPath("./")},
		{Input: "./", Expect: NewPath(".")},
		{Input: ".//", Expect: NewPath(".")},
		{Input: "../", Expect: NewPath("..")},
		{Input: "../.", Expect: NewPath("../")},
		{Input: "../..", Expect: NewPath("../../")},
		{Input: "../../", Expect: NewPath("../../")},
		{Input: "../../.", Expect: NewPath("../../")},
		{Input: "/..", Expect: NewPath("/")},
		{Input: "/../..", Expect: NewPath("/")},
		{Input: "foo", Expect: NewPath("foo")},
		{Input: "foo/", Expect: NewPath("foo")},
		{Input: "foo/.", Expect: NewPath("foo")},
		{Input: "foo/..", Expect: NewPath(".")},
		{Input: "foo/../..", Expect: NewPath("..")},
		{Input: "foo/../bar", Expect: NewPath("bar")},
		{Input: "foo/../bar/..", Expect: NewPath(".")},
		{Input: "foo/../../bar", Expect: NewPath("../bar")},
		{Input: "foo/../../bar/..", Expect: NewPath("..")},
		{Input: "foo/bar", Expect: NewPath("foo/bar")},
		{Input: "    foo/bar", Expect: NewPath("foo/bar")},
		{Input: "  \t foo/bar", Expect: NewPath("foo/bar")},
		{Input: "foo/bar\n", Expect: NewPath("foo/bar")},
		{Input: "/foo/bar", Expect: NewPath("/foo/bar")},
		{Input: "/foo/bar/baz.yz", Expect: NewPath("/foo/bar/baz.yz")},
		{Input: "./foo/bar/baz.yz", Expect: NewPath("foo/bar/baz.yz")},
		{Input: "/foo/bar/baz.yz/..", Expect: NewPath("/foo/bar")},
		{Input: "some-random_thing", Expect: NewPath("some-random_thing")},
		{Input: "some-random_/thing/", Expect: NewPath("some-random_/thing")},
		{Input: "c:", Expect: NewPath("c:")},
		{Input: "c:/", Expect: NewPath("c:")},
		{Input: "c://", Expect: NewPath("c:")},
		{Input: "c://hello", Expect: NewPath("c:/hello")},
		{Input: "c:hello", Expect: NewPath("c:hello")},
		{Input: "c:\\", Expect: NewPath("c:/")},
		{Input: "c:\\\\", Expect: NewPath("c:/")},
		{Input: "c:\\\\hello", Expect: NewPath("c:/hello")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input string, expect *Path) {
		inputPath := NewPath(input)

		t.Run("fromString", func(t *testing.T) {
			require.Equal(t, *expect, *inputPath)
		})

		t.Run("toString", func(t *testing.T) {
			require.Equal(t, expect.path, inputPath.String())
		})

		t.Run("text unmarshalling", func(t *testing.T) {
			var emptyPath = &Path{}

			err := emptyPath.UnmarshalText([]byte(input))
			require.NoError(t, err)

			require.Equal(t, *expect, *emptyPath)
		})

		t.Run("text marshalling", func(t *testing.T) {
			marshaled, err := inputPath.MarshalText()
			require.NoError(t, err)

			require.Equal(t, expect.String(), string(marshaled))
		})

		t.Run("json unmarshalling", func(t *testing.T) {
			var emptyPaths []*Path
			input := fmt.Sprintf(`["%s"]`, inputPath.String())

			err := json.Unmarshal([]byte(input), &emptyPaths)
			require.NoError(t, err)

			require.Len(t, emptyPaths, 1)
			require.Equal(t, *expect, *emptyPaths[0])
		})

		t.Run("json marshalling", func(t *testing.T) {
			marshaled, err := json.Marshal([]*Path{inputPath})
			require.NoError(t, err)

			require.Equal(t, fmt.Sprintf(`["%s"]`, expect.String()), string(marshaled))
		})
	})
}

func TestPathWhiteSpaceRepresentation(t *testing.T) {
	cases := []TestCase[string, []string]{
		{Input: "path/with\\ whitespace", Expect: []string{"path/with whitespace", "path/with\\ whitespace"}},
		{Input: "\\  whitespace", Expect: []string{"  whitespace", "\\ \\ whitespace"}},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input string, expect []string) {
		require.Equal(t, len(expect), 2)
		internalRepr := expect[0]
		stringRepr := expect[1]

		inputPath := NewPath(input)

		t.Run("fromString", func(t *testing.T) {
			require.Equal(t, internalRepr, inputPath.path)
		})

		t.Run("toString", func(t *testing.T) {
			require.Equal(t, stringRepr, inputPath.String())
		})
	})
}

func TestNewCwd(t *testing.T) {
	// call library function
	pathlibCwdPath, err := NewCwd()
	require.NoError(t, err)

	// recreate cwd path using stdlib
	localCwd, err := os.Getwd()
	require.NoError(t, err)
	localCwdPath := NewPath(localCwd)

	// assert
	require.Equal(t, localCwdPath, pathlibCwdPath)
}

func TestNewHome(t *testing.T) {
	pathlibHomePath, err := NewHome()
	require.NoError(t, err)

	localHome, err := os.UserHomeDir()
	require.NoError(t, err)
	localHomePath := NewPath(localHome)

	require.Equal(t, localHomePath, pathlibHomePath)
}

func TestPathFromParts(t *testing.T) {
	cases := []TestCase[[]string, *Path]{
		{Input: []string{"."}, Expect: NewPath(".")},
		{Input: []string{".."}, Expect: NewPath("..")},
		{Input: []string{"a", "b", "c"}, Expect: NewPath("a/b/c")},
		{Input: []string{"a", "..", "c"}, Expect: NewPath("c")},
		{Input: []string{"..", "..", "c"}, Expect: NewPath("../../c")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", strings.Join(testCase.Input, ","))
	}

	runForResults(t, cases, func(t *testing.T, input []string, expect *Path) {
		require.Equal(t, *expect, *PathFromParts(input...))
	})
}

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
		{Input: NewPath("."), Expect: ""},
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

func TestPath_Root(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: "."},
		{Input: NewPath(".."), Expect: ".."},
		{Input: NewPath("/"), Expect: "/"},
		{Input: NewPath("c:/"), Expect: "c:"},
		{Input: NewPath("c://"), Expect: "c:"},
		{Input: NewPath("c:\\"), Expect: "c:"},
		{Input: NewPath("c:\\\\"), Expect: "c:"},
		{Input: NewPath("c:/foo"), Expect: "c:"},
		{Input: NewPath("foo/bar"), Expect: "foo"},
		{Input: NewPath("foo/bar.js"), Expect: "foo"},
		{Input: NewPath("/foo/bar.js"), Expect: "/"},
		{Input: NewPath("bar.js"), Expect: "bar.js"},
		{Input: NewPath("../bar"), Expect: "../bar"},
		{Input: NewPath("../../bar.js"), Expect: "../../bar.js"},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		require.Equal(t, expect, input.Root())
	})
}

func TestPath_AbsoluteAndRelative(t *testing.T) {
	cases := []TestCase[*Path, bool]{
		{Input: NewPath("."), Expect: false},
		{Input: NewPath(".."), Expect: false},
		{Input: NewPath("/"), Expect: true},
		{Input: NewPath("c:"), Expect: false},
		{Input: NewPath("c:/"), Expect: false},
		{Input: NewPath("c://"), Expect: false},
		{Input: NewPath("c:\\"), Expect: false},
		{Input: NewPath("c:\\\\"), Expect: false},
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
	cases := []TestCase[[]*Path, *Path]{
		{Input: []*Path{NewPath("/a/b"), NewPath("/")}, Expect: NewPath("a/b")},
		{Input: []*Path{NewPath("/a/b"), NewPath("/a")}, Expect: NewPath("b")},
		{Input: []*Path{NewPath("a/b"), NewPath("a")}, Expect: NewPath("b")},
		{Input: []*Path{NewPath("a/b/d"), NewPath("a/b/c")}, Expect: NewPath("../d")},
		{Input: []*Path{NewPath("/b"), NewPath("/a")}, Expect: NewPath("../b")},
		{Input: []*Path{NewPath("/b/d"), NewPath("/a/c")}, Expect: NewPath("../../b/d")},
		{Input: []*Path{NewPath("/"), NewPath("/a/b")}, Expect: NewPath("../..")},
		{Input: []*Path{NewPath(""), NewPath("/a/b")}, Error: true},
		{Input: []*Path{NewPath("../"), NewPath("/a/b")}, Error: true},
		{Input: []*Path{NewPath("../b"), NewPath("a/b")}, Expect: NewPath("../../../b")},
		{Input: []*Path{NewPath("a/b\\ whitespace/c"), NewPath("a/d")}, Expect: NewPath("../b whitespace/c")},
	}

	for i := range cases {
		cases[i].Name = fmt.Sprintf("[%d]", i+1)
	}

	runForResultsE(t, cases, func(t *testing.T, input []*Path, expect *Path, error bool) {
		require.Equal(t, len(input), 2)

		basePath := input[0]
		otherPath := input[1]
		relativePath, err := basePath.RelativeTo(otherPath)

		require.Equal(t, error, err != nil)
		if !error {
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
		absolutePath, err := input.Absolute()
		require.NoError(t, err)

		require.Equal(t, expect, absolutePath)
	})
}

func TestPath_AbsoluteTo(t *testing.T) {
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

	runForResultsE(t, cases, func(t *testing.T, input []*Path, expect *Path, error bool) {
		require.Equal(t, len(input), 2)

		base := input[0]
		other := input[1]
		absolutePath, err := base.AbsoluteTo(other)
		require.Equal(t, error, err != nil)

		if !error {
			require.Equal(t, expect, absolutePath)
		}
	})
}

func TestPath_Joins(t *testing.T) {
	cases := []TestCase[[]string, *Path]{
		{Input: []string{"/", "."}, Expect: NewPath("/")},
		{Input: []string{"/", "foo"}, Expect: NewPath("/foo")},
		{Input: []string{"/", "../"}, Expect: NewPath("/")},
		{Input: []string{"/", "../b"}, Expect: NewPath("/b")},
		{Input: []string{"a", "b"}, Expect: NewPath("a/b")},
		{Input: []string{"a", "../b"}, Expect: NewPath("b")},
		{Input: []string{"../a", "../b"}, Expect: NewPath("../b")},
		{Input: []string{"../a", "../../b"}, Expect: NewPath("../../b")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input []string, expect *Path) {
		require.True(t, len(input) > 0)

		basePath := NewPath(input[0])
		joinedStrPath := basePath.JoinStrings(input[1:]...)

		var strCvtPath []*Path = nil
		for _, pathStr := range input[1:] {
			strCvtPath = append(strCvtPath, NewPath(pathStr))
		}
		joinedPathsPath := basePath.Join(strCvtPath...)

		require.Equal(t, expect, joinedStrPath)
		require.Equal(t, expect, joinedPathsPath)
	})
}

func TestPath_Equals(t *testing.T) {
	cases := []TestCase[[]string, bool]{
		{Input: []string{"", ""}, Expect: true},
		{Input: []string{"", "a"}, Expect: false},
		{Input: []string{"foo", "foo"}, Expect: true},
		{Input: []string{"   foo", "foo"}, Expect: true},
		{Input: []string{"foo", "Foo"}, Expect: false},
		{Input: []string{"./foo", "foo"}, Expect: true},
		{Input: []string{"./foo", "\tfoo"}, Expect: true},
		{Input: []string{"./foo", "/foo"}, Expect: false},
		{Input: []string{"/foo", "/foo"}, Expect: true},
		{Input: []string{"/foo", "/foo\n"}, Expect: true},
		{Input: []string{"/foo", "/Foo"}, Expect: false},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input []string, expect bool) {
		require.Len(t, input, 2)

		basePath := NewPath(input[0])
		pathEquals := basePath.Equals(NewPath(input[1]))
		stringEquals := basePath.EqualsString(input[1])

		require.Equal(t, expect, pathEquals)
		require.Equal(t, expect, stringEquals)
	})
}

func TestPath_EqualsFlat(t *testing.T) {
	cases := []TestCase[[]string, bool]{
		{Input: []string{"", ""}, Expect: true},
		{Input: []string{"", "a"}, Expect: false},
		{Input: []string{"foo", "foo"}, Expect: true},
		{Input: []string{"   foo", "foo"}, Expect: true},
		{Input: []string{"foo", "Foo"}, Expect: true},
		{Input: []string{"./foo", "foo"}, Expect: true},
		{Input: []string{"./foo", "\tfoo"}, Expect: true},
		{Input: []string{"./foo", "/foo"}, Expect: false},
		{Input: []string{"/foo", "/foo"}, Expect: true},
		{Input: []string{"/foo", "/foo\n"}, Expect: true},
		{Input: []string{"/foo", "/Foo"}, Expect: true},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input []string, expect bool) {
		require.Len(t, input, 2)

		basePath := NewPath(input[0])
		pathEquals := basePath.EqualsFlat(NewPath(input[1]))
		stringEquals := basePath.EqualsStringFlat(input[1])

		require.Equal(t, expect, pathEquals)
		require.Equal(t, expect, stringEquals)
	})
}

func TestPath_ToPosix(t *testing.T) {
	cases := []TestCase[*Path, string]{
		{Input: NewPath("."), Expect: "."},
		{Input: NewPath(".."), Expect: ".."},
		{Input: NewPath("/foo"), Expect: "/foo"},
		{Input: NewPath("\\\\foo"), Expect: "/foo"},
		{Input: NewPath("\\\\foo\\bar"), Expect: "/foo/bar"},
		{Input: NewPath("\\\\foo\\\\bar"), Expect: "/foo/bar"},
		{Input: NewPath("/foo/bar"), Expect: "/foo/bar"},
		{Input: NewPath("//foo/bar"), Expect: "/foo/bar"},
		{Input: NewPath("//foo//bar"), Expect: "/foo/bar"},
		{Input: NewPath("/foo/with\\ whitespace"), Expect: "/foo/with\\ whitespace"},
		{Input: NewPath("\\foo\\with\\ whitespace"), Expect: "/foo/with\\ whitespace"},
		{Input: NewPath("\\\\foo\\\\with\\ whitespace"), Expect: "/foo/with\\ whitespace"},
		{Input: NewPath("C:\\foo\\with\\ whitespace"), Expect: "C:/foo/with\\ whitespace"},
		{Input: NewPath("C:/foo/with\\ whitespace"), Expect: "C:/foo/with\\ whitespace"},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect string) {
		toPosix := input.ToPosix()

		require.Equal(t, expect, toPosix)
	})
}

func TestPath_WithName(t *testing.T) {
	cases := []TestCase[[]string, *Path]{
		{Input: []string{"", "foo"}, Expect: NewPath("foo")},
		{Input: []string{"/", "foo"}, Expect: NewPath("/foo")},
		{Input: []string{"../", "foo"}, Expect: NewPath("foo")},
		{Input: []string{"../..", "foo"}, Expect: NewPath("../foo")},
		{Input: []string{"foo/bar", "foo"}, Expect: NewPath("foo/foo")},
		{Input: []string{"/foo/bar", "foo"}, Expect: NewPath("/foo/foo")},
		{Input: []string{"foo/file.txt", "bar.txt"}, Expect: NewPath("foo/bar.txt")},
		{Input: []string{"foo/.txt", ".json"}, Expect: NewPath("foo/.json")},
		{Input: []string{"/foo/.txt", ".json"}, Expect: NewPath("/foo/.json")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input []string, expect *Path) {
		require.True(t, len(input) == 2)

		// call function and assert
		path := NewPath(input[0])
		changedName := path.WithName(input[1])

		require.Equal(t, expect, changedName)
	})
}

func TestPath_Copy(t *testing.T) {
	cases := []TestCase[*Path, interface{}]{
		{Input: NewPath("foo/bar")},
		{Input: NewPath("../foo/bar")},
		{Input: NewPath("..")},
		{Input: NewPath("/foo")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect interface{}) {
		pointerCopy := input
		copiedPath := input.Copy()

		// compare pointers
		require.True(t, input == pointerCopy)
		require.False(t, input == copiedPath)

		// ensure copied path has same contents as original
		require.Equal(t, input, copiedPath)
	})
}

/*
 * TestPath_flipCase tests the underlying function for case-sensitivity checks.
 */
func TestPath_flipCase(t *testing.T) {
	cases := []TestCase[string, string]{
		{Input: "", Expect: ""},
		{Input: "/", Expect: "/"},
		{Input: "Aaa", Expect: "aaa"},
		{Input: "AAA", Expect: "aAA"},
		{Input: "aAA", Expect: "AAA"},
		{Input: "aaa", Expect: "Aaa"},
		{Input: "aaa ads", Expect: "Aaa ads"},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input string, expect string) {
		require.Equal(t, expect, flipCase(input))
	})
}

func mergeTestInputWithExpected[I any, E any](t *testing.T, testInputs []TestInput[I], testExpected []TestExpect[E]) []TestCase[I, E] {
	if len(testInputs) != len(testExpected) {
		t.Fatalf("Unequal number of given inputs (%d) and expected results (%d)", len(testInputs), len(testExpected))
	}

	cases := make([]TestCase[I, E], len(testInputs))
	for i, input := range testInputs {
		cases[i] = TestCase[I, E]{
			Name:   input.Name,
			Input:  input.Input,
			Expect: testExpected[i].Expect,
			Error:  testExpected[i].Error,
		}
	}

	return cases
}

func runForResultsE[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expect E, error bool)) {
	for _, test := range cases {

		caseName := test.Name
		if strings.TrimSpace(caseName) == "" {
			caseName = fmt.Sprintf("case--\"%v\"", test.Input)
		}

		t.Run(fmt.Sprint(caseName), func(t *testing.T) {
			testFunc(t, test.Input, test.Expect, test.Error)
		})
	}
}

func runForResults[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expect E)) {
	runForResultsE(t, cases, func(t *testing.T, input I, expect E, error bool) {
		testFunc(t, input, expect)
	})
}
