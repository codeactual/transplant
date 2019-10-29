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
)

type FileMatchSuite struct {
	suite.Suite
}

func (s *FileMatchSuite) TestAny() {
	t := s.T()

	t.Run("return output describing file match", func(t *testing.T) {
		expectedInclude := "/path/to/**/*.go"
		in := cage_filepath.MatchAnyInput{
			Name:    "/path/to/some/go/file.go",
			Include: []string{expectedInclude},
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.NoError(t, err)
		require.True(t, out.Match)
		require.Exactly(t, expectedInclude, out.Include)
		require.Exactly(t, "", out.Exclude)
	})

	t.Run("return output describing dir match", func(t *testing.T) {
		expectedInclude := "/path/to/**/cmd"
		in := cage_filepath.MatchAnyInput{
			Name:    "/path/to/some/go/cmd",
			Include: []string{expectedInclude},
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.NoError(t, err)
		require.True(t, out.Match)
		require.Exactly(t, expectedInclude, out.Include)
		require.Exactly(t, "", out.Exclude)
	})

	t.Run("return output describing no file match", func(t *testing.T) {
		expectedExclude := "/path/to/**/*.go"
		in := cage_filepath.MatchAnyInput{
			Name:    "/path/to/some/go/file.go",
			Include: []string{"/path/to/**/*.go"},
			Exclude: []string{expectedExclude},
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.NoError(t, err)
		require.False(t, out.Match)
		require.Exactly(t, "", out.Include)
		require.Exactly(t, expectedExclude, out.Exclude)
	})

	t.Run("return output describing no dir match", func(t *testing.T) {
		expectedExclude := "/path/to/**/cmd"
		in := cage_filepath.MatchAnyInput{
			Name:    "/path/to/some/go/cmd",
			Include: []string{"/path/to/**/*.go"},
			Exclude: []string{expectedExclude},
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.NoError(t, err)
		require.False(t, out.Match)
		require.Exactly(t, "", out.Include)
		require.Exactly(t, expectedExclude, out.Exclude)
	})

	t.Run("handle include pattern error", func(t *testing.T) {
		badInclude := `/path/to/[`
		in := cage_filepath.MatchAnyInput{
			Name:    "/path/to/some/go/file.go",
			Include: []string{badInclude},
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax error in pattern")
		require.False(t, out.Match)
		require.Exactly(t, "", out.Include)
		require.Exactly(t, "", out.Exclude)
	})

	t.Run("handle exclude pattern error", func(t *testing.T) {
		badExclude := `/path/to/[`
		in := cage_filepath.MatchAnyInput{
			Name:    "/path/to/some/go/file.go",
			Include: []string{"/path/to/**/*.go"},
			Exclude: []string{badExclude},
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax error in pattern")
		require.False(t, out.Match)
		require.Exactly(t, "", out.Include)
		require.Exactly(t, "", out.Exclude)
	})

	t.Run("handle missing name", func(t *testing.T) {
		in := cage_filepath.MatchAnyInput{
			Include: []string{"/path/to/**/*.go"},
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.NoError(t, err)
		require.False(t, out.Match)
		require.Exactly(t, "", out.Include)
		require.Exactly(t, "", out.Exclude)
	})

	t.Run("handle missing include", func(t *testing.T) {
		in := cage_filepath.MatchAnyInput{
			Name: "/path/to/some/go/file.go",
		}

		out, err := cage_filepath.PathMatchAny(in)
		require.NoError(t, err)
		require.False(t, out.Match)
		require.Exactly(t, "", out.Include)
		require.Exactly(t, "", out.Exclude)
	})
}

func TestFileMatchSuite(t *testing.T) {
	suite.Run(t, new(FileMatchSuite))
}
