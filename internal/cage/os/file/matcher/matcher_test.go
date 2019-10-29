// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package matcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_file_matcher "github.com/codeactual/transplant/internal/cage/os/file/matcher"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

type MatcherSuite struct {
	suite.Suite
}

func (s *MatcherSuite) TestPopulatedDir() {
	t := s.T()

	_, fixturePath := testkit_file.FixturePath(t, "populated_dir")

	require.NoError(t, os.MkdirAll(filepath.Join(fixturePath, "empty"), 0755))

	finder := cage_file.NewFinder().
		Dir(filepath.Join(fixturePath, "empty")).
		DirMatcher(cage_file_matcher.PopulatedDir)

	dirs, dirsErr := finder.GetDirnameMatches()
	require.NoError(t, dirsErr)

	require.Exactly(t, 0, dirs.Len())

	files, filesErr := finder.GetFilenameMatches()
	require.NoError(t, filesErr)

	require.Exactly(t, 0, files.Len())

	finder = cage_file.NewFinder().
		Dir(filepath.Join(fixturePath, "non_empty")).
		DirMatcher(cage_file_matcher.PopulatedDir)

	dirs, dirsErr = finder.GetDirnameMatches()
	require.NoError(t, dirsErr)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "non_empty"),
			filepath.Join(fixturePath, "non_empty", "a"),
			filepath.Join(fixturePath, "non_empty", "a", "b"),
			filepath.Join(fixturePath, "non_empty", "a", "b", "c"),
		},
		dirs.Slice(),
	)

	files, filesErr = finder.GetFilenameMatches()
	require.NoError(t, filesErr)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "non_empty", "a", "a0"),
			filepath.Join(fixturePath, "non_empty", "a", "a1"),
			filepath.Join(fixturePath, "non_empty", "a", "b", "b0"),
			filepath.Join(fixturePath, "non_empty", "a", "b", "b1"),
			filepath.Join(fixturePath, "non_empty", "a", "b", "c", "c0"),
			filepath.Join(fixturePath, "non_empty", "a", "b", "c", "c1"),
			filepath.Join(fixturePath, "non_empty", "root0"),
			filepath.Join(fixturePath, "non_empty", "root1"),
		},
		files.Slice(),
	)
}

func TestMatcherSuite(t *testing.T) {
	suite.Run(t, new(MatcherSuite))
}
