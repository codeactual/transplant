// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/pkg/errors"

	cage_ast "github.com/codeactual/transplant/internal/cage/go/ast"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

type TraitType string

const (
	// TraitDotImport rationale:
	//
	// It's unknown whether their usage is common enough to justify updating the
	// global-usage-related code to ensure that if there a package which is imported
	// only for initialization, then ensure its globals and also its transitive dependencies
	// are all inspected.
	TraitDotImport TraitType = "dot import name"

	// TraitDuplicateImport rationale:
	//
	// Allowing duplicates would complicate import pruning which follows global declaration pruning
	// because the usage of all import names would need to be tracked, and only the ast.ImportSpec
	// referring to the unused import name(s) would be eligible for pruning. Also, this trait
	// is reasonably considered lint and a potential source of bugs.
	TraitDuplicateImport TraitType = "multiple imports with the same path"
)

type UnsupportedTrait struct {
	// Type identifies the trait, e.g. blank identifier.
	Type TraitType

	// FileOrDir is the site's filename if possible, otherwise its directory.
	FileOrDir string

	// PkgPath is the site's import path
	PkgPath string

	// Msg provides additional detail specific to the trait.
	Msg string
}

const (
	GlobalIdSeparator = "."
)

type GlobalNode struct {
	Id          GlobalId
	InspectInfo NodeInspectInfo
}

type Node struct {
	Ast         ast.Node
	InspectInfo NodeInspectInfo
}

type InspectFunc func(n ast.Node, i NodeInspectInfo) bool

// PathImport describes a package imported by a file.
//
// It is indexed by import path.
type PathImport map[string]Import

// FileImports describes the packages imported by a file.
//
// It is indexed by source file absolute path.
//
// If x/tools/go/packages.NeedSyntax, the filenames will instead be directory names because the former
// are not available. Also, Import.UsedName may be inaccurate because the ast.ImportSpec values
// were not available.
type FileImports map[string]PathImport

// PkgImports describes the packages imported by another package.
//
// It is indexed by package name (as declared in the source file).
type PkgImports map[string]FileImports

// DirImports describes the packages imported by all files in a directory.
//
// It is indexed by directory absolute path.
type DirImports map[string]PkgImports

// pkgFile holds the absolute paths of source files of a package.
//
// It is indexed by package name (as declared in the source file).
type pkgFile map[string]*cage_strings.Set

// dirFiles holds the absolute paths of all files in a directory.
//
// It is indexed by directory absolute path.
type dirFiles map[string]pkgFile

// pkgGlobalIdNames holds the names of a package's identifiers.
//
// It is indexed by package name (as declared in the source file).
type pkgGlobalIdNames map[string]*cage_strings.Set

// dirGlobalIdNames holds the names of a directory's identifiers.
//
// It is indexed by package name (as declared in the source file).
type dirGlobalIdNames map[string]pkgGlobalIdNames

// IdToNode holds a Node associated with an identifier.
//
// It is indexed by the global's name (as declared in the source file).
type IdToNode map[GlobalIdName]Node

func (m IdToNode) SortedIds() []string {
	var ids []string
	for k := range m {
		ids = append(ids, k)
	}
	cage_strings.SortStable(ids)
	return ids
}

func (m IdToNode) Contains(id string) bool {
	_, ok := m[id]
	return ok
}

// ImportPathGlobalIdNodes holds a Node for each of a package's identifiers.
//
// It is indexed by import paths.
type ImportPathGlobalIdNodes map[string]IdToNode

// PkgGlobalIdNodes holds a Node for each of a package's identifiers.
//
// It is indexed by package name (as declared in the source file).
type PkgGlobalIdNodes map[string]IdToNode

// SortedPkgNames returns the map's keys in sorted order.
func (n PkgGlobalIdNodes) SortedPkgNames() []string {
	s := cage_strings.NewSet()
	for dir := range n {
		s.Add(dir)
	}
	return s.SortedSlice()
}

// DirGlobalIdNodes holds a Node for each of a package's identifiers.
//
// It is indexed by directory absolute path.
type DirGlobalIdNodes map[string]PkgGlobalIdNodes

// SortedDirs returns the map's keys in sorted order.
func (n DirGlobalIdNodes) SortedDirs() []string {
	s := cage_strings.NewSet()
	for dir := range n {
		s.Add(dir)
	}
	return s.SortedSlice()
}

// FileNodes holds Node values indexed by file absolute paths.
type FileNodes map[string]Node

// FilePkgs holds Package values indexed by file absolute paths.
type FilePkgs map[string]*Package

// Shadows contains the names of global identifiers which are shadowed in a global function/method.
//
// It is indexed by names of of global functions/methods.
type Shadows map[string]*cage_strings.Set

// FileGlobalIdShadows holds Shadow values in each file which contains at least one
// global function/method containing at least one shadow.
//
// It is indexed by file absolute paths.
type FileGlobalIdShadows map[string]Shadows

// PkgGlobalIdShadows holds Shadow values in each package which contains at least one
// global function/method containing at least one shadow.
//
// It is indexed by package name (as declared in the source file).
type PkgGlobalIdShadows map[string]FileGlobalIdShadows

// DirGlobalIdShadows holds Shadow values in each directory which contains at least one
// global function/method containing at least one shadow.
//
// It is indexed by directory absolute path.
type DirGlobalIdShadows map[string]PkgGlobalIdShadows

// IdToGlobalNode indexes GlobalNode values by the associated GlobalNode.Id.Name string.
type IdToGlobalNode map[string]GlobalNode

func (i IdToGlobalNode) GetByBaseName(name string) *GlobalNode {
	for fullName, global := range i {
		if global.Id.BaseName() == name {
			found := i[fullName]
			return &found
		}
	}
	return nil
}

// GlobalNodes holds one GlobalNode per identifier declared by the ast.Node.
//
// GenDecl is an example of an ast.Node which can declare multiple identifiers, e.g. types
// declared in a "type (...)" block. In such examples we store those identifiers in a group
// in order to support use cases like AST rewriting, specifically node removal, which may
// require more than simply remove an ast.Ident from the tree but instead modifying and
// replacing the GenDecl as a whole.
type GlobalNodes map[ast.Node]IdToGlobalNode

// FuncDeclWithShadow describes a function/method declaration which contains a parameter and/or
// receiver identifier which shadows the name of a global.
type FuncDeclWithShadow struct {
	// Name is the function/method name. Method names take this form: "<type name>.<method name>".
	Name string
}

type FuncDeclShadows map[*ast.Ident]FuncDeclWithShadow

// FileImportPaths holds paths imported by a single file.
//
// It is indexed by file absolute paths.
type FileImportPaths map[string]*cage_strings.Set

// PkgImportPaths holds packages' per-file import path sets.
//
// It is indexed by package name (as declared in the source file).
type PkgImportPaths map[string]FileImportPaths

// DirImportPaths holds directories' per-file import path sets.
//
// It is indexed by directory absolute paths.
type DirImportPaths map[string]PkgImportPaths

// IdFilenames holds file absolute paths.
//
// It is indexed by global identifer name.
type IdFilenames map[string]string

// PkgFilenames holds per-package global identifier filenames.
//
// It is indexed by import paths.
type PkgFilenames map[string]IdFilenames

type Inspector struct {
	// BlankImports indexes `import _ "path/to/pkg"` occurrences.
	//
	// It is only populated if the query config includes x/tools/go/packages.NeedSyntax because
	// access to ast.ImportSpec nodes is required to know the import names.
	BlankImports DirImportPaths

	// DotImports indexes `import . "path/to/pkg"` occurrence.
	//
	// It is only populated if the query config includes x/tools/go/packages.NeedSyntax because
	// access to ast.ImportSpec nodes is required to know the import names.
	DotImports DirImportPaths

	// FuncDeclShadows indexes function/method declaration info by parameter and/or receiver identifiers.
	//
	// It supports queries to check if an ast.Ident in a declaration is known to shadow a global.
	FuncDeclShadows FuncDeclShadows

	// FileNodes holds the ast.Node of each file in discovered packages.
	FileNodes FileNodes

	// FilePkgs holds the Package of each file in discovered packages.
	FilePkgs FilePkgs

	// FileSet is the source file set that was parsed as a group during Inspect.
	FileSet *token.FileSet

	// GlobalIdNodes is first indexed by input directory names and then package names at the second level.
	//
	// It provides hierarchical access to the ast.Node of all global identifiers found during PreInspect.
	//
	// Unlike firstPassGlobalIdNames, it includes type methods using the format "<type name>.<method name>".
	//
	// Its leaf Node values may hold types such as ast.GenDecl. Therefore multiple FileNodes keys may
	// return the same Node value.
	//
	// GenDecl is an example of an ast.Node which can declare multiple identifiers, e.g. types
	// declared in a "type (...)" block. In such examples we store those identifiers in a group
	// in order to support use cases like AST rewriting, specifically node removal, which may
	// require more than simply remove an ast.Ident from the tree but instead modifying and
	// replacing the GenDecl as a whole.
	GlobalIdNodes DirGlobalIdNodes

	// ImportPathGlobalIdNodes holds the same node inventory as GlobalIdNodes but indexed by import path
	// instead of dir and package name.
	//
	// It supports cases such as verifying that a global identifier name exists in a package.
	ImportPathGlobalIdNodes ImportPathGlobalIdNodes

	// GlobalIdShadows is first indexed by input directory names and then package names at the second level.
	GlobalIdShadows DirGlobalIdShadows

	// GoFiles holds the files found in each input directory.
	GoFiles dirFiles

	// StdImports indexes Import values for standard library packages directly/transitively imported by
	// directories passed to NewInspector.
	//
	// If x/tools/go/packages.NeedSyntax, the filenames will instead be directory names because the former
	// are not available. Also, Import.UsedName may be inaccurate because the ast.ImportSpec values
	// were not available.
	//
	// Its Import values do not intersect with NonStdImports.
	StdImports DirImports

	// NonStdImports indexes Import values for packages declared in directories passed to NewInspector,
	// or directly/transitively imported by packages in those directories.
	//
	// If x/tools/go/packages.NeedSyntax, the filenames will instead be directory names because the former
	// are not available. Also, Import.UsedName may be inaccurate because the ast.ImportSpec values
	// were not available.
	//
	// Its Import values do not intersect with StdImports.
	NonStdImports DirImports

	// ImportPathToPkg is indexes packages import path.
	ImportPathToPkg map[string]*Package

	// pkgs describes the packages declared in each input directory.
	Pkgs DirPkgs

	// GlobalNodes holds details about AST nodes found in the global scope.
	GlobalNodes GlobalNodes

	// GlobalIdFilenames holds the filename of each global identifier declaration.
	GlobalIdFilenames PkgFilenames

	// UnsupportedTraits holds detected code traits which are not supported and may
	// cause the inspection results to be incorrect/incomplete.
	//
	// Instances of these traits are collected, instead of emitted in errors, to allow the
	// client to decide how to proceed based on its own needs.
	UnsupportedTraits []UnsupportedTrait

	// globalImportPaths is the cache for the GlobalImportPaths method.
	globalImportPaths map[*ast.File][]string

	// globalRefs is the cache for the FindGlobalRef method.
	globalRefs map[*ast.Ident]*GlobalRef

	// identContexts is the cache for the IdentContext method.
	identContexts map[*ast.File]*IdentContext

	// findIdsUsedByNodeCache is the cache for the findIdsUsedByNode method.
	findIdsUsedByNodeCache map[token.Pos][]IdUsedByNode

	// firstPassGlobalIdNames contains the names of identifiers from ast.File.Scope.Objects
	// walked during inspectPackageMeta, and also custom identifiers for each init function
	// found in ast.File.Decls (with names in format "init.<file absolute path>").
	//
	// Its purpose is to support creation of globalIdNodes which contains the complete set of identifiers.
	//
	// It does not include type methods.
	firstPassGlobalIdNames dirGlobalIdNames

	// inspectedDirs holds absolute paths of source directories already processed by Inspect.
	inspectedDirs *cage_strings.Set

	// loadConfig allows clients to customize the amount of x/tools/go/packages.Package
	// to collect. For example, if Mode does not x/tools/go/packages.NeedSyntax, GlobalIdNodes and others
	// that are dependent on AST data will be incomplete or empty.
	loadConfig *Config

	// pkgCache loads packages via x/tools/go/packages.
	pkgCache *Cache
}

// NewInspector analyzes the packages found in the input directories.
func NewInspector(loadConfig *Config, dirs ...string) *Inspector {
	i := &Inspector{
		loadConfig: loadConfig,
		pkgCache:   NewCache(),
	}
	return i.Reset(dirs...)
}

// Reset is equivalent to NewInspector except that the package caches are retained.
func (i *Inspector) Reset(dirs ...string) *Inspector {
	i.FileNodes = make(FileNodes)
	i.FilePkgs = make(FilePkgs)
	i.FuncDeclShadows = make(FuncDeclShadows)
	i.GlobalIdNodes = make(DirGlobalIdNodes)
	i.ImportPathGlobalIdNodes = make(ImportPathGlobalIdNodes)
	i.GlobalIdShadows = make(DirGlobalIdShadows)
	i.GlobalNodes = make(GlobalNodes)
	i.GoFiles = make(dirFiles)
	i.StdImports = make(DirImports)
	i.NonStdImports = make(DirImports)
	i.ImportPathToPkg = make(map[string]*Package)
	i.Pkgs = make(DirPkgs)
	i.BlankImports = make(DirImportPaths)
	i.DotImports = make(DirImportPaths)
	i.GlobalIdFilenames = make(PkgFilenames)

	// method caches
	i.globalImportPaths = make(map[*ast.File][]string)
	i.globalRefs = make(map[*ast.Ident]*GlobalRef)
	i.identContexts = make(map[*ast.File]*IdentContext)
	i.findIdsUsedByNodeCache = make(map[token.Pos][]IdUsedByNode)

	i.firstPassGlobalIdNames = make(dirGlobalIdNames)
	i.inspectedDirs = cage_strings.NewSet()

	i.AddDir(dirs...)

	return i
}

// AddDir appends the list of directories initialized by NewInspector.
//
// It returns true if at least one input directory is new/unique to the list.
//
// It supports incremental expansion of results, such as GlobalIdNodes, after additional Inspect calls.
func (i *Inspector) AddDir(dirs ...string) bool {
	var added bool

	for _, d := range dirs {
		if _, ok := i.Pkgs[d]; ok {
			continue
		}
		if !i.inspectedDirs.Contains(d) {
			i.Pkgs[d] = make(PkgsByName)
			added = true
		}
	}

	return added
}

// Inspect analyzes the packages in the directories passed to NewInspector, populates records such as
// GlobalIdNodes, and returns errors such as unsupported identifier shadowing.
func (i *Inspector) Inspect() []error {
	if errs := i.inspectPackageMeta(); len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.Wrap(errs[n], "failed to collect global identifiers")
		}
		return errs
	}

	if errs := i.inspectPackageNodes(i.filterGlobalIdDecls); len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.Wrap(errs[n], "failed to collect AST nodes")
		}
		return errs
	}

	for dir := range i.Pkgs {
		i.inspectedDirs.Add(dir)
	}

	return []error{}
}

func (i *Inspector) SetPackageCache(c *Cache) *Inspector {
	i.pkgCache = c
	return i
}

// inspectPackageMeta inspects package and file object fields in preparation for inspectPackageNodes.
//
// It initializes many of the data structures used, and/or completed, by inspectPackageNodes
// which index entities including files, imports, and ASTs.
//
// It collects but does not walk file ASTs.
func (i *Inspector) inspectPackageMeta() []error {
	var errs []error

	// importPathToPkgObj indexes Inspector.pkgCache directory-based results by import path.
	importPathToPkgObj := make(map[string]*Package)

	// Sort the dirs passed to NewInspector for a predictable iteration order for debugging.
	var dirs []string
	for d := range i.Pkgs {
		if i.inspectedDirs.Contains(d) {
			continue
		}

		dirs = append(dirs, d)
	}
	cage_strings.SortStable(dirs)

	// Load the dirs in a separate initial pass in order to collect importPathToPkgObj so its
	// full before collecting StdImports/NonStdImports of the first directory.
	dirPkgs, loadErrs := i.pkgCache.LoadDirs(i.loadConfig, dirs...)
	if len(loadErrs) > 0 {
		for _, loadErr := range loadErrs {
			errs = append(errs, errors.WithStack(loadErr))
		}
		return errs
	}

	// Perform an extra walk to populate importPathToPkgObj.
	for d, pkgsByName := range dirPkgs {
		i.Pkgs[d] = pkgsByName
		for pkgName, pkg := range i.Pkgs[d] {
			importPathToPkgObj[pkg.PkgPath] = i.Pkgs[d][pkgName]

			if i.FileSet == nil {
				i.FileSet = pkg.Fset
			}
		}
	}

	for _, d := range dirs {
		i.GoFiles[d] = make(pkgFile)
		i.StdImports[d] = make(PkgImports)
		i.NonStdImports[d] = make(PkgImports)
		i.firstPassGlobalIdNames[d] = make(pkgGlobalIdNames)
		i.BlankImports[d] = make(PkgImportPaths)
		i.DotImports[d] = make(PkgImportPaths)

		sortablePkgNames := cage_strings.NewSet()
		for pkgName := range i.Pkgs[d] {
			sortablePkgNames.Add(pkgName)
		}

		for _, pkgName := range sortablePkgNames.SortedSlice() {
			pkg := i.Pkgs[d][pkgName]

			i.StdImports[d][pkg.Name] = make(FileImports)
			i.NonStdImports[d][pkg.Name] = make(FileImports)
			i.firstPassGlobalIdNames[d][pkg.Name] = cage_strings.NewSet()
			i.ImportPathToPkg[pkg.PkgPath] = pkg
			i.BlankImports[d][pkg.Name] = make(FileImportPaths)
			i.DotImports[d][pkg.Name] = make(FileImportPaths)

			i.GoFiles[d][pkg.Name] = cage_strings.NewSet()
			for _, filename := range cage_strings.NewSet().AddSlice(pkg.GoFiles).SortedSlice() {
				i.GoFiles[d][pkg.Name].Add(filename)
				i.BlankImports[d][pkg.Name][filename] = cage_strings.NewSet()
				i.DotImports[d][pkg.Name][filename] = cage_strings.NewSet()
			}

			sortPackageImports := func(inspectDir, fileOrDir, importPath string, spec *ast.ImportSpec) {
				if _, ok := i.StdImports[inspectDir][pkg.Name][fileOrDir]; !ok {
					i.StdImports[inspectDir][pkg.Name][fileOrDir] = make(PathImport)
				}
				if _, ok := i.NonStdImports[inspectDir][pkg.Name][fileOrDir]; !ok {
					i.NonStdImports[inspectDir][pkg.Name][fileOrDir] = make(PathImport)
				}

				var p string
				if spec == nil {
					p = importPath
				} else {
					p = spec.Path.Value[1 : len(spec.Path.Value)-1]
				}

				// Hits are expected on packages in the directories passed to NewInspector.
				cached, ok := importPathToPkgObj[p]

				if ok {
					if spec != nil && spec.Name != nil && spec.Name.Name == "." {
						i.UnsupportedTraits = append(i.UnsupportedTraits, UnsupportedTrait{
							Type:      TraitDotImport,
							FileOrDir: fileOrDir,
							PkgPath:   pkg.PkgPath,
							Msg:       "imported path [" + spec.Path.Value + "]",
						})
					}

					nonStdImport := NewImportFromPkg(cached)
					if spec != nil && spec.Name != nil {
						nonStdImport.SetUsedName(spec.Name.Name) // apply custom name selected by importer
					}
					i.NonStdImports[inspectDir][pkg.Name][fileOrDir][p] = nonStdImport
				} else {
					// The imported package is located outside the directories passed to NewInspector, e.g. standard
					// libraries and other external dependencies of those directories' packages.

					nonLoadedPkgs, buildPkgErr := i.pkgCache.LoadImportPathWithBuild(p, inspectDir, 0)
					if buildPkgErr != nil {
						errs = append(errs, errors.WithStack(buildPkgErr))
						return
					}

					for _, nonLoadedPkg := range nonLoadedPkgs {
						if strings.HasSuffix(nonLoadedPkg.Name, "_test") {
							continue
						}

						if nonLoadedPkg.Goroot { // standard library
							stdImport := NewImportFromPkg(nonLoadedPkg)
							if spec != nil && spec.Name != nil {
								stdImport.SetUsedName(spec.Name.Name) // apply custom name selected by importer
							}
							i.StdImports[inspectDir][pkg.Name][fileOrDir][p] = stdImport
						} else {
							nonStdImport := NewImportFromPkg(nonLoadedPkg)
							if spec != nil && spec.Name != nil {
								nonStdImport.SetUsedName(spec.Name.Name) // apply custom name selected by importer
							}
							i.NonStdImports[inspectDir][pkg.Name][fileOrDir][p] = nonStdImport
						}
					}
				}
			}

			if len(pkg.Syntax) == 0 {
				// If x/tools/go/packages.NeedSyntax, the filenames will not be available.
				// Reuse the directory name for StdImports/NonStdImports leaf keys.

				importedPaths := cage_strings.NewSet() // Sort iteration to make error lists more stable.
				for p := range pkg.Imports {
					importedPaths.Add(p)
				}

				for _, p := range importedPaths.SortedSlice() {
					sortPackageImports(d, d, p, nil)
				}
			} else {
				// If x/tools/go/packages.NeedSyntax, collect the available filename and AST data.

				for _, f := range pkg.Syntax {
					for _, o := range f.Scope.Objects { // collect only the names of globals declared in the file
						nameSet := i.firstPassGlobalIdNames[d][pkg.Name]
						nameSet.Add(o.Name)
					}

					seenImportPaths := cage_strings.NewSet()
					filename := pkg.FileToName[f]

					for _, spec := range f.Imports {
						sortPackageImports(d, filename, "", spec)

						if seenImportPaths.Contains(spec.Path.Value) {
							i.UnsupportedTraits = append(i.UnsupportedTraits, UnsupportedTrait{
								Type:      TraitDuplicateImport,
								FileOrDir: filename,
								PkgPath:   pkg.PkgPath,
								Msg:       "imported path [" + spec.Path.Value + "]",
							})
						} else {
							seenImportPaths.Add(spec.Path.Value)
						}

						if spec.Name != nil {
							if spec.Name.Name == "_" {
								i.BlankImports[d][pkg.Name][filename].Add(spec.Path.Value[1 : len(spec.Path.Value)-1])
							} else if spec.Name.Name == "." {
								i.DotImports[d][pkg.Name][filename].Add(spec.Path.Value[1 : len(spec.Path.Value)-1])
							}
						}
					}

					fileImports := make(map[string]Import)
					for importPath, im := range i.NonStdImports[d][pkg.Name][filename] {
						fileImports[importPath] = im
					}

					inspectInfo := NewNodeInspectInfo()
					inspectInfo.Dirname = d
					inspectInfo.Filename = filename
					inspectInfo.PkgPath = pkg.PkgPath
					inspectInfo.PkgName = pkg.Name
					inspectInfo.NonStdImports = fileImports

					i.FileNodes[filename] = Node{Ast: f, InspectInfo: inspectInfo}
					i.FilePkgs[filename] = pkg

					var fileBlankIdPos, initFuncPos int

					for _, fileDecl := range f.Decls { // collect info on globals

						switch fd := fileDecl.(type) {

						// const/type/var declaration

						case *ast.GenDecl:
							var genDeclBlankIdPos int

							for _, spec := range fd.Specs {
								switch spec := spec.(type) {

								case *ast.ValueSpec:
									for _, ident := range spec.Names {
										if ident.Name != "_" {
											continue
										}

										idName := NewBlankId(filename, fileBlankIdPos, genDeclBlankIdPos).String()
										fileBlankIdPos++
										genDeclBlankIdPos++

										i.addGlobalNode(fd, ident, idName, inspectInfo)
									}
								}
							}

						// init functions

						case *ast.FuncDecl:
							if fd.Name.Name == "init" {
								initInspectInfo := inspectInfo
								initInspectInfo.InitFuncPos = initFuncPos
								initFuncPos++

								idName := NewInitIdName(filename)

								nameSet := i.firstPassGlobalIdNames[d][pkg.Name]
								nameSet.Add(idName)

								i.addGlobalNode(fd, fd.Name, idName, initInspectInfo)
							}
						}
					}
				}
			}
		}
	}

	return []error(errs)
}

// addGlobalNode indexes the global identifier for later querying.
//
// An example declNode is an ast.GenDecl or ast.FuncDecl which represent the identifier's declaration
// "parent" or "group", while the ast.Ident represents the identifier itself.
//
// Index both the ast.ident and its parent in order to support both in query keys and results. For example,
// a parent ast.GenDecl and one or more ast.Ident nodes of the identifiers declared within it, or a parent
// ast.FuncDecl and the one ast.Ident representing the function's name.
func (i *Inspector) addGlobalNode(declNode ast.Node, ident *ast.Ident, idName GlobalIdName, info NodeInspectInfo) {
	if i.GlobalNodes[declNode] == nil {
		i.GlobalNodes[declNode] = make(IdToGlobalNode)
	}
	i.GlobalNodes[declNode][idName] = GlobalNode{
		Id:          NewGlobalId(info.PkgPath, info.PkgName, info.Filename, idName),
		InspectInfo: info,
	}

	if i.GlobalIdNodes[info.Dirname] == nil {
		i.GlobalIdNodes[info.Dirname] = make(PkgGlobalIdNodes)
	}
	if i.GlobalIdNodes[info.Dirname][info.PkgName] == nil {
		i.GlobalIdNodes[info.Dirname][info.PkgName] = make(IdToNode)
	}
	i.GlobalIdNodes[info.Dirname][info.PkgName][idName] = Node{Ast: declNode, InspectInfo: info}

	if i.ImportPathGlobalIdNodes[info.PkgPath] == nil {
		i.ImportPathGlobalIdNodes[info.PkgPath] = make(IdToNode)
	}
	i.ImportPathGlobalIdNodes[info.PkgPath][idName] = Node{
		Ast:         declNode,
		InspectInfo: info,
	}

	if i.GlobalIdFilenames[info.PkgPath] == nil {
		i.GlobalIdFilenames[info.PkgPath] = make(IdFilenames)
	}
	i.GlobalIdFilenames[info.PkgPath][idName] = info.Filename
}

// inspectPackageNodes visits all AST nodes, in all packages located in input directories, with InspectFunc.
//
// It delegates actual AST node inspection to the ast.Inspect walk function returned by newFileInspector.
func (i *Inspector) inspectPackageNodes(fn InspectFunc) []error {
	var errs []error

	// Sort the dirs passed to NewInspector for a predictable iteration order for debugging.
	var dirs []string
	for d := range i.Pkgs {
		if i.inspectedDirs.Contains(d) {
			continue
		}

		dirs = append(dirs, d)
	}
	cage_strings.SortStable(dirs)

	for _, d := range dirs {
		if i.inspectedDirs.Contains(d) {
			continue
		}

		sortablePkgNames := cage_strings.NewSet()
		for pkgName := range i.Pkgs[d] {
			sortablePkgNames.Add(pkgName)
		}

		for _, pkgName := range sortablePkgNames.SortedSlice() {
			pkg := i.Pkgs[d][pkgName]

			// Inspect the init functions separately because they're not included as an ast.File node
			// (but instead in the ast.File.[]Decls collected earlier by inspectPackageMeta).
			for _, idName := range i.GlobalIdNodes[d][pkg.Name].SortedIds() {
				n := i.GlobalIdNodes[d][pkg.Name][idName]

				filename := ParseInitIdName(idName)
				if filename != "" {
					ast.Inspect(n.Ast, i.newFileInspector(d, pkg, filename, fn, n.InspectInfo.NonStdImports))
				}
			}

			for _, f := range pkg.Syntax {
				filename := pkg.FileToName[f]

				fileImports := make(map[string]Import) // make a copy for init function InspectInfo
				for importPath, im := range i.NonStdImports[d][pkg.Name][filename] {
					fileImports[importPath] = im
				}

				ast.Inspect(f, i.newFileInspector(d, pkg, filename, fn, fileImports))
			}
		}
	}

	return []error(errs)
}

// newFileInspector provides an ast.Inspect walk function which wraps the input InspectFunc. It collects
// and maintains state about the current walk in order to inject a NodeInspectInfo into the InspectFunc,
// providing it with contextual detail such as the name of the current global function.
func (i *Inspector) newFileInspector(dir string, pkg *Package, filename string, fn InspectFunc, imports map[string]Import) func(ast.Node) bool {
	var globalFuncEndPos token.Pos
	var globalFuncName string
	var globalMethodTypeName *ast.Ident

	return func(n ast.Node) bool {
		if n == nil {
			return false
		}

		info := NewNodeInspectInfo()
		info.Dirname = dir
		info.Filename = filename
		info.PkgName = pkg.Name
		info.PkgPath = pkg.PkgPath
		info.NonStdImports = imports

		if globalFuncEndPos > 0 && n.Pos() > globalFuncEndPos { // exiting a global-scope func/method
			globalFuncEndPos = 0
			globalFuncName = ""
			globalMethodTypeName = nil
		}

		info.GlobalScope = globalFuncEndPos == 0

		if info.GlobalScope {
			switch x := n.(type) {

			case *ast.FuncDecl:
				globalFuncEndPos = n.End()
				globalFuncName = x.Name.Name

				if x.Name.Name == "init" {
					globalFuncName = NewInitIdName(filename)
				}

				var funcDecl FuncDeclWithShadow
				var shadowIdents []*ast.Ident

				if x.Recv != nil && len(x.Recv.List) > 0 { // method detection
					// It's unclear why there's a receiver list instead of a single value. We assume there will only
					// be one iteration.
					for _, field := range x.Recv.List {
						// Determine the method's type by looking at the first/only receiver type.
						switch f := field.Type.(type) { // receiver types
						case *ast.Ident: // value receiver
							globalMethodTypeName = f
						case *ast.StarExpr: // pointer receiver
							switch ptrRecvIdent := f.X.(type) {
							case *ast.Ident:
								globalMethodTypeName = ptrRecvIdent
							}
						}

						shadowIdents = append(shadowIdents, field.Names...) // receiver variable names
					}

					funcDecl.Name = globalMethodTypeName.Name + GlobalIdSeparator + globalFuncName
				} else {
					funcDecl.Name = globalFuncName
				}

				for _, list := range x.Type.Params.List { // function/method parameter names
					shadowIdents = append(shadowIdents, list.Names...)
				}

				for _, ident := range shadowIdents {
					i.FuncDeclShadows[ident] = funcDecl
				}

			case *ast.GenDecl:
				if x.Tok != token.CONST {
					break
				}

				var iotaValue bool // true if the previous identifier is iota-valued

				for _, spec := range x.Specs {
					switch spec := spec.(type) {
					case *ast.ValueSpec:
						valuesLen := len(spec.Values)

						for nPos := range spec.Names {

							if valuesLen == 0 || nPos >= valuesLen { // identifier has an implicit value
								if iotaValue {
									info.IotaValuedNames.Add(spec.Names[nPos].Name)
								}
								continue
							}

							switch valType := spec.Values[nPos].(type) {

							case *ast.Ident:
								if valType.Name == "iota" {
									info.IotaValuedNames.Add(spec.Names[nPos].Name)
									iotaValue = true
								} else if valType.Name != "" {
									iotaValue = false
								}

							case *ast.BinaryExpr: // like ast.Ident handling above but we look in both sides of the expression
								switch valXType := valType.X.(type) {
								case *ast.Ident:
									if valXType.Name == "iota" {
										info.IotaValuedNames.Add(spec.Names[nPos].Name)
										iotaValue = true
									}
									switch valYType := valType.Y.(type) {
									case *ast.Ident:
										if valYType.Name == "iota" {
											info.IotaValuedNames.Add(spec.Names[nPos].Name)
											iotaValue = true
										}
									}
								}

							default: // not one of the types in which we expect the "iota" identifier
								iotaValue = false
							}
						}
					}
				}
			}
		}

		info.GlobalFuncName = globalFuncName

		if globalMethodTypeName == nil {
			info.GlobalMethodTypeName = ""
		} else {
			info.GlobalMethodTypeName = globalMethodTypeName.Name
		}

		return fn(n, info)
	}
}

// filterGlobalIdDecls collects Node values for each discovered global identifier declaration
// (or shadow declaration).
//
// It is an InspectFunc passed to inspectPackageNodes in Inspector.Inspect.
func (i *Inspector) filterGlobalIdDecls(n ast.Node, info NodeInspectInfo) bool {
	// candidateGlobalIdDeclsByNode is the means by which the logic below can
	// assume each declaration shares an identifier name with a known global.
	decls := i.candidateGlobalIdDeclsByNode(info.Dirname, info.PkgName, info, n)
	declsLen := len(decls)

	if declsLen == 0 {
		return true
	}

	for _, decl := range decls {
		receiverOrParam := decl.DeclType == methodRecvDeclType || decl.DeclType == funcOrMethodParamDeclType

		if info.GlobalScope {
			if !receiverOrParam {
				i.addGlobalNode(n, decl.Ident, decl.Name, info)
			}
			continue
		}

		if _, ok := i.GlobalIdShadows[info.Dirname]; !ok {
			i.GlobalIdShadows[info.Dirname] = make(PkgGlobalIdShadows)
		}
		if _, ok := i.GlobalIdShadows[info.Dirname][info.PkgName]; !ok {
			i.GlobalIdShadows[info.Dirname][info.PkgName] = make(FileGlobalIdShadows)
		}
		if _, ok := i.GlobalIdShadows[info.Dirname][info.PkgName][info.Filename]; !ok {
			i.GlobalIdShadows[info.Dirname][info.PkgName][info.Filename] = make(Shadows)
		}

		// The location of the shadow is always in a function or method. Global scope shadows
		// are already detected by the compiler as a redeclaration error.
		var funcOrMethodName string
		if decl.StructPropType == "" {
			funcOrMethodName = info.GlobalFuncName
		} else {
			funcOrMethodName = decl.StructPropType + GlobalIdSeparator + decl.StructPropName
		}

		if _, ok := i.GlobalIdShadows[info.Dirname][info.PkgName][info.Filename][funcOrMethodName]; !ok {
			i.GlobalIdShadows[info.Dirname][info.PkgName][info.Filename][funcOrMethodName] = cage_strings.NewSet()
		}
		i.GlobalIdShadows[info.Dirname][info.PkgName][info.Filename][funcOrMethodName].Add(decl.Name)
	}

	return true
}

// candidateGlobalIdDeclsByNode non-recursively collects the identifiers, declared by the input AST node,
// whose names match a known global identifier.
//
// It does not detect shadowing of a declared package name when 1) the name differs from the base of the
// import path, and 2) the import statement does not specify the declared name (or a custom one). goimports
// automatically fixes import statements to specify the declared name if a custom one is not present,
// so the lack of detection support here should be low impact.
//
// Typically it will only collect one identifier per node. But if the node declares a method,
// it will collect both the method and receiver names. (The latter is collected in case it
// shadows a global identifier.)
func (i *Inspector) candidateGlobalIdDeclsByNode(dir, pkgName string, info NodeInspectInfo, n ast.Node) (found []globalIdDecl) {
	isCandidate := func(declType globalIdDeclType, idName GlobalIdName) (isCandidate bool, decl globalIdDecl) {
		if idName == "_" { // blank identifiers already collected in inspectPackageMeta
			return false, globalIdDecl{}
		}

		decl.Name = idName
		decl.DeclType = declType

		if i.firstPassGlobalIdNames[dir][pkgName].Contains(idName) { // const/func/var/type
			return true, decl
		}

		// Import name shadows
		//
		// We do not need to assign a custom globalIdDeclType here for filterGlobalIdDecls to catch and
		// automatically consider the declaration a shadow. If an import name shadow is in the global scope,
		// the compiler will emit an "already declared through import of package" error. So here we only
		// catch shadows in non-global scope, which filterGlobalIdDecls assumes are shadows.

		for importPath, importObj := range i.NonStdImports[dir][pkgName][info.Filename] {
			if importObj.UsedName == idName {
				decl.ImportPath = importPath
				return true, decl
			}
		}

		for importPath, importObj := range i.StdImports[dir][pkgName][info.Filename] {
			if importObj.UsedName == idName {
				decl.ImportPath = importPath
				return true, decl
			}
		}

		return false, globalIdDecl{}
	}

	switch x := n.(type) {
	case *ast.FuncDecl:
		var methodName, methodType string

		checkParams := func() { // function/method parameter names
			if x.Type != nil && x.Type.Params != nil {
				for _, field := range x.Type.Params.List {
					for _, name := range field.Names {
						if candidate, decl := isCandidate(funcOrMethodParamDeclType, name.Name); candidate {
							decl.Ident = name
							decl.StructPropName = methodName
							decl.StructPropType = methodType
							found = append(found, decl)
						}
					}
				}
			}
		}

		// global-scope type method
		//
		// Inspect methods first to handle the case where there exists a function named "F" and also a
		// a method named "F". If we assume an isCandidate() match on "F" is a function, we won't catch
		// the method of the same name (due to how we break from the ast.FuncDecl block as soon as we
		// decide it is one or the other).

		if x.Recv != nil && len(x.Recv.List) > 0 {
			methodName = x.Name.Name

			// It's unclear why there's a receiver list instead of a single value. We assume there will only
			// be one iteration.
			for _, field := range x.Recv.List {
				var recvIdent *ast.Ident

				// Determine the method's type by looking at the first/only receiver type.

				var recvTypeName string
				switch f := field.Type.(type) { // receiver types
				case *ast.Ident: // value receiver
					recvIdent = f
					recvTypeName = f.Name
				case *ast.StarExpr: // pointer receiver
					switch ptrRecvIdent := f.X.(type) {
					case *ast.Ident:
						recvIdent = ptrRecvIdent
						recvTypeName = ptrRecvIdent.Name
					}
				}

				if recvTypeName != "" {
					if candidate, decl := isCandidate(methodDeclType, recvTypeName); candidate {
						decl.Ident = recvIdent

						methodType = recvTypeName
						decl.StructPropName = methodName
						decl.StructPropType = methodType

						// Overwrite the Name assigned automatically (f.Name) in isCandidate because we need
						// to collect a name for the method, not its type. We passed its type, f.Name, to isCandidate
						// in order for the latter just to check if it was a known global.
						decl.Name = recvTypeName + GlobalIdSeparator + methodName

						found = append(found, decl)
					}
				}

				if methodType == "" {
					break
				}

				for _, fieldName := range field.Names {
					if candidate, decl := isCandidate(methodRecvDeclType, fieldName.Name); candidate {
						decl.Ident = fieldName
						decl.StructPropName = methodName
						decl.StructPropType = methodType
						found = append(found, decl)
					}
				}
			}

			checkParams()

			break
		}

		// global-scope function

		if candidate, decl := isCandidate(funcDeclType, x.Name.Name); candidate {
			decl.Ident = x.Name
			found = append(found, decl)
			checkParams()
		}

	// short variable declarations
	//
	// While globals won't be declared in this form, we'll collect the declarations anyway
	// for detecting shadows declared in this form.
	case *ast.AssignStmt:
		if x.Tok != token.DEFINE {
			break
		}
		for _, expr := range x.Lhs {
			switch exprNode := expr.(type) {
			case *ast.Ident:
				if candidate, decl := isCandidate(nonFuncOrMethodDeclType, exprNode.Name); candidate {
					if info.GlobalMethodTypeName != "" {
						decl.StructPropName = info.GlobalFuncName
						decl.StructPropType = info.GlobalMethodTypeName
					}
					decl.Ident = exprNode
					found = append(found, decl)
				}
			}
		}
		return found

	case *ast.GenDecl: // const/type/var declaration
		for _, spec := range x.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				// collect the type itself

				if candidate, decl := isCandidate(nonFuncOrMethodDeclType, spec.Name.Name); candidate {
					decl.Ident = spec.Name
					found = append(found, decl)
				}

				switch specTypeType := spec.Type.(type) {

				case *ast.StructType:
					// if it's a struct, collect its fields
					for _, field := range specTypeType.Fields.List {
						for _, fieldIdent := range field.Names {
							if candidate, decl := isCandidate(fieldDeclType, spec.Name.Name); candidate {
								decl.Ident = fieldIdent

								decl.StructPropName = fieldIdent.Name
								decl.StructPropType = spec.Name.Name

								// Overwrite the Name assigned automatically (f.Name) in isCandidate because we need
								// to collect a name for the field, not its type. We passed its type, f.Name, to isCandidate
								// in order for the latter just to check if it was a known global.
								decl.Name = spec.Name.Name + GlobalIdSeparator + fieldIdent.Name

								found = append(found, decl)
							}
						}
					}

				case *ast.InterfaceType: // if it's an interface, collect its methods
					if specTypeType.Methods == nil {
						break
					}

					for _, method := range specTypeType.Methods.List {
						for _, methodIdent := range method.Names {
							switch method.Type.(type) {
							case *ast.FuncType:
								if candidate, decl := isCandidate(methodDeclType, spec.Name.Name); candidate {
									decl.Ident = spec.Name

									decl.StructPropName = methodIdent.Name
									decl.StructPropType = spec.Name.Name

									// Overwrite the Name assigned automatically (f.Name) in isCandidate because we need
									// to collect a name for the method, not its type. We passed its type, f.Name, to isCandidate
									// in order for the latter just to check if it was a known global.
									decl.Name = spec.Name.Name + GlobalIdSeparator + methodIdent.Name

									found = append(found, decl)
								}
							}
						}
					}

				} // specTypeType
			case *ast.ValueSpec:
				for _, ident := range spec.Names {
					if candidate, decl := isCandidate(nonFuncOrMethodDeclType, ident.Name); candidate {
						decl.Ident = ident
						found = append(found, decl)
					}
				}
			}
		}
	}

	return found
}

func (i *Inspector) NodeToFilename(n ast.Node) string {
	return i.FileSet.File(n.Pos()).Name()
}

func (i *Inspector) NodeToString(n ast.Node) string {
	return cage_ast.FileSetNodeToString(i.FileSet, n)
}
