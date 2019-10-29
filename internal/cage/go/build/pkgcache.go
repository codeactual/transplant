// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package build

import (
	std_build "go/build"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

// packageCacheValue is a cache value for import path queries.
//
// In addition to the primary value field/content, it stores fields such as mode in order
// to support post-read logic that verifies the value can satisify a query.
type packageCacheValue struct {
	value *std_build.Package
	mode  std_build.ImportMode
}

// Satisfies returns true if the cache value can satisfy a query based on the latter's configuration.
//
// If the cache value's query was configured to return the same or greater amount of package information,
// then it can satisfy queries with the same or lesser scope.
func (v packageCacheValue) Satisfies(mode std_build.ImportMode) bool {
	// Unlike golang.org/x/tools/go/package where each LoadMode higher iota value indicates the
	// mode provides mores detail in addition to the lower-mode detail, go/build ImportMode
	// is a bitmask.
	//
	// Currently there is only one case covered: where we want to reuse mode=0 results
	// when FindOnly is requested.
	if mode == std_build.FindOnly && v.mode == 0 {
		return true
	}

	return mode == v.mode
}

type PackageCache struct {
	cache       map[string]packageCacheValue
	context     std_build.Context
	srcDirInKey bool
}

func NewPackageCache() *PackageCache {
	return &PackageCache{
		cache:   make(map[string]packageCacheValue),
		context: std_build.Default,
	}
}

func (c *PackageCache) SrcDirInKey(enable bool) {
	c.srcDirInKey = enable
}

func (c *PackageCache) SetContext(context std_build.Context) {
	c.context = context
}

func (c *PackageCache) Import(importPath, srcDir string, mode std_build.ImportMode) (*std_build.Package, error) {
	var key string
	if c.srcDirInKey {
		key = importPath + srcDir + strconv.Itoa(int(mode))
	} else {
		key = importPath + strconv.Itoa(int(mode))
	}

	if read, ok := c.cache[key]; ok {
		if read.Satisfies(mode) {
			return read.value, nil
		}
	}

	pkg, err := c.context.Import(importPath, srcDir, mode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find package [%s] from dir [%s]", importPath, srcDir)
	}

	for n := range pkg.GoFiles {
		pkg.GoFiles[n] = filepath.Join(pkg.Dir, pkg.GoFiles[n])
	}
	for n := range pkg.TestGoFiles {
		pkg.TestGoFiles[n] = filepath.Join(pkg.Dir, pkg.TestGoFiles[n])
	}
	for n := range pkg.XTestGoFiles {
		pkg.XTestGoFiles[n] = filepath.Join(pkg.Dir, pkg.XTestGoFiles[n])
	}

	c.cache[key] = packageCacheValue{
		mode:  mode,
		value: pkg,
	}

	return c.cache[key].value, nil
}

func (c *PackageCache) GetDeps(rootImportPath, srcDir string) (deps []*std_build.Package, errs []error) {
	seen := make(map[string]struct{})

	var walk func(string)
	walk = func(importPath string) {
		pkg, err := c.Import(importPath, srcDir, 0)
		if err != nil {
			errs = append(errs, errors.WithStack(err))
			return
		}

		if rootImportPath != importPath {
			deps = append(deps, pkg)
		}

		for _, transPath := range pkg.Imports {
			if _, ok := seen[transPath]; ok {
				continue
			}

			seen[transPath] = struct{}{}

			walk(transPath)
		}
	}

	walk(rootImportPath)

	if len(errs) > 0 {
		return []*std_build.Package{}, errs
	}

	return deps, []error{}
}
