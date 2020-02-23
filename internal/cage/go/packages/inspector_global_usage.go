// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/pkg/errors"
)

// GlobalInAstNode describes a global found during an ast.Inspect of an ast.Node.
type GlobalInAstNode struct {
	// Ast is the identifier of the global's name.
	Ast ast.Node

	// BlankIdAssignPos is the position of the global in a blank identifier assignment, if applicable.
	BlankIdAssignPos AssignPos
}

// GlobalIdsUsedByGlobal returns the set of global identifiers used by the target global node.
//
// The set is indexed by GlobalId.String() values.
func (i *Inspector) GlobalIdsUsedByGlobal(dir, pkgName, idName GlobalIdName) (usedMap map[string]IdUsedByNode, errs []error) {
	usedMap = make(map[string]IdUsedByNode)

	errs = i.WalkGlobalIdsUsedByGlobal(dir, pkgName, idName, func(used IdUsedByNode) {
		globalId := NewGlobalId("", used.IdentInfo.PkgName, used.IdentInfo.Position.Filename, used.Name)
		usedMap[globalId.String()] = used
	})

	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return nil, errs
	}

	return usedMap, []error{}
}

// WalkGlobalIdsUsedByGlobal provides walkFn with every global identifier used in the target global node.
func (i *Inspector) WalkGlobalIdsUsedByGlobal(dir, pkgName, idName GlobalIdName, walkFn IdUsedByNodeWalkFunc) (errs []error) {

	node, err := i.GlobalIdNode(dir, pkgName, idName)
	if err != nil {
		return []error{errors.WithStack(err)}
	}

	return i.walkIdsUsedByNode(dir, pkgName, node.Ast, idName, func(used IdUsedByNode) {
		isPkgLocalId := used.IdentInfo.PkgPath == node.InspectInfo.PkgPath && i.GlobalIdNodes[dir][pkgName].Contains(used.Name)
		if isPkgLocalId {
			// If a package-local identifier has the same name as the input value, assume it is the inspection root
			// and continue. Shadowing of globals is not supported by Inspector in order to allow use of this type of
			// assumption.
			if idName != used.Name {
				walkFn(used)
			}
		} else {
			// verify used.Name exists in this non-local package
			if _, ok := i.ImportPathGlobalIdNodes[used.IdentInfo.PkgPath]; !ok {
				return
			}
			if !i.ImportPathGlobalIdNodes[used.IdentInfo.PkgPath].Contains(used.Name) {
				return
			}

			walkFn(used)
		}
	})
}

// WalkIdsUsedByGlobal provides walkFn with every identifier used in the target node.
func (i *Inspector) WalkIdsUsedByGlobal(dir, pkgName, idName GlobalIdName, walkFn IdUsedByNodeWalkFunc) (errs []error) {
	node, err := i.GlobalIdNode(dir, pkgName, idName)
	if err != nil {
		return []error{errors.WithStack(err)}
	}
	return i.walkIdsUsedByNode(dir, pkgName, node.Ast, idName, walkFn)
}

// walkIdsUsedByNode provides walkFn with every identifier used in the target node.
//
// If the input global name is empty, all internal nodes will be inspected. Otherwise only the identified
// internal node will be inspected, e.g. when the input node is an ast.GenDecl which can contain multiple
// const/var/type declarations.
func (i *Inspector) walkIdsUsedByNode(dir, pkgName string, node ast.Node, idName GlobalIdName, walkFn IdUsedByNodeWalkFunc) (errs []error) {
	var queue []GlobalInAstNode

	if idName == "" {
		queue = append(queue, GlobalInAstNode{Ast: node})
	} else {
		// In the ideal case, we would simply walk the node with ast.Inspect and look for any ast.Ident
		// which matches idName.
		//
		// The problem is when the node is of type ast.GenDecl which can contain the declaration of
		// multiple globals (e.g. `const (...)`), even though we only want to walk one of them (idName).
		//
		// In such cases we need to prevent ast.Inspect from passing walkFn every global used in the
		// declaration of each identifier in the group. We perform this filtering by calling ast.Inspect
		// only on manually selected sub-nodes.
		var findErr error
		queue, findErr = i.FindGlobalInAstNode(node, idName)
		if findErr != nil {
			return []error{errors.WithStack(findErr)}
		}
	}

	for len(queue) > 0 {
		var item GlobalInAstNode

		item, queue = queue[0], queue[1:]
		used, findErrs := i.findIdsUsedByNode(item.Ast)

		if len(findErrs) > 0 {
			for _, err := range findErrs {
				errs = append(errs, errors.WithStack(err))
			}
			break
		}

		for n := range used {
			used[n].BlankIdAssignPos = item.BlankIdAssignPos
			walkFn(used[n])
		}
	}

	return errs
}

// findIdsUsedByNode returns the identifiers used in the target node.
func (i *Inspector) findIdsUsedByNode(node ast.Node) (used []IdUsedByNode, errs []error) {
	pos := node.Pos()

	hit, ok := i.findIdsUsedByNodeCache[pos]
	if ok {
		return hit, []error{}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		switch ident := n.(type) {
		case *ast.Ident:
			// Do not consider receivers/parameters, which shadow a global, to count as usage of the global.
			_, shadowRecvOrParam := i.FuncDeclShadows[ident]
			if shadowRecvOrParam {
				break
			}

			identInfo, identInfoErr := NewIdentInfo(i, ident)
			if identInfoErr != nil {
				errs = append(errs, errors.Wrapf(identInfoErr, "failed to get type info of node: %s", i.NodeToString(ident)))
				break
			}

			if identInfo == nil {
				break
			}

			// Register the global to which the current ast.Ident refers, rather than the latter itself.
			// (However when the current ast.Ident is itself a declaration, the two will be the same.)
			if identInfo.GlobalRef != nil {
				used = append(used, IdUsedByNode{
					IdentInfo: identInfo.GlobalRef,
					Name:      identInfo.GlobalRef.Name,
				})

				// Register the type dependencies.
				for _, usedType := range identInfo.GlobalRef.UniqueTypes() {
					used = append(used, IdUsedByNode{
						IdentInfo: usedType,
						Name:      usedType.Name,
					})
				}
			}
		}

		return true
	})

	i.findIdsUsedByNodeCache[pos] = used

	return i.findIdsUsedByNodeCache[pos], []error{}
}

// PackagesUsedByNode returns the set of packages, indexed by import name, used in the input nodes.
func (i *Inspector) PackagesUsedByNode(dir, pkgName string, nodes ...ast.Node) (pkgsByName map[string]PackageUsedByNode, pkgsByPath map[string]PackageUsedByNode, errs []error) {

	pkgsByName = make(map[string]PackageUsedByNode)
	pkgsByPath = make(map[string]PackageUsedByNode)

	walkFn := func(used PackageUsedByNode) {
		if _, ok := pkgsByName[used.Name]; !ok {
			pkgsByName[used.Name] = PackageUsedByNode{Name: used.Name, Path: used.Path}
		}

		if _, ok := pkgsByPath[used.Path]; !ok {
			pkgsByPath[used.Path] = PackageUsedByNode{Name: used.Name, Path: used.Path}
		}
	}

	for _, node := range nodes {
		if walkErrs := i.walkPackagesUsedByNode(dir, pkgName, node, "", walkFn); len(walkErrs) > 0 {
			for _, walkErr := range walkErrs {
				errs = append(errs, errors.WithStack(walkErr))
			}
		}
	}

	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return nil, nil, errs
	}

	return pkgsByName, pkgsByPath, []error{}
}

// walkPackagesUsedByNode provides walkFn with every package used in the target node (except in import statements).
func (i *Inspector) walkPackagesUsedByNode(dir, pkgName string, node ast.Node, idName GlobalIdName, walkFn PackageUsedByNodeWalkFunc) (errs []error) {
	var queue []GlobalInAstNode

	if idName == "" {
		queue = append(queue, GlobalInAstNode{Ast: node})
	} else {
		// In the ideal case, we would simply walk the node with ast.Inspect and look for any ast.Ident
		// which matches idName.
		//
		// The problem is when the node is of type ast.GenDecl which can contain the declaration of
		// multiple globals (e.g. `const (...)`), even though we only want to walk one of them (idName).
		//
		// In such cases we need to prevent ast.Inspect from passing walkFn every global used in the
		// declaration of each identifier in the group. We perform this filtering by calling ast.Inspect
		// only on manually selected sub-nodes.
		var findErr error
		queue, findErr = i.FindGlobalInAstNode(node, idName)
		if findErr != nil {
			return []error{errors.WithStack(findErr)}
		}
	}

	for len(queue) > 0 {
		var item GlobalInAstNode

		item, queue = queue[0], queue[1:]
		used, findErrs := i.findPackagesUsedByGlobal(item.Ast)

		if len(findErrs) > 0 {
			for _, err := range findErrs {
				errs = append(errs, errors.WithStack(err))
			}
			break
		}

		for _, u := range used {
			walkFn(u)
		}
	}

	return errs
}

// findPackagesUsedByGlobal returns the inspected packages used in the target node (except in import statements).
func (i *Inspector) findPackagesUsedByGlobal(node ast.Node) (used []PackageUsedByNode, errs []error) {
	// importSpecEndPos allows the collection to skip import names found in import statements.
	var importSpecEndPos token.Pos

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false

		}
		if importSpecEndPos > 0 && n.Pos() > importSpecEndPos { // exited an ast.ImportSpec
			importSpecEndPos = 0
		}

		switch nodeType := n.(type) {

		case *ast.ImportSpec:
			importSpecEndPos = n.End()

		case *ast.Ident:
			if n == nil {
				break
			}

			if importSpecEndPos > 0 { // skip import names from import statements
				break
			}

			identPkg, identFile, _ := i.FindAstNode(nodeType)
			if identPkg == nil {
				break
			}

			typesObj, _, _ := i.IdentObjectOf(identPkg.PkgPath, identFile, nodeType)
			if typesObj == nil {
				break
			}

			switch pkgNameObj := typesObj.(type) {
			case *types.PkgName:
				used = append(used, PackageUsedByNode{
					Path: pkgNameObj.Imported().Path(),
					Name: pkgNameObj.Imported().Name(),
				})
			}

		} // nodeType

		return true
	})

	if len(errs) > 0 {
		return []PackageUsedByNode{}, errs
	}

	return used, errs
}

// FuncDeclTypeNames returns each type name ast.Ident found in the declaration which refers to a global
// of an inspected package.
func (i *Inspector) FuncDeclTypeNames(n *ast.FuncDecl) []*ast.Ident {
	var queryIdents []*ast.Ident

	// method receiver type

	if n.Recv != nil {
		for _, recv := range n.Recv.List {
			queryIdents = append(queryIdents, i.GlobalRefsInNode(recv.Type)...)
		}
	}

	// function/method parameter types

	if n.Type != nil {
		if n.Type.Params != nil {
			for _, param := range n.Type.Params.List {
				queryIdents = append(queryIdents, i.GlobalRefsInNode(param.Type)...)
			}
		}

		// function/method return types

		if n.Type.Results != nil {
			for _, param := range n.Type.Results.List {
				queryIdents = append(queryIdents, i.GlobalRefsInNode(param.Type)...)
			}
		}
	}

	return queryIdents
}

// GenDeclSpecTypeNames returns each type name ast.Ident found in the declaration which refers to a global
// of an inspected package.
func (i *Inspector) GenDeclSpecTypeNames(t ast.Expr) []*ast.Ident {
	return i.GlobalRefsInNode(t)
}
