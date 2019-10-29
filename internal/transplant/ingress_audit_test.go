// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

// Per-case comments may refer to configuration file sections such as Ops.From and Ops.Dep.
// The file is located in ./fixture/ingress/transplant.yml.

type IngressAuditSuite struct {
	Suite
}

func TestIngressAuditSuite(t *testing.T) {
	suite.Run(t, new(IngressAuditSuite))
}

// TestDepPathsExcluded asserts that Ops.Dep.From file trees are neither inspected for/as Go packages
// nor included in file lists supporting the copy operation.
func (s *IngressAuditSuite) TestDepPathsExcluded() {
	t := s.T()

	fixture := s.MustCopyFixture("ingress", "ingress", "IngressAuditSuite", "yml", "dep_paths_excluded")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "copy", "copy_only", "copy_only.go"),
			filepath.Join(fixture.Path, "copy", "testdata", "fixture", "fixture.go"),
		},
		fixture.Audit.LocalCopyOnlyFiles.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Audit.LocalGoDescendantFiles.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "copy"),
		},
		fixture.Audit.LocalInspectDirs.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Audit.DepCopyOnlyFiles.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Audit.DepGoDescendantFiles.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Audit.DepInspectDirs.SortedSlice(),
	)
}
