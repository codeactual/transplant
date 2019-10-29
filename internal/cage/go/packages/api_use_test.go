// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages_test

import (
	"fmt"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_file_matcher "github.com/codeactual/transplant/internal/cage/os/file/matcher"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	testkit "github.com/codeactual/transplant/internal/cage/testkit"
)

type expectId struct {
	FullName string
}

func (s *ApiInspectorSuite) requireGlobalIdsUsedByNode(i *cage_pkgs.Inspector, dir, pkgName, nodeId string, expectIds []expectId) {
	t := s.T()

	node, err := i.GlobalIdNode(dir, pkgName, nodeId)
	require.NoError(t, err)

	assertId := fmt.Sprintf(
		"AST:\n%s\n\n"+
			"CODE:\n%s\n\n"+
			"ID: [%s]\n",
		spew.Sdump(node.Ast),
		i.NodeToString(node.Ast),
		nodeId,
	)

	usedMap, errs := i.GlobalIdsUsedByGlobal(dir, pkgName, nodeId)
	testkit.RequireNoErrors(t, errs)

	// Assert GlobalId.String() lists are the same.

	var expectIdStrs []string
	for _, id := range expectIds {
		expectIdStrs = append(expectIdStrs, id.FullName)
	}
	cage_strings.SortStable(expectIdStrs)

	var actualIdStrs []string
	for idStr := range usedMap {
		actualIdStrs = append(actualIdStrs, idStr)
	}
	cage_strings.SortStable(actualIdStrs)

	require.Exactly(t, expectIdStrs, actualIdStrs, assertId)
}

func (s *ApiInspectorSuite) TestGlobalIdsUsedByNode() {
	t := s.T()

	baseDir := s.FixturePath("use_and_shadowing")
	goDirs, goDirsErr := cage_file.NewFinder().Dir(baseDir).DirMatcher(cage_file_matcher.GoDir).GetDirnameMatches()
	require.NoError(t, goDirsErr)
	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, goDirs.Slice()...)

	pkgName := "use"
	mainDir := filepath.Join(baseDir, "main")

	// const: use of package-local

	pkglocalConstDir := filepath.Join(baseDir, "pkglocal", "const", pkgName)
	pkglocalFile := filepath.Join(pkglocalConstDir, pkgName+".go")

	initFilename := filepath.Join(pkglocalConstDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	initFilename = filepath.Join(pkglocalConstDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedConst1").String()},
	})

	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "ExportedConst4", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "ExportedConst5", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "ExportedConst6", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "ExportedConst7", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst5").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InCall", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InSwitchCase", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InAssign", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InAssignShort", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InCompositeLit", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InSingleReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InMultiReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InLocalConst", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InIf", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "InSwitch", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "DefinedIntNonIota0", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "DefinedIntNonIota1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "DefinedIntNonIota2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "UseDefinedIntNonIota", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntNonIota0").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntNonIota1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntNonIota2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "DefinedIntIota0", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "DefinedIntIota1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "DefinedIntIota2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "UseDefinedIntIota", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntIota0").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntIota1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntIota2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota0", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota3", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota4", nil)
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota5", nil)
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota6", nil)
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "MultiValIota7", nil)
	s.requireGlobalIdsUsedByNode(i, pkglocalConstDir, pkgName, "UseMultiValIota", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota0").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota5").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota6").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "MultiValIota7").String()},
	})

	// const: use of imported

	importerDir := filepath.Join(baseDir, "importer", "const", pkgName)
	importerFile := filepath.Join(importerDir, pkgName+".go")

	initFilename = filepath.Join(importerDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	initFilename = filepath.Join(importerDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})

	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst4", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst5", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst6", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst7", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst5").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCall", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitchCase", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssign", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignShort", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCompositeLit", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InLocalConst", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InIf", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitch", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseDefinedIntNonIota", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntNonIota0").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntNonIota1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntNonIota2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseDefinedIntIotaInImportedConst", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntIota0").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntIota1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedIntIota2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "DefinedIntIota0", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "DefinedIntIota1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "DefinedIntIota2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseDefinedIntIotaInLocalConst", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "DefinedIntIota0").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "DefinedIntIota1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "DefinedIntIota2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "DefinedInt").String()},
	})

	// func: use of package-local

	pkglocalFuncDir := filepath.Join(baseDir, "pkglocal", "func", pkgName)
	pkglocalFile = filepath.Join(pkglocalFuncDir, pkgName+".go")

	initFilename = filepath.Join(pkglocalFuncDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	initFilename = filepath.Join(pkglocalFuncDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})

	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InCallAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InCallForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InIfForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InSwitchForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InSwitchCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InSelectCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InSelectCaseAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InAssignAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InAssignShortAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InAssignShortForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InCompositeLitAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InCompositeLitForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InSingleReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InMultiReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InSingleReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InMultiReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InDefer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalFuncDir, pkgName, "InGoroutine", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})

	// func: use of imported

	importerDir = filepath.Join(baseDir, "importer", "func", pkgName)
	importerFile = filepath.Join(importerDir, pkgName+".go")

	initFilename = filepath.Join(importerDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	initFilename = filepath.Join(importerDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})

	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCallAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCallForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InIfForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitchForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitchCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSelectCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSelectCaseAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignShortAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignShortForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCompositeLitAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCompositeLitForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InDefer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InGoroutine", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedFunc1").String()},
	})

	// method: use of package-local
	//
	// The type associated with every method should be included in every GlobalIdsUsedByNode
	// assertion. The latter collects separate identifier for types and their methods as a
	// a convenience rather than force type use to be inferred from one or more method identifiers.

	pkglocalMethodDir := filepath.Join(baseDir, "pkglocal", "method", pkgName)
	pkglocalFile = filepath.Join(pkglocalMethodDir, pkgName+".go")

	initFilename = filepath.Join(pkglocalMethodDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})

	initFilename = filepath.Join(pkglocalMethodDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})

	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InCallAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InCallForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InIfForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InSwitchForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InSwitchCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InSelectCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InSelectCaseAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InAssignAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InAssignShortAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InAssignShortForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InCompositeLitAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InCompositeLitForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InSingleReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InMultiReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InSingleReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InMultiReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InDefer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "InGoroutine", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "StructWithBothReceiverTypes.PointerReceiver", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "StructWithBothReceiverTypes.ValueReceiver", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "UseStructWithBothReceiverTypes", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "UseStructWithAnonEmbeddedValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithAnonEmbeddedValue").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "UseStructWithAnonEmbeddedPointer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithAnonEmbeddedPointer").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "UseStructWithNamedEmbeddedValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithNamedEmbeddedValue").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "UseStructWithNamedEmbeddedPointer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithNamedEmbeddedPointer").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "NoReceiverName.Method", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "NoReceiverName").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalMethodDir, pkgName, "UseNoReceiverName", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "NoReceiverName").String()},
	})

	// method: use of imported

	importerDir = filepath.Join(baseDir, "importer", "method", pkgName)

	initFilename = filepath.Join(importerDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	initFilename = filepath.Join(importerDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})

	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCallAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCallForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InIfForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitchForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitchCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSelectCaseForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSelectCaseAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignShortAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignShortForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCompositeLitAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCompositeLitForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiReturnAsValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiReturnForReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InDefer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InGoroutine", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseStructWithBothReceiverTypes", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseStructWithAnonEmbeddedValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithAnonEmbeddedValue").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseStructWithAnonEmbeddedPointer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithAnonEmbeddedPointer").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseStructWithNamedEmbeddedValue", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithNamedEmbeddedValue").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseStructWithNamedEmbeddedPointer", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithBothReceiverTypes").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "StructWithNamedEmbeddedPointer").String()},
	})

	// type: use of package-local

	pkglocalTypeDir := filepath.Join(baseDir, "pkglocal", "type", pkgName)
	pkglocalFile = filepath.Join(pkglocalTypeDir, pkgName+".go")

	initFilename = filepath.Join(pkglocalTypeDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	initFilename = filepath.Join(pkglocalTypeDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})

	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedType4Named", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedType4NamedTwice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedType4NamedThrice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedAliasedIntAlias", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedAliasedIntAliasTwice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedAliasedIntAliasThrice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedVar1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedVar2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedConst1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedConst2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedConst3", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedConst4", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedVar3", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedVar4", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedVar5", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "ExportedVar6", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InCall", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InCallee").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InCallee", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InSwitchCase", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InAssign", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InAssignShort", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InCompositeLit", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InSingleReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InMultiReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InSingleTypeListFuncDecl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InMultiTypeListFuncDecl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "inMethodDeclTypeList.Single", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "inMethodDeclTypeList").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "inMethodDeclTypeList.Multi", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "inMethodDeclTypeList").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InTypeAssertion", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InTypeConversion", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InAliasType", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InNamedType", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InMakeMap", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InMakeChan", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InInterfaceMethod", []expectId{
		// from Method's param types
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},

		// from Method's return type
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "InInterfaceMethod.Method", []expectId{
		// from Method's param types
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},

		// from Method's return type
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "PassInInterfaceMethodImpl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethod").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethodImpl").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "UseInInterfaceMethodImpl").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "UseInInterfaceMethodImpl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethod").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "UseHasAnonEmbeddedStruct", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "HasAnonEmbeddedStruct").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "embedded").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "UseHasAnonEmbeddedStructMethod", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "HasAnonEmbeddedStruct").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "embedded").String()},
	})
	// TODO  UseInterfaceAsVarType contains two globals with overlapping type dependencies.
	//       For AssignUsagePos expectations below, it's unclear what they should be based on:
	//       the first global or second global. It matters because the LHS/RHS/Non-assign state
	//       depends on it.
	//       - eval: How the used IDs are deduplicated in the first place.
	//         - i.e. it may be last-usage wins and determines the AssignUsagePos value.
	s.requireGlobalIdsUsedByNode(i, pkglocalTypeDir, pkgName, "UseInterfaceAsVarType", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethod").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethodImpl").String()},
	})

	// type: use of imported

	importerDir = filepath.Join(baseDir, "importer", "type", pkgName)
	importerFile = filepath.Join(importerDir, pkgName+".go")

	initFilename = filepath.Join(importerDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	initFilename = filepath.Join(importerDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})

	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedType4Named", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedType4NamedTwice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "ExportedType4Named").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedType4NamedThrice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "ExportedType4NamedTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedAliasedIntAlias", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedAliasedIntAliasTwice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedAliasedIntAliasThrice", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "ExportedAliasedIntAliasTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedVar1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedVar2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst2", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst3", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst4", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst5", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedConst6", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedVar3", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedVar4", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedVar5", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "ExportedVar6", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCall", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "InCallee").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCallee", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitchCase", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssign", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignShort", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCompositeLit", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleTypeListFuncDecl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiTypeListFuncDecl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "inMethodDeclTypeList.Single", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "inMethodDeclTypeList").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "inMethodDeclTypeList.Multi", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "inMethodDeclTypeList").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InTypeAssertion", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InTypeConversion", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAliasType", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InNamedType", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMakeMap", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMakeChan", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InInterfaceMethod", []expectId{
		// from Method's param types
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},

		// from Method's return type
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InInterfaceMethod.Method", []expectId{
		// from Method's param types
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},

		// from Method's return type
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "PassInInterfaceMethodImpl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, importerFile, "UseInInterfaceMethodImpl").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethod").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethodImpl").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseInInterfaceMethodImpl", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethod").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseHasAnonEmbeddedStruct", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "HasAnonEmbeddedStruct").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "embedded").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseHasAnonEmbeddedStructMethod", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "HasAnonEmbeddedStruct").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "embedded").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "UseInterfaceAsVarType", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedInt").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAlias").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedAliasedIntAliasTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType3").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4Named").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedThrice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedType4NamedTwice").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethod").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "InInterfaceMethodImpl").String()},
	})

	// var: use of package-local

	pkglocalVarDir := filepath.Join(baseDir, "pkglocal", "var", pkgName)
	pkglocalFile = filepath.Join(pkglocalVarDir, pkgName+".go")

	initFilename = filepath.Join(pkglocalVarDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
	})
	initFilename = filepath.Join(pkglocalVarDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedVar1").String()},
	})

	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "strVar1", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "strType1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "strVar2", []expectId{})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "intWithoutExplicitType", []expectId{})

	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InCall", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedVar1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InSwitchCase", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedVar1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InAssign", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedVar1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InAssignShort", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InCompositeLit", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "nonExportedVar1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InSingleReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InMultiReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InIf", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, pkglocalVarDir, pkgName, "InSwitch", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
	})

	// var: use of imported

	importerDir = filepath.Join(baseDir, "importer", "var", pkgName)
	importerFile = filepath.Join(importerDir, pkgName+".go")

	initFilename = filepath.Join(importerDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
	})
	initFilename = filepath.Join(importerDir, pkgName+".go")
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, s.InitId(initFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
	})

	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCall", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitchCase", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssign", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InAssignShort", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InCompositeLit", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSingleReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InMultiReturn", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InIf", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar2").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, importerDir, pkgName, "InSwitch", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, pkglocalFile, "ExportedVar1").String()},
	})

	pkgName = "main"
	importerPkgName := "use"

	// main: use of package-local and imported

	mainFilename := filepath.Join(mainDir, pkgName+".go")

	constFilename := filepath.Join(pkglocalConstDir, importerPkgName+".go")
	funcFilename := filepath.Join(pkglocalFuncDir, importerPkgName+".go")
	typeFilename := filepath.Join(pkglocalTypeDir, importerPkgName+".go")
	varFilename := filepath.Join(pkglocalVarDir, importerPkgName+".go")

	mainInitFilename := filepath.Join(mainDir, "init.go")
	s.requireGlobalIdsUsedByNode(i, mainDir, pkgName, s.InitId(mainInitFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, mainFilename, "ExportedConst2").String()},
		{FullName: cage_pkgs.NewGlobalId("", importerPkgName, constFilename, "ExportedConst3").String()},
	})
	s.requireGlobalIdsUsedByNode(i, mainDir, pkgName, s.InitId(mainFilename), []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, mainFilename, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", importerPkgName, constFilename, "ExportedConst2").String()},
	})

	s.requireGlobalIdsUsedByNode(i, mainDir, pkgName, "main", []expectId{
		{FullName: cage_pkgs.NewGlobalId("", pkgName, mainFilename, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, mainFilename, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, mainFilename, "ExportedType1").String()},
		{FullName: cage_pkgs.NewGlobalId("", pkgName, mainFilename, "ExportedVar1").String()},
		{FullName: cage_pkgs.NewGlobalId("", importerPkgName, constFilename, "ExportedConst1").String()},
		{FullName: cage_pkgs.NewGlobalId("", importerPkgName, funcFilename, "ExportedFunc1").String()},
		{FullName: cage_pkgs.NewGlobalId("", importerPkgName, typeFilename, "ExportedType2").String()},
		{FullName: cage_pkgs.NewGlobalId("", importerPkgName, varFilename, "ExportedVar1").String()},
	})
}
