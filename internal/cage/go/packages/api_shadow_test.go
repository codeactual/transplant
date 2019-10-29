// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages_test

import (
	"fmt"
	"path/filepath"

	"github.com/stretchr/testify/require"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_file_matcher "github.com/codeactual/transplant/internal/cage/os/file/matcher"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

func (s *ApiInspectorSuite) requireGlobalShadowSourceDirs(shadows cage_pkgs.DirGlobalIdShadows, expected []string) {
	var actual []string
	for k := range shadows {
		actual = append(actual, k)
	}
	cage_strings.SortStable(actual)
	require.Exactly(s.T(), expected, actual)
}

func (s *ApiInspectorSuite) requireGlobalShadowSourcePkgs(shadows cage_pkgs.DirGlobalIdShadows, dir string, expected []string) {
	var actual []string
	for k := range shadows[dir] {
		actual = append(actual, k)
	}
	cage_strings.SortStable(actual)
	require.Exactly(s.T(), expected, actual)
}

func (s *ApiInspectorSuite) requireGlobalShadowSourceIds(shadows cage_pkgs.DirGlobalIdShadows, dir, pkg string, expected []string) {
	t := s.T()

	_, ok := shadows[dir]
	require.True(t, ok, dir)
	_, ok = shadows[dir][pkg]
	require.True(t, ok, dir+"."+pkg)

	var actual []string
	for _, fileShadows := range shadows[dir][pkg] {
		for k := range fileShadows {
			actual = append(actual, k)
		}
	}
	cage_strings.SortStable(actual)
	require.Exactly(t, expected, actual)
}

func (s *ApiInspectorSuite) requireNoGlobalShadowIds(shadows cage_pkgs.DirGlobalIdShadows, dir, pkg, sourceId string) {
	t := s.T()

	_, ok := shadows[dir]
	require.True(t, ok, dir)
	_, ok = shadows[dir][pkg]
	require.True(t, ok, dir+"."+pkg)

	for filename, fileShadows := range shadows[dir][pkg] {
		actual, found := fileShadows[sourceId]
		require.False(t, found, fmt.Sprintf("func/method source [%s] found in file [%s]: %q", dir+"."+pkg+"."+sourceId, filename, actual))
	}
}

func (s *ApiInspectorSuite) requireGlobalShadowIds(shadows cage_pkgs.DirGlobalIdShadows, dir, pkg, sourceId string, expected []string) {
	t := s.T()

	_, ok := shadows[dir]
	require.True(t, ok, dir)
	_, ok = shadows[dir][pkg]
	require.True(t, ok, dir+"."+pkg)

	var found bool
	var filenames []string
	for filename, fileShadows := range shadows[dir][pkg] {
		actual, ok := fileShadows[sourceId]
		if ok {
			found = true
			require.Exactly(t, expected, actual.SortedSlice())
			break
		}
		filenames = append(filenames, filename)
	}
	require.True(t, found, fmt.Sprintf("func/method source [%s] not found in files %v", dir+"."+pkg+"."+sourceId, filenames))
}

func (s *ApiInspectorSuite) TestGlobalIdShadows() {
	t := s.T()

	baseDir := s.FixturePath("use_and_shadowing")
	goDirs, goDirsErr := cage_file.NewFinder().Dir(baseDir).DirMatcher(cage_file_matcher.GoDir).GetDirnameMatches()
	require.NoError(t, goDirsErr)
	i := s.MustInspect(baseDir, cage_pkgs.LoadSyntax, goDirs.Slice()...)

	shadows := i.GlobalIdShadows
	pkg := "shadow"
	mainDir := filepath.Join(baseDir, "main")

	pkglocalConstShadowDir := filepath.Join(baseDir, "pkglocal", "const", pkg)
	pkglocalFuncShadowDir := filepath.Join(baseDir, "pkglocal", "func", pkg)
	pkglocalImportNameShadowDir := filepath.Join(baseDir, "pkglocal", "import_name", pkg)
	pkglocalMethodShadowDir := filepath.Join(baseDir, "pkglocal", "method", pkg)
	pkglocalTypeShadowDir := filepath.Join(baseDir, "pkglocal", "type", pkg)
	pkglocalVarShadowDir := filepath.Join(baseDir, "pkglocal", "var", pkg)

	s.requireGlobalShadowSourceDirs(shadows, []string{
		mainDir,
		pkglocalConstShadowDir,
		pkglocalFuncShadowDir,
		pkglocalImportNameShadowDir,
		pkglocalMethodShadowDir,
		pkglocalTypeShadowDir,
		pkglocalVarShadowDir,
	})

	// const

	s.requireGlobalShadowSourcePkgs(shadows, pkglocalConstShadowDir, []string{pkg})
	s.requireGlobalShadowSourceIds(shadows, pkglocalConstShadowDir, pkg, []string{
		"WithConst",
		"WithType",
		"WithVar",
		s.InitId(filepath.Join(pkglocalConstShadowDir, "init.go")),
		s.InitId(filepath.Join(pkglocalConstShadowDir, "shadow.go")),
	})
	s.requireGlobalShadowIds(shadows, pkglocalConstShadowDir, pkg, "WithConst", []string{
		"ExportedConst1",
		"ExportedConst2",
		"nonExportedConst1",
		"nonExportedConst2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalConstShadowDir, pkg, "WithType", []string{
		"ExportedConst1",
	})
	s.requireGlobalShadowIds(shadows, pkglocalConstShadowDir, pkg, "WithVar", []string{
		"ExportedConst1",
		"ExportedConst2",
		"ExportedConst3",
		"nonExportedConst1",
		"nonExportedConst2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalConstShadowDir, pkg, s.InitId(filepath.Join(pkglocalConstShadowDir, "init.go")), []string{
		"ExportedConst2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalConstShadowDir, pkg, s.InitId(filepath.Join(pkglocalConstShadowDir, "shadow.go")), []string{
		"ExportedConst1",
	})

	// func

	s.requireGlobalShadowSourcePkgs(shadows, pkglocalFuncShadowDir, []string{pkg})
	s.requireGlobalShadowSourceIds(shadows, pkglocalFuncShadowDir, pkg, []string{
		"InParamName",
		"WithConst",
		"WithType",
		"WithVar",
		s.InitId(filepath.Join(pkglocalFuncShadowDir, "init.go")),
		s.InitId(filepath.Join(pkglocalFuncShadowDir, "shadow.go")),
	})
	s.requireGlobalShadowIds(shadows, pkglocalFuncShadowDir, pkg, "WithConst", []string{
		"ExportedFunc1",
		"ExportedFunc2",
		"nonExportedFunc1",
		"nonExportedFunc2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalFuncShadowDir, pkg, "WithType", []string{
		"ExportedFunc1",
	})
	s.requireGlobalShadowIds(shadows, pkglocalFuncShadowDir, pkg, "WithVar", []string{
		"ExportedFunc1",
		"ExportedFunc2",
		"ExportedFunc3",
		"nonExportedFunc1",
		"nonExportedFunc2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalFuncShadowDir, pkg, "InParamName", []string{
		"ExportedFunc2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalFuncShadowDir, pkg, s.InitId(filepath.Join(pkglocalFuncShadowDir, "init.go")), []string{
		"ExportedFunc2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalFuncShadowDir, pkg, s.InitId(filepath.Join(pkglocalFuncShadowDir, "shadow.go")), []string{
		"ExportedFunc1",
	})

	// import name

	s.requireGlobalShadowSourcePkgs(shadows, pkglocalImportNameShadowDir, []string{pkg})
	s.requireGlobalShadowSourceIds(shadows, pkglocalImportNameShadowDir, pkg, []string{
		"ShadowConst",
		"ShadowLongVar",
		"ShadowShortVar",
		"ShadowType",
	})
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "FuncParamDeclName")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "FuncParamDeclNameDiffersFromDir")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "FuncParamUsedNameStdlib")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "FuncParamUsedNameNonStdlib")
	s.requireGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "ShadowLongVar", []string{
		"pkg_name_is_not_dir_name",
		"strings",
		"used_name_non_stdlib",
		"used_name_stdlib",
	})
	s.requireGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "ShadowShortVar", []string{
		"pkg_name_is_not_dir_name",
		"strings",
		"used_name_non_stdlib",
		"used_name_stdlib",
	})
	s.requireGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "ShadowConst", []string{
		"pkg_name_is_not_dir_name",
		"strings",
		"used_name_non_stdlib",
		"used_name_stdlib",
	})
	s.requireGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "ShadowType", []string{
		"pkg_name_is_not_dir_name",
		"strings",
		"used_name_non_stdlib",
		"used_name_stdlib",
	})
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowParam.DeclName")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowParam.DeclNameDiffersFromDir")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowParam.UsedNameStdlib")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowParam.UsedNameNonStdlib")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowReceiver.DeclName")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowReceiver.DeclNameDiffersFromDir")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowReceiver.UsedNameStdlib")
	s.requireNoGlobalShadowIds(shadows, pkglocalImportNameShadowDir, pkg, "typeWithShadowReceiver.UsedNameNonStdlib")

	// method

	s.requireGlobalShadowSourcePkgs(shadows, pkglocalMethodShadowDir, []string{pkg})
	s.requireGlobalShadowSourceIds(shadows, pkglocalMethodShadowDir, pkg, []string{
		"typeWithShadowParam.Method1",
		"typeWithShadowReceiver.Method1",
	})
	s.requireGlobalShadowIds(shadows, pkglocalMethodShadowDir, pkg, "typeWithShadowReceiver.Method1", []string{
		"ExportedConst2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalMethodShadowDir, pkg, "typeWithShadowParam.Method1", []string{
		"ExportedConst2",
	})

	// type

	s.requireGlobalShadowSourcePkgs(shadows, pkglocalTypeShadowDir, []string{pkg})
	s.requireGlobalShadowSourceIds(shadows, pkglocalTypeShadowDir, pkg, []string{
		"WithConst",
		"WithType",
		"WithVar",
		s.InitId(filepath.Join(pkglocalTypeShadowDir, "init.go")),
		s.InitId(filepath.Join(pkglocalTypeShadowDir, "shadow.go")),
	})
	s.requireGlobalShadowIds(shadows, pkglocalTypeShadowDir, pkg, "WithConst", []string{
		"ExportedType1",
		"ExportedType2",
		"nonExportedType1",
		"nonExportedType2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalTypeShadowDir, pkg, "WithType", []string{
		"ExportedType1",
	})
	s.requireGlobalShadowIds(shadows, pkglocalTypeShadowDir, pkg, "WithVar", []string{
		"ExportedType1",
		"ExportedType2",
		"ExportedType3",
		"nonExportedType1",
		"nonExportedType2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalTypeShadowDir, pkg, s.InitId(filepath.Join(pkglocalTypeShadowDir, "init.go")), []string{
		"ExportedType2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalTypeShadowDir, pkg, s.InitId(filepath.Join(pkglocalTypeShadowDir, "shadow.go")), []string{
		"ExportedType1",
	})

	// var

	s.requireGlobalShadowSourcePkgs(shadows, pkglocalVarShadowDir, []string{pkg})
	s.requireGlobalShadowSourceIds(shadows, pkglocalVarShadowDir, pkg, []string{
		"WithConst",
		"WithType",
		"WithVar",
		s.InitId(filepath.Join(pkglocalVarShadowDir, "init.go")),
		s.InitId(filepath.Join(pkglocalVarShadowDir, "shadow.go")),
	})
	s.requireGlobalShadowIds(shadows, pkglocalVarShadowDir, pkg, "WithConst", []string{
		"ExportedVar1",
		"ExportedVar2",
		"nonExportedVar1",
		"nonExportedVar2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalVarShadowDir, pkg, "WithType", []string{
		"ExportedVar1",
	})
	s.requireGlobalShadowIds(shadows, pkglocalVarShadowDir, pkg, "WithVar", []string{
		"ExportedVar1",
		"ExportedVar2",
		"ExportedVar3",
		"nonExportedVar1",
		"nonExportedVar2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalVarShadowDir, pkg, s.InitId(filepath.Join(pkglocalVarShadowDir, "init.go")), []string{
		"ExportedVar2",
	})
	s.requireGlobalShadowIds(shadows, pkglocalVarShadowDir, pkg, s.InitId(filepath.Join(pkglocalVarShadowDir, "shadow.go")), []string{
		"ExportedVar1",
	})

	// main (reuse const assertions)

	pkg = "main"

	s.requireGlobalShadowSourcePkgs(shadows, mainDir, []string{pkg})
	s.requireGlobalShadowSourceIds(shadows, mainDir, pkg, []string{
		"WithConst",
		"WithType",
		"WithVar",
		s.InitId(filepath.Join(mainDir, "shadow.go")),
	})
	s.requireGlobalShadowIds(shadows, mainDir, pkg, "WithConst", []string{
		"ExportedConst1",
		"ExportedConst2",
		"nonExportedConst1",
		"nonExportedConst2",
	})
	s.requireGlobalShadowIds(shadows, mainDir, pkg, "WithType", []string{
		"ExportedConst1",
	})
	s.requireGlobalShadowIds(shadows, mainDir, pkg, "WithVar", []string{
		"ExportedConst1",
		"ExportedConst2",
		"ExportedConst3",
		"nonExportedConst1",
		"nonExportedConst2",
	})
	s.requireGlobalShadowIds(shadows, mainDir, pkg, s.InitId(filepath.Join(mainDir, "shadow.go")), []string{
		"ExportedConst1",
	})
}
