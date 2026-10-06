package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobOptionFilterType(t *testing.T) {
	t.Parallel()

	type Expect struct {
		Valid  bool
		String string
	}

	cases := []TestCase[GlobOptionFilterType, Expect]{
		{Name: "all", Input: GlobOptionFilterAll, Expect: Expect{Valid: true, String: "all"}},
		{Name: "files", Input: GlobOptionFilterFiles, Expect: Expect{Valid: true, String: "files"}},
		{Name: "directories", Input: GlobOptionFilterDirectories, Expect: Expect{Valid: true, String: "directories"}},
		{Name: "above range", Input: 3, Expect: Expect{Valid: false, String: "GlobOptionFilterType(3)"}},
		{Name: "negative", Input: -1, Expect: Expect{Valid: false, String: "GlobOptionFilterType(-1)"}},
	}

	runForResults(t, cases, func(t *testing.T, input GlobOptionFilterType, expect Expect) {
		t.Helper()

		require.Equal(t, expect.Valid, input.Valid())
		require.Equal(t, expect.String, input.String())
	})
}
