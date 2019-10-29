// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages_test

import (
	"go/ast"
	"path"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	std_packages "golang.org/x/tools/go/packages"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	testkit "github.com/codeactual/transplant/internal/cage/testkit"
)

type ApiInspectorSuite struct {
	BaseInspectorSuite
}

func TestApiInspectorSuite(t *testing.T) {
	suite.Run(t, new(ApiInspectorSuite))
}

func (s *ApiInspectorSuite) TestNonStdImportsWithoutSyntax() {
	t := s.T()

	// reuse a fixture with a simple import structure:
	//
	//   use.go and init.go are in the same package
	//   use.go and init.go both import importPkg
	pkg := "use"
	baseDir := s.FixturePath("use_and_shadowing")
	dir := filepath.Join(baseDir, "importer", "const", pkg)
	importPkg := "fixture.tld/use_and_shadowing/pkglocal/const/use"

	i := s.MustInspect(baseDir, std_packages.NeedImports, dir)

	testkit.RequireNoErrors(t, i.Inspect())

	// assert import info was collected for init.go's and use.go's dir

	var actualDirs []string
	for k := range i.NonStdImports {
		actualDirs = append(actualDirs, k)
	}
	cage_strings.SortStable(actualDirs)
	require.Exactly(t, []string{dir}, actualDirs)

	// assert import info was collected for init.go's and use.go's package

	var actualPkgNames []string
	for k := range i.NonStdImports[dir] {
		actualPkgNames = append(actualPkgNames, k)
	}
	require.Exactly(t, []string{pkg}, actualPkgNames)

	// assert import info was collected for init.go and use.go (indexed
	// by their dir due to missing AST data)

	var actualFilenames []string
	for k := range i.NonStdImports[dir][pkg] {
		actualFilenames = append(actualFilenames, k)
	}
	cage_strings.SortStable(actualFilenames)
	require.Exactly(t, []string{dir}, actualFilenames)

	// assert path imported by init.go was collected

	var actualPaths []string
	for k := range i.NonStdImports[dir][pkg][dir] {
		actualPaths = append(actualPaths, k)
	}
	cage_strings.SortStable(actualPaths)
	require.Exactly(t, []string{importPkg}, actualPaths)
}

func (s *ApiInspectorSuite) TestNonStdImportsWithSyntax() {
	t := s.T()

	// reuse a fixture with a simple import structure:
	//
	//   useFile and initFile are in the same package
	//   useFile and initFile both import importPkg
	pkg := "use"
	baseDir := s.FixturePath("use_and_shadowing")
	dir := filepath.Join(baseDir, "importer", "const", pkg)
	useFile := filepath.Join(dir, "use.go")
	initFile := filepath.Join(dir, "init.go")
	importPkg := "fixture.tld/use_and_shadowing/pkglocal/const/use"

	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, dir)

	// assert import info was collected for initFile's and useFile's dir

	var actualDirs []string
	for k := range i.NonStdImports {
		actualDirs = append(actualDirs, k)
	}
	cage_strings.SortStable(actualDirs)
	require.Exactly(t, []string{dir}, actualDirs)

	// assert import info was collected for initFile's and useFile's package

	var actualPkgNames []string
	for k := range i.NonStdImports[dir] {
		actualPkgNames = append(actualPkgNames, k)
	}
	require.Exactly(t, []string{pkg}, actualPkgNames)

	// assert import info was collected for initFile and useFile

	var actualFilenames []string
	for k := range i.NonStdImports[dir][pkg] {
		actualFilenames = append(actualFilenames, k)
	}
	cage_strings.SortStable(actualFilenames)
	require.Exactly(t, []string{initFile, useFile}, actualFilenames)

	// assert path imported by initFile was collected

	var actualPaths []string
	for k := range i.NonStdImports[dir][pkg][initFile] {
		actualPaths = append(actualPaths, k)
	}
	cage_strings.SortStable(actualPaths)
	require.Exactly(t, []string{importPkg}, actualPaths)

	// assert path imported by useFile was collected

	actualPaths = []string{}
	for k := range i.NonStdImports[dir][pkg][useFile] {
		actualPaths = append(actualPaths, k)
	}
	cage_strings.SortStable(actualPaths)
	require.Exactly(t, []string{importPkg}, actualPaths)
}

func (s *ApiInspectorSuite) TestGlobalDeclCollectionBaseline() {
	t := s.T()

	pkgName := "baseline"
	baseDir := s.FixturePath("global_decl", "baseline")
	filename := filepath.Join(baseDir, pkgName+".go")

	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, baseDir)

	requireGenDecl := func(idName string, n ast.Node) {
		switch n.(type) {
		case *ast.GenDecl:
			return
		}
		t.Fatalf("expected node [%s] to be GenDecl: %s\n", idName, spew.Sdump(n))
	}
	requireFuncDecl := func(idName string, n ast.Node) {
		switch n.(type) {
		case *ast.FuncDecl:
			return
		}
		t.Fatalf("expected node [%s] to be FuncDecl: %s\n", idName, spew.Sdump(n))
	}

	globalNodeIds := func(m cage_pkgs.IdToGlobalNode) (ids []string) {
		for k := range m {
			ids = append(ids, k)
		}
		cage_strings.SortStable(ids)
		return ids
	}

	idName := "Const1"
	node, err := i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 1)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Const2"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Const2", "Const3"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Const3"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Const2", "Const3"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Const4"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Const4", "Const5"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Const5"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Const4", "Const5"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Var1"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 1)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Var2"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Var2", "Var3"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Var3"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Var2", "Var3"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Var4"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Var4", "Var5"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Var5"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Var4", "Var5"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Struct"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 2)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName+".S").String(),
		i.GlobalNodes[node.Ast][idName+".S"].Id.String(),
	)

	idName = "Struct.Method"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireFuncDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 1)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Type1"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Type1", "Type2"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Type2"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Exactly(
		t,
		[]string{"Type1", "Type2"},
		globalNodeIds(i.GlobalNodes[node.Ast]),
	)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "NamedType"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 1)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "NamedType"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 1)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "AliasType"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireGenDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 1)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)

	idName = "Func"
	node, err = i.GlobalIdNode(baseDir, pkgName, idName)
	require.NoError(t, err)
	requireFuncDecl(idName, node.Ast)
	require.Len(t, i.GlobalNodes[node.Ast], 1)
	require.Exactly(
		t,
		cage_pkgs.NewGlobalId("", pkgName, filename, idName).String(),
		i.GlobalNodes[node.Ast][idName].Id.String(),
	)
}

func (s *ApiInspectorSuite) TestMainGlobalIdNodes() {
	t := s.T()

	baseDir := s.FixturePath("use_and_shadowing")
	mainDir := filepath.Join(baseDir, "main")

	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, mainDir)

	var keys []string
	for k := range i.GlobalIdNodes {
		keys = append(keys, k)
	}
	require.Exactly(t, []string{mainDir}, keys)

	keys = []string{}
	for k := range i.GlobalIdNodes[mainDir] {
		keys = append(keys, k)
	}
	require.Exactly(t, []string{"main"}, keys)

	require.Exactly(
		t,
		[]string{
			// main.go
			"ExportedConst1",
			"ExportedConst2",
			"ExportedConst3",
			"ExportedFunc1",
			"ExportedType1",
			"ExportedVar1",

			// shadow.go
			"WithConst",
			"WithType",
			"WithVar",

			s.InitId(filepath.Join(mainDir, "init.go")),
			s.InitId(filepath.Join(mainDir, "main.go")),
			s.InitId(filepath.Join(mainDir, "shadow.go")),

			// main.go
			"main",
			"nonExportedConst1",
			"nonExportedConst2",
		},
		i.GlobalIdNodes[mainDir]["main"].SortedIds(),
	)
}

func (s *ApiInspectorSuite) TestBlankImports() {
	t := s.T()

	baseDir := s.FixturePath("blank_imports")
	dirs := []string{
		filepath.Join(baseDir, "pkg0"),
		filepath.Join(baseDir, "pkg1"),
	}
	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, dirs...)

	testkit.RequireNoErrors(t, i.Inspect())

	require.Len(t, i.BlankImports, 2)

	require.Len(t, i.BlankImports[dirs[0]], 1)
	require.Len(t, i.BlankImports[dirs[0]]["pkg0"], 2)
	require.Exactly(
		t,
		[]string{
			"io/ioutil",
			"path/filepath",
		},
		i.BlankImportsInFile(dirs[0], "pkg0", filepath.Join(dirs[0], "pkg0a.go")).SortedSlice(),
	)
	require.Exactly(
		t,
		[]string{
			"runtime/debug",
			"text/template",
		},
		i.BlankImportsInFile(dirs[0], "pkg0", filepath.Join(dirs[0], "pkg0b.go")).SortedSlice(),
	)

	require.Len(t, i.BlankImports[dirs[1]], 1)
	require.Len(t, i.BlankImports[dirs[1]]["pkg1"], 2)
	require.Exactly(
		t,
		[]string{
			"bytes",
			"strings",
		},
		i.BlankImportsInFile(dirs[1], "pkg1", filepath.Join(dirs[1], "pkg1a.go")).SortedSlice(),
	)
	require.Exactly(
		t,
		[]string{
			"log",
			"net/http",
		},
		i.BlankImportsInFile(dirs[1], "pkg1", filepath.Join(dirs[1], "pkg1b.go")).SortedSlice(),
	)
}

func (s *ApiInspectorSuite) TestDotImports() {
	t := s.T()

	baseDir := s.FixturePath("dot_imports")
	dirs := []string{
		filepath.Join(baseDir, "pkg0"),
		filepath.Join(baseDir, "pkg1"),
	}
	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, dirs...)

	require.Len(t, i.DotImports, 2)

	require.Len(t, i.DotImports[dirs[0]], 1)
	require.Len(t, i.DotImports[dirs[0]]["pkg0"], 2)
	require.Exactly(
		t,
		[]string{
			"io/ioutil",
			"path/filepath",
		},
		i.DotImportsInFile(dirs[0], "pkg0", filepath.Join(dirs[0], "pkg0a.go")).SortedSlice(),
	)
	require.Exactly(
		t,
		[]string{
			"runtime/debug",
			"text/template",
		},
		i.DotImportsInFile(dirs[0], "pkg0", filepath.Join(dirs[0], "pkg0b.go")).SortedSlice(),
	)

	require.Len(t, i.DotImports[dirs[1]], 1)
	require.Len(t, i.DotImports[dirs[1]]["pkg1"], 2)
	require.Exactly(
		t,
		[]string{
			"crypto/sha256",
			"strings",
		},
		i.DotImportsInFile(dirs[1], "pkg1", filepath.Join(dirs[1], "pkg1a.go")).SortedSlice(),
	)
	require.Exactly(
		t,
		[]string{
			"log",
			"net/http",
		},
		i.DotImportsInFile(dirs[1], "pkg1", filepath.Join(dirs[1], "pkg1b.go")).SortedSlice(),
	)
}

type WalkGlobalIdsUsedByGlobalSuite struct {
	BaseInspectorSuite
}

func TestWalkGlobalIdsUsedByGlobalSuite(t *testing.T) {
	suite.Run(t, new(WalkGlobalIdsUsedByGlobalSuite))
}

func (s *WalkGlobalIdsUsedByGlobalSuite) TestTargetNodeOmitted() {
	t := s.T()

	fixtureId := "target_node_omitted"
	baseDir := s.FixturePath("walk_global_ids_used_by_node")
	dir := filepath.Join(baseDir, fixtureId)

	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, dir, filepath.Join(dir, "dep1"))

	usedIds := cage_strings.NewSet()
	walkFn := func(used cage_pkgs.IdUsedByNode) {
		usedIds.Add(used.IdentInfo.PkgName + "." + used.IdentInfo.Name)
	}

	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "Const", walkFn))
	require.Exactly(t, []string{}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "ConstAssignedWithImported", walkFn))
	require.Exactly(t, []string{"dep1.Const"}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "Var", walkFn))
	require.Exactly(t, []string{}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "VarAssignedWithImported", walkFn))
	require.Exactly(t, []string{"dep1.Var"}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "Struct", walkFn))
	require.Exactly(t, []string{}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "Struct.Method", walkFn))
	require.Exactly(
		t,
		[]string{"dep1.Struct", "target_node_omitted.Struct"},
		usedIds.SortedSlice(),
	)

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "NamedType", walkFn))
	require.Exactly(t, []string{"target_node_omitted.Struct"}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "NamedImportedType", walkFn))
	require.Exactly(t, []string{"dep1.Struct"}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "AliasType", walkFn))
	require.Exactly(t, []string{"target_node_omitted.Struct"}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "AliasImportedType", walkFn))
	require.Exactly(t, []string{"dep1.Struct"}, usedIds.SortedSlice())

	usedIds.Clear()
	testkit.RequireNoErrors(t, i.WalkGlobalIdsUsedByGlobal(dir, fixtureId, "Func", walkFn))
	require.Exactly(t, []string{"dep1.Func"}, usedIds.SortedSlice())
}

func (s *WalkGlobalIdsUsedByGlobalSuite) TestPackagesUsedByNode() {
	t := s.T()

	fixtureId := "baseline"
	pkgName := fixtureId
	basePkgPath := "fixture.tld/packages_used_by_node/baseline"
	baseDir := s.FixturePath("packages_used_by_node")
	fixtureDir := filepath.Join(baseDir, fixtureId)
	targetDirs := []string{
		fixtureDir,
		filepath.Join(fixtureDir, "dep1"),
		filepath.Join(fixtureDir, "dep2"),
		filepath.Join(fixtureDir, "pkg_name_differs"),
	}

	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, targetDirs...)

	pkgsByName, pkgsByPath, errs := i.PackagesUsedByNode(fixtureDir, pkgName, i.GlobalIdNodes[fixtureDir][pkgName]["ExportedFunc1"].Ast)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(
		t,
		map[string]cage_pkgs.PackageUsedByNode{
			"dep1":    {Name: "dep1", Path: path.Join(basePkgPath, "dep1")},
			"strings": {Name: "strings", Path: "strings"},
		},
		pkgsByName,
	)
	require.Exactly(
		t,
		map[string]cage_pkgs.PackageUsedByNode{
			path.Join(basePkgPath, "dep1"): {Name: "dep1", Path: path.Join(basePkgPath, "dep1")},
			"strings":                      {Name: "strings", Path: "strings"},
		},
		pkgsByPath,
	)

	pkgsByName, pkgsByPath, errs = i.PackagesUsedByNode(fixtureDir, pkgName, i.GlobalIdNodes[fixtureDir][pkgName]["ExportedFunc2"].Ast)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(
		t,
		map[string]cage_pkgs.PackageUsedByNode{
			"dep2":    {Name: "dep2", Path: path.Join(basePkgPath, "dep2")},
			"runtime": {Name: "runtime", Path: "runtime"},
		},
		pkgsByName,
	)
	require.Exactly(
		t,
		map[string]cage_pkgs.PackageUsedByNode{
			path.Join(basePkgPath, "dep2"): {Name: "dep2", Path: path.Join(basePkgPath, "dep2")},
			"runtime":                      {Name: "runtime", Path: "runtime"},
		},
		pkgsByPath,
	)

	pkgsByName, pkgsByPath, errs = i.PackagesUsedByNode(fixtureDir, pkgName, i.GlobalIdNodes[fixtureDir][pkgName]["ExportedFunc3"].Ast)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(
		t,
		map[string]cage_pkgs.PackageUsedByNode{
			"pkg_name_differs_from_dir_name": {
				Name: "pkg_name_differs_from_dir_name", Path: path.Join(basePkgPath, "pkg_name_differs"),
			},
		},
		pkgsByName,
	)
	require.Exactly(
		t,
		map[string]cage_pkgs.PackageUsedByNode{
			path.Join(basePkgPath, "pkg_name_differs"): {
				Name: "pkg_name_differs_from_dir_name", Path: path.Join(basePkgPath, "pkg_name_differs"),
			},
		},
		pkgsByPath,
	)
}
