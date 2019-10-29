// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"fmt"
	"go/build"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	std_packages "golang.org/x/tools/go/packages"

	cage_build "github.com/codeactual/transplant/internal/cage/go/build"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

type CacheValue struct {
	*Package

	// WriteId is the value of Cache.writeId at creation time.
	WriteId int

	// Hits is a counter.
	Hits int
}

// PkgIdx provides the second-level and final index based on LoadMode values.
// Any LoadMode key which can satisfy the query's bit flags, either exactly
// or as a superset, contains a cached Package with sufficient data for the query.
type PkgIdx map[std_packages.LoadMode]CacheValue

// ImportIdx provides the first-level index based on import paths.
type ImportIdx map[string]PkgIdx

// CacheHit describes the query and cached data.
type CacheHit struct {
	// Key is the NewCacheKey value used in the query.
	Key string

	// Mode is the value used in the past query which populated the cached Package.
	//
	// It is a PkgIdx map key.
	Mode std_packages.LoadMode

	// Package is the cached query result.
	Value CacheValue
}

// OnHitFunc receives information about Cache hits.
type OnHitFunc func(CacheHit)

// CacheMiss describes the query.
type CacheMiss struct {
	// Key is the NewCacheKey value used in the query.
	Key string

	// Mode is the value used in the current query.
	Mode std_packages.LoadMode
}

// OnMissFunc receives information about Cache misses.
type OnMissFunc func(CacheMiss)

// Cache serves x/tools/go/packages.Load queries via methods like LoadImportPath
// which write to a cache shared by all query methods.
type Cache struct {
	// Enabled is true if Cache.data reads and writes should be performed. If false, full queries always occur.
	Enabled bool

	// OnHit is called for each hit processed by LoadImportPath/LoadDir.
	OnHit OnHitFunc

	// OnMiss is called for each hit processed by LoadImportPath/LoadDir.
	OnMiss OnMissFunc

	// data resolves NewCacheKey and x/tools/go/packages.LoadMode values to *Package entries.
	data ImportIdx

	// dirToImportPath resolves directory absolute paths to import paths.
	dirToImportPath map[string]string

	// writeId is incremented once per write to data.
	writeId int

	// buildCache supports falling back to go/build in LoadImportPath.
	buildCache *cage_build.PackageCache
}

func NewCache() *Cache {
	buildCache := cage_build.NewPackageCache()

	c := &Cache{
		Enabled:         true,
		OnHit:           func(_ CacheHit) {},
		OnMiss:          func(_ CacheMiss) {},
		buildCache:      buildCache,
		data:            make(ImportIdx),
		dirToImportPath: make(map[string]string),
	}

	return c
}

// LoadImportPathWithBuild uses go/build to generate the Package instead of x/tools/go/packages.
//
// Its main benefit over x/tools/go/packages is non-recursive queries, i.e. abiility to find
// the package's imported paths without also finding the latter's imported paths and so on.
// Elements of x/tools/go/packages.Package.Imports only contain the PkgPath field.
//
// If the import path selects a standard library, only these fields populated:
// Dir (if go/build provides it), Goroot, ImportPaths (if mode provides them), PkgPath, and Name.
func (c *Cache) LoadImportPathWithBuild(importPath, srcDir string, mode build.ImportMode) (PkgsByName, error) {
	pkgs := make(PkgsByName)

	if stdlibImportPaths.Contains(importPath) {
		// Assumes all stdlib packages use this convention.
		// Can verify with: go list -f "{{.ImportPath}} {{.Name}}" ./... | egrep -v "^(internal|cmd)"
		stdlibName := path.Base(importPath)

		pkgs[stdlibName] = &Package{
			Dir:    StdlibDir(importPath),
			Goroot: true,
			Package: &std_packages.Package{
				Name:    stdlibName,
				PkgPath: importPath,
			},
		}

		return pkgs, nil
	}

	buildPkg, err := c.buildCache.Import(importPath, srcDir, mode)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// implementation

	pkgs[buildPkg.Name] = &Package{
		Dir:         buildPkg.Dir,
		Goroot:      buildPkg.Goroot,
		ImportPaths: cage_strings.NewSet(),
		Package: &std_packages.Package{
			Name:    buildPkg.Name,
			PkgPath: importPath,
		},
		Vendor: strings.Contains(buildPkg.Dir, string(filepath.Separator)+"vendor"+string(filepath.Separator)),
	}

	pkgs[buildPkg.Name].Imports = make(map[string]*std_packages.Package)
	for _, i := range buildPkg.Imports {
		pkgs[buildPkg.Name].ImportPaths.Add(i)
		pkgs[buildPkg.Name].Imports[i] = &std_packages.Package{
			PkgPath: i,
		}
	}

	// test

	testGoFilesLen := len(buildPkg.TestGoFiles)
	xTestGoFilesLen := len(buildPkg.XTestGoFiles)

	if testGoFilesLen > 0 || xTestGoFilesLen > 0 {
		testPkgName := buildPkg.Name + "_test"
		pkgs[testPkgName] = &Package{
			Dir:         buildPkg.Dir,
			Goroot:      buildPkg.Goroot,
			ImportPaths: cage_strings.NewSet(),
			Package: &std_packages.Package{
				Name:    testPkgName,
				PkgPath: importPath + "_test",
			},
			Vendor: pkgs[buildPkg.Name].Vendor,
		}
		pkgs[testPkgName].Imports = make(map[string]*std_packages.Package)

		if testGoFilesLen > 0 {
			for _, i := range buildPkg.TestImports {
				pkgs[testPkgName].ImportPaths.Add(i)
				pkgs[testPkgName].Imports[i] = &std_packages.Package{
					PkgPath: i,
				}
			}
			for _, f := range buildPkg.TestGoFiles {
				pkgs[testPkgName].GoFiles = append(pkgs[testPkgName].GoFiles, f)
			}
		} else {
			for _, i := range buildPkg.XTestImports {
				pkgs[testPkgName].ImportPaths.Add(i)
				pkgs[testPkgName].Imports[i] = &std_packages.Package{
					PkgPath: i,
				}
			}
			for _, f := range buildPkg.XTestGoFiles {
				pkgs[testPkgName].GoFiles = append(pkgs[testPkgName].GoFiles, f)
			}
		}
	}

	return pkgs, nil
}

// LoadImportPaths is the more generic method behind LoadImportPath and LoadDir queries
// which reads from the cache, performs the actual x/tools/go/packages.Load query,
// and writes to the cache.
//
// It adds NeedName|NeedFiles to all Config.Mode values in order to index the returned map.
func (c *Cache) LoadImportPaths(cfg *Config, _importPaths ...string) (loaded PkgsByImportPath, errs []error) {
	loaded = make(PkgsByImportPath)

	// patterns holds import paths and file directories which will be passed to x/tools/go/packages.Load.
	//
	// If an input import path cannot be served from the cache, it is added to this set.
	patterns := cage_strings.NewSet()

	for _, p := range _importPaths {
		patterns.Add(TrimVendorPathPrefix(p))
	}

	// Enforce NeedName and NeedFiles to support indexing of LoadDir results.

	cfg.Mode |= std_packages.NeedFiles
	cfg.Mode |= std_packages.NeedName

	// Avoid queries for standard library

	for _, importPath := range patterns.Slice() {
		if stdlibImportPaths.Contains(importPath) {
			loaded[importPath] = &Package{
				Dir:    StdlibDir(importPath),
				Goroot: true,
				Package: &std_packages.Package{
					Name:    path.Base(importPath), // Assumes all stdlib packages use this convention.
					PkgPath: importPath,
				},
			}

			patterns.Remove(importPath)
		}
	}

	// Attempt to read implementation/test packages from the cache.

	missingImplPaths, missingTestPaths := cage_strings.NewSet(), cage_strings.NewSet()

	if c.Enabled {
		for _, importPath := range patterns.SortedSlice() {
			implKey := NewCacheKey(cfg, importPath)
			var implSatisfied, testSatisfied bool

			if pkgsByMode, ok := c.data[implKey]; ok {
				for mode, val := range pkgsByMode {
					if NeedSatisfied(cfg.Mode, mode) {
						loaded[TrimVendorPathPrefix(val.Package.PkgPath)] = val.Package
						c.OnHit(CacheHit{Key: implKey, Mode: mode, Value: val})
						implSatisfied = true
						val.Hits++
						c.data[implKey][mode] = val
						break
					}
				}
			}

			if !implSatisfied {
				c.OnMiss(CacheMiss{Key: implKey, Mode: cfg.Mode})
				missingImplPaths.Add(importPath)
			}

			if cfg.Tests {
				testKey := NewCacheKey(cfg, importPath+"_test")

				if pkgsByMode, ok := c.data[testKey]; ok {
					for mode, val := range pkgsByMode {
						if NeedSatisfied(cfg.Mode, mode) {
							loaded[TrimVendorPathPrefix(val.Package.PkgPath)] = val.Package
							c.OnHit(CacheHit{Key: testKey, Mode: mode, Value: val})
							testSatisfied = true
							val.Hits++
							c.data[implKey][mode] = val
							break
						}
					}
				}

				if !testSatisfied {
					c.OnMiss(CacheMiss{Key: testKey, Mode: cfg.Mode})
					missingTestPaths.Add(importPath + "_test")
				}
			} else {
				testSatisfied = true
			}

			if implSatisfied && testSatisfied {
				patterns.Remove(importPath)
			}
		}
	}

	if patterns.Len() == 0 {
		return loaded, []error{}
	}

	// Cache does not contain a satisfying entry for the implementation and/or test package.
	//
	// In either case we need to query for the implementation's import path. Even if Config.Tests=true,
	// x/tools/go/packages.Load will only include the test package in queries for the implementation,
	// i.e. "/path/to/<package name>_test" + Config.Tests=true will not yield that test package.

	pkgList, loadErrs := LoadWithConfig(cfg, patterns.SortedSlice()...)

	if len(loadErrs) > 0 {
		for _, loadErr := range loadErrs {
			errs = append(errs, errors.WithStack(loadErr))
		}
		return nil, errs
	}

	// Write results to the cache that were unsatisfied earlier.

	for _, importPath := range pkgList.SortedPkgPaths() {
		pkg := pkgList[importPath]

		importPath = TrimVendorPathPrefix(importPath)

		isTest := strings.HasSuffix(importPath, "_test")
		key := NewCacheKey(cfg, importPath)

		var missingAcquired bool

		if isTest {
			loaded[importPath] = pkg
			if missingTestPaths.Contains(importPath) {
				missingAcquired = true
			}
		} else {
			loaded[importPath] = pkg
			if missingImplPaths.Contains(importPath) {
				missingAcquired = true
			}
		}

		if missingAcquired && c.Enabled {
			if c.data[key] == nil {
				c.data[key] = make(PkgIdx)
			}
			c.data[key][cfg.Mode] = CacheValue{Package: pkg, WriteId: c.writeId}
			c.writeId++
		}
	}

	return loaded, []error{}
}

// LoadImportPath performs a single-package query.
//
// It adds NeedName|NeedFiles to all Config.Mode values in order to index the returned map.
//
// It does not support Config.Tests because test x/tools/go/packages.Load does not support direct
// queries for "_test" packages, even with Config.Tests enabled. (Instead, the query must specify
// the implementation's import path or the file directory containing both packages.)
func (c *Cache) LoadImportPath(cfg *Config, importPath string) (*Package, []error) {
	pkgsByImportPath, errs := c.LoadImportPaths(cfg, importPath)
	if len(errs) > 0 {
		for e := range errs {
			errs[e] = errors.WithStack(errs[e])
		}
		return nil, errs
	}

	match := pkgsByImportPath[TrimVendorPathPrefix(importPath)]

	if match == nil {
		return nil, []error{errors.Errorf("package [%s] not found", importPath)}
	}

	return match, []error{}
}

// LoadDirs performs a query for the packages in one or more directories.
//
// It adds NeedName|NeedFiles to all Config.Mode values in order to index the returned map.
func (c *Cache) LoadDirs(cfg *Config, dirs ...string) (_ DirPkgs, errs []error) {
	importPaths, unresolvedDirs := cage_strings.NewSet(), cage_strings.NewSet()

	// Determine import paths of packages in the directory.

	if c.Enabled {
		for _, d := range dirs {
			if p := c.dirToImportPath[d]; p == "" {
				unresolvedDirs.Add(d)
			} else {
				importPaths.Add(p)
			}
		}
	}

	if unresolvedDirs.Len() > 0 {
		dirCfg := cfg.Copy() // only need to resolve the dir to the implementation's import path
		dirCfg.Tests = false // x/tools/go/packages.Load only needs implemetnation path for both cases

		// NeedName for x/tools/go/packages.Package.{Name,PkgPath}, NeedFiles for cage/go/packages.Package.Dir
		dirCfg.Mode = std_packages.NeedName | std_packages.NeedFiles

		pkgList, errs := LoadWithConfig(dirCfg, unresolvedDirs.SortedSlice()...)
		if len(errs) > 0 {
			for n := range errs {
				errs[n] = errors.WithStack(errs[n])
			}
			return nil, errs
		}

		for _, pkg := range pkgList {
			if pkg.Dir != "" {
				c.dirToImportPath[pkg.Dir] = pkg.PkgPath
			}
			importPaths.Add(pkg.PkgPath)
		}
	}

	// Delegate full query to more generic load method.

	cfg.Mode |= std_packages.NeedFiles // NeedFiles for cage/go/packages.Package.Dir

	pkgsByImportPath, errs := c.LoadImportPaths(cfg, importPaths.SortedSlice()...)
	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return nil, errs
	}

	dirPkgs := make(DirPkgs)
	for _, pkg := range pkgsByImportPath {
		if dirPkgs[pkg.Dir] == nil {
			dirPkgs[pkg.Dir] = make(PkgsByName)
		}
		dirPkgs[pkg.Dir][pkg.Name] = pkg
	}

	return dirPkgs, []error{}
}

// LoadDir performs a single-directory query.
//
// It adds NeedName|NeedFiles to all Config.Mode values in order to index the returned map.
//
// It does not support Config.Tests because test x/tools/go/packages.Load does not support direct
// queries for "_test" packages, even with Config.Tests enabled. (Instead, the query must specify
// the implementation's import path or the file directory containing both packages.)
func (c *Cache) LoadDir(cfg *Config, dir string) (PkgsByName, []error) {
	dirPkgs, errs := c.LoadDirs(cfg, dir)
	if len(errs) > 0 {
		for e := range errs {
			errs[e] = errors.WithStack(errs[e])
		}
		return nil, errs
	}

	match := dirPkgs[dir]

	if match == nil {
		return nil, []error{errors.Errorf("package(s) in directory [%s] not found", dir)}
	}

	return match, []error{}
}

func (c *Cache) Data() ImportIdx {
	return c.data
}

func (c *Cache) String() string {
	importPaths := cage_strings.NewSet()
	grouped := make(map[string][]string) // import path -> data strings

	var allBuilder strings.Builder

	for key, pkgsByMode := range c.data {
		importPath := key[:strings.Index(key, " ")]

		importPaths.Add(importPath)

		var entryBuilder strings.Builder
		for mode, val := range pkgsByMode {
			entryBuilder.WriteString(fmt.Sprintf("\tKey [%s] Write [%d] Hits [%d] Mode [%s]:\n", key, val.WriteId, val.Hits, LoadModeString(mode)))
			entryBuilder.WriteString(StdSdump(val.Package.Package, 2))
		}
		grouped[importPath] = append(grouped[importPath], entryBuilder.String())
	}

	for _, importPath := range importPaths.SortedSlice() {
		allBuilder.WriteString(importPath)
		allBuilder.WriteString("\n")
		for _, s := range grouped[importPath] {
			allBuilder.WriteString(s)
		}
	}

	return allBuilder.String()
}

func NewCacheKey(cfg *Config, pattern string) (key string) {
	key = pattern
	key += " Dir=" + cfg.Dir
	key += " Env=" + strings.Join(cfg.Env, ",")
	key += " BuildFlags=" + strings.Join(cfg.BuildFlags, ",")
	return key
}
