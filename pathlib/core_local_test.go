package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPath_IsLocal(t *testing.T) {
	t.Parallel()

	cases := []TestCase[*Path, bool]{
		{Name: "Name", Input: NewPathFromPosix("a"), Expect: true},
		{Name: "Nested names", Input: NewPathFromPosix("a/b/c.txt"), Expect: true},
		{Name: "Current directory", Input: NewPathFromPosix("."), Expect: true},
		{Name: "Empty string is the current directory", Input: NewPathFromPosix(""), Expect: true},
		{Name: "Resolved parent stays inside", Input: NewPathFromPosix("a/../b"), Expect: true},
		{Name: "Dot name", Input: NewPathFromPosix(".hidden"), Expect: true},
		{Name: "Name starting with dots", Input: NewPathFromPosix("..a"), Expect: true},
		{Name: "Name with spaces", Input: NewPathFromPosix("my file.txt"), Expect: true},
		{Name: "Backslash separates names inside", Input: NewPathFromPosix(`a\b`), Expect: true},

		{Name: "Parent", Input: NewPathFromPosix(".."), Expect: false},
		{Name: "Parent prefix", Input: NewPathFromPosix("../a"), Expect: false},
		{Name: "Resolved parent leaves", Input: NewPathFromPosix("a/../../b"), Expect: false},
		{Name: "Posix absolute", Input: NewPathFromPosix("/a"), Expect: false},
		{Name: "Posix root", Input: NewPathFromPosix("/"), Expect: false},

		{Name: "Backslash parent", Input: NewPathFromPosix(`..\a`), Expect: false},
		{Name: "Backslash parent in a name", Input: NewPathFromPosix(`a\..\..\b`), Expect: false},
		{Name: "Leading backslash", Input: NewPathFromPosix(`\a`), Expect: false},
		{Name: "Posix UNC string", Input: NewPathFromPosix(`\\host\share\a`), Expect: false},

		{Name: "Windows rooted volume", Input: NewPathFromWindows(`C:\a`), Expect: false},
		{Name: "Windows drive-relative volume", Input: NewPathFromWindows(`C:a`), Expect: false},
		{Name: "Windows UNC", Input: NewPathFromWindows(`\\host\share\a`), Expect: false},
		{Name: "Windows rooted in the current drive", Input: NewPathFromWindows(`\a`), Expect: false},
		{Name: "Windows relative", Input: NewPathFromWindows(`a\b`), Expect: true},
		{Name: "Windows parent", Input: NewPathFromWindows(`a\..\..`), Expect: false},
		{Name: "Volume string parsed as Posix", Input: NewPathFromPosix("C:/a"), Expect: false},
		{Name: "Colon in a name", Input: NewPathFromPosix("a/b:c"), Expect: false},

		{Name: "Reserved name", Input: NewPathFromPosix("NUL"), Expect: false},
		{Name: "Reserved name in lower case", Input: NewPathFromPosix("a/con"), Expect: false},
		{Name: "Reserved name with extension", Input: NewPathFromPosix("nul.txt"), Expect: false},
		{Name: "Reserved name with trailing space", Input: NewPathFromPosix("aux "), Expect: false},
		{Name: "Reserved name inside", Input: NewPathFromPosix("a/prn/b"), Expect: false},
		{Name: "Reserved numbered name", Input: NewPathFromPosix("COM1"), Expect: false},
		{Name: "Reserved numbered name with superscript", Input: NewPathFromPosix("lpt\u00b3"), Expect: false},
		{Name: "Reserved console name", Input: NewPathFromPosix("conin$"), Expect: false},
		{Name: "Name starting with a reserved name", Input: NewPathFromPosix("console"), Expect: true},
		{Name: "Numbered name with zero", Input: NewPathFromPosix("COM0"), Expect: true},
		{Name: "Numbered name with two digits", Input: NewPathFromPosix("COM10"), Expect: true},
	}

	runForResults(t, cases, func(t *testing.T, input *Path, expect bool) {
		t.Helper()

		require.Equal(t, expect, input.IsLocal())
	})
}
