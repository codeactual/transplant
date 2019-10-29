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

	cage_os "github.com/codeactual/transplant/internal/cage/os"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

// Per-case comments may refer to configuration file sections such as Ops.From and Ops.Dep.
// The file is located in ./fixture/gomod/transplant.yml.

type GomodSuite struct {
	Suite
}

func TestGomodSuite(t *testing.T) {
	suite.Run(t, new(GomodSuite))
}

// TestEgressSyncBaseline asserts that the copy's go.mod contains versions in its "require" and "replace"
// directives which match those in the origin's go.mod.
//
// The golden/go.mod contains an older golang.org/x/sync than origin/go.mod because github.com/onsi/gomega@1.4.3
// specifies that version in its go.mod as an indirect.
//
// The versions in the fixture go.mod files should remain out-of-date in order to assert that the toolchain
// use neither modifies the origin go.mod at all nor updates the copy's go.mod to use versions newer/different
// than the origin.
//
// Half of the "replace" dependencies are from the "require" directive, half are transitive. All select an
// older version (than found in the "require" or go.sum) as a replacement.
func (s *GomodSuite) TestEgressSyncBaseline() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "gomod", "GomodSuite", "yml", "egress_sync_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestEgressVendor asserts that if GOFLAGS contains "-mod=vendor" and <ModuleFilePath>/{go.sum,vendor/modules.txt}
// exist, then "go mod vendor" output is included in the copy.
//
// The origin/vendor/ tree contains fake packages in order to make the origin/ fixtures loadable.
//
// The golden/go.mod contains an older golang.org/x/sync than origin/go.mod because github.com/onsi/gomega@1.4.3
// specifies that version in its go.mod as an indirect.
//
// The versions in the fixture go.mod files should remain out-of-date in order to assert that the toolchain
// use neither modifies the origin go.mod at all nor updates the copy's go.mod to use versions newer/different
// than the origin.
func (s *GomodSuite) TestEgressVendor() {
	t := s.T()

	origGoflags, _, err := cage_os.AppendEnv("GOFLAGS", "-mod=vendor", " ")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Setenv("GOFLAGS", origGoflags))
	}()

	fixture := s.MustCopyFixtureWithGomod("egress", "gomod", "GomodSuite", "yml", "egress_vendor")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	// assert stage content

	testkit_require.FileStringContains(
		t,
		filepath.Join(fixture.Plan.StagePath, "vendor", "modules.txt"),
		"github.com/mitchellh/mapstructure",
		"github.com/stretchr/testify",
	)
	testkit_require.FileStringContains(
		t,
		filepath.Join(fixture.Plan.StagePath, "vendor", "github.com", "mitchellh", "mapstructure", "mapstructure.go"),
		"DecoderConfig",
	)
	testkit_require.FileStringContains(
		t,
		filepath.Join(fixture.Plan.StagePath, "vendor", "github.com", "stretchr", "testify", "suite", "suite.go"),
		"Suite",
	)

	testkit_require.FilesMatch(
		t,
		filepath.Join(fixture.GoldenPath, "go.mod"),
		filepath.Join(fixture.Plan.StagePath, "go.mod"),
		s.GomodVersionReplacer(FixtureGomodVersion),
	)
	testkit_require.FilesMatch(
		t,
		filepath.Join(fixture.GoldenPath, "go.sum"),
		filepath.Join(fixture.Plan.StagePath, "go.sum"),
	)
	testkit_require.DirsMatch(
		t,
		filepath.Join(fixture.GoldenPath, "local"),
		filepath.Join(fixture.Plan.StagePath, "local"),
	)
	testkit_require.DirsMatch(
		t,
		filepath.Join(fixture.GoldenPath, "internal"),
		filepath.Join(fixture.Plan.StagePath, "internal"),
	)

	// assert copy content

	testkit_require.FileStringContains(
		t,
		filepath.Join(fixture.OutputPath, "vendor", "modules.txt"),
		"github.com/mitchellh/mapstructure",
		"github.com/stretchr/testify",
	)
	testkit_require.FileStringContains(
		t,
		filepath.Join(fixture.OutputPath, "vendor", "github.com", "mitchellh", "mapstructure", "mapstructure.go"),
		"DecoderConfig",
	)
	testkit_require.FileStringContains(
		t,
		filepath.Join(fixture.OutputPath, "vendor", "github.com", "stretchr", "testify", "suite", "suite.go"),
		"Suite",
	)

	testkit_require.DirsMatch(
		t,
		filepath.Join(fixture.GoldenPath, "local"),
		filepath.Join(fixture.OutputPath, "local"),
	)
	testkit_require.DirsMatch(
		t,
		filepath.Join(fixture.GoldenPath, "internal"),
		filepath.Join(fixture.OutputPath, "internal"),
	)
	testkit_require.FilesMatch(
		t,
		filepath.Join(fixture.GoldenPath, "go.mod"),
		filepath.Join(fixture.OutputPath, "go.mod"),
		s.GomodVersionReplacer(FixtureGomodVersion),
	)
	testkit_require.FilesMatch(
		t,
		filepath.Join(fixture.GoldenPath, "go.sum"),
		filepath.Join(fixture.OutputPath, "go.sum"),
	)
}
