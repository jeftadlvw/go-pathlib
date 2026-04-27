package pathlib

import (
	"encoding"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
			AsPosixOnPosix: "c:", AsPosixOnWindows: "c:", AsWindowsOnPosix: "c:/", AsWindowsOnWindows: "c:\\"}},
		{Input: "c://", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:", AsPosixOnWindows: "c:", AsWindowsOnPosix: "c:/", AsWindowsOnWindows: "c:\\"}},
		{Input: "c://hello", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:/hello", AsPosixOnWindows: "c:\\hello", AsWindowsOnPosix: "c:/hello", AsWindowsOnWindows: "c:\\hello"}},
		{Input: "c:hello", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:hello", AsPosixOnWindows: "c:hello", AsWindowsOnPosix: "c:hello", AsWindowsOnWindows: "c:hello"}},
		{Input: "c:\\", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\", AsPosixOnWindows: "c:\\", AsWindowsOnPosix: "c:/", AsWindowsOnWindows: "c:\\"}},
		{Input: "\\", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\", AsPosixOnWindows: "\\", AsWindowsOnPosix: "/", AsWindowsOnWindows: "\\"}},
		{Input: "\\foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\foo", AsPosixOnWindows: "\\foo", AsWindowsOnPosix: "/foo", AsWindowsOnWindows: "\\foo"}},
		{Input: "c:\\\\", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\\\", AsPosixOnWindows: "c:\\", AsWindowsOnPosix: "c:/", AsWindowsOnWindows: "c:\\"}},
		{Input: "c:\\\\hello", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\\\hello", AsPosixOnWindows: "c:\\hello", AsWindowsOnPosix: "c:/hello", AsWindowsOnWindows: "c:\\hello"}},
		{Input: "c:\\hello\\world", Expect: ExpectMatrix{
			AsPosixOnPosix: "c:\\hello\\world", AsPosixOnWindows: "c:\\hello\\world", AsWindowsOnPosix: "c:/hello/world", AsWindowsOnWindows: "c:\\hello\\world"}},
		{Input: "\\\\host\\share", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share", AsPosixOnWindows: "\\host\\share", AsWindowsOnPosix: "//host/share/", AsWindowsOnWindows: "\\\\host\\share"}},
		{Input: "\\\\host\\share\\foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share\\foo", AsPosixOnWindows: "\\host\\share\\foo", AsWindowsOnPosix: "//host/share/foo", AsWindowsOnWindows: "\\\\host\\share\\foo"}},
		{Input: "\\\\host\\share\\  foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share\\  foo", AsPosixOnWindows: "\\host\\share\\  foo", AsWindowsOnPosix: "//host/share/  foo", AsWindowsOnWindows: "\\\\host\\share\\  foo"}},
		{Input: "\\\\host\\share/foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "\\\\host\\share/foo", AsPosixOnWindows: "\\host\\share\\foo", AsWindowsOnPosix: "//host/share/foo", AsWindowsOnWindows: "\\\\host\\share\\foo"}},
		{Input: "//host/share/foo", Expect: ExpectMatrix{
			AsPosixOnPosix: "/host/share/foo", AsPosixOnWindows: "\\host\\share\\foo", AsWindowsOnPosix: "//host/share/foo", AsWindowsOnWindows: "\\\\host\\share\\foo"}},
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
				require.Equal(t, expect, inputPath.ToWindows())
			})

			t.Run("internal_toWindows", func(t *testing.T) {
				if !expectMatrixMask.OnWindows {
					t.Skip("Test not targeted to Windows runtime")
				}

				require.Equal(t, expect, inputPath.ToWindows())
			})

			t.Run("matching_posix_repr", func(t *testing.T) {
				// This test fails if a Posix path contains "\\" and is checked on Windows,
				// because on Posix, "\\" is allowed and not escaped.

				if expectMatrixMask.OnWindows && expectMatrixMask.AsPosix && strings.Contains(input, "\\") {
					t.Skip("Undefined path part state due to existing backslash read as Posix on Windows")
				}

				if expectMatrixMask.OnPosix && expectMatrixMask.AsWindows && inputPath.isWindowsAnchoredPath() {
					t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
				}

				require.Equal(t, expectPath.ToPosix(), inputPath.ToPosix())
			})

			t.Run("TextMarshalling", func(t *testing.T) {
				if expectMatrixMask.OnWindows && expectMatrixMask.AsPosix && strings.Contains(input, "\\") {
					t.Skip("Undefined path part state due to existing backslash read as Posix on Windows")
				}

				if expectMatrixMask.OnPosix && expectMatrixMask.AsWindows && inputPath.isWindowsAnchoredPath() {
					t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
				}

				marshaled, err := inputPath.MarshalText()
				require.NoError(t, err)

				require.Equal(t, expectPath.ToPosix(), string(marshaled))
			})

			t.Run("TextUnmarshalling", func(t *testing.T) {
				if strings.Contains(input, "\\") && expectMatrixMask.AsPosix {
					t.Skip("Posix paths with literal backslashes cannot round-trip through text unmarshalling")
				}

				if expectMatrixMask.OnPosix && expectMatrixMask.AsWindows && inputPath.isWindowsAnchoredPath() {
					t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
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

				if expectMatrixMask.OnPosix && expectMatrixMask.AsWindows && inputPath.isWindowsAnchoredPath() {
					t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
				}

				marshaled, err := json.Marshal([]*Path{inputPath})
				require.NoError(t, err)

				require.Equal(t, fmt.Sprintf(`["%s"]`, expectPath.ToPosix()), string(marshaled))
			})

			t.Run("JsonUnmarshalling", func(t *testing.T) {
				if strings.ContainsAny(input, escapeSequences) {
					t.Skip("Input contained escape sequences")
				}

				if expectMatrixMask.OnPosix && expectMatrixMask.AsWindows && inputPath.isWindowsAnchoredPath() {
					t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
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

			inputPath := NewPathFromPosix(input)
			expectPath := NewPathFromPosix(expect.AsPosixOnPosix)
			runTests(t, expect.AsPosixOnPosix, input, expectPath, inputPath, ExpectMatrixMask{
				AsPosix: true,
				OnPosix: true,
			})
		})

		t.Run("AsPosixOnWindows", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromPosix(input)
			expectPath := NewPathFromWindows(expect.AsPosixOnWindows)
			runTests(t, expect.AsPosixOnWindows, input, expectPath, inputPath, ExpectMatrixMask{
				AsPosix:   true,
				OnWindows: true,
			})
		})

		t.Run("AsWindowsOnPosix", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromWindows(input)
			expectPath := NewPathFromPosix(expect.AsWindowsOnPosix)
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
		{Input: []*Path{NewPath("a/d"), NewPath("a/b\\ whitespace/c")}, Expect: NewPath("../b\\ whitespace/c")},
	}

	for i := range cases {
		cases[i].Name = fmt.Sprintf("[%d]", i+1)
	}

	runForResultsE(t, cases, func(t *testing.T, input []*Path, expect *Path, expectError bool) {
		require.Equal(t, len(input), 2)

		basePath := input[0]
		originalPath := input[1]
		relativePath, err := originalPath.RelativeTo(basePath)

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

	runForResultsE(t, cases, func(t *testing.T, input []*Path, expect *Path, expectError bool) {
		require.Equal(t, len(input), 2)

		base := input[0]
		other := input[1]
		absolutePath, err := base.AbsoluteTo(other)
		require.Equal(t, expectError, err != nil)

		if !expectError {
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

func TestPath_WithName(t *testing.T) {
	type Input struct {
		Original string
		NewName  string
	}

	cases := []TestCase[Input, *Path]{
		{Input: Input{Original: "", NewName: "foo"}, Expect: NewPath("foo")},
		{Input: Input{Original: "/", NewName: "foo"}, Expect: NewPath("/foo")},
		{Input: Input{Original: "../", NewName: "foo"}, Expect: NewPath("foo")},
		{Input: Input{Original: "../..", NewName: "foo"}, Expect: NewPath("../foo")},
		{Input: Input{Original: "foo/bar", NewName: "foo"}, Expect: NewPath("foo/foo")},
		{Input: Input{Original: "/foo/bar", NewName: "foo"}, Expect: NewPath("/foo/foo")},
		{Input: Input{Original: "foo/file.txt", NewName: "bar.txt"}, Expect: NewPath("foo/bar.txt")},
		{Input: Input{Original: "foo/.txt", NewName: ".json"}, Expect: NewPath("foo/.json")},
		{Input: Input{Original: "/foo/.txt", NewName: ".json"}, Expect: NewPath("/foo/.json")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[original:%s__newname:%s]", testCase.Input.Original, testCase.Input.NewName)
	}

	runForResults(t, cases, func(t *testing.T, input Input, expect *Path) {
		path := NewPath(input.Original)
		changedName := path.WithName(input.NewName)

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

// platformNativeUNC returns the expected platform-native representation of a UNC anchor.
// On Posix it returns forward slashes, on Windows backslashes.
func platformNativeUNC(posixForm string) string {
	if runningOnWindows {
		return toWindowsSeparators(posixForm)
	}
	return posixForm
}

// onWindows returns windowsVal on Windows, posixVal on other platforms.
func onWindows[T any](posixVal, windowsVal T) T {
	if runningOnWindows {
		return windowsVal
	}
	return posixVal
}

func runForResultsE[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expect E, expectError bool)) {
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

func runForResults[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expectError E)) {
	runForResultsE(t, cases, func(t *testing.T, input I, expect E, error bool) {
		testFunc(t, input, expect)
	})
}

func TestMatchesPatternE(t *testing.T) {
	type matchInput struct {
		Path    string
		Pattern string
		Opts    []CompareOption
	}

	type matchExpect struct {
		Match bool
		Error bool
	}

	cases := []struct {
		Name   string
		Input  matchInput
		Expect matchExpect
	}{
		{"simple match", matchInput{"foo.go", "*.go", nil}, matchExpect{true, false}},
		{"simple no match", matchInput{"foo.go", "*.txt", nil}, matchExpect{false, false}},
		{"exact match", matchInput{"foo", "foo", nil}, matchExpect{true, false}},
		{"double asterisk", matchInput{"a/b/c.go", "**/*.go", nil}, matchExpect{true, false}},
		{"double asterisk matches all", matchInput{"a/b/c", "**", nil}, matchExpect{true, false}},
		{"single asterisk no cross slash", matchInput{"a/b.go", "*.go", nil}, matchExpect{false, false}},
		{"empty pattern", matchInput{"foo", "", nil}, matchExpect{false, true}},
		{"empty pattern on dot path", matchInput{".", "", nil}, matchExpect{false, true}},
		{"bad pattern syntax", matchInput{"foo", "[", nil}, matchExpect{false, true}},

		// case-sensitive (default)
		{"case sensitive default", matchInput{"Foo.GO", "*.go", nil}, matchExpect{false, false}},
		{"case sensitive explicit", matchInput{"Foo.GO", "*.go", []CompareOption{CaseSensitive}}, matchExpect{false, false}},

		// case-insensitive
		{"case insensitive match", matchInput{"Foo.GO", "*.go", []CompareOption{CaseInsensitive}}, matchExpect{true, false}},
		{"case insensitive exact", matchInput{"FOO", "foo", []CompareOption{CaseInsensitive}}, matchExpect{true, false}},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			p := NewPath(tc.Input.Path)
			match, err := p.MatchesPatternE(tc.Input.Pattern, tc.Input.Opts...)
			matchNoErr := p.MatchesPattern(tc.Input.Pattern, tc.Input.Opts...)

			require.Equal(t, match, matchNoErr)

			if tc.Expect.Error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.Expect.Match, match)
		})
	}
}
