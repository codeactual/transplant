// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant

import (
	"bytes"
	"go/ast"
	"go/types"
	"path"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
	"github.com/pkg/errors"

	cage_errors "github.com/codeactual/transplant/internal/cage/errors"
	cage_dst "github.com/codeactual/transplant/internal/cage/go/dst"
	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// PrunableFile stores details about a single-file which will be copied to the stage and
// final destination, supporting copy-related tasks such as content updates and renaming.
//
// It serves the same purpose as File but only for files which require may have its nodes
// pruned and printed by github.com/dave/dst for improved comment handling.
type PrunableFile struct {
	*File

	// DecoratedFile is a converted ast.File that contains additional information which supports
	// such things as AST editing w/ comment position retention.
	DecoratedFile *dst.File

	// Decorator converts the ast.File to Decorated file, maps dst.Node values to ast.Node values, etc.
	Decorator *decorator.Decorator
}

// NewPrunableFile returns an instance representing a file in the tree which will be copied.
//
// It performs the same role as File except for supporting github.com/dave/dst ASTs
// which provides improved comment handling when nodes are removed.
//
// All file path parameters must be absolute.
func NewPrunableFile(audit *Audit, newFilePath string) (f *PrunableFile, err error) {
	baseFile, err := NewFile(audit, newFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create base file object [%s]", newFilePath)
	}

	f = &PrunableFile{File: baseFile}
	f.Decorator = decorator.NewDecorator(f.Pkg.Fset)

	f.DecoratedFile, err = f.Decorator.DecorateFile(f.Node.Ast.(*ast.File))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decorate AST file [%s]", newFilePath)
	}

	return f, nil
}

// UpdateDepAst modifies import paths/names to point to destination paths, prunes declarations
// of unused globals, and removes unused import declarations.
func (f *PrunableFile) UpdateDepAst(audit *Audit, op Op) (prunedGlobalIds *cage_strings.Set, errs []error) {
	// Track how many "_" globals we encounter in the file during pruneDepNodes in order
	// for the latter to use cage_pkgs.NewBlankIdName to obtain the globals' transplant-specific IDs.
	var blankIdFilePos int

	prunedGlobalIds = cage_strings.NewSet()

	f.DecoratedFile = f.Apply(func(cursor *dstutil.Cursor) bool { //nolint:errcheck
		astNode := f.Decorator.Ast.Nodes[cursor.Node()]

		if !op.Ingress {
			globalIds, pruneErr := pruneDepNodes(audit, f.Node.InspectInfo, &blankIdFilePos, op, cursor, astNode)
			if cage_errors.Append(&errs, errors.WithStack(pruneErr)) {
				return false
			}
			prunedGlobalIds.AddSlice(globalIds)
		}

		err := f.RewriteImportsInNode(audit, op, cursor, astNode, false)

		return !cage_errors.Append(&errs, errors.WithStack(err))
	}).(*dst.File)

	if len(errs) > 0 {
		return nil, errs
	}

	pathsUsedAfterPrune, pathsErrs := f.GetImportedPaths(audit, op, true)
	if len(pathsErrs) > 0 {
		for n := range pathsErrs {
			cage_errors.Append(&errs, errors.Wrapf(pathsErrs[n], "failed to get paths imported in file [%s]", f.AbsPath))
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	f.DecoratedFile = f.Apply(func(cursor *dstutil.Cursor) bool { //nolint:errcheck
		if cursor.Index() < 0 {
			return true
		}
		switch genDecl := cursor.Node().(type) {
		case *dst.GenDecl:
			var updatedSpecs []dst.Spec
			var removeLen int
			for _, spec := range genDecl.Specs {
				switch s := spec.(type) {
				case *dst.ImportSpec:
					importPath := s.Path.Value[1 : len(s.Path.Value)-1]

					// Retain imports paths which are blank-named, dot-named, or detected as used.
					if (s.Name != nil && (s.Name.Name == "_" || s.Name.Name == ".")) || pathsUsedAfterPrune.Contains(importPath) {
						updatedSpecs = append(updatedSpecs, s)
					} else {
						removeLen++
					}
				}
			}
			if removeLen == 0 {
				break
			}
			if len(updatedSpecs) == 0 {
				cursor.Delete()
			} else {
				cage_dst.SpecsTrimEmptyLine(updatedSpecs)
				genDecl.Specs = updatedSpecs
				cursor.Replace(genDecl)
			}
		}
		return true
	}).(*dst.File)

	return prunedGlobalIds, errs
}

// Apply is an astutil.Apply alternative that retains comment position after AST mutation.
func (f *PrunableFile) Apply(fn func(cursor *dstutil.Cursor) bool) dst.Node {
	return dstutil.Apply(f.DecoratedFile, fn, nil)
}

// GetNodeBytes converts the input node into a []byte using the file's decorator.
func (f *PrunableFile) GetNodeBytes(node dst.Node) ([]byte, error) {
	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, node.(*dst.File)); err != nil {
		return []byte{}, errors.Wrapf(err, "failed to convert decorated file [%s] string", f.AbsPath)
	}
	return buf.Bytes(), nil
}

// GetImportedPaths returns the import paths used in the file based on inspection data.
func (f *PrunableFile) GetImportedPaths(audit *Audit, op Op, isLocal bool) (paths *cage_strings.Set, errs []error) {
	paths = cage_strings.NewSet()

	dstutil.Apply(f.DecoratedFile, nil, func(cursor *dstutil.Cursor) bool {
		if len(errs) > 0 {
			return false
		}

		if cursor.Index() < 0 {
			return true
		}

		astNode := f.Decorator.Ast.Nodes[cursor.Node()]
		if astNode == nil {
			return false
		}

		_, usedPkgsByPath, pkgsUsedErrs := audit.inspector.PackagesUsedByNode(f.Dir, f.Pkg.Name, astNode)
		if len(pkgsUsedErrs) > 0 {
			for _, err := range pkgsUsedErrs {
				errs = append(errs, errors.WithStack(err))
			}
			return false
		}

		for p := range usedPkgsByPath {
			p = cage_pkgs.TrimVendorPathPrefix(p)

			rewritten := f.RewriteImportPath(audit, op, p, isLocal)
			if p == rewritten {
				paths.Add(p)
			} else {
				paths.Add(rewritten)
			}
		}

		return true
	})

	return paths, []error{}
}

// RewriteImportsInNode updates an AST node if it imports or references a package whose location will
// change in the copy operation.
//
// Import declarations and identifiers which include an import name of a Op.From.LocalFilePath/Op.Dep.From.FilePath
// package will be updated to the associated Op.To.LocalFilePath/Op.Dep.To.FilePath path/name.
func (f *PrunableFile) RewriteImportsInNode(audit *Audit, op Op, cursor *dstutil.Cursor, astNode ast.Node, isLocal bool) (err error) {
	switch decorNode := cursor.Node().(type) {

	case *dst.ImportSpec:
		decorNode.Path.Value = `"` + f.RewriteImportPath(audit, op, decorNode.Path.Value[1:len(decorNode.Path.Value)-1], isLocal) + `"`

	case *dst.Ident:
		identPkg, identFile, _ := audit.inspector.FindAstNode(astNode)
		if identPkg == nil {
			return errors.Wrapf(err, "failed to get File for node [%s]\n", audit.inspector.NodeToString(astNode))
		}

		typesObj, _, _ := audit.inspector.IdentObjectOf(identPkg.PkgPath, identFile, astNode.(*ast.Ident))

		if typesObj == nil {
			return nil
		}

		switch pkgNameObj := typesObj.(type) {
		case *types.PkgName:
			if pkgNameObj.Imported().Path() == op.From.LocalImportPath {
				decorNode.Name = path.Base(op.To.LocalImportPath)
				cursor.Replace(decorNode)
				break
			}

			for _, dep := range op.Dep {
				if pkgNameObj.Imported().Path() == dep.From.ImportPath {
					decorNode.Name = path.Base(dep.To.ImportPath)
					cursor.Replace(decorNode)
					break
				}
			}
		}
	}

	return nil
}

// pruneDepNodes removes global identifiers from the cursor-defined AST node if they were not detected
// as directly/transitively used by Ops.From packages during the audit step.
//
// For ast.GenDecl nodes, which may declare multiple globals, each of their ast.Spec nodes are evaluated
// for their use by Ops.From packages. Individual ast.Spec nodes are removed from the ast.GenDecl and
// then the latter is replaced as a whole. Direct removal of ast.Spec nodes is not performed because
// the AST cursors provided by github.com/dave/dst/dstutil (and also Go's astutil, IIRC) do not support
// that granularity.
//
// For all other node types, it performs a simple deletion of the whole node at the cursor.
//
// github.com/dave/dst is used instead of Go's astutil to support removal of leading/inline comments
// of pruned nodes.
func pruneDepNodes(audit *Audit, fileInspectInfo cage_pkgs.NodeInspectInfo, blankIdFilePos *int, op Op, cursor *dstutil.Cursor, astNode ast.Node) (prunedGlobalIds []string, err error) {
	if cursor.Index() < 0 { // cursor.Delete requires cursor.Index() >= 0 {
		return []string{}, nil
	}

	globals, ok := audit.inspector.GlobalNodes[astNode]
	if !ok {
		return []string{}, nil
	}

	var genDecl *dst.GenDecl

	switch dstNode := cursor.Node().(type) {

	// GenDecl
	//
	// Collect the node for processing it below.

	case *dst.GenDecl:
		genDecl = dstNode

	// All other node types:
	//
	// Delete unused nodes directly.

	default:
		for _, global := range globals {
			if !audit.IsDepGlobalUsedInLocal(global.Id) {
				cursor.Delete()
				prunedGlobalIds = append(prunedGlobalIds, global.Id.String())
			}
		}
		return prunedGlobalIds, nil
	}

	// Index all globals declared in the GenDecl, names mapped to their full transplant-specific IDs.
	usedGlobalIdNames := make(map[string]string)
	for _, global := range globals {
		if audit.IsDepGlobalUsedInLocal(global.Id) {
			usedGlobalIdNames[global.Id.BaseName()] = global.Id.String()
		}
	}

	var updatedSpecs []dst.Spec // specs to define the GenDecl that is replaced as a whole
	var specRemoveLen int       // specs pruned from the GenDecl

	// Track many "_" identifiers have been encountered so far. Transplant-specific global IDs
	// of blank identifiers incorporate their order of occurrence in the AST walk of their
	// specific files and also order in the GenDecl itself. To obtain the transplant-specific ID,
	// the current file order (maintained across all pruneDepNodes calls for the file) and this
	// GenDecl order are passed to cage_pkgs.NewBlankIdName.
	var blankIdGenDeclPos int

	for _, spec := range genDecl.Specs {
		switch s := spec.(type) {

		// Type declarations:
		//
		// Prune if we did not store the global's name above in usedGlobalIdNames.

		case *dst.TypeSpec:
			globalId := usedGlobalIdNames[s.Name.Name]

			if globalId == "" {
				specRemoveLen++
			} else {
				updatedSpecs = append(updatedSpecs, s)
			}

		// Variable declarations:
		//
		// - For blank identifiers, obtain their transplant-specific global ID (see blankIdGenDeclPos notes above)
		//   and prune them if IsDepGlobalUsedInLocal returns false.
		// - For all other node types, prune them if we did not store the global's name above in usedGlobalIdNames.

		case *dst.ValueSpec:
			var updatedNames []*dst.Ident // ValueSpec.Names for the GenDecl replacement
			var updatedValues []dst.Expr  // ValueSpec.Values for the GenDecl replacement
			var namePruneLen int          // ValueSpec.Names elements pruned

			valuesLen := len(s.Values)

			for n := range s.Names {
				var globalIdStr string

				if s.Names[n].Name == "_" {
					globalIdObj := cage_pkgs.NewGlobalId(
						fileInspectInfo.PkgPath,
						fileInspectInfo.PkgName,
						fileInspectInfo.Filename,
						cage_pkgs.NewBlankId(fileInspectInfo.Filename, *blankIdFilePos, blankIdGenDeclPos).String(),
					)

					if audit.IsDepGlobalUsedInLocal(globalIdObj) {
						globalIdStr = globalIdObj.String()
					}

					*blankIdFilePos++
					blankIdGenDeclPos++
				} else {
					globalIdStr = usedGlobalIdNames[s.Names[n].Name]
				}

				if globalIdStr == "" { // global ID not found, so the ValueSpec.Names element will be pruned
					namePruneLen++
				} else {
					updatedNames = append(updatedNames, s.Names[n])
					if valuesLen > 0 && n < valuesLen { // e.g. valuesLen will be 0 for "var x, y, z int"
						updatedValues = append(updatedValues, s.Values[n])
					}
				}
			}

			if len(updatedNames) == 0 { // no ValueSpec.Names elements retained, whole ValueSpec will be pruned
				specRemoveLen++
			} else {
				// Add the retained global declarations to the final specs used in the GenDecl replacement
				s.Names = updatedNames
				s.Values = updatedValues
				updatedSpecs = append(updatedSpecs, s)
			}
		}
	}

	if specRemoveLen == 0 { // all GenDecl.Specs elements retained
		return []string{}, nil
	}

	if len(updatedSpecs) == 0 { // all GenDecl.Specs pruned
		cursor.Delete()
	} else {
		cage_dst.SpecsTrimEmptyLine(updatedSpecs)
		genDecl.Specs = updatedSpecs
		cursor.Replace(genDecl)
	}

	return prunedGlobalIds, nil
}
