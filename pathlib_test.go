package pathlib

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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

func TestNewPath(t *testing.T) {
	cases := []TestCase[string, *Path]{
		{Input: ".", Expect: NewPath(".")},
		{Input: "..", Expect: NewPath("../")},
		{Input: "/", Expect: NewPath("/")},
		{Input: "//", Expect: NewPath("/")},
		{Input: "./", Expect: NewPath(".")},
		{Input: ".//", Expect: NewPath(".")},
		{Input: "../.", Expect: NewPath("../")},
		{Input: "../..", Expect: NewPath("../../")},
		{Input: "../../", Expect: NewPath("../../")},
		{Input: "../../.", Expect: NewPath("../../")},
		{Input: "foo", Expect: NewPath("foo")},
		{Input: "foo/", Expect: NewPath("foo")},
		{Input: "foo/.", Expect: NewPath("foo")},
		{Input: "foo/..", Expect: NewPath(".")},
		{Input: "foo/../..", Expect: NewPath("..")},
		{Input: "foo/../bar", Expect: NewPath("bar")},
		{Input: "foo/../bar/..", Expect: NewPath(".")},
		{Input: "foo/../../bar", Expect: NewPath("../bar")},
		{Input: "foo/../../bar/..", Expect: NewPath("..")},
		{Input: "foo/bar", Expect: NewPath("foo/bar")},
		{Input: "/foo/bar", Expect: NewPath("/foo/bar")},
		{Input: "/foo/bar/baz.yz", Expect: NewPath("/foo/bar/baz.yz")},
		{Input: "./foo/bar/baz.yz", Expect: NewPath("foo/bar/baz.yz")},
		{Input: "/foo/bar/baz.yz/..", Expect: NewPath("/foo/bar")},
	}

	runForResults(t, cases, func(t *testing.T, input string, expect *Path) {
		inputPath := NewPath(input)

		assert.Equal(t, *inputPath, *expect)
	})
}

func TestNewCwd(t *testing.T) {
	// call library function
	pathlibCwdPath, err := NewCwd()
	assert.NoError(t, err)

	// recreate cwd path using stdlib
	localCwd, err := os.Getwd()
	assert.NoError(t, err)
	localCwdPath := NewPath(localCwd)

	// assert
	assert.Equal(t, localCwdPath, pathlibCwdPath)
}

func TestNewHome(t *testing.T) {
	pathlibHomePath, err := NewHome()
	assert.NoError(t, err)

	localHome, err := os.UserHomeDir()
	assert.NoError(t, err)
	localHomePath := NewPath(localHome)

	assert.Equal(t, localHomePath, pathlibHomePath)
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

func runForResults[I any, E any](t *testing.T, cases []TestCase[I, E], testFunc func(t *testing.T, input I, expect E)) {
	for _, test := range cases {

		caseName := test.Name
		if strings.TrimSpace(caseName) == "" {
			caseName = fmt.Sprintf("case--\"%v\"", test.Input)
		}

		t.Run(fmt.Sprint(caseName), func(t *testing.T) {
			testFunc(t, test.Input, test.Expect)
		})
	}
}
