// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tp_file "github.com/codeactual/transplant/internal/third_party/gist.github.com/os/file"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/tools/go/packages"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_testkit "github.com/codeactual/transplant/internal/cage/testkit"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
	"github.com/codeactual/transplant/internal/transplant"
)

const (
	FixtureGomodVersion = "1.12"
)

var gomodVersionRe *regexp.Regexp

func init() {
	gomodVersionRe = regexp.MustCompile(`^go [0-9.]+$`)
}

type Fixture struct {
	Audit *transplant.Audit
	OpId  string
	Path  string
}

type CopyFixture struct {
	Fixture

	Copier     *transplant.Copier
	GoldenPath string
	OutputPath string
	Plan       transplant.CopyPlan

	// OutputStats indexes, by absolute path, stat details of preexisting files in the OutputPath.
	OutputStats map[string]os.FileInfo
}

type Suite struct {
	suite.Suite

	NilStringSlice []string
	PkgCache       *cage_pkgs.Cache
	Wd             string

	Env map[string]string
}

func (s *Suite) SetupTest() {
	var err error

	t := s.T()

	s.Wd, err = os.Getwd()
	require.NoError(t, err)

	s.Wd, err = filepath.Abs(s.Wd)
	require.NoError(t, err)

	s.PkgCache = cage_pkgs.NewCache()
}

func (s *Suite) ImportPath(pathParts ...string) string {
	return "origin.tld/user/proj/" + strings.Join(pathParts, "/")
}

func (s *Suite) FixturePath(groupId string, fixtureIdParts ...string) string {
	return filepath.Join(append([]string{s.Wd, "testdata", "fixture", groupId}, fixtureIdParts...)...)
}

func (s *Suite) Op(modeId, groupId, opId, configExt string, fixtureIdParts ...string) transplant.Op {
	t := s.T()

	file := s.FixturePath(groupId, "transplant."+configExt)

	var config transplant.Config
	cage_testkit.RequireNoErrors(t, config.ReadFile(file, opId))

	op, ok := config.Ops[opId]
	require.True(t, ok, "missing Ops key in config file: "+opId)

	// Similar to how SeedOutputTree will ensure the copy operation outputs to a testdata dir
	// ignored by git, here we will adjust the config for ingress mode. During ingress, we
	// reuse the egress config by simply reversing the To/From values, rather than requiring users
	// to declare a separate ingress-specific section. In preparation for that reversal,
	// here we complement SeedOutputTree by updating the config with two main changes. First,
	// the "From" file path now points to the same output path selected by SeedOutputTree,
	// which means that it becomes the "To" file path after the config reversal (in finalizeIngressOp).
	// Second, the "To" file path now points to the conventional location of the fixture tree
	// which defines what the destination looks like prior to performing the copy. These steps
	// allow the configs for ingress test cases to look exactly like egress test cases.
	if modeId == "ingress" {
		op.From.ModuleFilePath = strings.Replace(
			op.From.ModuleFilePath,
			filepath.Join("testdata", "fixture", groupId, opId, "origin"),
			filepath.Join("testdata", "dynamic"),
			1,
		)

		var copySubDir string
		if s.IsSharedModeFixture(groupId, fixtureIdParts...) {
			copySubDir = "copy_ingress"
		} else {
			copySubDir = "copy"
		}

		op.To.ModuleFilePath = strings.Replace(
			op.To.ModuleFilePath,
			filepath.Join("testdata", "dynamic"),
			filepath.Join("testdata", "fixture", groupId, opId, copySubDir),
			1,
		)
	}

	return op
}

func (s *Suite) NewImport(groupId string, f Fixture, pathParts ...string) cage_pkgs.Import {
	t := s.T()

	pkg, errs := s.PkgCache.LoadImportPath(cage_pkgs.NewConfig(&packages.Config{}), s.ImportPath(pathParts...))
	cage_testkit.RequireNoErrors(t, errs)

	return cage_pkgs.NewImportFromPkg(pkg)
}

func (s *Suite) LoadFixture(modeId, groupId, suiteId, configExt string, fixtureIdParts ...string) (f Fixture, errs []error) {
	f.Path = s.FixturePath(groupId, fixtureIdParts...)
	f.OpId = filepath.Join(fixtureIdParts...)
	s.SeedOutputTree(modeId, groupId, fixtureIdParts...)
	f.Audit = s.newAudit(modeId, s.Op(modeId, groupId, f.OpId, configExt, fixtureIdParts...))
	return f, f.Audit.Generate()
}

func (s *Suite) MustLoadFixture(modeId, groupId, suiteId, configExt string, fixtureIdParts ...string) Fixture {
	f, errs := s.LoadFixture(modeId, groupId, suiteId, configExt, fixtureIdParts...)
	cage_testkit.RequireNoErrors(s.T(), errs)
	return f
}

// IsSharedModeFixture returns true if ./copy_ingress is found in the fixture dir, indicating that
// the current test case that exercises both copy modes using the same config, origin, etc.
func (s *Suite) IsSharedModeFixture(groupId string, fixtureIdParts ...string) bool {
	exists, _, existsErr := cage_file.Exists(filepath.Join(s.FixturePath(groupId, fixtureIdParts...), "copy_ingress"))
	require.NoError(s.T(), existsErr)
	return exists
}

// SeedOutputTree copies the fixture destination tree ("<fixture root>/copy" during egress, "<fixture root>/origin"
// during ingress) to a temporary test data dir so we can assert changes to the tree w/o modifying
// repository files.
func (s *Suite) SeedOutputTree(modeId, groupId string, fixtureIdParts ...string) (destFixtureExists bool, outputPath string) {
	t := s.T()

	outputPath = testkit_file.DynamicDataDirAbs(t)
	require.NoError(t, cage_file.RemoveAllSafer(outputPath))

	// If the fixture directory defined the preexisting content of the copy's destination, then seed the
	// output directory with that content to support assertions about additions, overwrites, and removals.
	//
	// Evaluate multiple egress destination dirs to support both mode-specific dirs and also ones which
	// contain both egress and ingress fixtures to be used in a single test/config.
	var destFixturePaths []string
	if modeId == "egress" {
		destFixturePaths = append(
			destFixturePaths,
			filepath.Join(s.FixturePath(groupId, fixtureIdParts...), "copy"),
			filepath.Join(s.FixturePath(groupId, fixtureIdParts...), "copy_egress"),
		)
	} else {
		destFixturePaths = append(destFixturePaths, filepath.Join(s.FixturePath(groupId, fixtureIdParts...), "origin"))
	}

	for _, destFixturePath := range destFixturePaths {
		var existsErr error
		destFixtureExists, _, existsErr = cage_file.Exists(destFixturePath)
		require.NoError(t, existsErr)

		if destFixtureExists {
			require.NoError(t, tp_file.CopyDir(destFixturePath, outputPath))
			break
		}
	}

	if modeId == "ingress" {
		require.NoError(t, tp_file.CopyFile(filepath.Join(s.FixturePath(groupId, fixtureIdParts...), "origin", "go.mod"), filepath.Join(outputPath, "go.mod")))
	}

	return destFixtureExists, outputPath
}

// CopyFixture creates an Audit, Copier, and performs the egress/ingress copy.
func (s *Suite) CopyFixture(modeId, groupId, suiteId, configExt string, fixtureIdParts ...string) (f *CopyFixture, errs []error) {
	f, errs = s.NewCopier(modeId, groupId, suiteId, configExt, fixtureIdParts...)
	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return nil, errs
	}

	f.Plan, errs = f.Copier.Run()
	return f, errs
}

func (s *Suite) MustCopyFixture(modeId, groupId, suiteId, configExt string, fixtureIdParts ...string) *CopyFixture {
	f, errs := s.CopyFixture(modeId, groupId, suiteId, configExt, fixtureIdParts...)
	cage_testkit.RequireNoErrors(s.T(), errs)
	return f
}

// CopyFixtureWithGomod is the same as CopyFixture except that, in egress mode, the copy's go.mod (and go.sum
// if applicable) is created/updated, versions are synced between the go.mod "require" directives,
// and go.sum (if applicable) sums are compared for differences.
func (s *Suite) CopyFixtureWithGomod(modeId, groupId, suiteId, configExt string, fixtureIdParts ...string) (f *CopyFixture, errs []error) {
	f, errs = s.NewCopier(modeId, groupId, suiteId, configExt, fixtureIdParts...)
	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return nil, errs
	}

	f.Copier.ModuleRequire = true
	f.Plan, errs = f.Copier.Run()
	return f, errs
}

func (s *Suite) MustCopyFixtureWithGomod(modeId, groupId, suiteId, configExt string, fixtureIdParts ...string) *CopyFixture {
	f, errs := s.CopyFixtureWithGomod(modeId, groupId, suiteId, configExt, fixtureIdParts...)
	cage_testkit.RequireNoErrors(s.T(), errs)
	return f
}

func (s *Suite) NewCopier(modeId, groupId, suiteId, configExt string, fixtureIdParts ...string) (f *CopyFixture, errs []error) {
	t := s.T()

	f = &CopyFixture{}
	f.Path = s.FixturePath(groupId, fixtureIdParts...)
	f.GoldenPath = filepath.Join(f.Path, "golden")
	f.OpId = filepath.Join(fixtureIdParts...)

	var outputPathExists bool
	outputPathExists, f.OutputPath = s.SeedOutputTree(modeId, groupId, fixtureIdParts...)

	// collect stat info to support assertions about the effect of a copy operation

	f.OutputStats = make(map[string]os.FileInfo)
	if outputPathExists {
		walkErrs := cage_filepath.WalkAbs(f.OutputPath, func(absPath string, fi os.FileInfo, walkErr error) []error {
			if walkErr != nil {
				return []error{errors.WithStack(walkErr)}
			}
			f.OutputStats[absPath] = fi
			return nil
		})
		cage_testkit.RequireNoErrors(s.T(), walkErrs)
	}

	f.Audit = s.newAudit(modeId, s.Op(modeId, groupId, f.OpId, configExt, fixtureIdParts...))

	if errs = f.Audit.Generate(); len(errs) > 0 {
		return f, errs
	}

	var err error

	f.Copier, err = transplant.NewCopier(context.Background(), f.Audit)
	require.NoError(t, err)
	f.Copier.ProgressModule = os.Stderr

	return f, []error{}
}

func (s *Suite) newAudit(modeId string, op transplant.Op) (a *transplant.Audit) {
	switch modeId { // mainly to reverse the From/To configs for ingress
	case "egress":
		a = transplant.NewEgressAudit(op)
	case "ingress":
		a = transplant.NewIngressAudit(op)
	default:
		s.T().Fatalf("invalid fixture mode ID [%s]", modeId)
	}

	return a
}

func (s *Suite) GomodVersionReplacer(expectedVersion string) testkit_require.ReaderLineReplacer {
	return func(_, _, actualLine string) (replacement string) {
		if gomodVersionRe.MatchString(actualLine) {
			return "go " + expectedVersion
		}
		return actualLine
	}
}

func (s *Suite) DirsMatchExceptGomod(expectPath, actualPath string) {
	testkit_require.DirsMatch(s.T(), expectPath, actualPath, s.GomodVersionReplacer(FixtureGomodVersion))
}
