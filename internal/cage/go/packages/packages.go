// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	std_packages "golang.org/x/tools/go/packages"

	cage_build "github.com/codeactual/transplant/internal/cage/go/build"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

const (
	// LoadSyntax approximates the deprecated tools/go/packages.LoadSyntax mode.
	//
	// - NeedDeps is required unless https://github.com/golang/tools/pull/139/files is accepted.
	LoadSyntax = std_packages.NeedName |
		std_packages.NeedFiles |
		std_packages.NeedImports |
		std_packages.NeedTypes |
		std_packages.NeedTypesInfo |
		std_packages.NeedSyntax

	// StdNeedMin is the current lowest tools/go/packages.Need* constant value.
	StdNeedMin = std_packages.NeedName

	// StdNeedMax is the current highest tools/go/packages.Need* constant value.
	StdNeedMax = std_packages.NeedTypesSizes
)

type Package struct {
	*std_packages.Package

	// Dir is derived from x/tools/go/packages.Package.GoFiles[0]
	Dir string

	// FileToName provides a mapping from ast.File pointers, returned by Load*, to the source filenames.
	FileToName map[*ast.File]string

	ImportPaths *cage_strings.Set

	// ImportPathToDir maps import paths to absolute paths of source directories.
	ImportPathToDir map[string]string

	// Goroot is true if the package is located under GOROOT.
	Goroot bool

	// Vendor is true if the package is located under a vendor directory.
	Vendor bool
}

func (p *Package) IdentTypesObj(ident *ast.Ident) types.Object {
	if p.TypesInfo.Uses[ident] != nil {
		return p.TypesInfo.Uses[ident]
	}
	return p.TypesInfo.Defs[ident]
}

func StdSdump(p *std_packages.Package, indentTabs int) string {
	var b strings.Builder
	seen := cage_strings.NewSet()

	var indent string
	for t := 0; t < indentTabs; t++ {
		indent += "\t"
	}

	var sdump func(importer string, _p *std_packages.Package)
	sdump = func(importer string, _p *std_packages.Package) {
		// Use ID instead of PkgPath because _p may be from Package.Imports without NeedDeps to
		// populate PkgPath/Name/etc.
		b.WriteString(fmt.Sprintf("%sID [%s]\n", indent, _p.ID))

		if importer != "" {
			b.WriteString(fmt.Sprintf("%s\t(%s Package.Imports)\n", indent, importer))
		}
		if l := len(_p.Errors); l > 0 {
			b.WriteString(fmt.Sprintf("%s\tlen(Errors) [%d]\n", indent, l))
		}
		if l := len(_p.GoFiles); l > 0 {
			b.WriteString(fmt.Sprintf("%s\tlen(GoFiles) [%d]\n", indent, l))
		}
		if l := len(_p.Imports); l > 0 {
			b.WriteString(fmt.Sprintf("%s\tlen(Imports) [%d]\n", indent, l))
		}
		if _p.Types != nil {
			b.WriteString(fmt.Sprintf("%s\tTypes: [%p]\n", indent, _p.Types))
		}
		if _p.TypesInfo != nil {
			b.WriteString(fmt.Sprintf("%s\tTypesInfo: [%p]\n", indent, _p.TypesInfo))
		}
		if l := len(_p.Syntax); l > 0 {
			b.WriteString(fmt.Sprintf("%s\tlen(Syntax) [%d]\n", indent, l))
			for _, v := range _p.Syntax {
				b.WriteString(fmt.Sprintf("%s\t\tAST [%s]\n", indent, _p.Fset.File(v.Package).Name()))
			}
		}
		for _, v := range _p.Imports {
			if seen.Contains(v.PkgPath) {
				continue
			}
			seen.Add(v.PkgPath)
			sdump(_p.PkgPath, v)
		}
	}

	sdump("", p)

	return b.String()
}

// PkgsByString indexes packages by a string key such as import path.
type PkgsByString map[string]*Package

// PkgsByName indexes packages by names as they appear in the package clause.
type PkgsByName PkgsByString

// SortedPkgNames returns the map's keys in sorted order.
func (p PkgsByName) SortedPkgNames() []string {
	s := cage_strings.NewSet()
	for dir := range p {
		s.Add(dir)
	}
	return s.SortedSlice()
}

// PkgsByImportPath indexes packages by import paths.
type PkgsByImportPath PkgsByString

// SortedPkgPaths returns the map's keys in sorted order.
func (p PkgsByImportPath) SortedPkgPaths() []string {
	s := cage_strings.NewSet()
	for dir := range p {
		s.Add(dir)
	}
	return s.SortedSlice()
}

// DirPkgs indexes the packages found in directories by their absolute paths.
type DirPkgs map[string]PkgsByName

type Config struct {
	*std_packages.Config
}

func NewConfig(cfg *std_packages.Config) *Config {
	return &Config{
		Config: cfg,
	}
}

func (c *Config) Copy() *Config {
	cpy := &Config{
		Config: c.Config,
	}

	cfg := *c.Config
	cpy.Config = &cfg

	return cpy
}

// LoadWithConfig wraps x/tools/go/packages.Load to address some common-case needs such as
// returning all encountered errors and all available package information.
//
// The packages returned are indexed by the import path of the package.
//
// The errors returned will include all x/tools/go/packages.Package.[]Errors, if any,
// each wrapped with a message indicating its origin package. In that event, the Package map
// is also returned from which the same per-Package errors are available.
func LoadWithConfig(cfg *Config, patterns ...string) (pkgMap PkgsByImportPath, errs []error) {
	pkgMap = make(PkgsByImportPath)
	needSyntax := NeedSatisfied(std_packages.NeedSyntax, cfg.Mode)

	var allPkgsFileToName sync.Map

	if needSyntax {
		// Adjust the same default function from x/tools/go/package to build the ast.File-to-filename map.
		cfg.ParseFile = func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			var isrc interface{}
			if src != nil {
				isrc = src
			}

			file, err := parser.ParseFile(fset, filename, isrc, parser.AllErrors|parser.ParseComments)

			if err != nil {
				return nil, err
			}

			// - Skip GOCACHE files by checking for ".go" extension.
			// - Skip standard library files.
			if cage_filepath.IsGoFile(filename) && !strings.HasPrefix(filename, cage_build.Goroot()) {
				allPkgsFileToName.Store(file, filename)
			}

			return file, nil
		}
	}

	pkgList, err := std_packages.Load(cfg.Config, patterns...)
	if err != nil {
		return nil, []error{errors.Wrapf(err, "failed to load packages with pattern %q", patterns)}
	}

	for _, pkg := range pkgList {
		// Skip the two extra packages included when cfg.Tests is true:
		// "the package as compiled for the test" and "the test binary".
		if strings.HasSuffix(pkg.PkgPath, ".test") {
			continue
		}

		importPathToDir := make(map[string]string)
		fileToName := make(map[*ast.File]string)
		mapVal := &Package{
			FileToName: fileToName,
			Package:    pkg,
		}

		if needSyntax {
			for _, file := range pkg.Syntax {
				filename, ok := allPkgsFileToName.Load(file)
				if !ok { // e.g. standard library
					continue
				}
				fileToName[file] = filename.(string) //nolint:errcheck
				importPathToDir[pkg.PkgPath] = filepath.Dir(filename.(string))
				mapVal.Goroot = strings.HasPrefix(filename.(string), cage_build.Goroot())
				mapVal.Dir = filepath.Dir(filename.(string))
			}
		}

		if NeedSatisfied(std_packages.NeedFiles, cfg.Mode) {
			if len(pkg.GoFiles) == 0 {
				if len(pkg.Errors) > 0 {
					for _, pkgErr := range pkg.Errors {
						errs = append(errs, errors.Wrapf(pkgErr, "failed to load Package.GoFiles [%s]", pkg.PkgPath))
					}
				} else {
					errs = append(errs, errors.Errorf("failed to load Package.GoFiles [%s]", pkg.PkgPath))
				}
				continue
			}
			importPathToDir[pkg.PkgPath] = filepath.Dir(pkg.GoFiles[0])
			mapVal.Goroot = strings.HasPrefix(pkg.GoFiles[0], cage_build.Goroot())
			mapVal.Dir = filepath.Dir(pkg.GoFiles[0])
		}

		mapVal.ImportPathToDir = importPathToDir
		pkgMap[pkg.PkgPath] = mapVal

		for _, pkgErr := range pkg.Errors {
			errs = append(errs, errors.Wrapf(pkgErr, "failed to load package [%s]", pkg.PkgPath))
		}
	}

	return pkgMap, errs
}

// TrimVendorPathPrefix trims "path/to/vendor/repo/user/proj" to "repo/user/proj".
func TrimVendorPathPrefix(importPath string) string {
	vendorIdx := strings.LastIndex(importPath, "vendor/")
	if vendorIdx == -1 || vendorIdx+7 >= len(importPath) {
		return importPath
	}
	return importPath[vendorIdx+7:]
}

// NeedSatisfied returns true if the "needer" mode's enabled bits are also enabled in
// the "satisifier" mode, where the satisfier can provide an exact or superset of what's needed.
//
// If the needer has no enabled bits, true is returned.
func NeedSatisfied(needer, satisfier std_packages.LoadMode) bool {
	for m := StdNeedMin; m <= StdNeedMax; m <<= 1 { // keep doubling to reach the next higher Need*
		if needer&m != 0 { // mode is enabled in needer
			if satisfier&m == 0 { // mode is is NOT enabled by the satisifier
				return false
			}
		}
	}
	return true
}

func LoadModeString(m std_packages.LoadMode) string {
	if m == 0 {
		// zero behavior: https://github.com/golang/tools/blob/aac0b97cf93b20107ae688cd90e54cdc211fd043/go/packages/packages.go#L434
		return "<default: Name|Files|CompiledGoFiles>"
	}

	var needs []string
	if m&std_packages.NeedName != 0 {
		needs = append(needs, "Name")
	}
	if m&std_packages.NeedFiles != 0 {
		needs = append(needs, "Files")
	}
	if m&std_packages.NeedCompiledGoFiles != 0 {
		needs = append(needs, "CompiledGoFiles")
	}
	if m&std_packages.NeedImports != 0 {
		needs = append(needs, "Imports")
	}
	if m&std_packages.NeedDeps != 0 {
		needs = append(needs, "Deps")
	}
	if m&std_packages.NeedExportsFile != 0 {
		needs = append(needs, "ExportsFile")
	}
	if m&std_packages.NeedTypes != 0 {
		needs = append(needs, "Types")
	}
	if m&std_packages.NeedSyntax != 0 {
		needs = append(needs, "Syntax")
	}
	if m&std_packages.NeedTypesInfo != 0 {
		needs = append(needs, "TypesInfo")
	}
	if m&std_packages.NeedTypesSizes != 0 {
		needs = append(needs, "TypesSizes")
	}

	return strings.Join(needs, "|")
}
