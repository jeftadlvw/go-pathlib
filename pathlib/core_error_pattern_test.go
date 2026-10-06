package pathlib

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPatternError(t *testing.T) {
	t.Parallel()

	type Expect struct {
		Kind    *PathlibError
		Pattern string
	}

	cases := []struct {
		Name   string
		Run    func(t *testing.T) error
		Expect Expect
	}{
		{
			Name: "MatchesPatternE with an empty pattern",
			Run: func(_ *testing.T) error {
				_, err := NewPath("a/b").MatchesPatternE("")
				return err
			},
			Expect: Expect{Kind: ErrEmptyPattern, Pattern: ""},
		},
		{
			Name: "MatchesPatternE with a malformed pattern",
			Run: func(_ *testing.T) error {
				_, err := NewPath("a/b").MatchesPatternE("a/[")
				return err
			},
			Expect: Expect{Kind: ErrBadPattern, Pattern: "a/["},
		},
		{
			Name: "MatchesPatternE keeps the casing of a case-insensitive pattern",
			Run: func(_ *testing.T) error {
				_, err := NewPath("a/b").MatchesPatternE("A/[", CaseInsensitive)
				return err
			},
			Expect: Expect{Kind: ErrBadPattern, Pattern: "A/["},
		},
		{
			Name: "Glob with an empty pattern",
			Run: func(t *testing.T) error {
				t.Helper()

				_, err := setupTempDir(t).Glob("")
				return err
			},
			Expect: Expect{Kind: ErrEmptyPattern, Pattern: ""},
		},
		{
			Name: "Glob with a malformed pattern",
			Run: func(t *testing.T) error {
				t.Helper()

				_, err := setupTempDir(t).Glob("src/**/[")
				return err
			},
			Expect: Expect{Kind: ErrBadPattern, Pattern: "src/**/["},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			err := c.Run(t)
			require.ErrorIs(t, err, c.Expect.Kind)

			var cause *PatternError
			require.ErrorAs(t, err, &cause)
			require.Equal(t, c.Expect.Pattern, cause.Pattern())
			require.Len(t, cause.Paths(), 1)
			require.NotErrorAs(t, err, new(*PathError))

			if c.Expect.Kind == ErrBadPattern {
				require.ErrorIs(t, err, path.ErrBadPattern)
			}
		})
	}
}
