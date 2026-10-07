package pathlib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenMode_Valid(t *testing.T) {
	t.Parallel()

	cases := []TestCase[OpenMode, bool]{
		{Name: "OpenRead", Input: OpenRead, Expect: true},
		{Name: "OpenWrite", Input: OpenWrite, Expect: true},
		{Name: "OpenWriteTruncate", Input: OpenWriteTruncate, Expect: true},
		{Name: "OpenReadWrite", Input: OpenReadWrite, Expect: true},
		{Name: "OpenReadWriteTruncate", Input: OpenReadWriteTruncate, Expect: true},
		{Name: "OpenAppend", Input: OpenAppend, Expect: true},
		{Name: "OpenReadAppend", Input: OpenReadAppend, Expect: true},
		{Name: "Negative", Input: OpenMode(-1), Expect: false},
		{Name: "Above the last constant", Input: OpenReadAppend + 1, Expect: false},
	}

	runForResults(t, cases, func(t *testing.T, input OpenMode, expect bool) {
		t.Helper()

		require.Equal(t, expect, input.Valid())
	})
}

func TestOpenMode_String(t *testing.T) {
	t.Parallel()

	cases := []TestCase[OpenMode, string]{
		{Name: "Zero value", Input: OpenMode(0), Expect: "OpenRead"},
		{Name: "OpenWrite", Input: OpenWrite, Expect: "OpenWrite"},
		{Name: "OpenWriteTruncate", Input: OpenWriteTruncate, Expect: "OpenWriteTruncate"},
		{Name: "OpenReadWrite", Input: OpenReadWrite, Expect: "OpenReadWrite"},
		{Name: "OpenReadWriteTruncate", Input: OpenReadWriteTruncate, Expect: "OpenReadWriteTruncate"},
		{Name: "OpenAppend", Input: OpenAppend, Expect: "OpenAppend"},
		{Name: "OpenReadAppend", Input: OpenReadAppend, Expect: "OpenReadAppend"},
		{Name: "Invalid", Input: OpenMode(99), Expect: "OpenMode(99)"},
	}

	runForResults(t, cases, func(t *testing.T, input OpenMode, expect string) {
		t.Helper()

		require.Equal(t, expect, input.String())
	})
}
