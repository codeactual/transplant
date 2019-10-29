// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_file_matcher "github.com/codeactual/transplant/internal/cage/os/file/matcher"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

type FinderSuite struct {
	suite.Suite
}

func (s *FinderSuite) TestBaseline() {
	t := s.T()

	_, fixturePath := testkit_file.FixturePath(t, "finder", "baseline")
	f := cage_file.NewFinder()

	require.NoError(t, os.MkdirAll(filepath.Join(fixturePath, "both_types", "empty"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(fixturePath, "empty", "empty_sub"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(fixturePath, "md_only", "empty"), 0755))

	// Find all dirs.
	// Find all files in those dirs.

	dirs, err := f.Dir(fixturePath).DirMatcher().GetDirnameMatches()
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			fixturePath,
			filepath.Join(fixturePath, "both_types"),
			filepath.Join(fixturePath, "both_types", "empty"),
			filepath.Join(fixturePath, "empty"),
			filepath.Join(fixturePath, "empty", "empty_sub"),
			filepath.Join(fixturePath, "go_only"),
			filepath.Join(fixturePath, "go_only", "go_only_sub"),
			filepath.Join(fixturePath, "md_only"),
			filepath.Join(fixturePath, "md_only", "empty"),
			filepath.Join(fixturePath, "md_only", "md_only_sub"),
		},
		dirs.Slice(),
	)

	files, err := f.GetFilenameMatches()
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "baseline.go"),
			filepath.Join(fixturePath, "baseline.md"),
			filepath.Join(fixturePath, "both_types", "both_types.go"),
			filepath.Join(fixturePath, "both_types", "both_types.md"),
			filepath.Join(fixturePath, "go_only", "go_only.go"),
			filepath.Join(fixturePath, "go_only", "go_only_sub", "go_only_sub.go"),
			filepath.Join(fixturePath, "md_only", "md_only.md"),
			filepath.Join(fixturePath, "md_only", "md_only_sub", "md_only_sub.md"),
		},
		files.Slice(),
	)

	// Find all dirs with at least one Go file.
	// Find all Go files in those dirs.

	dirs, err = f.Dir(fixturePath).DirMatcher(cage_file_matcher.GoDir).GetDirnameMatches()
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			fixturePath,
			filepath.Join(fixturePath, "both_types"),
			filepath.Join(fixturePath, "go_only"),
			filepath.Join(fixturePath, "go_only", "go_only_sub"),
		},
		dirs.Slice(),
	)

	files, err = f.GetFilenameMatches(cage_file_matcher.GoFile)
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "baseline.go"),
			filepath.Join(fixturePath, "both_types", "both_types.go"),
			filepath.Join(fixturePath, "go_only", "go_only.go"),
			filepath.Join(fixturePath, "go_only", "go_only_sub", "go_only_sub.go"),
		},
		files.Slice(),
	)

	// Find all dirs with at least one Go file (under both_types/ and go_only/).
	// Find all Go files in those dirs.

	f.Clear()
	dirs, err = f.GetDirnameMatches()
	require.NoError(t, err)
	require.Zero(t, dirs.Len())

	searchDirs := []string{
		filepath.Join(fixturePath, "both_types"),
		filepath.Join(fixturePath, "go_only"),
	}

	dirs, err = f.Dir(searchDirs...).DirMatcher(cage_file_matcher.GoDir).GetDirnameMatches()
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "both_types"),
			filepath.Join(fixturePath, "go_only"),
			filepath.Join(fixturePath, "go_only", "go_only_sub"),
		},
		dirs.Slice(),
	)

	files, err = f.GetFilenameMatches(cage_file_matcher.GoFile)
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "both_types", "both_types.go"),
			filepath.Join(fixturePath, "go_only", "go_only.go"),
			filepath.Join(fixturePath, "go_only", "go_only_sub", "go_only_sub.go"),
		},
		files.Slice(),
	)

	// Find all dirs with zero Go files.
	// Find all non-Go files in those dirs.

	f.Clear()
	dirs, err = f.GetDirnameMatches()
	require.NoError(t, err)
	require.Zero(t, dirs.Len())

	dirs, err = f.Dir(fixturePath).DirMatcher(
		cage_file_matcher.InvertDir(cage_file_matcher.GoDir),
	).GetDirnameMatches()
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "both_types", "empty"),
			filepath.Join(fixturePath, "empty"),
			filepath.Join(fixturePath, "empty", "empty_sub"),
			filepath.Join(fixturePath, "md_only"),
			filepath.Join(fixturePath, "md_only", "empty"),
			filepath.Join(fixturePath, "md_only", "md_only_sub"),
		},
		dirs.Slice(),
	)

	files, err = f.GetFilenameMatches(cage_file_matcher.InvertFile(cage_file_matcher.GoFile))
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "md_only", "md_only.md"),
			filepath.Join(fixturePath, "md_only", "md_only_sub", "md_only_sub.md"),
		},
		files.Slice(),
	)

	// Find all dirs with at least one file

	f.Clear()
	dirs, err = f.GetDirnameMatches()
	require.NoError(t, err)
	require.Zero(t, dirs.Len())

	dirs, err = f.Dir(fixturePath).DirMatcher(cage_file_matcher.DirWithFile).GetDirnameMatches()
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			fixturePath,
			filepath.Join(fixturePath, "both_types"),
			filepath.Join(fixturePath, "go_only"),
			filepath.Join(fixturePath, "go_only", "go_only_sub"),
			filepath.Join(fixturePath, "md_only"),
			filepath.Join(fixturePath, "md_only", "md_only_sub"),
		},
		dirs.Slice(),
	)

	// Find all dirs without at least one file

	f.Clear()
	dirs, err = f.GetDirnameMatches()
	require.NoError(t, err)
	require.Zero(t, dirs.Len())

	dirs, err = f.Dir(fixturePath).DirMatcher(
		cage_file_matcher.InvertDir(cage_file_matcher.DirWithFile),
	).GetDirnameMatches()
	require.NoError(t, err)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixturePath, "both_types", "empty"),
			filepath.Join(fixturePath, "empty"),
			filepath.Join(fixturePath, "empty", "empty_sub"),
			filepath.Join(fixturePath, "md_only", "empty"),
		},
		dirs.Slice(),
	)
}

func (s *FinderSuite) TestDirMatchErr() {
	t := s.T()

	_, fixturePath := testkit_file.FixturePath(t, "finder", "baseline")
	f := cage_file.NewFinder()

	errMsg := "errMsg"
	errOnlyMatcher := func(absName string, files cage_file.FinderMatchFiles) (bool, error) {
		return false, errors.New(errMsg)
	}

	dirs, err := f.Dir(fixturePath).DirMatcher(errOnlyMatcher).GetDirnameMatches()
	require.Nil(t, dirs)
	require.Contains(t, err.Error(), errMsg)
}

func (s *FinderSuite) TestFileMatchErr() {
	t := s.T()

	_, fixturePath := testkit_file.FixturePath(t, "finder", "baseline")
	f := cage_file.NewFinder()

	errMsg := "errMsg"
	errOnlyMatcher := func(candidate cage_file.FinderFile) (bool, error) {
		return false, errors.New(errMsg)
	}

	files, err := f.Dir(fixturePath).GetFilenameMatches(errOnlyMatcher)
	require.Nil(t, files)
	require.Contains(t, err.Error(), errMsg)
}

func TestFinderSuite(t *testing.T) {
	suite.Run(t, new(FinderSuite))
}
