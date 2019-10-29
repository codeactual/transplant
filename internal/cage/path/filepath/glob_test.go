// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package filepath_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
	testkit_filepath "github.com/codeactual/transplant/internal/cage/testkit/path/filepath"
)

type GlobSuite struct {
	suite.Suite

	absPath1 string
	absPath2 string
	absPath3 string

	ancestorRoot string
}

func (s *GlobSuite) SetupTest() {
	t := s.T()

	testkit_file.ResetTestdata(t)

	s.ancestorRoot = testkit_filepath.Abs(t, filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj"))

	_, s.absPath1 = testkit_file.CreateFile(t, "path", "to", "proj", "cmd", "proj", "main.go")
	_, s.absPath2 = testkit_file.CreateFile(t, "path", "to", "proj", "file.go")
	_, _ = testkit_file.CreateFile(t, "path", "to", "proj", "ci")
	_, s.absPath3 = testkit_file.CreateFile(t, "path", "to", "proj", "README.md")
	_, _ = testkit_file.CreateFile(t, "path", "to", "proj", "LICENSE")
}

func (s *GlobSuite) TestAny() {
	t := s.T()

	t.Run("return output describing file inclusion matches", func(t *testing.T) {
		include := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "**", "*.go"),
		}
		in := cage_filepath.GlobAnyInput{
			Include: []cage_filepath.Glob{include},
		}
		expectedInclude := map[string]cage_filepath.Glob{
			s.absPath1: include,
			s.absPath2: include,
		}
		expectedExclude := map[string]cage_filepath.Glob{}

		out, err := cage_filepath.GlobAny(in)
		require.NoError(t, err)
		require.Exactly(t, expectedInclude, out.Include)
		require.Exactly(t, expectedExclude, out.Exclude)
	})

	t.Run("return output describing dir inclusion matches", func(t *testing.T) {
		include := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "c*"),
		}
		in := cage_filepath.GlobAnyInput{
			Include: []cage_filepath.Glob{include},
		}
		expectedInclude := map[string]cage_filepath.Glob{
			testkit_filepath.Abs(t, filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "cmd")): include,
			testkit_filepath.Abs(t, filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "ci")):  include,
		}
		expectedExclude := map[string]cage_filepath.Glob{}

		out, err := cage_filepath.GlobAny(in)
		require.NoError(t, err)
		require.Exactly(t, expectedInclude, out.Include)
		require.Exactly(t, expectedExclude, out.Exclude)
	})

	t.Run("return output describing file exclusion matches", func(t *testing.T) {
		include := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "**", "*.go"),
		}
		exclude := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "**", "main.go"),
		}
		in := cage_filepath.GlobAnyInput{
			Include: []cage_filepath.Glob{include},
			Exclude: []cage_filepath.Glob{exclude},
		}
		expectedInclude := map[string]cage_filepath.Glob{
			s.absPath2: include,
		}
		expectedExclude := map[string]cage_filepath.Glob{
			s.absPath1: exclude,
		}

		out, err := cage_filepath.GlobAny(in)
		require.NoError(t, err)
		require.Exactly(t, expectedInclude, out.Include)
		require.Exactly(t, expectedExclude, out.Exclude)
	})

	t.Run("return output describing dir exclusion matches", func(t *testing.T) {
		include := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "c*"),
		}
		exclude := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "ci"),
		}
		in := cage_filepath.GlobAnyInput{
			Include: []cage_filepath.Glob{include},
			Exclude: []cage_filepath.Glob{exclude},
		}
		expectedInclude := map[string]cage_filepath.Glob{
			testkit_filepath.Abs(t, filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "cmd")): include,
		}
		expectedExclude := map[string]cage_filepath.Glob{
			testkit_filepath.Abs(t, filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "ci")): exclude,
		}

		out, err := cage_filepath.GlobAny(in)
		require.NoError(t, err)
		require.Exactly(t, expectedInclude, out.Include)
		require.Exactly(t, expectedExclude, out.Exclude)
	})

	t.Run("handle include pattern error", func(t *testing.T) {
		include := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "["),
		}
		out, err := cage_filepath.GlobAny(cage_filepath.GlobAnyInput{
			Include: []cage_filepath.Glob{include},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax error in pattern")
		require.Exactly(t, cage_filepath.GlobAnyOutput{}, out)
	})

	t.Run("handle exclude pattern error", func(t *testing.T) {
		include := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "**", "*.go"),
		}
		exclude := cage_filepath.Glob{
			Pattern: filepath.Join(testkit_file.DynamicDataDir(), "path", "to", "proj", "["),
		}
		out, err := cage_filepath.GlobAny(cage_filepath.GlobAnyInput{
			Include: []cage_filepath.Glob{include},
			Exclude: []cage_filepath.Glob{exclude},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax error in pattern")
		require.Exactly(t, cage_filepath.GlobAnyOutput{}, out)
	})

	t.Run("handle missing include", func(t *testing.T) {
		out, err := cage_filepath.GlobAny(cage_filepath.GlobAnyInput{})
		require.NoError(t, err)
		require.Exactly(t, cage_filepath.GlobAnyOutput{}, out)
	})
}

func TestGlobSuite(t *testing.T) {
	suite.Run(t, new(GlobSuite))
}
