// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant_test

import (
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_dag "github.com/codeactual/transplant/internal/cage/graph/dag"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

// Per-case comments may refer to configuration file sections such as Ops.From and Ops.Dep.
// The file is located in ./fixture/egress/transplant.yml.

type EgressAuditSuite struct {
	Suite
}

func TestEgressAuditSuite(t *testing.T) {
	suite.Run(t, new(EgressAuditSuite))
}

func (s *EgressAuditSuite) TestDuplicateImports() {
	t := s.T()
	_ = t

	_, errs := s.LoadFixture("egress", "egress", "EgressAuditSuite", "yml", "dupe_import")
	require.Len(t, errs, 2)
	testkit_require.MatchRegexp(
		t,
		errs[0].Error(),
		"duplicate imports are not currently supported",
		`dep1\.go`,
	)
	testkit_require.MatchRegexp(
		t,
		errs[1].Error(),
		"duplicate imports are not currently supported",
		`local\.go`,
	)
}

// TestUnconfiguredLocalDirs asserts the content Audit.UnconfiguredLocalDirs when findUsedDepPkgs encounters
// dependencies which were excluded by Ops.From.GoFilePath.Exclude or simply not covered by any inclusion
// pattern in the first place.
func (s *EgressAuditSuite) TestUnconfiguredLocalDirs() {
	t := s.T()

	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", "unconfigured_local_dirs")

	expectedImporter := path.Join(fixture.Audit.Op().From.LocalImportPath, "with_inclusion")

	testkit_require.StringSliceExactly(
		t,
		[]string{
			expectedImporter,
		},
		fixture.Audit.UnconfiguredDirImporters.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "local", "with_inclusion", "auto_detect_exclusion"),
			filepath.Join(fixture.Path, "origin", "local", "without_inclusion"),
		},
		fixture.Audit.UnconfiguredDirs[expectedImporter].SortedSlice(),
	)
}

// TestUnconfiguredDepDirs asserts the content Audit.UnconfiguredDepDirs when findUsedDepPkgs encounters
// dependencies which were excluded by Ops.Dep.From.GoFilePath.Exclude or simply not covered by
// any inclusion pattern in the first place.
func (s *EgressAuditSuite) TestUnconfiguredDepDirs() {
	t := s.T()

	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", "unconfigured_dep_dirs")

	op := fixture.Audit.Op()
	expectedLocalImporter := op.From.LocalImportPath
	expectedDepImporter := op.Dep[0].From.ImportPath

	testkit_require.StringSliceExactly(
		t,
		[]string{
			expectedDepImporter,
			expectedLocalImporter,
		},
		fixture.Audit.UnconfiguredDirImporters.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep_without_inclusion2"),
		},
		fixture.Audit.UnconfiguredDirs[expectedLocalImporter].SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1", "auto_detect_exclusion"),
			filepath.Join(fixture.Path, "origin", "dep_without_inclusion1"),
		},
		fixture.Audit.UnconfiguredDirs[expectedDepImporter].SortedSlice(),
	)
}

// TestFindUsedDepPkgs asserts the output Audit.findUsedDepPkgs.
func (s *EgressAuditSuite) TestFindUsedDepPkgs() {
	t := s.T()

	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", "dep_dirs")

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1"),
			filepath.Join(fixture.Path, "origin", "dep1", "dep1a"),
			filepath.Join(fixture.Path, "origin", "dep2"),
			filepath.Join(fixture.Path, "origin", "dep2", "dep2a"),
		},
		fixture.Audit.DepInspectDirs.SortedSlice(),
	)
}

// TestDepConfiguredButNotImported asserts that Ops.Dep package "unused1" is not included
// in Audit.DirectDepImportsIntoLocal because its not directly/transitively used by Audit.LocalGoFiles
// package "local".
func (s *EgressAuditSuite) TestDepConfiguredButNotImported() {
	t := s.T()

	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", "unused_dep")

	testkit_require.StringSliceExactly(
		t,
		[]string{
			s.ImportPath("dep1"),
		},
		fixture.Audit.DirectDepImportsIntoLocal.Paths(),
	)
}

// TestDirectlyUsedDepExportsAcrossMultiDep asserts that Audit.UsedDepExports contains exports from
// two Ops.Dep packages which are directly used by Audit.LocalGoFiles package "local".
func (s *EgressAuditSuite) TestDirectlyUsedDepExportsAcrossMultiDep() {
	t := s.T()

	fixtureId := "dep_export_use_across_multi_dep"
	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", fixtureId)

	dep1PkgPath := s.ImportPath("dep1")
	dep1Filename := s.FixturePath("egress", fixtureId, "origin", "dep1", "dep1.go")
	dep2PkgPath := s.ImportPath("dep2")
	dep2Filename := s.FixturePath("egress", fixtureId, "origin", "dep2", "dep2.go")

	i := fixture.Audit.DirectDepImportsIntoLocal.Get(s.ImportPath("dep1"))
	require.NotNil(t, i)

	testkit_require.StringSliceExactly(
		t,
		[]string{
			dep1PkgPath,
			dep2PkgPath,
		},
		fixture.Audit.DirectDepImportsIntoLocal.Paths(),
	)

	require.Exactly(
		t,
		map[string]cage_pkgs.GlobalId{
			"ExportedFunc1": cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedFunc1"),
			"ExportedFunc2": cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedFunc2"),
		},
		fixture.Audit.UsedDepExports[i.Path],
	)

	i = fixture.Audit.DirectDepImportsIntoLocal.Get(s.ImportPath("dep2"))
	require.NotNil(t, i)

	require.Exactly(
		t,
		map[string]cage_pkgs.GlobalId{
			"ExportedType1": cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "ExportedType1"),
			"ExportedType2": cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "ExportedType2"),
		},
		fixture.Audit.UsedDepExports[i.Path],
	)
}

// TestDirectlyUsedDepExportsAcrossMultiLocal asserts that Audit.UsedDepExports contains exports from
// a Ops.Dep package which is directly used by Audit.LocalGoFiles package "local" across two files.
func (s *EgressAuditSuite) TestDirectlyUsedDepExportsAcrossMultiLocal() {
	t := s.T()

	fixtureId := "dep_export_use_across_multi_local"
	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", fixtureId)

	dep1PkgPath := s.ImportPath("dep1")
	dep1Filename := s.FixturePath("egress", fixtureId, "origin", "dep1", "dep1.go")

	testkit_require.StringSliceExactly(
		t,
		[]string{
			s.ImportPath("dep1"),
		},
		fixture.Audit.DirectDepImportsIntoLocal.Paths(),
	)

	i := fixture.Audit.DirectDepImportsIntoLocal.Get(s.ImportPath("dep1"))
	require.NotNil(t, i)

	require.Exactly(
		t,
		map[string]cage_pkgs.GlobalId{
			"ExportedFunc1": cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedFunc1"),
			"ExportedFunc2": cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedFunc2"),
			"ExportedType1": cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedType1"),
			"ExportedType2": cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedType2"),
		},
		fixture.Audit.UsedDepExports[i.Path],
	)
}

// TestDepUse asserts the content of various Audit structures which are populated based on
// use of Ops.Dep packages in Audit.LocalGoFiles.
func (s *EgressAuditSuite) TestDepUse() {
	t := s.T()

	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", "dep_use")

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1", "dep1.go"),
			filepath.Join(fixture.Path, "origin", "dep2", "dep2.go"),
			filepath.Join(fixture.Path, "origin", "dep3", "dep3.go"), // transitive
			filepath.Join(fixture.Path, "origin", "dep4", "dep4.go"), // transitive
		},
		fixture.Audit.UsedDepGoFiles.SortedSlice(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{
			s.ImportPath("dep1"),
			s.ImportPath("dep2"),
		},
		fixture.Audit.DirectDepImportsIntoLocal.Paths(),
	)

	testkit_require.StringSliceExactly(
		t,
		[]string{
			s.ImportPath("dep1"),
			s.ImportPath("dep2"),
			s.ImportPath("dep3"), // transitive
			s.ImportPath("dep4"), // transitive
		},
		fixture.Audit.AllDepImportsIntoLocal.Paths(),
	)
}

// TestDagBaseline asserts the structure of Audit.DepGlobalIdUsageDag reflects the dependency
// relationships between the Audit.LocalGoFiles package "local" and Ops.Dep packages "dep[1-3]".
// The overall relationships can be describe as simply "from->dep1->dep2->dep3" but the case
// asserts graph content down to the vertices and the globals they represent.
func (s *EgressAuditSuite) TestDagBaseline() {
	t := s.T()

	fixture := s.MustLoadFixture("egress", "egress", "EgressAuditSuite", "yml", "dag_baseline")

	// initial edges connected to the same root

	rootIds := s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, fixture.Audit.LocalGoFilesDagRoot)
	dep1Filename := filepath.Join(fixture.Path, "origin", "dep1", "dep1.go")
	dep2Filename := filepath.Join(fixture.Path, "origin", "dep2", "dep2.go")
	dep3Filename := filepath.Join(fixture.Path, "origin", "dep3", "dep3.go")

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{
			{Filename: filepath.Dir(dep1Filename)},
			{Filename: filepath.Dir(dep2Filename)},
			{Filename: filepath.Dir(dep3Filename)},
		},
		cage_pkgs.NewGlobalIdList().Add(rootIds...).SortedSlice(),
	)

	basePkgPath := "origin.tld/user/proj"

	dep1Vertex := rootIds[0]
	dep1PkgPath := path.Join(basePkgPath, "dep1")
	dep2Vertex := rootIds[1]
	dep2PkgPath := path.Join(basePkgPath, "dep2")
	dep3Vertex := rootIds[2]
	dep3PkgPath := path.Join(basePkgPath, "dep3")

	// edges rooted in dep1's directory

	dep1AliasTypeChain := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "AliasTypeChain")
	dep1ConstChain := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ConstChain")
	dep1ExportedType1 := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedType1")
	dep1ExportedType1CompositeLit := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedType1.MethodCallChainFromCompositeLit")
	dep1ExportedType1Value := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "ExportedType1.MethodCallChainFromValue")
	dep1FieldTypeChain := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "FieldTypeChain")
	dep1FieldTypeChainField := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "FieldTypeChain.f")
	dep1FuncCallChain := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "FuncCallChain")
	dep1NamedTypeChain := cage_pkgs.NewGlobalId(dep1PkgPath, "dep1", dep1Filename, "NamedTypeChain")

	// Assert individually because require.Exactly in v1.3.1 outputs an empty diff on inequality.
	expectVertices := []cage_pkgs.GlobalId{
		dep1AliasTypeChain,
		dep1ConstChain,
		dep1ExportedType1,
		dep1ExportedType1CompositeLit,
		dep1ExportedType1Value,
		dep1FieldTypeChain,
		dep1FieldTypeChainField,
		dep1FuncCallChain,
		dep1NamedTypeChain,
	}
	actualVertices := s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1Vertex)

	require.Exactly(t, len(expectVertices), len(actualVertices))
	for n := range expectVertices {
		require.Exactly(t, expectVertices[n], actualVertices[n])
	}

	// edges rooted in dep2's directory

	dep2AliasTypeChain := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "AliasTypeChain")
	dep2ConstChain := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "ConstChain")
	dep2ExportedType1 := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "ExportedType1")
	dep2ExportedType1CompositeLit := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "ExportedType1.MethodCallChainFromCompositeLit")
	dep2ExportedType1Value := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "ExportedType1.MethodCallChainFromValue")
	dep2FieldTypeChain := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "FieldTypeChain")
	dep2FieldTypeChainField := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "FieldTypeChain.f")
	dep2FuncCallChain := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "FuncCallChain")
	dep2NamedTypeChain := cage_pkgs.NewGlobalId(dep2PkgPath, "dep2", dep2Filename, "NamedTypeChain")

	expectVertices = []cage_pkgs.GlobalId{
		dep2AliasTypeChain,
		dep2ConstChain,
		dep2ExportedType1,
		dep2ExportedType1CompositeLit,
		dep2ExportedType1Value,
		dep2FieldTypeChain,
		dep2FieldTypeChainField,
		dep2FuncCallChain,
		dep2NamedTypeChain,
	}
	actualVertices = s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2Vertex)

	require.Exactly(t, len(expectVertices), len(actualVertices))
	for n := range expectVertices {
		require.Exactly(t, expectVertices[n], actualVertices[n])
	}

	// edges rooted in dep3's directory

	dep3AliasTypeChain := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "AliasTypeChain")
	dep3ConstChain := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "ConstChain")
	dep3ExportedType1 := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "ExportedType1")
	dep3ExportedType1CompositeLit := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "ExportedType1.MethodCallChainFromCompositeLit")
	dep3ExportedType1Value := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "ExportedType1.MethodCallChainFromValue")
	dep3FieldTypeChain := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "FieldTypeChain")
	dep3FuncCallChain := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "FuncCallChain")
	dep3NamedTypeChain := cage_pkgs.NewGlobalId(dep3PkgPath, "dep3", dep3Filename, "NamedTypeChain")

	expectVertices = []cage_pkgs.GlobalId{
		dep3AliasTypeChain,
		dep3ConstChain,
		dep3ExportedType1,
		dep3ExportedType1CompositeLit,
		dep3ExportedType1Value,
		dep3FieldTypeChain,
		dep3FuncCallChain,
		dep3NamedTypeChain,
	}
	actualVertices = s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep3Vertex)

	require.Exactly(t, len(expectVertices), len(actualVertices))
	for n := range expectVertices {
		require.Exactly(t, expectVertices[n], actualVertices[n])
	}

	// Assert dep*.ExportedType1 struct types are connected to their methods.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep1ExportedType1CompositeLit, dep1ExportedType1Value},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1ExportedType1),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2ExportedType1CompositeLit, dep2ExportedType1Value},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2ExportedType1),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep3ExportedType1CompositeLit, dep3ExportedType1Value},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep3ExportedType1),
	)

	// Assert the AliasTypeChain usage chain includes both the direct and transitive type dependency.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2AliasTypeChain, dep3AliasTypeChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1AliasTypeChain),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep3AliasTypeChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2AliasTypeChain),
	)

	// Assert NamedTypeChain usage chains are built similarly to AliasTypeChain's
	// where each link is connected to direct and transitive dependencies.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2NamedTypeChain, dep3NamedTypeChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1NamedTypeChain),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep3NamedTypeChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2NamedTypeChain),
	)

	// Assert ConstChain usage chain links one direct value dependency to another.
	// - Unlike the type chain above where dep1.AliasTypeChain is connected to both its
	//   direct and transitive type dependencies, a constant is only linked to a direct
	//   dependency because Inspector.IdentInfo, by current design, returns full type chains
	//   but not full value chains.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2ConstChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1ConstChain),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep3ConstChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2ConstChain),
	)

	// Assert ExportedType1.MethodCallChainFromCompositeLit methods are linked to their
	// parent struct types and the struct type of the called method.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep1ExportedType1, dep2ExportedType1},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1ExportedType1CompositeLit),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2ExportedType1, dep3ExportedType1},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2ExportedType1CompositeLit),
	)

	// Assert ExportedType1.MethodCallChainFromValue methods are linked to their
	// parent struct types and the struct type of the called method.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep1ExportedType1, dep2ExportedType1},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1ExportedType1Value),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2ExportedType1, dep3ExportedType1},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2ExportedType1Value),
	)

	// Assert FieldTypeChain usage chains link the struct types to their methods,
	// and the direct/transitive struct types used in their methods.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep1FieldTypeChainField, dep2FieldTypeChain, dep3FieldTypeChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1FieldTypeChain),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2FieldTypeChainField, dep3FieldTypeChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2FieldTypeChain),
	)

	// Assert FuncCallChain usage chains are built similarly to ConstChain's
	// where each link is only connected to its direct dependency.

	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep2FuncCallChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep1FuncCallChain),
	)
	require.Exactly(
		t,
		[]cage_pkgs.GlobalId{dep3FuncCallChain},
		s.verticesFrom(fixture.Audit.DepGlobalIdUsageDag, dep2FuncCallChain),
	)
}

// verticesFrom returns the DAG vertices connected to the input "origin" vertex.
func (s *EgressAuditSuite) verticesFrom(dag cage_dag.Graph, origin cage_pkgs.GlobalId) []cage_pkgs.GlobalId {
	list := cage_pkgs.NewGlobalIdList()
	for _, id := range dag.VerticesFrom(origin) {
		list.Add(id.(cage_pkgs.GlobalId))
	}
	return list.SortedSlice()
}
