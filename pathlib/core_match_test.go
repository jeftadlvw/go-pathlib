package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
