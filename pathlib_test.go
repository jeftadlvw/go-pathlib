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

func TestPathInputOutputDisplay(t *testing.T) {
	type ExpectMatrix struct {
		AsPosixOnPosix     string
		AsPosixOnWindows   string
		AsWindowsOnPosix   string
		AsWindowsOnWindows string
	}

	type ExpectMatrixMask struct {
		AsPosix   bool
		AsWindows bool
		OnPosix   bool
		OnWindows bool
	}

	cases := []TestCase[string, ExpectMatrix]{
		{Input: "", Expect: ExpectMatrix{
			AsPosixOnPosix: ".", AsPosixOnWindows: ".", AsWindowsOnPosix: ".", AsWindowsOnWindows: "."}},
		{Input: ".", Expect: ExpectMatrix{
			AsPosixOnPosix: ".", AsPosixOnWindows: ".", AsWindowsOnPosix: ".", AsWindowsOnWindows: "."}},
		{Input: "..", Expect: ExpectMatrix{
			AsPosixOnPosix: "..", AsPosixOnWindows: "..", AsWindowsOnPosix: "..", AsWindowsOnWindows: ".."}},
		{Input: "/", Expect: ExpectMatrix{
			AsPosixOnPosix: "/", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "//", Expect: ExpectMatrix{
			AsPosixOnPosix: "/", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "\\", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "./", Expect: ExpectMatrix{
			AsPosixOnPosix: ".", AsPosixOnWindows: ".", AsWindowsOnPosix: ".", AsWindowsOnWindows: "."}},
		{Input: ".//", Expect: ExpectMatrix{
			AsPosixOnPosix: ".", AsPosixOnWindows: ".", AsWindowsOnPosix: ".", AsWindowsOnWindows: "."}},
		{Input: "../", Expect: ExpectMatrix{
			AsPosixOnPosix: "..", AsPosixOnWindows: "..", AsWindowsOnPosix: "..", AsWindowsOnWindows: ".."}},
		{Input: "../.", Expect: ExpectMatrix{
			AsPosixOnPosix: "..", AsPosixOnWindows: "..", AsWindowsOnPosix: "..", AsWindowsOnWindows: ".."}},
		{Input: "../..", Expect: ExpectMatrix{
			AsPosixOnPosix: "../..", AsPosixOnWindows: "..\\..", AsWindowsOnPosix: "../..", AsWindowsOnWindows: "..\\.."}},
		{Input: "../../", Expect: ExpectMatrix{
			AsPosixOnPosix: "../..", AsPosixOnWindows: "..\\..", AsWindowsOnPosix: "../..", AsWindowsOnWindows: "..\\.."}},
		{Input: "../../.", Expect: ExpectMatrix{
			AsPosixOnPosix: "../..", AsPosixOnWindows: "..\\..", AsWindowsOnPosix: "../..", AsWindowsOnWindows: "..\\.."}},
		{Input: "../../foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "../../foo", AsPosixOnWindows: "..\\..\\foo", AsWindowsOnPosix: "../../foo", AsWindowsOnWindows: "..\\..\\foo"}},
		{Input: "/..", Expect: ExpectMatrix{
			AsPosixOnPosix: "/", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "/.", Expect: ExpectMatrix{
			AsPosixOnPosix: "/", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "/../..", Expect: ExpectMatrix{
			AsPosixOnPosix: "/", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo", AsPosixOnWindows: "foo", AsWindowsOnPosix: "foo", AsWindowsOnWindows: "foo"}},
		{Input: "foo  ", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo  ", AsPosixOnWindows: "foo  ", AsWindowsOnPosix: "foo  ", AsWindowsOnWindows: "foo  "}},
		{Input: "foo/", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo", AsPosixOnWindows: "foo", AsWindowsOnPosix: "foo", AsWindowsOnWindows: "foo"}},
		{Input: "foo/.", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo", AsPosixOnWindows: "foo", AsWindowsOnPosix: "foo", AsWindowsOnWindows: "foo"}},
		{Input: "foo/..", Expect: ExpectMatrix{
			AsPosixOnPosix: ".", AsPosixOnWindows: ".", AsWindowsOnPosix: ".", AsWindowsOnWindows: "."}},
		{Input: "foo/../..", Expect: ExpectMatrix{
			AsPosixOnPosix: "..", AsPosixOnWindows: "..", AsWindowsOnPosix: "..", AsWindowsOnWindows: ".."}},
		{Input: "foo/../bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "bar", AsPosixOnWindows: "bar", AsWindowsOnPosix: "bar", AsWindowsOnWindows: "bar"}},
		{Input: "foo/../bar/..", Expect: ExpectMatrix{
			AsPosixOnPosix: ".", AsPosixOnWindows: ".", AsWindowsOnPosix: ".", AsWindowsOnWindows: "."}},
		{Input: "foo/../../bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "../bar", AsPosixOnWindows: "..\\bar", AsWindowsOnPosix: "../bar", AsWindowsOnWindows: "..\\bar"}},
		{Input: "foo/../../bar/..", Expect: ExpectMatrix{
			AsPosixOnPosix: "..", AsPosixOnWindows: "..", AsWindowsOnPosix: "..", AsWindowsOnWindows: ".."}},
		{Input: "foo/bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo/bar", AsPosixOnWindows: "foo\\bar", AsWindowsOnPosix: "foo/bar", AsWindowsOnWindows: "foo\\bar"}},
		{Input: "  foo  ", Expect: ExpectMatrix{
			AsPosixOnPosix: "  foo  ", AsPosixOnWindows: "  foo  ", AsWindowsOnPosix: "  foo  ", AsWindowsOnWindows: "  foo  "}},
		{Input: "  \t foo/bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "  \t foo/bar", AsPosixOnWindows: "  \t foo\\bar", AsWindowsOnPosix: "  \t foo/bar", AsWindowsOnWindows: "  \t foo\\bar"}},
		{Input: "foo/bar\n", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo/bar\n", AsPosixOnWindows: "foo\\bar\n", AsWindowsOnPosix: "foo/bar\n", AsWindowsOnWindows: "foo\\bar\n"}},
		{Input: "foo/  bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo/  bar", AsPosixOnWindows: "foo\\  bar", AsWindowsOnPosix: "foo/  bar", AsWindowsOnWindows: "foo\\  bar"}},
		{Input: "/ foo/  bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "/ foo/  bar", AsPosixOnWindows: "\\ foo\\  bar", AsWindowsOnPosix: "/ foo/  bar", AsWindowsOnWindows: "\\ foo\\  bar"}},
		{Input: "/ foo/ \\ bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "/ foo/ \\ bar", AsPosixOnWindows: "\\ foo\\ \\ bar", AsWindowsOnPosix: "/ foo/ / bar", AsWindowsOnWindows: "\\ foo\\ \\ bar"}},
		{Input: "/ foo/\\ \\ bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "/ foo/\\ \\ bar", AsPosixOnWindows: "\\ foo\\ \\ bar", AsWindowsOnPosix: "/ foo/ / bar", AsWindowsOnWindows: "\\ foo\\ \\ bar"}},
		{Input: "/ foo/ \\ \\ bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "/ foo/ \\ \\ bar", AsPosixOnWindows: "\\ foo\\ \\ \\ bar", AsWindowsOnPosix: "/ foo/ / / bar", AsWindowsOnWindows: "\\ foo\\ \\ \\ bar"}},
		{Input: "/ foo/\\bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "/ foo/\\bar", AsPosixOnWindows: "\\ foo\\bar", AsWindowsOnPosix: "/ foo/bar", AsWindowsOnWindows: "\\ foo\\bar"}},
		{Input: "/ foo/\\ \\\\ bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "/ foo/\\ \\\\ bar", AsPosixOnWindows: "\\ foo\\ \\ bar", AsWindowsOnPosix: "/ foo/ / bar", AsWindowsOnWindows: "\\ foo\\ \\ bar"}},
		{Input: "/foo/bar", Expect: ExpectMatrix{
			AsPosixOnPosix: "/foo/bar", AsPosixOnWindows: "\\foo\\bar", AsWindowsOnPosix: "/foo/bar", AsWindowsOnWindows: "\\foo\\bar"}},
		{Input: "/foo/bar/baz.yz", Expect: ExpectMatrix{
			AsPosixOnPosix: "/foo/bar/baz.yz", AsPosixOnWindows: "\\foo\\bar\\baz.yz", AsWindowsOnPosix: "/foo/bar/baz.yz", AsWindowsOnWindows: "\\foo\\bar\\baz.yz"}},
		{Input: "./foo/bar/baz.yz", Expect: ExpectMatrix{
			AsPosixOnPosix: "foo/bar/baz.yz", AsPosixOnWindows: "foo\\bar\\baz.yz", AsWindowsOnPosix: "foo/bar/baz.yz", AsWindowsOnWindows: "foo\\bar\\baz.yz"}},
		{Input: "/foo/bar/baz.yz/..", Expect: ExpectMatrix{
			AsPosixOnPosix: "/foo/bar", AsPosixOnWindows: "\\foo\\bar", AsWindowsOnPosix: "/foo/bar", AsWindowsOnWindows: "\\foo\\bar"}},
		{Input: "some-random_thing", Expect: ExpectMatrix{
			AsPosixOnPosix: "some-random_thing", AsPosixOnWindows: "some-random_thing", AsWindowsOnPosix: "some-random_thing", AsWindowsOnWindows: "some-random_thing"}},
		{Input: "some-random_/thing/", Expect: ExpectMatrix{
			AsPosixOnPosix: "some-random_/thing", AsPosixOnWindows: "some-random_\\thing", AsWindowsOnPosix: "some-random_/thing", AsWindowsOnWindows: "some-random_\\thing"}},
		{Input: "c:", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:", AsPosixOnWindows: "c:", AsWindowsOnPosix: "c:", AsWindowsOnWindows: "c:"}},
		{Input: "c:/", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:", AsPosixOnWindows: "c:", AsWindowsOnPosix: "c:", AsWindowsOnWindows: "c:\\"}},
		{Input: "c://", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:", AsPosixOnWindows: "c:", AsWindowsOnPosix: "c:", AsWindowsOnWindows: "c:\\"}},
		{Input: "c://hello", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:/hello", AsPosixOnWindows: "c:\\hello", AsWindowsOnPosix: "c:/hello", AsWindowsOnWindows: "c:\\hello"}},
		{Input: "c:hello", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:hello", AsPosixOnWindows: "c:hello", AsWindowsOnPosix: "c:hello", AsWindowsOnWindows: "c:hello"}},
		{Input: "c:\\", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\", AsPosixOnWindows: "c:\\", AsWindowsOnPosix: "c:", AsWindowsOnWindows: "c:\\"}},
		{Input: "\\", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "\\foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\foo", AsPosixOnWindows: "\\foo", AsWindowsOnPosix: "/foo", AsWindowsOnWindows: "\\foo"}},
		{Input: "c:\\\\", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\\\", AsPosixOnWindows: "c:\\", AsWindowsOnPosix: "c:", AsWindowsOnWindows: "c:\\"}},
		{Input: "c:\\\\hello", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\\\hello", AsPosixOnWindows: "c:\\hello", AsWindowsOnPosix: "c:/hello", AsWindowsOnWindows: "c:\\hello"}},
		{Input: "c:\\hello\\world", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\hello\\world", AsPosixOnWindows: "c:\\hello\\world", AsWindowsOnPosix: "c:/hello/world", AsWindowsOnWindows: "c:\\hello\\world"}},
		{Input: "\\\\host\\share", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share", AsPosixOnWindows: "\\host\\share", AsWindowsOnPosix: "/host/share", AsWindowsOnWindows: "\\\\host\\share"}},
		{Input: "\\\\host\\share\\foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share\\foo", AsPosixOnWindows: "\\host\\share\\foo", AsWindowsOnPosix: "/host/share/foo", AsWindowsOnWindows: "\\\\host\\share\\foo"}},
		{Input: "\\\\host\\share\\  foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share\\  foo", AsPosixOnWindows: "\\host\\share\\  foo", AsWindowsOnPosix: "/host/share/  foo", AsWindowsOnWindows: "\\\\host\\share\\  foo"}},
		{Input: "\\\\host\\share/foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share/foo", AsPosixOnWindows: "\\host\\share\\foo", AsWindowsOnPosix: "/host/share/foo", AsWindowsOnWindows: "\\\\host\\share\\foo"}},
		{Input: "//host/share/foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "/host/share/foo", AsPosixOnWindows: "\\host\\share\\foo", AsWindowsOnPosix: "/host/share/foo", AsWindowsOnWindows: "\\\\host\\share\\foo"}},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("%d-[%s]", i, testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input string, expect ExpectMatrix) {
		runTests := func(t *testing.T, expect string, input string, expectPath *Path, inputPath *Path, expectMatrixMask ExpectMatrixMask) {
			/*
				// Do not test internal representation, because they may differ based on the constructor.
				t.Run("internal repr", func(t *testing.T) {
					require.Equal(t, *expectPath, *inputPath)
				})
			*/

			t.Run("ToString_posix", func(t *testing.T) {
				if !expectMatrixMask.OnPosix {
					t.Skip("Test not targeted to Posix runtime")
				}

				if runningOnWindows {
					t.Skip("Cannot run Posix test on Windows runtime")
				}

				require.Equal(t, expect, inputPath.String())
				require.Equal(t, expect, inputPath.ToPosix())
			})

			t.Run("ToString_windows", func(t *testing.T) {
				if !expectMatrixMask.OnWindows {
					t.Skip("Test not targeted to Windows runtime")
				}

				if notRunningOnWindows {
					t.Skip("Running Windows test in Non-Windows runtime")
				}

				require.Equal(t, expect, inputPath.String())
				require.Equal(t, expect, inputPath.toWindows())
			})

			t.Run("internal_toWindows", func(t *testing.T) {
				if !expectMatrixMask.OnWindows {
					t.Skip("Test not targeted to Windows runtime")
				}

				fmt.Println(input)

				require.Equal(t, expect, inputPath.toWindows())
			})

			t.Run("matching_posix_repr", func(t *testing.T) {
				// This test fails if a Posix path contains "\\" and is checked on Windows,
				// because on Posix, "\\" is allowed and not escaped.

				if strings.Contains(input, "\\") && expectMatrixMask.AsPosix && expectMatrixMask.OnWindows {
					t.Skip("Undefined path part state due to existing backslash read as Posix on Windows")
				}

				require.Equal(t, expectPath.ToPosix(), inputPath.ToPosix())
			})

			t.Run("TextMarshalling", func(t *testing.T) {
				if strings.Contains(input, "\\") && expectMatrixMask.AsPosix && expectMatrixMask.OnWindows {
					t.Skip("Undefined path part state due to existing backslash read as Posix on Windows")
				}

				marshaled, err := inputPath.MarshalText()
				require.NoError(t, err)

				require.Equal(t, expectPath.ToPosix(), string(marshaled))
			})

			t.Run("TextUnmarshalling", func(t *testing.T) {
				if strings.Contains(input, "\\") && expectMatrixMask.AsPosix && expectMatrixMask.OnWindows {
					t.Skip("Undefined path part state due to existing backslash read as Posix on Windows")
				}

				var emptyPath = &Path{}

				marshaled, err := inputPath.MarshalText()
				require.NoError(t, err)

				err = emptyPath.UnmarshalText(marshaled)
				require.NoError(t, err)

				require.Equal(t, expectPath.ToPosix(), emptyPath.ToPosix())
			})

			escapeSequences := "\n\r\t\\"

			t.Run("JsonMarshalling", func(t *testing.T) {
				if strings.ContainsAny(input, escapeSequences) {
					t.Skip("Input contained escape sequences")
				}

				marshaled, err := json.Marshal([]*Path{inputPath})
				require.NoError(t, err)

				require.Equal(t, fmt.Sprintf(`["%s"]`, expectPath.ToPosix()), string(marshaled))
			})

			t.Run("JsonUnmarshalling", func(t *testing.T) {
				if strings.ContainsAny(input, escapeSequences) {
					t.Skip("Input contained escape sequences")
				}

				var emptyPaths []*Path

				jsonInput := fmt.Sprintf(`["%s"]`, inputPath.ToPosix())

				err := json.Unmarshal([]byte(jsonInput), &emptyPaths)
				require.NoError(t, err)

				require.Len(t, emptyPaths, 1)
				require.Equal(t, expectPath.ToPosix(), (*emptyPaths[0]).ToPosix())
			})
		}

		t.Run("AsPosixOnPosix", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPath(input)
			expectPath := NewPath(expect.AsPosixOnPosix)
			runTests(t, expect.AsPosixOnPosix, input, expectPath, inputPath, ExpectMatrixMask{
				AsPosix: true,
				OnPosix: true,
			})
		})

		t.Run("AsPosixOnWindows", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPath(input)
			expectPath := NewPathFromWindows(expect.AsPosixOnWindows)
			runTests(t, expect.AsPosixOnWindows, input, expectPath, inputPath, ExpectMatrixMask{
				AsPosix:   true,
				OnWindows: true,
			})
		})

		t.Run("AsWindowsOnPosix", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromWindows(input)
			expectPath := NewPath(expect.AsWindowsOnPosix)
			runTests(t, expect.AsWindowsOnPosix, input, expectPath, inputPath, ExpectMatrixMask{
				AsWindows: true,
				OnPosix:   true,
			})
		})

		t.Run("AsWindowsOnWindows", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromWindows(input)
			expectPath := NewPathFromWindows(expect.AsWindowsOnWindows)
			runTests(t, expect.AsWindowsOnWindows, input, expectPath, inputPath, ExpectMatrixMask{
				AsWindows: true,
				OnWindows: true,
			})
		})
	})
}

func TestNewCwd(t *testing.T) {
	// call standard library function
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
		{Input: NewPath("c:/"), Expect: ""},
		{Input: NewPath("c:/foo"), Expect: ""},
		{Input: NewPath("c://"), Expect: ""},
		{Input: NewPath("c:\\"), Expect: ""},
		{Input: NewPath("c:\\foo"), Expect: ""},
		{Input: NewPath("c:\\\\"), Expect: ""},
		{Input: NewPath("//host/share"), Expect: "/"},
		{Input: NewPath("//host/share/"), Expect: "/"},
		{Input: NewPath("//host/share/foo"), Expect: "/"},
		{Input: NewPath("\\\\host\\share"), Expect: ""},
		{Input: NewPath("\\\\host\\share\\"), Expect: ""},
		{Input: NewPath("\\\\host\\share\\foo"), Expect: ""},
		{Input: NewPathFromWindows("\\\\host\\share"), Expect: "\\\\host\\share"},
		{Input: NewPathFromWindows("\\\\host\\share\\"), Expect: "\\\\host\\share"},
		{Input: NewPathFromWindows("\\\\host\\share\\foo"), Expect: "\\\\host\\share"},
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
		{Input: []*Path{NewPath("a/b\\ whitespace/c"), NewPath("a/d")}, Expect: NewPath("../b\\ whitespace/c")},
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

		// TODO Test with n randomly generated strings

		require.Equal(t, expect, joinedStrPath)
		require.Equal(t, expect, joinedPathsPath)
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
		pathEqualsCaseSensitive := basePath.Equals(NewPath(input[1]), true)
		stringEqualsCaseSensitive := basePath.EqualsString(input[1], true)

		require.Equal(t, expect, pathEqualsCaseSensitive)
		require.Equal(t, expect, stringEqualsCaseSensitive)
	})
}

func TestPath_EqualsCaseInSensitive(t *testing.T) {
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
		pathEqualsCaseInSensitive := basePath.Equals(NewPath(strings.ToLower(input[1])), false)
		stringEqualsCaseInSensitive := basePath.EqualsString(strings.ToLower(input[1]), false)

		require.Equal(t, expect, pathEqualsCaseInSensitive)
		require.Equal(t, expect, stringEqualsCaseInSensitive)
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
