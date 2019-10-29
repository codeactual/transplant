// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/ast/astutil"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// File stores details about a single-file which will be copied to the stage and
// final destination, supporting copy-related tasks such as content updates and renaming.
//
// For files which require node pruning, use PrunableFile instead.
type File struct {
	// AbsPath is the absolute path to the file being copied.
	AbsPath string

	// Dir is the parent directory of the file.
	Dir string

	// Go is true of the file is a Go file.
	Go bool

	// Node is the inspection data about the associated ast.File.
	Node cage_pkgs.Node

	// FileSet contains the subject file if it is of Go type.
	FileSet *token.FileSet

	// Pkg is the inspection data about the file's package.
	Pkg *cage_pkgs.Package

	// Mode represents the file's mode/permission bits.
	Mode os.FileMode
}

type FileConfig struct {
	Audit            *Audit
	ToFilePathPrefix string
}

// NewFile returns a new File instance.
//
// All file path parameters must be absolute.
func NewFile(audit *Audit, newFilePath string) (f *File, err error) {
	f = &File{
		AbsPath: newFilePath,
		Dir:     filepath.Dir(newFilePath),
	}

	stat, err := os.Stat(newFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat [%s]", newFilePath)
	}
	f.Mode = stat.Mode()

	f.Go = cage_filepath.IsGoFile(newFilePath)

	if f.Go {
		f.FileSet = audit.inspector.FileSet
	}

	inspected := f.Go &&
		(audit.LocalGoFiles.Contains(newFilePath) ||
			audit.LocalGoTestFiles.Contains(newFilePath) ||
			audit.UsedDepGoFiles.Contains(newFilePath) ||
			audit.DepGoTestFiles.Contains(newFilePath))

	if !inspected {
		return f, nil
	}

	var ok bool

	f.Node, ok = audit.inspector.FileNodes[newFilePath]
	if !ok {
		return nil, errors.Errorf("failed to load ast.File [%s]", newFilePath)
	}

	f.Pkg, ok = audit.inspector.FilePkgs[newFilePath]
	if !ok {
		return nil, errors.Errorf("failed to load Package [%s]", newFilePath)
	}

	return f, nil
}

func (f *File) Apply(fn func(cursor *astutil.Cursor) bool) ast.Node {
	return astutil.Apply(f.Node.Ast.(*ast.File), fn, nil)
}

// RenameIfGoFileNamedAfterRootPackage alters the file's base name to match the package name of its
// destination directory if: the file's base name currently matches its package, the parent
// directory is the local/dep root dir, and a file with the new name does not conflict.
//
// Example: Ops.From.ImportPath is "path/to/proj", Ops.To.ImportPath is "path/to/other",
// and the file's path relative to Ops.From.FilePath is "proj.go". "proj.go" is renamed
// to "other.go". However if the file's relative path is deep in the tree, e.g. "subdir/proj.go",
// then the file would not be renamed because the attempt to persist the naming convention
// should only happen at the top of the tree.
func (f *File) RenameIfGoFileNamedAfterRootPackage(fromImportPath, toImportPath string, allAbsPaths *cage_strings.Set) {
	fromImportPathBase := path.Base(fromImportPath)
	toImportPathBase := path.Base(toImportPath)

	if fromImportPathBase == toImportPathBase { // desired state already present
		return
	}

	if !cage_filepath.IsGoFile(f.AbsPath) { // we're only targeting package source files
		return
	}

	var toAbsPath string

	if cage_filepath.IsGoTestFile(f.AbsPath) && cage_filepath.BaseWithoutExt(f.AbsPath) == fromImportPathBase+"_test" {
		toAbsPath = strings.Replace(f.AbsPath, fromImportPathBase+"_test.go", toImportPathBase+"_test.go", 1)
	} else if cage_filepath.IsGoFile(f.AbsPath) && cage_filepath.BaseWithoutExt(f.AbsPath) == fromImportPathBase {
		toAbsPath = strings.Replace(f.AbsPath, fromImportPathBase+".go", toImportPathBase+".go", 1)
	} else { // base name mismatch
		return
	}

	if allAbsPaths.Contains(toAbsPath) { // another file to be copied already has the proposed name
		return
	}

	f.AbsPath = toAbsPath
}

// LocalDestPaths returns a Ops.From file's copy-destination (Ops.To) paths.
func (f *File) LocalDestPaths(op Op) (absPath, relPath string) {
	fromRelToPrefix := strings.TrimPrefix(f.AbsPath, FromAbs(op, op.From.LocalFilePath)+string(filepath.Separator))
	absPath = ToAbs(op, op.To.LocalFilePath, fromRelToPrefix)
	relPath = strings.TrimPrefix(absPath, op.To.ModuleFilePath+"/")
	return absPath, relPath
}

// DepDestPaths returns a Ops.Dep.From file's copy-destination (Ops.To) paths.
func (f *File) DepDestPaths(op Op, dep Dep) (absPath, relPath string) {
	fromRelToPrefix := strings.TrimPrefix(f.AbsPath, FromAbs(op, dep.From.FilePath)+string(filepath.Separator))
	absPath = ToAbs(op, dep.To.FilePath, fromRelToPrefix)
	relPath = strings.TrimPrefix(absPath, op.To.ModuleFilePath+"/")
	return absPath, relPath
}

func (c *Copier) logFileActivity(name, msg string) {
	if c.WhyLog != nil {
		c.WhyLog[name] = append(c.WhyLog[name], msg)
	}
}

// RenamePackageClause alters the `package X` line if the file is at the top level
// of the origin file path by renaming the package to match the destination import path's base name.
//
// It allows files which are named after parent, top-level directory to continue being named after
// the parent, e.g. files in Op.From.LocalFilePath or Op.Dep.From.FilePath.
//
// E.g. if origin file "from1" and the destination import base is "f1", then the new package clause
// will contain the name "f1".
func (f *File) RenamePackageClause(topLevelFilePath, fromImportPath, toImportPath string, fileText []byte) []byte {
	if f.Dir != topLevelFilePath {
		return fileText
	}

	var fromBase, toBase string

	if cage_filepath.IsGoFile(f.AbsPath) {
		fromBase = path.Base(fromImportPath)
		toBase = path.Base(toImportPath)
	} else if cage_filepath.IsGoTestFile(f.AbsPath) {
		fromBase = path.Base(fromImportPath) + "_test"
		toBase = path.Base(toImportPath) + "_test"
	} else {
		return fileText
	}

	return bytes.Replace(fileText, []byte("package "+fromBase), []byte("package "+toBase), 1)
}

// GetNodeBytes converts the input node into a []byte using the file's decorator.
func (f *File) GetNodeBytes(node ast.Node) ([]byte, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, f.FileSet, node.(*ast.File)); err != nil {
		return []byte{}, errors.Wrapf(err, "failed to convert decorated file [%s] string", f.AbsPath)
	}
	return buf.Bytes(), nil
}

// RewriteImportsInNode updates an AST node if it imports or references a package whose location will
// change in the copy operation.
//
// Import declarations and identifiers which include an import name of a Op.From.LocalFilePath/Op.Dep.From.FilePath
// package will be updated to the associated Op.To.LocalFilePath/Op.Dep.To.FilePath path/name.
func (f *File) RewriteImportsInNode(audit *Audit, op Op, cursor *astutil.Cursor, isLocal bool) (err error) {
	node := cursor.Node()

	switch nodeType := node.(type) {

	case *ast.ImportSpec:
		nodeType.Path.Value = `"` + f.RewriteImportPath(audit, op, nodeType.Path.Value[1:len(nodeType.Path.Value)-1], isLocal) + `"`

	case *ast.Ident:
		identPkg, identFile, _ := audit.inspector.FindAstNode(node)
		if identPkg == nil {
			return errors.Wrapf(err, "failed to get File for nodeType [%s]\n", audit.inspector.NodeToString(node))
		}

		typesObj, _, _ := audit.inspector.IdentObjectOf(identPkg.PkgPath, identFile, node.(*ast.Ident))

		if typesObj == nil {
			return nil
		}

		switch pkgNameObj := typesObj.(type) {
		case *types.PkgName:
			if pkgNameObj.Imported().Path() == op.From.LocalImportPath {
				nodeType.Name = path.Base(op.To.LocalImportPath)
				cursor.Replace(nodeType)
				break
			}

			for _, dep := range op.Dep {
				if pkgNameObj.Imported().Path() == dep.From.ImportPath {
					nodeType.Name = path.Base(dep.To.ImportPath)
					cursor.Replace(nodeType)
					break
				}
			}
		}
	}

	return nil
}

// RewriteImportPath detects Op.From.LocalImportPath/Op.Dep.From.ImportPath prefixes and replaces them
// with  Op.To.LocalImportPath/Op.Dep.To.ImportPath.
func (f *File) RewriteImportPath(audit *Audit, op Op, importPath string, isLocal bool) string {
	if isLocal {
		return audit.AllImportPathReplacer.InString(importPath)
	}
	return audit.DepImportPathReplacer.InString(importPath)
}

// MatchAnyFileRelPath requires candidate paths to contain the input prefix, and only matches input
// patterns against the variable, relative suffixes. If the prefix itself is a candidate path,
// it will be included in the matched results.
//
// For example, "**/dirname/*" would normally match any file under a "dirname" directory
// regardless of the prefix before the "dirname" path part.
//
// This function only returns candidates which contain the input prefix, and only matches patterns
// against the remaining suffix.
func MatchAnyFileRelPath(prefix string, config cage_filepath.MatchAnyInput) cage_file.FileMatcher {
	prefix = filepath.Clean(prefix)

	return func(candidate cage_file.FinderFile) (bool, error) {
		if !strings.HasPrefix(candidate.AbsPath, prefix) {
			return false, nil
		}

		res, err := cage_filepath.PathMatchAny(cage_filepath.MatchAnyInput{
			// Trim both the prefix and leading separator of the remaining suffix so the patterns
			// are matched against a relative path.
			Name: strings.TrimPrefix(candidate.AbsPath, prefix+string(filepath.Separator)),

			Include: config.Include,
			Exclude: config.Exclude,
		})

		if err != nil {
			return false, errors.WithStack(err)
		}

		return res.Match, nil
	}
}

// MatchAnyDirRelPath requires candidate paths to contain the input prefix, and only matches input
// patterns against the variable, relative suffixes. If the prefix itself is a candidate path,
// it will be included in the matched results.
//
// For example, "**/dirname/*" would normally match any directory under a "dirname" directory
// regardless of the prefix before the "dirname" path part.
//
// This function only returns candidates which contain the input prefix, and only matches patterns
// against the remaining suffix.
func MatchAnyDirRelPath(prefix string, config cage_filepath.MatchAnyInput) cage_file.DirMatcher {
	prefix = filepath.Clean(prefix)

	return func(candidateAbsPath string, _ cage_file.FinderMatchFiles) (bool, error) {
		res, err := cage_filepath.PathMatchAny(cage_filepath.MatchAnyInput{
			// Trim both the prefix and leading separator of the remaining suffix so the patterns
			// are matched against a relative path.
			Name: strings.TrimPrefix(candidateAbsPath, prefix+string(filepath.Separator)),

			Include: config.Include,
			Exclude: config.Exclude,
		})

		if err != nil {
			return false, errors.WithStack(err)
		}

		return res.Match, nil
	}
}
