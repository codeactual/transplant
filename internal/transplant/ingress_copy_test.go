// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

// Per-case comments may refer to configuration file sections such as Ops.From and Ops.Dep.
// The file is located in ./fixture/ingress/transplant.yml.

type IngressCopySuite struct {
	Suite
}

func TestIngressCopySuite(t *testing.T) {
	suite.Run(t, new(IngressCopySuite))
}

// TestBaseline simulates an copy operation in which there are files which are expected to be
// added, overwritten, or removed, and that import/file path strings are rewritten to point to
// the origin.
func (s *IngressCopySuite) TestBaseline() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("ingress", "ingress", "IngressCopySuite", "yml", "copy_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath+"_stage", fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath+"_output", fixture.OutputPath)
}

// TestGoDescendantFilePath asserts that GoDescendantFilePath patterns are applied during ingress
// as they are during egress, and that import/file path strings are rewritten to point to the origin
// In the fixtures, the copy targets are *.md files, relying on Go ancestor files named match.go
// to trigger their inclusion.
func (s *IngressCopySuite) TestGoDescendantFilePath() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("ingress", "ingress", "IngressCopySuite", "yml", "go_descendant")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath+"_stage", fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath+"_output", fixture.OutputPath)
}

// TestGoFilePath asserts that custom GoFilePath patterns are applied during ingress
// as they are during egress.
func (s *IngressCopySuite) TestGoFilePath() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("ingress", "ingress", "IngressCopySuite", "yml", "auto_detect")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath+"_stage", fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath+"_output", fixture.OutputPath)
}

// TestRenameFileAtFilePathRoot asserts that files which share base filenames with top-level
// Ops.From.FilePath directories will be renamed to match associated "To" names.
func (s *IngressCopySuite) TestRenameFileAtFilePathRoot() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("ingress", "ingress", "IngressCopySuite", "yml", "rename_file_at_filepath_root")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath+"_stage", fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath+"_output", fixture.OutputPath)
}

// TestRenamedFilesRetained asserts that files which are renamed are retained during ingress
// as long as their counterparts in the copy are retained.
func (s *IngressCopySuite) TestRenamedFilesRetained() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("ingress", "ingress", "IngressCopySuite", "yml", "renamed_files_retained")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath+"_stage", fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath+"_output", fixture.OutputPath)
}

// TestOverwriteMinimum asserts that the stage/output dirs and CopyPlan only receive regular file
// overwrites where the destination contains different content.
//
// Fixtures contain the string "(edit)" to represent a content change.
func (s *IngressCopySuite) TestOverwriteMinimum() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("ingress", "ingress", "ingressCopySuite", "yml", "copy_overwrite_minimal")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	changedPaths := []string{
		filepath.Join(fixture.OutputPath, "local", "auto_detect", "changed.go"),
		filepath.Join(fixture.OutputPath, "local", "copy_only", "changed.go"),
		filepath.Join(fixture.OutputPath, "local", "go_descendant", "descendant", "changed.md"),
	}

	skipPaths := []string{
		filepath.Join(fixture.OutputPath, "local", "auto_detect", "skip.go"),
		filepath.Join(fixture.OutputPath, "local", "copy_only", "skip.go"),
		filepath.Join(fixture.OutputPath, "local", "go_descendant", "descendant", "skip.md"),
		filepath.Join(fixture.OutputPath, "local", "go_descendant", "go_descendant.go"),
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Add,
	)
	testkit_require.StringSliceExactly(
		t,
		skipPaths,
		fixture.Plan.OverwriteSkip,
	)
	testkit_require.StringSliceExactly(
		t,
		changedPaths,
		fixture.Plan.Overwrite,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Remove,
	)

	for _, p := range changedPaths {
		fi, err := os.Stat(p)
		require.NoError(t, err)
		require.Greater(t, fi.ModTime().UnixNano(), fixture.OutputStats[p].ModTime().UnixNano()) // above plan to overwrite was enacted
	}
	for _, p := range skipPaths {
		fi, err := os.Stat(p)
		require.NoError(t, err)
		require.Exactly(t, fixture.OutputStats[p].ModTime().UnixNano(), fi.ModTime().UnixNano()) // above plan to skip was enacted
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath+"_stage", fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath+"_output", fixture.OutputPath)
}
