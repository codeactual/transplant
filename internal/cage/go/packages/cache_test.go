// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	std_packages "golang.org/x/tools/go/packages"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	"github.com/codeactual/transplant/internal/cage/testkit"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
)

const (
	enforcedMinLoadMode = std_packages.NeedName | std_packages.NeedFiles
)

type CacheSuite struct {
	suite.Suite

	cache  *cage_pkgs.Cache
	hits   []cage_pkgs.CacheHit
	misses []cage_pkgs.CacheMiss

	modDir string

	baselineFixtureImportPath string
	baselineFixtureDir        string
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheSuite))
}

func (s *CacheSuite) SetupTest() {
	t := s.T()

	s.cache = cage_pkgs.NewCache()

	s.cache.OnHit = func(hit cage_pkgs.CacheHit) {
		s.hits = append(s.hits, hit)
	}
	s.cache.OnMiss = func(miss cage_pkgs.CacheMiss) {
		s.misses = append(s.misses, miss)
	}

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	s.baselineFixtureImportPath = "fixture.tld/cache/baseline"
	_, s.baselineFixtureDir = testkit_file.FixturePath(t, "cache", "baseline")
	require.NoError(t, cage_filepath.Abs(&s.baselineFixtureDir))

	_, s.modDir = testkit_file.FixturePath(t)
}

func (s *CacheSuite) TestLoadImportPathWrite() {
	t := s.T()

	filesCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedFiles})
	importsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedImports})
	typesCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedTypes})

	inputFilesAndImportsMode := std_packages.NeedFiles | std_packages.NeedImports
	filesAndImportsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputFilesAndImportsMode})

	expectedKey := fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir)

	// NeedName is applied to all queries in order to index the returned map by package name.
	expectedFilesAndImportMode := inputFilesAndImportsMode | enforcedMinLoadMode
	expectedTypesMode := typesCfg.Mode | enforcedMinLoadMode

	// first query: initial write w/ NeedFiles|NeedImports

	_, errs := s.cache.LoadImportPath(filesAndImportsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	importIdx := s.cache.Data()

	require.Len(t, importIdx, 1)
	require.Exactly(t, s.baselineFixtureImportPath, importIdx[expectedKey][expectedFilesAndImportMode].PkgPath)

	// second query: NeedFiles hit, no data change

	_, errs = s.cache.LoadImportPath(filesCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	importIdx = s.cache.Data()

	require.Len(t, importIdx, 1)
	require.Exactly(t, s.baselineFixtureImportPath, importIdx[expectedKey][expectedFilesAndImportMode].PkgPath)

	// third query: NeedImports hit, no data change

	_, errs = s.cache.LoadImportPath(importsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	importIdx = s.cache.Data()

	require.Len(t, importIdx, 1)
	require.Exactly(t, s.baselineFixtureImportPath, importIdx[expectedKey][expectedFilesAndImportMode].PkgPath)

	// fourth query: NeedTypes miss, write new mode

	_, errs = s.cache.LoadImportPath(typesCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	importIdx = s.cache.Data()

	require.Len(t, importIdx, 1)
	require.Len(t, importIdx[expectedKey], 2)
	require.Exactly(t, s.baselineFixtureImportPath, importIdx[expectedKey][expectedFilesAndImportMode].PkgPath)
	require.Exactly(t, s.baselineFixtureImportPath, importIdx[expectedKey][expectedTypesMode].PkgPath)
}

func (s *CacheSuite) TestLoadImportPath() {
	t := s.T()

	inputMode := std_packages.NeedFiles
	cfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputMode})

	// NeedName is applied to all queries in order to index the returned map by package name.
	expectedMode := inputMode | enforcedMinLoadMode

	// first query: expect miss

	pkg, errs := s.cache.LoadImportPath(cfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, expectedMode, s.misses[0].Mode)

	// second query: expect hit

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(cfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.hits[0].Key,
	)
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)

	// third query: expect hit

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(cfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.hits[0].Key,
	)
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)
}

func (s *CacheSuite) TestLoadDir() {
	t := s.T()

	inputMode := std_packages.NeedFiles
	cfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputMode})

	// NeedName is applied to all queries in order to index the returned map by package name.
	expectedMode := inputMode | enforcedMinLoadMode

	// first query: expect miss

	dirPkgs, errs := s.cache.LoadDir(cfg, s.baselineFixtureDir)
	testkit.RequireNoErrors(t, errs)
	require.Len(t, dirPkgs, 1)
	pkg := dirPkgs["baseline"]

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, expectedMode, s.misses[0].Mode)

	// second query: expect hit

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	dirPkgs, errs = s.cache.LoadDir(cfg, s.baselineFixtureDir)
	testkit.RequireNoErrors(t, errs)
	require.Len(t, dirPkgs, 1)
	pkg = dirPkgs["baseline"]

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.hits[0].Key,
	)
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)

	// third query: expect hit

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	dirPkgs, errs = s.cache.LoadDir(cfg, s.baselineFixtureDir)
	testkit.RequireNoErrors(t, errs)
	require.Len(t, dirPkgs, 1)
	pkg = dirPkgs["baseline"]

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.hits[0].Key,
	)
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)
}

func (s *CacheSuite) TestCachedTestsQuerySatisfiesNonTestsQuery() {
	t := s.T()

	inputMode := std_packages.NeedFiles
	nonTestsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputMode})
	testsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputMode, Tests: true})

	// NeedName is applied to all queries in order to index the returned map by package name.
	expectedMode := inputMode | enforcedMinLoadMode

	// first query: expect miss, populate cache with both implementation and test packages

	pkg, errs := s.cache.LoadImportPath(testsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 2, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, expectedMode, s.misses[0].Mode)
	require.Exactly(
		t,
		fmt.Sprintf("%s_test Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[1].Key,
	)
	require.Exactly(t, expectedMode, s.misses[1].Mode)

	// second query: expect hit, query for only the implementation package

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(nonTestsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)

	// third query: expect hit, query for both

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(testsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 2, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)
	require.Exactly(t, expectedMode, s.hits[1].Mode)
	require.Exactly(t, s.baselineFixtureImportPath+"_test", s.hits[1].Value.PkgPath)
}

func (s *CacheSuite) TestCachedNonTestsQueryDoesNotSatisfyTestsQuery() {
	t := s.T()

	inputMode := std_packages.NeedFiles
	nonTestsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputMode})
	testsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputMode, Tests: true})

	// NeedName is applied to all queries in order to index the returned map by package name.
	expectedMode := inputMode | enforcedMinLoadMode

	// first query: expect miss, populate cache with both implementation and test packages

	pkg, errs := s.cache.LoadImportPath(nonTestsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, expectedMode, s.misses[0].Mode)

	// second query: expect hit, query for both

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(testsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)
	require.Exactly(
		t,
		fmt.Sprintf("%s_test Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, expectedMode, s.misses[0].Mode)

	// third query: expect hit, query for only the implementation package

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(nonTestsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(t, expectedMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)
}

func (s *CacheSuite) TestCachedModeQuerySatisfiesFlagSubsetQuery() {
	t := s.T()

	filesCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedFiles})
	importsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedImports})
	typesCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedTypes})
	inputFilesAndImportsMode := std_packages.NeedFiles | std_packages.NeedImports
	filesAndImportsCfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: inputFilesAndImportsMode})

	// NeedName is applied to all queries in order to index the returned map by package name.
	expectedFilesAndImportMode := inputFilesAndImportsMode | enforcedMinLoadMode
	expectedTypesMode := typesCfg.Mode | enforcedMinLoadMode

	// first query: expect miss, populate cache with superset

	pkg, errs := s.cache.LoadImportPath(filesAndImportsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, expectedFilesAndImportMode, s.misses[0].Mode)

	// second query: expect hit, query for subset

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(importsCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(t, expectedFilesAndImportMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)

	// third query: expect hit, query for different subset

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(filesCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(t, expectedFilesAndImportMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)

	// fourth query: expect miss, query for new mode

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(typesCfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, expectedTypesMode, s.misses[0].Mode)
}

func (s *CacheSuite) TestLoadImportPathHitAfterLoadDir() {
	t := s.T()

	cfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedName})

	// first query: expect miss via dir

	dirPkgs, errs := s.cache.LoadDir(cfg, s.baselineFixtureDir)
	testkit.RequireNoErrors(t, errs)
	require.Len(t, dirPkgs, 1)
	pkg := dirPkgs["baseline"]

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, enforcedMinLoadMode, s.misses[0].Mode)

	// second query: expect hit via import path

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	pkg, errs = s.cache.LoadImportPath(cfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)
	require.NotNil(t, pkg)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.hits[0].Key,
	)
	require.Exactly(t, enforcedMinLoadMode, s.hits[0].Mode)
	require.Exactly(t, s.baselineFixtureImportPath, s.hits[0].Value.PkgPath)
}

func (s *CacheSuite) TestLoadDirHitAfterLoadImportPath() {
	t := s.T()

	cfg := cage_pkgs.NewConfig(&std_packages.Config{Dir: s.modDir, Mode: std_packages.NeedName})

	// first query: expect miss via import path

	pkg, errs := s.cache.LoadImportPath(cfg, s.baselineFixtureImportPath)
	testkit.RequireNoErrors(t, errs)
	require.NotNil(t, pkg)

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 0, len(s.hits))
	require.Exactly(t, 1, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.misses[0].Key,
	)
	require.Exactly(t, enforcedMinLoadMode, s.misses[0].Mode)

	// second query: expect hit via dir

	s.hits, s.misses = []cage_pkgs.CacheHit{}, []cage_pkgs.CacheMiss{}

	dirPkgs, errs := s.cache.LoadDir(cfg, s.baselineFixtureDir)
	testkit.RequireNoErrors(t, errs)
	require.Len(t, dirPkgs, 1)
	pkg = dirPkgs["baseline"]

	require.Exactly(t, s.baselineFixtureImportPath, pkg.PkgPath)
	require.Exactly(t, 1, len(s.hits))
	require.Exactly(t, 0, len(s.misses))
	require.Exactly(
		t,
		fmt.Sprintf("%s Dir=%s Env= BuildFlags=", s.baselineFixtureImportPath, s.modDir),
		s.hits[0].Key,
	)
	require.Exactly(t, enforcedMinLoadMode, s.hits[0].Mode)
}
