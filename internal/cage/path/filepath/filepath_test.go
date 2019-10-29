// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package filepath_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

type FilepathSuite struct {
	suite.Suite
}

func (s *FilepathSuite) TestFileAncestor() {
	t := s.T()

	t.Run("fail on invalid descendant", func(t *testing.T) {
		descend := "/path/to/descendant"
		root := "/other/path"
		actual, err := cage_filepath.FileAncestor(descend, root)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a descendant")
		require.Len(t, actual, 0)
	})

	t.Run("find descendants", func(t *testing.T) {
		table := map[string][]string{
			"/path/to/proj/one/two/three": {
				"/path/to/proj",
				"/path/to/proj/one",
				"/path/to/proj/one/two",
			},
			"/path/to/proj/one/four": {
				"/path/to/proj",
				"/path/to/proj/one",
			},
			"/path/to/proj/one/four/five/six": {
				"/path/to/proj",
				"/path/to/proj/one",
				"/path/to/proj/one/four",
				"/path/to/proj/one/four/five",
			},
		}
		root := "/path/to/proj"
		for descend, expected := range table {
			actual, err := cage_filepath.FileAncestor(descend, root)
			cage_strings.SortStable(actual)
			require.NoError(t, err)
			require.Exactly(t, expected, actual)
		}
	})
}

func (s *FilepathSuite) TestAppend() {
	t := s.T()

	final, err := cage_filepath.Append("/a/b/c/d", "../e")
	require.EqualError(t, err, "path [../e] cannot be appended, escapes prefix [/a/b/c/d]")
	require.Exactly(t, "", final)

	final, err = cage_filepath.Append("/a/b/c/d", "e/../../f")
	require.EqualError(t, err, "path [e/../../f] cannot be appended, escapes prefix [/a/b/c/d]")
	require.Exactly(t, "", final)

	final, err = cage_filepath.Append("/a/b/c/d", "e/../f")
	require.NoError(t, err)
	require.Exactly(t, "/a/b/c/d/f", final)

	final, err = cage_filepath.Append("/a/b/c/d", "/e")
	require.NoError(t, err)
	require.Exactly(t, "/a/b/c/d/e", final)

	final, err = cage_filepath.Append("/a/b/c/d", "e/f")
	require.NoError(t, err)
	require.Exactly(t, "/a/b/c/d/e/f", final)
}

func (s *FilepathSuite) TestAncesorDirs() {
	t := s.T()

	require.Exactly(
		t,
		[]string{
			"/a/b/c/d",
			"/a/b/c",
			"/a/b",
		},
		cage_filepath.AncestorDirs("/a/b/c/d/e", "/a/b"),
	)
	require.Exactly(
		t,
		[]string{
			"/a/b/c/d",
			"/a/b/c",
			"/a/b",
			"/a",
			"/",
		},
		cage_filepath.AncestorDirs("/a/b/c/d/e", "invalid"),
	)
	require.Exactly(
		t,
		[]string(nil),
		cage_filepath.AncestorDirs("/a/b/c/d/e", "/a/b/c/d/e"),
	)
}

func (s *FilepathSuite) TestPathToSafeFilename() {
	t := s.T()

	require.Exactly(t, "a-b-c", cage_filepath.PathToSafeFilename("/a/b/c"))
	require.Exactly(t, "a-b-c", cage_filepath.PathToSafeFilename(`\a\b\c`))
	require.Exactly(t, "a-a-b-b-c-c", cage_filepath.PathToSafeFilename("/a a/b b/c c"))
}

func TestFilepathSuite(t *testing.T) {
	suite.Run(t, new(FilepathSuite))
}
