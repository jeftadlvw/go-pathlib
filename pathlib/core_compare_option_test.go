package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareOption_ZeroValueIsCaseSensitive(t *testing.T) {
	t.Parallel()

	var zero CompareOption
	require.Equal(t, CaseSensitive, zero)

	upper := NewPathFromPosix("Dir/File.TXT")
	lower := NewPathFromPosix("dir/file.txt")

	cases := []TestCase[[]CompareOption, bool]{
		{Name: "no option", Input: nil, Expect: false},
		{Name: "zero value", Input: []CompareOption{zero}, Expect: false},
		{Name: "CaseSensitive", Input: []CompareOption{CaseSensitive}, Expect: false},
		{Name: "CaseInsensitive", Input: []CompareOption{CaseInsensitive}, Expect: true},
	}

	runForResults(t, cases, func(t *testing.T, opts []CompareOption, expect bool) {
		t.Helper()

		require.Equal(t, expect, upper.Equals(lower, opts...), "Equals")
		require.Equal(t, expect, upper.EqualsString(lower.ToPosix(), opts...), "EqualsString")
		require.Equal(t, expect, upper.MatchesPattern("dir/*.txt", opts...), "MatchesPattern")

		// Exact casing always matches.
		require.True(t, upper.Equals(upper.Copy(), opts...))
		require.True(t, upper.MatchesPattern("Dir/*.TXT", opts...))
	})
}

func TestCompareOption(t *testing.T) {
	t.Parallel()

	type Expect struct {
		Valid  bool
		String string
	}

	cases := []TestCase[CompareOption, Expect]{
		{Name: "case-sensitive", Input: CaseSensitive, Expect: Expect{Valid: true, String: "case-sensitive"}},
		{Name: "case-insensitive", Input: CaseInsensitive, Expect: Expect{Valid: true, String: "case-insensitive"}},
		{Name: "above range", Input: 2, Expect: Expect{Valid: false, String: "CompareOption(2)"}},
		{Name: "negative", Input: -1, Expect: Expect{Valid: false, String: "CompareOption(-1)"}},
	}

	runForResults(t, cases, func(t *testing.T, input CompareOption, expect Expect) {
		t.Helper()

		require.Equal(t, expect.Valid, input.Valid())
		require.Equal(t, expect.String, input.String())
	})
}
