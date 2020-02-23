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
	"strings"

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

func NewIdentExpect(i *cage_pkgs.IdentInfo) *IdentExpect {
	return &IdentExpect{Name: i.Name, Info: i}
}

func (e *IdentExpect) Copy() *IdentExpect {
	c := IdentExpect{Name: e.Name}
	info := *e.Info
	c.Info = &info
	return &c
}

func (e *IdentExpect) AddType(expects ...*IdentExpect) *IdentExpect {
	for _, expect := range expects {
		e.Info.Types = append(e.Info.Types, expect.Info)
	}
	return e
}

func CycleOf(e *IdentExpect) *IdentExpect {
	info := *e.Info
	info.IsTypeDecl = false
	info.IsCycle = true
	info.Types = nil

	return &IdentExpect{
		Name: e.Name,
		Info: &info,
	}
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

	require.Exactly(
		t, expect.Info.IsTypeDecl, actualInfo.IsTypeDecl,
		fmt.Sprintf("%s: expected IdentInfo.IsTypeDecl to be %t", assertId, expect.Info.IsTypeDecl),
	)
	require.Exactly(t, expect.Info.PkgName, actualInfo.PkgName, assertId)
	require.Exactly(t, expect.Info.PkgPath, actualInfo.PkgPath, assertId)

	// Assert the IdentInfo of the node's type and underlying types (if any).

	indent := " -> "
	expectTypesString := expect.Info.TypesString(indent)
	if expectTypesString == "" {
		expectTypesString = "<none>\n"
	}
	actualTypesString := actualInfo.TypesString(indent)
	if actualTypesString == "" {
		actualTypesString = "<none>\n"
	}

	baseTypeAssertId := fmt.Sprintf(
		"%s\n\ntop-level expect IdentInfo.Types:\n\n%s\n"+
			"top-level actual IdentInfo.Types:\n\n%s",
		assertId, expectTypesString, actualTypesString,
	)

	expectTypeStack := cage_strings.NewStack()
	actualTypeStack := cage_strings.NewStack()

	var requireSameTypes func(e, a *cage_pkgs.IdentInfo, depth int)

	requireSameTypes = func(e, a *cage_pkgs.IdentInfo, depth int) {
		// Display the depth multiple types to emphasize where in the type chain the
		// failed expectation is located.
		typeAssertId := fmt.Sprintf(
			"\n\n****** queryAllPkgs DEPTH: %d ******\n\n%s\n\n"+
				"expect type stack (begins at depth 0):\n\t%s\n"+
				"actual type stack (begins at depth 0):\n\t%s\n\n"+
				"expect type: name [%s] pkg [%s] deps [%d] cycle [%t]\n"+
				"actual type: name [%s] pkg [%s] deps [%d] cycle [%t]",
			depth, baseTypeAssertId,
			strings.Join(expectTypeStack.Items(), "\n\t"),
			strings.Join(actualTypeStack.Items(), "\n\t"),
			e.Name, e.PkgPath, len(e.Types), e.IsCycle,
			a.Name, a.PkgPath, len(a.Types), a.IsCycle,
		)

		require.Exactly(t, a.IsCycle, a.IsCycle, typeAssertId)
		if e.IsCycle {
			require.False(t, e.IsTypeDecl, typeAssertId)
			require.Empty(t, e.Types, typeAssertId)
		}
		if a.IsCycle {
			require.False(t, a.IsTypeDecl, typeAssertId)
			require.Empty(t, a.Types, typeAssertId)
		}

		// Skip inspection of Types slice because it's always expected to be empty
		// and confirmed to be empty above.
		if e.IsCycle {
			return
		}

		if len(e.Types) == 0 {
			if len(a.Types) != 0 {
				for _, t := range a.Types {
					fmt.Printf("actual type: %s (cycle: %t)\n", t.IdShort(), t.IsCycle)
				}
			}
			require.Empty(t, a.Types, typeAssertId)
			return
		} else {
			if len(e.Types) != len(a.Types) {
				for _, t := range e.Types {
					fmt.Printf("expect type: %s\n", t.IdShort())
				}
				for _, t := range a.Types {
					fmt.Printf("actual type: %s\n", t.IdShort())
				}
			}
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

			// Skip inspection of Types slice because it's always expected to be empty
			// and confirmed to be empty below.
			if expectTypes[n].IsCycle {
				require.True(
					t, actualTypes[n].IsCycle, typeAssertId,
					fmt.Sprintf(
						"%s\n\nactual IdentInfo.Types[%d] (%s) is not a cycle start",
						typeAssertId, n, actualTypes[n].IdShort(),
					),
				)

				require.Empty(t, expectTypes[n].Types, typeAssertId)
				require.Empty(t, actualTypes[n].Types, typeAssertId)
				continue
			}

			// The type chain should only include IdentInfo values which describe type declarations,
			// not uses.
			require.True(
				t, expectTypes[n].IsTypeDecl,
				fmt.Sprintf(
					"%s\n\nexpect IdentInfo.Types[%d] (%s) is not a type declaration",
					typeAssertId, n, expectTypes[n].IdShort(),
				),
			)
			require.True(
				t, actualTypes[n].IsTypeDecl,
				fmt.Sprintf(
					"%s\n\nactual IdentInfo.Types[%d] (%s) is not a type declaration",
					typeAssertId, n, actualTypes[n].IdShort(),
				),
			)

			s.requireSimilarPosition(typeAssertId, expectTypes[n].Position.Filename, token.Position{}, actualTypes[n].Position)

			expectTypeStack.Push(expectTypes[n].PkgPath + "." + expectTypes[n].Name)
			actualTypeStack.Push(actualTypes[n].PkgPath + "." + actualTypes[n].Name)

			requireSameTypes(expectTypes[n], actualTypes[n], depth+1)

			expectTypeStack.Pop()
			actualTypeStack.Pop()
		}
	}

	expectTypeStack.Push(expect.Info.PkgPath + "." + expect.Info.Name)
	actualTypeStack.Push(actualInfo.PkgPath + "." + actualInfo.Name)

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
	cycleFilename := filepath.Join(baseDir, "cycle", "cycle.go")

	// Inspect the fixture packages.

	i := s.MustInspect(
		baseDir,
		cage_pkgs.LoadSyntax,
		[]string{
			filepath.Dir(localFilename),
			filepath.Dir(dotImpFilename),
			filepath.Dir(namedImpFilename),
			filepath.Dir(cycleFilename),
		}...,
	)

	// Provide common helpers.

	newTypeRef := func(identName, pkgName, pkgPath string, pos token.Position) *IdentExpect {
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
		e := newTypeRef(identName, pkgName, pkgPath, pos)
		e.Info.IsTypeDecl = true
		return e
	}

	// Assert results for all the identifiers from ident_info/local/local.go

	localPkgPath := "fixture.tld/ident_info/local"
	localPkgName := path.Base(localPkgPath)
	localPos := token.Position{Filename: localFilename}
	newLocalTypeRef := func(identName string) *IdentExpect {
		return newTypeRef(identName, localPkgName, localPkgPath, localPos)
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
		newLocalTypeRef("nonExportedStruct0").AddType(nonExportedStruct0),
		{Name: "Field0"},
		newLocalTypeRef("nonExportedStruct1").AddType(nonExportedStruct1),
		{Name: "Field1"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("DefinedFunc").AddType(definedFunc),
		{Name: "Field2"},
		{Name: "Shadowed0"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "Shadowed1"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "Shadowed0"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),

		{Name: "recv"},
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		{Name: "p0"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "r0"},
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),

		exportedStruct1,

		newLocalTypeDecl("nonExportedDefinedInt"),
		{Name: "int"},
		newLocalTypeDecl("anotherNonExportedDefinedInt").AddType(nonExportedDefinedInt),
		newLocalTypeRef("nonExportedDefinedInt").AddType(nonExportedDefinedInt),

		newLocalTypeDecl("DefinedInt").AddType(anotherNonExportedDefinedInt),
		newLocalTypeRef("anotherNonExportedDefinedInt").AddType(anotherNonExportedDefinedInt),

		newLocalTypeRef("DefinedIntIota0").AddType(definedInt),
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "iota"},
		newLocalTypeRef("DefinedIntIota1").AddType(definedInt),

		newLocalTypeDecl("nonExportedAliasedInt"),
		{Name: "int"},
		newLocalTypeDecl("anotherNonExportedAliasedInt").AddType(nonExportedAliasedInt),
		newLocalTypeRef("nonExportedAliasedInt").AddType(nonExportedAliasedInt),

		newLocalTypeDecl("AliasedInt").AddType(anotherNonExportedAliasedInt),
		newLocalTypeRef("anotherNonExportedAliasedInt").AddType(anotherNonExportedAliasedInt),

		newLocalTypeDecl("definedFunc").AddType(definedInt, aliasedInt),
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		newLocalTypeDecl("DefinedFunc").AddType(nonExportedDefinedFunc),
		newLocalTypeRef("definedFunc").AddType(nonExportedDefinedFunc),

		newLocalTypeDecl("aliasedFunc").AddType(definedInt, aliasedInt),
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		newLocalTypeDecl("AliasedFunc").AddType(nonExportedAliasedFunc),
		newLocalTypeRef("aliasedFunc").AddType(nonExportedAliasedFunc),

		newLocalTypeDecl("LocalInterface0").AddType(definedInt),
		{Name: "Method0"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),

		newLocalTypeDecl("LocalInterface0Impl0"),
		{Name: "recv"},
		newLocalTypeRef("LocalInterface0Impl0").AddType(newLocalTypeDecl("LocalInterface0Impl0")),
		{Name: "Method0"},
		{Name: "p0"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),

		newLocalTypeRef("IntLiteral100"),

		newLocalTypeRef("LocalFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		{Name: "p0"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "p1"},
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		newLocalTypeRef("DefinedFunc").AddType(definedFunc),
		{Name: "r1"},
		newLocalTypeRef("AliasedFunc").AddType(aliasedFunc),
		{Name: "nil"},

		newLocalTypeRef("LocalFunc1").AddType(definedInt, aliasedInt),
		{Name: "p0"},
		{Name: "p1"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "p2"},
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		{Name: "r1"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "r2"},
		{Name: "r3"},
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),

		newLocalTypeRef("LocalFunc2").AddType(
			definedInt, aliasedInt, definedFunc, aliasedFunc, nonExportedStruct0, nonExportedStruct1,
		),
		{Name: "p0"},
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		{Name: "p1"},
		newLocalTypeRef("DefinedFunc").AddType(definedFunc),
		newLocalTypeRef("AliasedFunc").AddType(aliasedFunc),
		newLocalTypeRef("nonExportedStruct0").AddType(nonExportedStruct0),
		newLocalTypeRef("nonExportedStruct1").AddType(nonExportedStruct1),
		{Name: "nil"},

		newLocalTypeRef("LocalNonGlobalUse"),

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
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "nonExportedStruct0"},
		newLocalTypeRef("nonExportedStruct0").AddType(nonExportedStruct0),
		{Name: "Field0"},
		newLocalTypeRef("nonExportedStruct1").AddType(nonExportedStruct1),
		{Name: "Shadowed0"},

		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "IntField"},
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field1"},
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Shadowed0"},

		newLocalTypeRef("DefinedInt").AddType(definedInt),
		newLocalTypeRef("AliasedInt").AddType(aliasedInt),
		newLocalTypeRef("DefinedIntIota0").AddType(definedInt),
		newLocalTypeRef("IntLiteral100"),
		newLocalTypeRef("LocalFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		newLocalTypeRef("LocalFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),

		{Name: "es0"},
		newLocalTypeRef("ExportedStruct0").AddType(exportedStruct0),
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
		newLocalTypeRef("DefinedInt").AddType(definedInt),
		{Name: "_"},
		{Name: "Shadowed0"},

		newLocalTypeRef("LocalScopeTest"),

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
	newDotImpTypeRef := func(identName string) *IdentExpect {
		return newTypeRef(identName, dotImpPkgName, dotImpPkgPath, dotImpPos)
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

		newDotImpTypeRef("ConstFromNonInspectedPkg"),
		{Name: "NonInspectedConst"},

		newDotImpTypeRef("CustomDefinedIntIota0").AddType(definedInt),
		newDotImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "iota"},
		newDotImpTypeRef("CustomDefinedIntIota1").AddType(definedInt),

		newDotImpTypeRef("DefinedIntIota0Copy").AddType(definedInt),
		newDotImpTypeRef("DefinedIntIota0").AddType(definedInt),

		dotImpCustomExportedStruct0,
		newDotImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		newDotImpTypeRef("ExportedStruct1").AddType(exportedStruct1),

		{Name: "recv"},
		newDotImpTypeRef("CustomExportedStruct0").AddType(dotImpCustomExportedStruct0),
		{Name: "Method0"},
		{Name: "p0"},
		newDotImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "r0"},
		newDotImpTypeRef("AliasedInt").AddType(aliasedInt),

		newDotImpTypeDecl("ReDefinedInt").AddType(definedInt),
		newDotImpTypeRef("DefinedInt").AddType(definedInt),

		newDotImpTypeDecl("ReDefinedIntTwice").AddType(dotImpReDefinedInt),
		newDotImpTypeRef("ReDefinedInt").AddType(dotImpReDefinedInt),

		newDotImpTypeDecl("ReDefinedIntThrice").AddType(dotImpReDefinedIntTwice),
		newDotImpTypeRef("ReDefinedIntTwice").AddType(dotImpReDefinedIntTwice),

		newDotImpTypeDecl("ReAliasedInt").AddType(aliasedInt),
		newDotImpTypeRef("AliasedInt").AddType(aliasedInt),

		newDotImpTypeDecl("ReAliasedIntTwice").AddType(dotImpReAliasedInt),
		newDotImpTypeRef("ReAliasedInt").AddType(dotImpReAliasedInt),

		newDotImpTypeDecl("ReAliasedIntThrice").AddType(dotImpReAliasedIntTwice),
		newDotImpTypeRef("ReAliasedIntTwice").AddType(dotImpReAliasedIntTwice),

		newDotImpTypeDecl("ReDefinedFunc").AddType(definedFunc),
		newDotImpTypeRef("DefinedFunc").AddType(definedFunc),
		newDotImpTypeDecl("ReAliasedFunc").AddType(aliasedFunc),
		newDotImpTypeRef("AliasedFunc").AddType(aliasedFunc),

		newDotImpTypeDecl("Interface0").AddType(definedInt),
		{Name: "Method0"},
		newDotImpTypeRef("DefinedInt").AddType(definedInt),

		newDotImpTypeRef("Func0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		{Name: "p0"},
		newDotImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "p1"},
		newDotImpTypeRef("AliasedInt").AddType(aliasedInt),
		newDotImpTypeRef("DefinedInt").AddType(definedInt),
		newDotImpTypeRef("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		newDotImpTypeRef("DefinedFunc").AddType(definedFunc),
		{Name: "r1"},
		newDotImpTypeRef("AliasedFunc").AddType(aliasedFunc),
		{Name: "nil"},

		newDotImpTypeRef("NonGlobalUse"),

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		newDotImpTypeRef("CustomExportedStruct0").AddType(
			newDotImpTypeDecl("CustomExportedStruct0").AddType(exportedStruct0, exportedStruct1),
		),
		{Name: "ExportedStruct0"},
		newDotImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		newDotImpTypeRef("ExportedStruct1").AddType(exportedStruct1),
		newDotImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		newDotImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},

		newDotImpTypeRef("CustomDefinedIntIota0").AddType(definedInt),
		newDotImpTypeRef("CustomDefinedIntIota1").AddType(definedInt),
		newDotImpTypeRef("ReDefinedIntThrice").AddType(dotImpReDefinedIntThrice),
		newDotImpTypeRef("ReAliasedIntThrice").AddType(dotImpReAliasedIntThrice),
		newDotImpTypeRef("DefinedIntIota0Copy").AddType(definedInt),

		{Name: "es0"},
		newDotImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "es0"},
		{Name: "Method0"},

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		newDotImpTypeRef("DefinedInt").AddType(definedInt),
		newDotImpTypeRef("AliasedInt").AddType(aliasedInt),
		newDotImpTypeRef("DefinedIntIota0").AddType(definedInt),

		newDotImpTypeRef("ScopeTest"),

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
	newNamedImpTypeRef := func(identName string) *IdentExpect {
		return newTypeRef(identName, namedImpPkgName, namedImpPkgPath, namedImpPos)
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
		newNamedImpTypeRef("ExportedStruct0").AddType(exportedStruct0),

		newNamedImpTypeRef("ConstFromNonInspectedPkg"),
		{Name: "other_pkg1"},
		{Name: "NonInspectedConst"},

		newNamedImpTypeRef("CustomDefinedIntIota0").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "iota"},
		newNamedImpTypeRef("CustomDefinedIntIota1").AddType(definedInt),

		newNamedImpTypeRef("DefinedIntIota0"),

		newNamedImpTypeRef("DefinedIntIota0Copy").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedIntIota0").AddType(definedInt),

		namedImpCustomExportedStruct0,
		{Name: "other_pkg0"},
		newNamedImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("ExportedStruct1").AddType(exportedStruct1),

		{Name: "recv"},
		newNamedImpTypeRef("CustomExportedStruct0").AddType(namedImpCustomExportedStruct0),
		{Name: "Method0"},
		{Name: "p0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "r0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("AliasedInt").AddType(aliasedInt),

		namedImpDefinedInt,
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),

		newNamedImpTypeDecl("ReDefinedInt").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),

		newNamedImpTypeDecl("ReDefinedIntTwice").AddType(namedImpReDefinedInt),
		newNamedImpTypeRef("ReDefinedInt").AddType(namedImpReDefinedInt),

		newNamedImpTypeDecl("ReDefinedIntThrice").AddType(namedImpReDefinedIntTwice),
		newNamedImpTypeRef("ReDefinedIntTwice").AddType(namedImpReDefinedIntTwice),

		newNamedImpTypeDecl("ReAliasedInt").AddType(aliasedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("AliasedInt").AddType(aliasedInt),

		newNamedImpTypeDecl("ReAliasedIntTwice").AddType(namedImpReAliasedInt),
		newNamedImpTypeRef("ReAliasedInt").AddType(namedImpReAliasedInt),

		newNamedImpTypeDecl("ReAliasedIntThrice").AddType(namedImpReAliasedIntTwice),
		newNamedImpTypeRef("ReAliasedIntTwice").AddType(namedImpReAliasedIntTwice),

		newNamedImpTypeDecl("ReDefinedFunc").AddType(definedFunc),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedFunc").AddType(definedFunc),
		newNamedImpTypeDecl("ReAliasedFunc").AddType(aliasedFunc),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("AliasedFunc").AddType(aliasedFunc),

		newNamedImpTypeDecl("Interface0").AddType(definedInt),
		{Name: "Method0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),

		newNamedImpTypeDecl("Interface1").AddType(namedImpDefinedInt),
		{Name: "Method0"},
		newNamedImpTypeRef("DefinedInt").AddType(namedImpDefinedInt),

		newNamedImpTypeRef("NamedFunc0").AddType(definedInt, aliasedInt, definedFunc, aliasedFunc),
		{Name: "p0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "p1"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("AliasedInt").AddType(aliasedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("AliasedInt").AddType(aliasedInt),
		{Name: "r0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedFunc").AddType(definedFunc),
		{Name: "r1"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("AliasedFunc").AddType(aliasedFunc),
		{Name: "nil"},

		newNamedImpTypeRef("NonGlobalUse"),

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		newNamedImpTypeRef("CustomExportedStruct0").AddType(
			newNamedImpTypeDecl("CustomExportedStruct0").AddType(exportedStruct0, exportedStruct1),
		),
		{Name: "ExportedStruct0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Field0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("ExportedStruct1").AddType(exportedStruct1),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "Method0"},

		newNamedImpTypeRef("CustomDefinedIntIota0").AddType(definedInt),
		newNamedImpTypeRef("ReDefinedIntThrice").AddType(namedImpReDefinedIntThrice),
		newNamedImpTypeRef("ReAliasedIntThrice").AddType(namedImpReAliasedIntThrice),

		{Name: "es0"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("ExportedStruct0").AddType(exportedStruct0),
		{Name: "es0"},
		{Name: "Method0"},

		{Name: "_"},
		{Name: "_"},
		{Name: "_"},
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedInt").AddType(definedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("AliasedInt").AddType(aliasedInt),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("DefinedIntIota0").AddType(definedInt),

		newNamedImpTypeRef("ScopeTest"),

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

		newNamedImpTypeRef("ImportedInterfaceImplUse"),
		{Name: "other_pkg0"},
		newNamedImpTypeRef("LocalInterface0Impl0").AddType(newLocalTypeDecl("LocalInterface0Impl0")),
		{Name: "Method0"},
	}
	s.requireFileIdentInfo(i, namedImpFilename, expect)

	// Assert results for all the identifiers from ident_info/cycle/cycle.go.

	cyclePkgPath := "fixture.tld/ident_info/cycle"
	cyclePkgName := path.Base(cyclePkgPath)
	cyclePos := token.Position{Filename: cycleFilename}
	newCycleTypeRef := func(identName string) *IdentExpect {
		return newTypeRef(identName, cyclePkgName, cyclePkgPath, cyclePos)
	}
	newCycleTypeDecl := func(identName string) *IdentExpect {
		return newTypeDecl(identName, cyclePkgName, cyclePkgPath, cyclePos)
	}

	aDeclTmpl := newCycleTypeDecl("A")
	bDeclTmpl := newCycleTypeDecl("B")
	cDeclTmpl := newCycleTypeDecl("C")

	aRefTmpl := newCycleTypeRef("A")
	bRefTmpl := newCycleTypeRef("B")
	cRefTmpl := newCycleTypeRef("C")

	copyTmpl := func(src *IdentExpect) func() *IdentExpect {
		return func() *IdentExpect {
			return src.Copy()
		}
	}

	aDecl := copyTmpl(aDeclTmpl)
	bDecl := copyTmpl(bDeclTmpl)
	cDecl := copyTmpl(cDeclTmpl)

	aRef := copyTmpl(aRefTmpl)
	bRef := copyTmpl(bRefTmpl)
	cRef := copyTmpl(cRefTmpl)

	// Expect an IdentInfo.Types to include a type the start of a cycle if NewIdentInfo's
	// recursive walk of type dependencies would revisit any type already encountered
	// during the current "path" traversed starting at NewIdentInfo's input identifier,
	// where the "path" is maintained as a stack.
	expect = []*IdentExpect{
		{Name: "cycle"},

		//
		// type A struct {
		//
		aDecl().
			AddType(CycleOf(aDecl())). // depends on itself due to "*A" field
			AddType(bDecl().
				AddType(CycleOf(aDecl())).
				AddType(CycleOf(bDecl())).
				AddType(cDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				),
			).
			AddType(cDecl().
				AddType(CycleOf(aDecl())).
				AddType(bDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(cDecl())),
			),

		//
		//   *A
		//
		aRef().AddType(aDecl().
			AddType(CycleOf(aDecl())).
			AddType(bDecl().
				AddType(CycleOf(aDecl())).
				AddType(CycleOf(bDecl())).
				AddType(cDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				),
			).
			AddType(cDecl().
				AddType(CycleOf(aDecl())).
				AddType(bDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(cDecl())),
			),
		),

		//
		//   *B
		//
		bRef().
			AddType(bDecl().
				AddType(aDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(cDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					),
				).
				AddType(CycleOf(bDecl())).
				AddType(cDecl().
					AddType(aDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				),
			),

		//
		//   *C
		//
		cRef().
			AddType(cDecl().
				AddType(aDecl().
					AddType(CycleOf(aDecl())).
					AddType(bDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					).
					AddType(CycleOf(cDecl())),
				).
				AddType(bDecl().
					AddType(aDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(cDecl())),
			),

		//
		// }
		//

		//
		// type B struct {
		//
		bDecl().
			AddType(aDecl().
				AddType(CycleOf(aDecl())).
				AddType(CycleOf(bDecl())).
				AddType(cDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				),
			).
			AddType(CycleOf(bDecl())). // depends on itself due to "*B" field
			AddType(cDecl().
				AddType(aDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(bDecl())).
				AddType(CycleOf(cDecl())),
			),

		//
		//   *A
		//
		aRef().
			AddType(aDecl().
				AddType(CycleOf(aDecl())).
				AddType(bDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(cDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					),
				).
				AddType(cDecl().
					AddType(CycleOf(aDecl())).
					AddType(bDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					).
					AddType(CycleOf(cDecl())),
				),
			),

		//
		//   *B
		//
		bRef().AddType(bDecl().
			AddType(aDecl().
				AddType(CycleOf(aDecl())).
				AddType(CycleOf(bDecl())).
				AddType(cDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				),
			).
			AddType(CycleOf(bDecl())).
			AddType(cDecl().
				AddType(aDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(bDecl())).
				AddType(CycleOf(cDecl())),
			),
		),

		//
		//   *C
		//
		cRef().
			AddType(cDecl().
				AddType(aDecl().
					AddType(CycleOf(aDecl())).
					AddType(bDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					).
					AddType(CycleOf(cDecl())),
				).
				AddType(bDecl().
					AddType(aDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
					).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(cDecl())),
			),

		//
		// }
		//

		//
		// type C struct {
		//
		cDecl().
			AddType(aDecl().
				AddType(CycleOf(aDecl())).
				AddType(bDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(cDecl())),
			).
			AddType(bDecl().
				AddType(
					aDecl().
						AddType(CycleOf(aDecl())).
						AddType(CycleOf(bDecl())).
						AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(bDecl())).
				AddType(CycleOf(cDecl())),
			).
			AddType(CycleOf(cDecl())), // depends on itself due to "*C" field

		//
		//   *A
		//
		aRef().AddType(aDecl().
			AddType(CycleOf(aDecl())).
			AddType(bDecl().
				AddType(CycleOf(aDecl())).
				AddType(CycleOf(bDecl())).
				AddType(cDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				),
			).
			AddType(cDecl().
				AddType(CycleOf(aDecl())).
				AddType(bDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(cDecl())),
			),
		),

		//
		//   *B
		//
		bRef().AddType(bDecl().
			AddType(aDecl().
				AddType(CycleOf(aDecl())).
				AddType(CycleOf(bDecl())).
				AddType(cDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				),
			).
			AddType(CycleOf(bDecl())).
			AddType(cDecl().
				AddType(aDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(bDecl())).
				AddType(CycleOf(cDecl())),
			),
		),

		//
		//   *C
		//
		cRef().AddType(cDecl().
			AddType(aDecl().
				AddType(CycleOf(aDecl())).
				AddType(bDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(cDecl())),
			).
			AddType(bDecl().
				AddType(aDecl().
					AddType(CycleOf(aDecl())).
					AddType(CycleOf(bDecl())).
					AddType(CycleOf(cDecl())),
				).
				AddType(CycleOf(bDecl())).
				AddType(CycleOf(cDecl())),
			).
			AddType(CycleOf(cDecl())),
		),

		//
		// }
		//
	}
	s.requireFileIdentInfo(i, cycleFilename, expect)
}
