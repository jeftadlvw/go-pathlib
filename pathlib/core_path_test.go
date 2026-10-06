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

func TestPathInterfaceImplementations(t *testing.T) {
	t.Parallel()

	path := NewPath("foo")

	t.Run("TextMarshaler", func(t *testing.T) {
		t.Parallel()

		_, ok := any(path).(encoding.TextMarshaler)
		require.True(t, ok)
	})

	t.Run("TextUnMarshaler", func(t *testing.T) {
		t.Parallel()

		_, ok := any(path).(encoding.TextUnmarshaler)
		require.True(t, ok)
	})
}

func TestPathInputOutputDisplay(t *testing.T) {
	t.Parallel()

	type ExpectMatrix struct {
		AsPosixOnPosix     string
		AsPosixOnWindows   string
		AsWindowsOnPosix   string
		AsWindowsOnWindows string
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
		t.Helper()

		t.Run("AsPosixOnPosix", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromPosix(input)
			expectPath := NewPathFromPosix(expect.AsPosixOnPosix)
			runDisplayTests(t, expect.AsPosixOnPosix, input, expectPath, inputPath, displayMask{
				AsPosix: true,
				OnPosix: true,
			})
		})

		t.Run("AsPosixOnWindows", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromPosix(input)
			expectPath := NewPathFromWindows(expect.AsPosixOnWindows)
			runDisplayTests(t, expect.AsPosixOnWindows, input, expectPath, inputPath, displayMask{
				AsPosix:   true,
				OnWindows: true,
			})
		})

		t.Run("AsWindowsOnPosix", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromWindows(input)
			expectPath := NewPathFromPosix(expect.AsWindowsOnPosix)
			runDisplayTests(t, expect.AsWindowsOnPosix, input, expectPath, inputPath, displayMask{
				AsWindows: true,
				OnPosix:   true,
			})
		})

		t.Run("AsWindowsOnWindows", func(t *testing.T) {
			t.Parallel()

			inputPath := NewPathFromWindows(input)
			expectPath := NewPathFromWindows(expect.AsWindowsOnWindows)
			runDisplayTests(t, expect.AsWindowsOnWindows, input, expectPath, inputPath, displayMask{
				AsWindows: true,
				OnWindows: true,
			})
		})
	})
}

func TestNewCwd(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	pathlibHomePath, err := NewHome()
	require.NoError(t, err)

	localHome, err := os.UserHomeDir()
	require.NoError(t, err)
	localHomePath := NewPath(localHome)

	require.Equal(t, localHomePath, pathlibHomePath)
}

func TestNewConfig(t *testing.T) {
	t.Parallel()

	pathlibConfigPath, err := NewConfig()
	require.NoError(t, err)

	localConfig, err := os.UserConfigDir()
	require.NoError(t, err)
	localConfigPath := NewPath(localConfig)

	require.Equal(t, localConfigPath, pathlibConfigPath)
}

func TestNewCache(t *testing.T) {
	t.Parallel()

	pathlibCachePath, err := NewCache()
	require.NoError(t, err)

	localCache, err := os.UserCacheDir()
	require.NoError(t, err)
	localCachePath := NewPath(localCache)

	require.Equal(t, localCachePath, pathlibCachePath)
}

func TestPathFromParts(t *testing.T) {
	t.Parallel()

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
		t.Helper()

		require.Equal(t, *expect, *PathFromParts(input...))
	})
}

func TestPath_Copy(t *testing.T) {
	t.Parallel()

	cases := []TestCase[*Path, any]{
		{Input: NewPath("foo/bar")},
		{Input: NewPath("../foo/bar")},
		{Input: NewPath("..")},
		{Input: NewPath("/foo")},
	}

	for i, testCase := range cases {
		cases[i].Name = fmt.Sprintf("[%s]", testCase.Input)
	}

	runForResults(t, cases, func(t *testing.T, input *Path, _ any) {
		t.Helper()

		pointerCopy := input
		copiedPath := input.Copy()

		// compare pointers
		require.Same(t, input, pointerCopy)
		require.NotSame(t, input, copiedPath)

		// ensure copied path has same contents as original
		require.Equal(t, input, copiedPath)
	})
}

// platformNativeUNC returns the expected platform-native representation of a UNC anchor.
// On Posix it returns forward slashes, on Windows backslashes.

// displayMask selects the representation and runtime a display test targets.
type displayMask struct {
	AsPosix   bool
	AsWindows bool
	OnPosix   bool
	OnWindows bool
}

// runDisplayTests checks that inputPath, built from input, displays and encodes
// as expect, and as expectPath does. The internal representations are not
// compared, because they differ between constructors.
func runDisplayTests(t *testing.T, expect, input string, expectPath, inputPath *Path, mask displayMask) {
	t.Helper()

	runDisplayStringTests(t, expect, inputPath, mask)

	skips := newDisplaySkips(input, inputPath, mask)
	runDisplayTextTests(t, expectPath, inputPath, skips)
	runDisplayJSONTests(t, expectPath, inputPath, skips)
}

// runDisplayStringTests checks the string forms of inputPath against expect.
func runDisplayStringTests(t *testing.T, expect string, inputPath *Path, mask displayMask) {
	t.Helper()

	t.Run("ToString_posix", func(t *testing.T) {
		t.Parallel()

		if !mask.OnPosix {
			t.Skip("Test not targeted to Posix runtime")
		}

		if runningOnWindows {
			t.Skip("Cannot run Posix test on Windows runtime")
		}

		require.Equal(t, expect, inputPath.String())
		require.Equal(t, expect, inputPath.ToPosix())
	})

	t.Run("ToString_windows", func(t *testing.T) {
		t.Parallel()

		if !mask.OnWindows {
			t.Skip("Test not targeted to Windows runtime")
		}

		if notRunningOnWindows {
			t.Skip("Running Windows test in Non-Windows runtime")
		}

		require.Equal(t, expect, inputPath.String())
		require.Equal(t, expect, inputPath.ToWindows())
	})

	t.Run("internal_toWindows", func(t *testing.T) {
		t.Parallel()

		if !mask.OnWindows {
			t.Skip("Test not targeted to Windows runtime")
		}

		require.Equal(t, expect, inputPath.ToWindows())
	})
}

// displaySkips holds the reasons to skip encoding tests of a display case.
type displaySkips struct {
	// backslashReadAsPosixOnWindows is set for a Posix path with a backslash on
	// Windows. On Posix, "\\" is allowed and not escaped, so its state is undefined.
	backslashReadAsPosixOnWindows bool
	// anchorNotPosix is set for a Windows anchor that Posix cannot represent.
	anchorNotPosix bool
	// backslashAsPosix is set for a Posix path with a literal backslash.
	backslashAsPosix bool
	// hasEscapeSequences is set for an input with escape sequences.
	hasEscapeSequences bool
}

// newDisplaySkips returns the reasons to skip encoding tests of inputPath,
// built from input for mask.
func newDisplaySkips(input string, inputPath *Path, mask displayMask) displaySkips {
	return displaySkips{
		backslashReadAsPosixOnWindows: mask.OnWindows && mask.AsPosix && strings.Contains(input, "\\"),
		anchorNotPosix:                mask.OnPosix && mask.AsWindows && inputPath.isWindowsAnchoredPath(),
		backslashAsPosix:              strings.Contains(input, "\\") && mask.AsPosix,
		hasEscapeSequences:            strings.ContainsAny(input, "\n\r\t\\"),
	}
}

// runDisplayTextTests checks that inputPath matches and text encodes as
// expectPath does.
func runDisplayTextTests(t *testing.T, expectPath, inputPath *Path, skips displaySkips) {
	t.Helper()

	t.Run("matching_posix_repr", func(t *testing.T) {
		t.Parallel()

		if skips.backslashReadAsPosixOnWindows {
			t.Skip("Undefined path part state due to existing backslash read as Posix on Windows")
		}

		if skips.anchorNotPosix {
			t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
		}

		require.Equal(t, expectPath.ToPosix(), inputPath.ToPosix())
	})

	t.Run("TextMarshalling", func(t *testing.T) {
		t.Parallel()

		if skips.backslashReadAsPosixOnWindows {
			t.Skip("Undefined path part state due to existing backslash read as Posix on Windows")
		}

		if skips.anchorNotPosix {
			t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
		}

		marshaled, err := inputPath.MarshalText()
		require.NoError(t, err)

		require.Equal(t, expectPath.ToPosix(), string(marshaled))
	})

	t.Run("TextUnmarshalling", func(t *testing.T) {
		t.Parallel()

		if skips.backslashAsPosix {
			t.Skip("Posix paths with literal backslashes cannot round-trip through text unmarshalling")
		}

		if skips.anchorNotPosix {
			t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
		}

		var emptyPath = &Path{}

		marshaled, err := inputPath.MarshalText()
		require.NoError(t, err)

		err = emptyPath.UnmarshalText(marshaled)
		require.NoError(t, err)

		require.Equal(t, expectPath.ToPosix(), emptyPath.ToPosix())
	})
}

// runDisplayJSONTests checks that inputPath JSON encodes as expectPath does.
func runDisplayJSONTests(t *testing.T, expectPath, inputPath *Path, skips displaySkips) {
	t.Helper()

	t.Run("JsonMarshalling", func(t *testing.T) {
		t.Parallel()

		if skips.hasEscapeSequences {
			t.Skip("Input contained escape sequences")
		}

		if skips.anchorNotPosix {
			t.Skip("NewPathFromPosix cannot represent Windows anchors for posix comparison")
		}

		marshaled, err := json.Marshal([]*Path{inputPath})
		require.NoError(t, err)

		require.Equal(t, fmt.Sprintf(`["%s"]`, expectPath.ToPosix()), string(marshaled))
	})

	t.Run("JsonUnmarshalling", func(t *testing.T) {
		t.Parallel()

		if skips.hasEscapeSequences {
			t.Skip("Input contained escape sequences")
		}

		if skips.anchorNotPosix {
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
