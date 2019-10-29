// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_testkit "github.com/codeactual/transplant/internal/cage/testkit"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
	"github.com/codeactual/transplant/internal/transplant"
)

// For more background on the topology contraints:
// https://github.com/codeactual/transplant/blob/master/doc/config.md#topologies
//
// Per-case comments may refer to configuration file sections such as Ops.From and Ops.Dep.
// The file is located in ./fixture/topology/transplant.yml.

type TopologySuite struct {
	Suite
}

func TestTopologySuite(t *testing.T) {
	suite.Run(t, new(TopologySuite))
}

// TestModuleRootedLocalAndDep asserts that Ops.From and a Ops.Dep.From trees cannot start at the root
// of the module.
func (s *TopologySuite) TestModuleRootedLocalAndDep() {
	t := s.T()
	opId := "module_rooted_local_and_dep"

	var config transplant.Config
	errs := config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId)

	require.Greater(t, len(errs), 0)
	testkit_require.MatchRegexp(t, errs[0].Error(), `Ops\[`+opId+`\].From.LocalFilePath is empty`)
}

func (s *TopologySuite) TestFromModuleFilePathEmpty() {
	t := s.T()
	opId := "from_modulefilepath_empty"

	var config transplant.Config
	errs := config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId)

	require.Greater(t, len(errs), 0)
	testkit_require.MatchRegexp(t, errs[0].Error(), `Ops\[`+opId+`\].From.ModuleFilePath is empty`)
}

func (s *TopologySuite) TestToModuleFilePathEmpty() {
	t := s.T()
	opId := "to_modulefilepath_empty"

	var config transplant.Config
	errs := config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId)

	require.Greater(t, len(errs), 0)
	testkit_require.MatchRegexp(t, errs[0].Error(), `Ops\[`+opId+`\].To.ModuleFilePath is empty`)
}

func (s *TopologySuite) TestToModuleImportPathEmpty() {
	t := s.T()
	opId := "to_moduleimportpath_empty"

	var config transplant.Config
	errs := config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId)

	require.Greater(t, len(errs), 0)
	testkit_require.MatchRegexp(t, errs[0].Error(), `Ops\[`+opId+`\].To.ModuleImportPath is empty`)
}

func (s *TopologySuite) TestLocalDepFilePathFromOverlap() {
	t := s.T()

	opId := "local_dep_filepath_from_overlap"

	var config transplant.Config
	cage_testkit.RequireNoErrors(t, config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId))

	audit := transplant.NewEgressAudit(config.Ops[opId])
	errs := audit.Generate()

	require.Len(t, errs, 1)
}

func (s *TopologySuite) TestLocalDepFilePathToOverlapBothSet() {
	t := s.T()
	opId := "local_dep_filepath_to_overlap_both_set"

	var config transplant.Config
	errs := config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId)

	require.Greater(t, len(errs), 0)
	testkit_require.MatchRegexp(t, errs[1].Error(), `Ops\[`+opId+`\].Dep\[0\].To.FilePath.*conflicts with Ops.To.LocalFilePath`)
}

func (s *TopologySuite) TestLocalDepFilePathToOverlapBothEmpty() {
	t := s.T()
	opId := "local_dep_filepath_to_overlap_both_emp"

	var config transplant.Config
	errs := config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId)

	require.Greater(t, len(errs), 0)
	testkit_require.MatchRegexp(t, errs[1].Error(), `Ops\[`+opId+`\].Dep\[0\].To.FilePath.*conflicts with Ops.To.LocalFilePath`)
}

func (s *TopologySuite) TestDepFilePathDupe() {
	t := s.T()

	opId := "dep_filepath_dupe"

	var config transplant.Config
	cage_testkit.RequireNoErrors(t, config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId))

	audit := transplant.NewEgressAudit(config.Ops[opId])
	errs := audit.Generate()

	require.Len(t, errs, 2)
	testkit_require.MatchRegexp(t, errs[0].Error(), "Dep.From.FilePath.*dep1.*is selected multiple times")
	testkit_require.MatchRegexp(t, errs[1].Error(), "Dep.To.FilePath.*internal/dep1.*is selected multiple times")
}

func (s *TopologySuite) TestDepFilePathOverlap() {
	t := s.T()

	opId := "dep_filepath_overlap"

	var config transplant.Config
	cage_testkit.RequireNoErrors(t, config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId))

	audit := transplant.NewEgressAudit(config.Ops[opId])
	errs := audit.Generate()

	require.Len(t, errs, 2)
	testkit_require.MatchRegexp(t, errs[0].Error(), "Dep.From.FilePath.*dep1/subpkg.*overlaps with another.*dep1]")
	testkit_require.MatchRegexp(t, errs[1].Error(), "Dep.To.FilePath.*internal/dep1/subpkg.*overlaps with another.*internal/dep1]")
}

// TestDotDotInRelativePath asserts that LocalFilePath/FilePath values cannot contain '..'.
func (s *TopologySuite) TestDotDotInRelativePath() {
	t := s.T()
	opId := "dotdot_in_relative_path"

	var config transplant.Config
	errs := config.ReadFile(s.FixturePath("topology", "transplant.yml"), opId)

	require.Len(t, errs, 4)
	testkit_require.MatchRegexp(t, errs[0].Error(), `Ops\[`+opId+`\].From.LocalFilePath \[../local\] cannot contain '..'`)
	testkit_require.MatchRegexp(t, errs[1].Error(), `Ops\[`+opId+`\].To.LocalFilePath \[../local\] cannot contain '..'`)
	testkit_require.MatchRegexp(t, errs[2].Error(), `Ops\[`+opId+`\].Dep\[../deps\].From.FilePath cannot contain '..'`)
	testkit_require.MatchRegexp(t, errs[3].Error(), `Ops\[`+opId+`\].Dep\[../internal\].To.FilePath cannot contain '..'`)
}

// RequireCopy asserts that the copy operation emitted no errors and the stage/output dir contents
// match those of their respective golden/expectation dirs.
func (s *TopologySuite) RequireCopy(modeId string, fixtureIdParts ...string) {
	s.Run(modeId+" "+strings.Join(fixtureIdParts, " "), func() {
		t := s.T()

		fixture := s.MustCopyFixtureWithGomod(modeId, "topology", "TopologySuite", "yml", fixtureIdParts...)
		if fixture.Plan.StagePath != "" {
			defer func() {
				require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
			}()
		}

		if modeId == "ingress" {
			s.DirsMatchExceptGomod(fixture.GoldenPath+"_"+modeId+"_stage", fixture.Plan.StagePath)
			s.DirsMatchExceptGomod(fixture.GoldenPath+"_"+modeId+"_output", fixture.OutputPath)
		} else {
			s.DirsMatchExceptGomod(fixture.GoldenPath+"_"+modeId, fixture.Plan.StagePath)
			s.DirsMatchExceptGomod(fixture.GoldenPath+"_"+modeId, fixture.OutputPath)
		}
	})
}

// TestLocalUnderDep asserts support for an Ops.From origin module under an Ops.Dep tree.
func (s *TopologySuite) TestPermutations() {
	for _, modeId := range []string{"egress", "ingress"} {
		s.RequireCopy(modeId, "loc_dep_non_moduleroot_siblings", "loc_from_set_to_emp", "dep_from_set_to_set")
		s.RequireCopy(modeId, "loc_dep_non_moduleroot_siblings", "loc_from_set_to_set", "dep_from_set_to_set")
		s.RequireCopy(modeId, "loc_under_dep", "loc_from_set_to_emp", "dep_from_emp_to_set")
		s.RequireCopy(modeId, "loc_under_dep", "loc_from_set_to_emp", "dep_from_set_to_set")
		s.RequireCopy(modeId, "loc_under_dep", "loc_from_set_to_set", "dep_from_emp_to_emp")
		s.RequireCopy(modeId, "loc_under_dep", "loc_from_set_to_set", "dep_from_emp_to_set")
		s.RequireCopy(modeId, "loc_under_dep", "loc_from_set_to_set", "dep_from_set_to_set")
	}
}
