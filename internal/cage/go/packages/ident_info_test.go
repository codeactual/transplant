// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"path/filepath"
	"sort"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_token "github.com/codeactual/transplant/internal/cage/go/token"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

type IdentExpect struct {
	Name string
	Info *cage_pkgs.IdentInfo
}

func (e *IdentExpect) AddType(expects ...*IdentExpect) *IdentExpect {
	for _, expect := range expects {
		e.Info.Types = append(e.Info.Types, expect.Info)
	}
	return e
}

// requireSimilarPosition is only a minimal check on the position because of two assumptions:
// it's relatively unlikely that the line/column will be incorrect, and more importantly
// it's too costly to update expected line/column values when the fixture's content changes.
func (s *ApiInspectorSuite) requireSimilarPosition(assertId string, filename string, prevActualPos, curActualPos token.Position) {
	t := s.T()

	require.Exactly(t, filename, curActualPos.Filename, assertId)

	if prevActualPos.IsValid() {
		require.GreaterOrEqual(t, curActualPos.Line, prevActualPos.Line, assertId)
		if curActualPos.Line == prevActualPos.Line {
			require.GreaterOrEqual(t, curActualPos.Column, prevActualPos.Column, assertId)
		}
	} else {
		require.GreaterOrEqual(t, curActualPos.Line, 0, assertId)
		require.Greater(t, curActualPos.Column, 0, assertId)
	}
}

// requireSimilarIdentInfo performs a mix of exact and approximate assertions of the node's core IdentInfo
// and type chain links.
func (s *ApiInspectorSuite) requireSimilarIdentInfo(assertId, filename string, expect *IdentExpect, prevActualPos token.Position, actualInfo *cage_pkgs.IdentInfo) {
	t := s.T()

	// Use a temporary assertion ID until we can augment it with the inspected node's position.

	s.requireSimilarPosition(assertId, filename, prevActualPos, actualInfo.Position)

	// Assert the all IdentInfo fields except Type.

	// Use the IdentExpect.Name instead of IdentExpect.Info.Name because the latter are all left
	// empty to avoid always setting two fields to the same value when defining new expectations.
	// (IdentExpect.Name exists so that IdentInfo.Info can be set to nil to express the expectation
	// that NewIdentInfo will return nil for the input node.)
	require.Exactly(t, expect.Name, actualInfo.Name, assertId)

	require.Exactly(t, expect.Info.IsTypeDecl, actualInfo.IsTypeDecl, assertId)
	require.Exactly(t, expect.Info.PkgName, actualInfo.PkgName, assertId)
	require.Exactly(t, expect.Info.PkgPath, actualInfo.PkgPath, assertId)

	// Assert the IdentInfo of the node's type and underlying types (if any).

	indent := "  -> "
	expectTypesString := expect.Info.TypesString(indent)
	if expectTypesString == "" {
		expectTypesString = "<none>"
	}
	actualTypesString := actualInfo.TypesString(indent)
	if actualTypesString == "" {
		actualTypesString = "<none>"
	}

	baseTypeAssertId := fmt.Sprintf(
		"%s\n\ntop-level expect IdentInfo.Types:\n\n%s\n"+
			"top-level actual IdentInfo.Types:\n\n%s",
		assertId, expectTypesString, actualTypesString,
	)

	var requireSameTypes func(e, a *cage_pkgs.IdentInfo, depth int)

	requireSameTypes = func(e, a *cage_pkgs.IdentInfo, depth int) {
		// Display the depth multiple types to emphasize where in the type chain the
		// failed expectation is located.
		typeAssertId := fmt.Sprintf(
			"\n\n****** queryAllPkgs DEPTH: %d ******\n\n%s\n\n"+
				"DEPTH %d expect type: name [%s] pkg [%s] len(Type): %d\n"+
				"DEPTH %d actual type: name [%s] pkg [%s] len(Type): %d\n",
			depth, baseTypeAssertId,
			depth, e.Name, e.PkgPath, len(e.Types),
			depth, a.Name, a.PkgPath, len(a.Types),
		)

		if len(e.Types) == 0 {
			require.Empty(t, a.Types, typeAssertId)
			return
		} else {
			require.Exactly(t, len(e.Types), len(a.Types), typeAssertId)
		}

		expectTypes := cage_pkgs.IdentInfoSlice(e.Types)
		sort.Stable(&expectTypes)
		actualTypes := cage_pkgs.IdentInfoSlice(a.Types)
		sort.Stable(&actualTypes)

		for n := range expectTypes {
			require.Exactly(t, expectTypes[n].Name, actualTypes[n].Name, typeAssertId)
			require.Exactly(t, expectTypes[n].PkgName, actualTypes[n].PkgName, typeAssertId)
			require.Exactly(t, expectTypes[n].PkgPath, actualTypes[n].PkgPath, typeAssertId)

			// The type chain should only include IdentInfo values which describe type declarations
			// not uses.
			require.True(t, expectTypes[n].IsTypeDecl, typeAssertId)
			require.True(t, actualTypes[n].IsTypeDecl, typeAssertId)

			s.requireSimilarPosition(typeAssertId, expectTypes[n].Position.Filename, token.Position{}, actualTypes[n].Position)

			requireSameTypes(expectTypes[n], actualTypes[n], depth+1)
		}
	}

	requireSameTypes(expect.Info, actualInfo, 0)
}

// requireFileIdentInfo asserts the IdentInfo values of all ast.Ident nodes found in the file.
func (s *ApiInspectorSuite) requireFileIdentInfo(i *cage_pkgs.Inspector, filename string, expect []*IdentExpect) {
	t := s.T()
	pkg := i.FilePkgs[filename]
	require.NotNil(t, pkg)
	require.Len(t, pkg.Syntax, 1)

	expectedIdentNames := cage_strings.NewSet()
	for _, e := range expect {
		require.NotEmpty(t, e.Name)
		expectedIdentNames.Add(e.Name)
	}

	// Find all ast.Ident nodes in the fixture file and collect their IdentInfo values.

	actualIdentNames := cage_strings.NewSet()
	actualInfos := []*cage_pkgs.IdentInfo{}
	actualIdents := []*ast.Ident{}

	ast.Inspect(pkg.Syntax[0], func(n ast.Node) bool {
		if n == nil {
			return false
		}

		assertId := fmt.Sprintf("file [%s] node: %s\n", filepath.Base(filename), spew.Sdump(n))

		switch ident := n.(type) {
		case *ast.Ident:
			if !expectedIdentNames.Contains(ident.Name) {
				t.Fatalf("unexpected ident [%s] in file [%s]", ident.Name, filename)
			}

			info, infoErr := cage_pkgs.NewIdentInfo(i, ident)
			require.NoError(t, infoErr, assertId)

			actualIdentNames.Add(ident.Name)
			actualInfos = append(actualInfos, info)
			actualIdents = append(actualIdents, ident)
		}

		return true
	})
	require.Exactly(t, expectedIdentNames.SortedSlice(), actualIdentNames.SortedSlice(), filename)

	// Compare the actual IdentInfo values to the expectations.

	var prevActualPos token.Position // track for eventual use in requireSimilarPosition

	for n := range expect {
		pos := i.FileSet.Position(actualIdents[n].Pos())
		assertId := fmt.Sprintf(
			"file [%s] expect name [%s] actual name [%s]",
			cage_token.ShortPositionString(pos), expect[n].Name, actualIdents[n].Name,
		)

		require.Exactly(t, expect[n].Name, actualIdents[n].Name, assertId)

		if expect[n].Info == nil {
			require.Nil(t, actualInfos[n], assertId)
			continue
		} else {
			require.NotNil(t, actualInfos[n], assertId)
		}

		s.requireSimilarIdentInfo(assertId, filename, expect[n], prevActualPos, actualInfos[n])
		prevActualPos = actualInfos[n].Position
	}
}

// TestIdentInfo asserts that NewIdentInfo returns correct details about ast.Ident nodes.
//
// The three fixture files are annotated to further describe the per-node expectations.
func (s *ApiInspectorSuite) TestNewIdentInfo() {
	baseDir := s.FixturePath("ident_info")
	localFilename := filepath.Join(baseDir, "local", "local.go")
	dotImpFilename := filepath.Join(baseDir, "importer", "dot", "dot.go")
	namedImpFilename := filepath.Join(baseDir, "importer", "named", "named.go")

	// Inspect the fixture packages.

	i := s.MustInspect(
		baseDir,
		cage_pkgs.LoadSyntax,
		[]string{
			filepath.Dir(localFilename),
			filepath.Dir(dotImpFilename),
			filepath.Dir(namedImpFilename),
		}...,
	)

	// Provide common helpers.

	newNonTypeDecl := func(identName, pkgName, pkgPath string, pos token.Position) *IdentExpect {
		return &IdentExpect{
			Name: identName,
			Info: &cage_pkgs.IdentInfo{
				Name:     identName,
				PkgName:  pkgName,
				PkgPath:  pkgPath,
				Position: pos,
			},
		}
	}
	newTypeDecl := func(identName, pkgName, pkgPath string, pos token.Position) *IdentExpect {
		e := newNonTypeDecl(identName, pkgName, pkgPath, pos)
		e.Info.IsTypeDecl = true
		return e
	}

	// Assert results for all the identifiers from ident_info/local/local.go

	localPkgPath := "fixture.tld/ident_info/local"
	localPkgName := path.Base(localPkgPath)
	localPos := token.Position{Filename: localFilename}
	newLocalNonTypeDecl := func(identName string) *IdentExpect {
		return newNonTypeDecl(identName, localPkgName, localPkgPath, localPos)
	}
	newLocalTypeDecl := func(identName string) *IdentExpect {
		return newTypeDecl(identName, localPkgName, localPkgPath, localPos)
	}

	nonExportedDefinedInt := newLocalTypeDecl("nonExportedDefinedInt")
	anotherNonExportedDefinedInt := newLocalTypeDecl("anotherNonExportedDefinedInt").AddType(nonExportedDefinedInt)
	definedInt := newLocalTypeDecl("DefinedInt").AddType(anotherNonExportedDefinedInt)

	nonExportedAliasedInt := newLocalTypeDecl("nonExportedAliasedInt")
	anotherNonExportedAliasedInt := newLocalTypeDecl("anotherNonExportedAliasedInt").AddType(nonExportedAliasedInt)
	aliasedInt := newLocalTypeDecl("AliasedInt").AddType(anotherNonExportedAliasedInt)

	nonExportedDefinedFunc := newLocalTypeDecl("definedFunc").AddType(definedInt, aliasedInt)
	definedFunc := newLocalTypeDecl("DefinedFunc").AddType(nonExportedDefinedFunc)

	nonExportedAliasedFunc := newLocalTypeDecl("aliasedFunc").AddType(definedInt, aliasedInt)
	aliasedFunc := newLocalTypeDecl("AliasedFunc").AddType(nonExportedAliasedFunc)

	nonExportedStruct0 := newLocalTypeDecl("nonExportedStruct0")
	nonExportedStruct1 := newLocalTypeDecl("nonExportedStruct1")

	exportedStruct0 := newLocalTypeDecl("ExportedStruct0").AddType(
		nonExportedStruct0, nonExportedStruct1, definedInt, definedFunc,
	)
	exportedStruct1 := newLocalTypeDecl("ExportedStruct1")

	expect := []*IdentExpect{
		{Name: "local"},

		newLocalTypeDecl("Shadowed0"),
		{Name: "int"},
		newLocalTypeDecl("Shadowed1"),
		{Name: "int"},

		nonExportedStruct0,
		{Name: "IntField"},
		{Name: "int"},
		nonExportedStruct1,

		exportedStruct0,
		newLocalNonTypeDecl("nonExportedStruct0").AddType(nonExportedStruct0),
		{Name: "Field0"},
		newLocalNonTypeDecl("nonExportedStruct1").AddType(nonExportedStruct1),
		{Name: "Field1"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("DefinedFunc").AddType(definedFunc),
		{Name: "Field2"},
		{Name: "Shadowed0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "Shadowed1"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "Shadowed0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),

		{Name: "recv"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		{Name: "p0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "r0"},
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),

		exportedStruct1,
		{Name: "Next"},
		newLocalNonTypeDecl("ExportedStruct1").AddType(exportedStruct1),

		newLocalTypeDecl("nonExportedDefinedInt"),
		{Name: "int"},
		newLocalTypeDecl("anotherNonExportedDefinedInt").AddType(nonExportedDefinedInt),
		newLocalNonTypeDecl("nonExportedDefinedInt").AddType(nonExportedDefinedInt),

		newLocalTypeDecl("DefinedInt").AddType(anotherNonExportedDefinedInt),
		newLocalNonTypeDecl("anotherNonExportedDefinedInt").AddType(anotherNonExportedDefinedInt),

		newLocalNonTypeDecl("DefinedIntIota0").AddType(definedInt),
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "iota"},
		newLocalNonTypeDecl("DefinedIntIota1").AddType(definedInt),

		newLocalTypeDecl("nonExportedAliasedInt"),
		{Name: "int"},
		newLocalTypeDecl("anotherNonExportedAliasedInt").AddType(nonExportedAliasedInt),
		newLocalNonTypeDecl("nonExportedAliasedInt").AddType(nonExportedAliasedInt),

		newLocalTypeDecl("AliasedInt").AddType(anotherNonExportedAliasedInt),
		newLocalNonTypeDecl("anotherNonExportedAliasedInt").AddType(anotherNonExportedAliasedInt),

		newLocalTypeDecl("definedFunc").AddType(definedInt, aliasedInt),
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newLocalTypeDecl("DefinedFunc").AddType(nonExportedDefinedFunc),
		newLocalNonTypeDecl("definedFunc").AddType(nonExportedDefinedFunc),

		newLocalTypeDecl("aliasedFunc").AddType(definedInt, aliasedInt),
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newLocalTypeDecl("AliasedFunc").AddType(nonExportedAliasedFunc),
		newLocalNonTypeDecl("aliasedFunc").AddType(nonExportedAliasedFunc),

		newLocalTypeDecl("LocalInterface0").AddType(definedInt),
		{Name: "Method0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),

		newLocalTypeDecl("LocalInterface0Impl0"),
		{Name: "recv"},
		newLocalNonTypeDecl("LocalInterface0Impl0").AddType(newLocalTypeDecl("LocalInterface0Impl0")),
		{Name: "Method0"},
		{Name: "p0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),

		newLocalNonTypeDecl("IntLiteral100"),

		newLocalNonTypeDecl("LocalFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		{Name: "p0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "p1"},
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		newLocalNonTypeDecl("DefinedFunc").AddType(definedFunc),
		{Name: "r1"},
		newLocalNonTypeDecl("AliasedFunc").AddType(aliasedFunc),
		{Name: "nil"},

		newLocalNonTypeDecl("LocalFunc1").AddType(definedInt, aliasedInt),
		{Name: "p0"},
		{Name: "p1"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "p2"},
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		{Name: "r1"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "r2"},
		{Name: "r3"},
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),

		newLocalNonTypeDecl("LocalFunc2").AddType(
			definedInt, aliasedInt, definedFunc, aliasedFunc, nonExportedStruct0, nonExportedStruct1,
		),
		{Name: "p0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		{Name: "p1"},
		newLocalNonTypeDecl("DefinedFunc").AddType(definedFunc),
		newLocalNonTypeDecl("AliasedFunc").AddType(aliasedFunc),
		newLocalNonTypeDecl("nonExportedStruct0").AddType(nonExportedStruct0),
		newLocalNonTypeDecl("nonExportedStruct1").AddType(nonExportedStruct1),
		{Name: "nil"},

		newLocalNonTypeDecl("LocalNonGlobalUse"),

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "nonExportedStruct0"},
		newLocalNonTypeDecl("nonExportedStruct0").AddType(nonExportedStruct0),
		{Name: "Field0"},
		newLocalNonTypeDecl("nonExportedStruct1").AddType(nonExportedStruct1),
		{Name: "Shadowed0"},

		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "IntField"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field1"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Shadowed0"},

		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		newLocalNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newLocalNonTypeDecl("DefinedIntIota0").AddType(definedInt),
		newLocalNonTypeDecl("IntLiteral100"),
		newLocalNonTypeDecl("LocalFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		newLocalNonTypeDecl("LocalFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),

		{Name: "es0"},
		newLocalNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "es0"},
		{Name: "IntField"},
		{Name: "es0"},
		{Name: "Field0"},
		{Name: "es0"},
		{Name: "Field1"},
		{Name: "es0"},
		{Name: "Method0"},
		{Name: "Shadowed0"},
		newLocalNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "_"},
		{Name: "Shadowed0"},

		newLocalNonTypeDecl("LocalScopeTest"),

		{Name: "DefinedInt"},
		{Name: "int"},
		{Name: "_"},
		{Name: "DefinedInt"},

		{Name: "ReDefinedInt"},
		{Name: "int"},
		{Name: "_"},
		{Name: "ReDefinedInt"},

		{Name: "DefinedIntIota0"},
		{Name: "_"},
		{Name: "DefinedIntIota0"},
	}
	s.requireFileIdentInfo(i, localFilename, expect)

	// Assert results for all the identifiers from ident_info/importer/dot/dot.go

	dotImpPkgPath := "fixture.tld/ident_info/importer/dot"
	dotImpPkgName := path.Base(dotImpPkgPath)
	dotImpPos := token.Position{Filename: dotImpFilename}
	newDotImpNonTypeDecl := func(identName string) *IdentExpect {
		return newNonTypeDecl(identName, dotImpPkgName, dotImpPkgPath, dotImpPos)
	}
	newDotImpTypeDecl := func(identName string) *IdentExpect {
		return newTypeDecl(identName, dotImpPkgName, dotImpPkgPath, dotImpPos)
	}

	dotImpReDefinedInt := newDotImpTypeDecl("ReDefinedInt").AddType(definedInt)
	dotImpReDefinedIntTwice := newDotImpTypeDecl("ReDefinedIntTwice").AddType(dotImpReDefinedInt)
	dotImpReDefinedIntThrice := newDotImpTypeDecl("ReDefinedIntThrice").AddType(dotImpReDefinedIntTwice)

	dotImpReAliasedInt := newDotImpTypeDecl("ReAliasedInt").AddType(aliasedInt)
	dotImpReAliasedIntTwice := newDotImpTypeDecl("ReAliasedIntTwice").AddType(dotImpReAliasedInt)
	dotImpReAliasedIntThrice := newDotImpTypeDecl("ReAliasedIntThrice").AddType(dotImpReAliasedIntTwice)

	dotImpCustomExportedStruct0 := newDotImpTypeDecl("CustomExportedStruct0").AddType(exportedStruct0, exportedStruct1)

	expect = []*IdentExpect{
		{Name: "dot"},

		{Name: "."},
		{Name: "."},

		newDotImpNonTypeDecl("ConstFromNonInspectedPkg"),
		{Name: "NonInspectedConst"},

		newDotImpNonTypeDecl("CustomDefinedIntIota0").AddType(definedInt),
		newDotImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "iota"},
		newDotImpNonTypeDecl("CustomDefinedIntIota1").AddType(definedInt),

		newDotImpNonTypeDecl("DefinedIntIota0Copy").AddType(definedInt),
		newDotImpNonTypeDecl("DefinedIntIota0").AddType(definedInt),

		dotImpCustomExportedStruct0,
		newDotImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		newDotImpNonTypeDecl("ExportedStruct1").AddType(exportedStruct1),
		{Name: "Next"},
		newDotImpNonTypeDecl("CustomExportedStruct0").AddType(dotImpCustomExportedStruct0),

		{Name: "recv"},
		newDotImpNonTypeDecl("CustomExportedStruct0").AddType(dotImpCustomExportedStruct0),
		{Name: "Method0"},
		{Name: "p0"},
		newDotImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "r0"},
		newDotImpNonTypeDecl("AliasedInt").AddType(aliasedInt),

		newDotImpTypeDecl("ReDefinedInt").AddType(definedInt),
		newDotImpNonTypeDecl("DefinedInt").AddType(definedInt),

		newDotImpTypeDecl("ReDefinedIntTwice").AddType(dotImpReDefinedInt),
		newDotImpNonTypeDecl("ReDefinedInt").AddType(dotImpReDefinedInt),

		newDotImpTypeDecl("ReDefinedIntThrice").AddType(dotImpReDefinedIntTwice),
		newDotImpNonTypeDecl("ReDefinedIntTwice").AddType(dotImpReDefinedIntTwice),

		newDotImpTypeDecl("ReAliasedInt").AddType(aliasedInt),
		newDotImpNonTypeDecl("AliasedInt").AddType(aliasedInt),

		newDotImpTypeDecl("ReAliasedIntTwice").AddType(dotImpReAliasedInt),
		newDotImpNonTypeDecl("ReAliasedInt").AddType(dotImpReAliasedInt),

		newDotImpTypeDecl("ReAliasedIntThrice").AddType(dotImpReAliasedIntTwice),
		newDotImpNonTypeDecl("ReAliasedIntTwice").AddType(dotImpReAliasedIntTwice),

		newDotImpTypeDecl("ReDefinedFunc").AddType(definedFunc),
		newDotImpNonTypeDecl("DefinedFunc").AddType(definedFunc),
		newDotImpTypeDecl("ReAliasedFunc").AddType(aliasedFunc),
		newDotImpNonTypeDecl("AliasedFunc").AddType(aliasedFunc),

		newDotImpTypeDecl("Interface0").AddType(definedInt),
		{Name: "Method0"},
		newDotImpNonTypeDecl("DefinedInt").AddType(definedInt),

		newDotImpNonTypeDecl("Func0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		{Name: "p0"},
		newDotImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "p1"},
		newDotImpNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newDotImpNonTypeDecl("DefinedInt").AddType(definedInt),
		newDotImpNonTypeDecl("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		newDotImpNonTypeDecl("DefinedFunc").AddType(definedFunc),
		{Name: "r1"},
		newDotImpNonTypeDecl("AliasedFunc").AddType(aliasedFunc),
		{Name: "nil"},

		newDotImpNonTypeDecl("NonGlobalUse"),

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		newDotImpNonTypeDecl("CustomExportedStruct0").AddType(
			newDotImpTypeDecl("CustomExportedStruct0").AddType(exportedStruct0, exportedStruct1),
		),
		{Name: "ExportedStruct0"},
		newDotImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		newDotImpNonTypeDecl("ExportedStruct1").AddType(exportedStruct1),
		newDotImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		newDotImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},

		newDotImpNonTypeDecl("CustomDefinedIntIota0").AddType(definedInt),
		newDotImpNonTypeDecl("CustomDefinedIntIota1").AddType(definedInt),
		newDotImpNonTypeDecl("ReDefinedIntThrice").AddType(dotImpReDefinedIntThrice),
		newDotImpNonTypeDecl("ReAliasedIntThrice").AddType(dotImpReAliasedIntThrice),
		newDotImpNonTypeDecl("DefinedIntIota0Copy").AddType(definedInt),

		{Name: "es0"},
		newDotImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "es0"},
		{Name: "Method0"},

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		newDotImpNonTypeDecl("DefinedInt").AddType(definedInt),
		newDotImpNonTypeDecl("AliasedInt").AddType(aliasedInt),
		newDotImpNonTypeDecl("DefinedIntIota0").AddType(definedInt),

		newDotImpNonTypeDecl("ScopeTest"),

		{Name: "DefinedInt"},
		{Name: "int"},
		{Name: "_"},
		{Name: "DefinedInt"},

		{Name: "ReDefinedInt"},
		{Name: "int"},
		{Name: "_"},
		{Name: "ReDefinedInt"},

		{Name: "DefinedIntIota0"},
		{Name: "_"},
		{Name: "DefinedIntIota0"},
	}
	s.requireFileIdentInfo(i, dotImpFilename, expect)

	// Assert results for all the identifiers from ident_info/importer/named/named.go

	namedImpPkgPath := "fixture.tld/ident_info/importer/named"
	namedImpPkgName := path.Base(namedImpPkgPath)
	namedImpPos := token.Position{Filename: namedImpFilename}
	newNamedImpNonTypeDecl := func(identName string) *IdentExpect {
		return newNonTypeDecl(identName, namedImpPkgName, namedImpPkgPath, namedImpPos)
	}
	newNamedImpTypeDecl := func(identName string) *IdentExpect {
		return newTypeDecl(identName, namedImpPkgName, namedImpPkgPath, namedImpPos)
	}

	namedImpDefinedInt := newNamedImpTypeDecl("DefinedInt").AddType(definedInt)

	namedImpReDefinedInt := newNamedImpTypeDecl("ReDefinedInt").AddType(definedInt)
	namedImpReDefinedIntTwice := newNamedImpTypeDecl("ReDefinedIntTwice").AddType(namedImpReDefinedInt)
	namedImpReDefinedIntThrice := newNamedImpTypeDecl("ReDefinedIntThrice").AddType(namedImpReDefinedIntTwice)

	namedImpReAliasedInt := newNamedImpTypeDecl("ReAliasedInt").AddType(aliasedInt)
	namedImpReAliasedIntTwice := newNamedImpTypeDecl("ReAliasedIntTwice").AddType(namedImpReAliasedInt)
	namedImpReAliasedIntThrice := newNamedImpTypeDecl("ReAliasedIntThrice").AddType(namedImpReAliasedIntTwice)

	namedImpCustomExportedStruct0 := newNamedImpTypeDecl("CustomExportedStruct0").AddType(exportedStruct0, exportedStruct1)

	expect = []*IdentExpect{
		{Name: "named"},

		{Name: "other_pkg0"},
		{Name: "other_pkg1"},

		newNamedImpTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),

		newNamedImpNonTypeDecl("ConstFromNonInspectedPkg"),
		{Name: "other_pkg1"},
		{Name: "NonInspectedConst"},

		newNamedImpNonTypeDecl("CustomDefinedIntIota0").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "iota"},
		newNamedImpNonTypeDecl("CustomDefinedIntIota1").AddType(definedInt),

		newNamedImpNonTypeDecl("DefinedIntIota0"),

		newNamedImpNonTypeDecl("DefinedIntIota0Copy").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedIntIota0").AddType(definedInt),

		namedImpCustomExportedStruct0,
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct1").AddType(exportedStruct1),
		{Name: "Next"},
		newNamedImpNonTypeDecl("CustomExportedStruct0").AddType(namedImpCustomExportedStruct0),

		{Name: "recv"},
		newNamedImpNonTypeDecl("CustomExportedStruct0").AddType(namedImpCustomExportedStruct0),
		{Name: "Method0"},
		{Name: "p0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "r0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("AliasedInt").AddType(aliasedInt),

		namedImpDefinedInt,
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),

		newNamedImpTypeDecl("ReDefinedInt").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),

		newNamedImpTypeDecl("ReDefinedIntTwice").AddType(namedImpReDefinedInt),
		newNamedImpNonTypeDecl("ReDefinedInt").AddType(namedImpReDefinedInt),

		newNamedImpTypeDecl("ReDefinedIntThrice").AddType(namedImpReDefinedIntTwice),
		newNamedImpNonTypeDecl("ReDefinedIntTwice").AddType(namedImpReDefinedIntTwice),

		newNamedImpTypeDecl("ReAliasedInt").AddType(aliasedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("AliasedInt").AddType(aliasedInt),

		newNamedImpTypeDecl("ReAliasedIntTwice").AddType(namedImpReAliasedInt),
		newNamedImpNonTypeDecl("ReAliasedInt").AddType(namedImpReAliasedInt),

		newNamedImpTypeDecl("ReAliasedIntThrice").AddType(namedImpReAliasedIntTwice),
		newNamedImpNonTypeDecl("ReAliasedIntTwice").AddType(namedImpReAliasedIntTwice),

		newNamedImpTypeDecl("ReDefinedFunc").AddType(definedFunc),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedFunc").AddType(definedFunc),
		newNamedImpTypeDecl("ReAliasedFunc").AddType(aliasedFunc),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("AliasedFunc").AddType(aliasedFunc),

		newNamedImpTypeDecl("Interface0").AddType(definedInt),
		{Name: "Method0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),

		newNamedImpTypeDecl("Interface1").AddType(namedImpDefinedInt),
		{Name: "Method0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(namedImpDefinedInt),

		newNamedImpNonTypeDecl("NamedFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		{Name: "p0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "p1"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("AliasedInt").AddType(aliasedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedFunc").AddType(definedFunc),
		{Name: "r1"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("AliasedFunc").AddType(aliasedFunc),
		{Name: "nil"},

		newNamedImpNonTypeDecl("NonGlobalUse"),

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		newNamedImpNonTypeDecl("CustomExportedStruct0").AddType(
			newNamedImpTypeDecl("CustomExportedStruct0").AddType(exportedStruct0, exportedStruct1),
		),
		{Name: "ExportedStruct0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct1").AddType(exportedStruct1),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},

		newNamedImpNonTypeDecl("CustomDefinedIntIota0").AddType(definedInt),
		newNamedImpNonTypeDecl("ReDefinedIntThrice").AddType(namedImpReDefinedIntThrice),
		newNamedImpNonTypeDecl("ReAliasedIntThrice").AddType(namedImpReAliasedIntThrice),

		{Name: "es0"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("ExportedStruct0").AddType(exportedStruct0),
		{Name: "es0"},
		{Name: "Method0"},

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedInt").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("AliasedInt").AddType(aliasedInt),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("DefinedIntIota0").AddType(definedInt),

		newNamedImpNonTypeDecl("ScopeTest"),

		{Name: "ReDefinedInt"},
		{Name: "int"},
		{Name: "_"},
		{Name: "ReDefinedInt"},

		{Name: "ReAliasedInt"},
		{Name: "int"},
		{Name: "_"},
		{Name: "ReAliasedInt"},

		{Name: "DefinedIntIota0Copy"},
		{Name: "_"},
		{Name: "DefinedIntIota0Copy"},

		newNamedImpNonTypeDecl("ImportedInterfaceImplUse"),
		{Name: "other_pkg0"},
		newNamedImpNonTypeDecl("LocalInterface0Impl0").AddType(newLocalTypeDecl("LocalInterface0Impl0")),
		{Name: "Method0"},
	}
	s.requireFileIdentInfo(i, namedImpFilename, expect)
}
