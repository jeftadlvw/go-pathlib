package pathlib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

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
