package pathlib

import (
	"fmt"
	"strings"
	"testing"
)

// TestCase is a named case of a table that maps an input to an expected
// result.
type TestCase[I any, E any] struct {
	// Name names the subtest. An empty name derives it from the input.
	Name string

	// Input is the input of the case.
	Input I

	// Expect is the expected result of the case.
	Expect E

	// Error reports whether the case expects an error.
	Error bool
}

// runForResultsE runs testFunc for every case in a parallel subtest, passing
// whether the case expects an error.
func runForResultsE[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expect E, expectError bool)) {
	t.Helper()

	for _, test := range cases {
		caseName := test.Name
		if strings.TrimSpace(caseName) == "" {
			caseName = fmt.Sprintf("case--\"%v\"", test.Input)
		}

		t.Run(caseName, func(t *testing.T) {
			t.Parallel()

			testFunc(t, test.Input, test.Expect, test.Error)
		})
	}
}

// runForResults runs testFunc for every case in a parallel subtest.
func runForResults[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expect E)) {
	t.Helper()

	runForResultsE(t, cases, func(t *testing.T, input I, expect E, _ bool) {
		t.Helper()

		testFunc(t, input, expect)
	})
}
