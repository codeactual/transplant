// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/pkg/errors"

	cage_token "github.com/codeactual/transplant/internal/cage/go/token"
	cage_types "github.com/codeactual/transplant/internal/cage/go/types"
)

// IdentInfo holds additional information about an ast.Ident node loaded by an Inspector.
type IdentInfo struct {
	// Name is a copy of ast.Ident.Name.
	Name string

	// PkgName is the declared name of the package which contains  he node.
	PkgName string

	// PkgPath is the path of the package which contains the node.
	PkgPath string

	// Position provides source-file location details about the node.
	Position token.Position

	// IsTypeDecl is true if the node names a new type.
	IsTypeDecl bool

	// GlobalRef belongs to the global ast.Ident to which subject node ast.Ident refers.
	GlobalRef *IdentInfo

	// Type provides the node's direct type dependencies on inspected packages.
	//
	// The IdentInfo.Types pointers provide type chains to determine all underlying types which
	// were also loaded by the same Inspector instance.
	//
	// Given "const C T = V", C's IdentInfo.Types would have one element because T is the
	// only direct/initial link in C's type chain.
	//
	// Given "type T1 func(T2, T3)", T2's IdentInfo.Types would have two elements because
	// T2 and T3 are both direct/initial links.
	//
	// If the node is just a use/reference of a type, rather than the declaration, then Type
	// will include the declaration.
	//
	// If the node is a struct field/method, then Type will include the struct type in addition
	// to the types directly used by the individual field/method.
	//
	// TypesString can assist with printing the type chains.
	Types []*IdentInfo
}

// NewIndentInfo provides location and type information about ast.Ident nodes in inspected packages
// which declare, or refer to, globals of inspected packages.
//
// IdentInfo will be nil for struct field names in type/struct-literal declarations,
// but non-nil for inspected type identifiers used in the declarations.
// IdentInfo will be nil for function/method parameter names, but non-nil for the type identifiers.
// IdentInfo will be nil for function/method return value names, but non-nil for the type identifiers.
//
// IdentInfo.Types will be non-nil when the node is a const/type/var and the type was declared
// in an inspected package.
//
// IdentInfo.GlobalRef will be nil in all type chains.
func NewIdentInfo(i *Inspector, ident *ast.Ident) (*IdentInfo, error) {
	// seenQuery prevents queryAllPkgs recursion cycles.
	seenQuery := make(map[*ast.Ident]*IdentInfo)

	// queryAllPkgs searches all inspected packages for the identifier.
	//
	// It performs recursive calls, so depth is tracked for debugging.
	var queryAllPkgs func(curIdent *ast.Ident, depth int) (*ast.Ident, *IdentInfo, error)

	queryAllPkgs = func(curIdent *ast.Ident, depth int) (globalRefName *ast.Ident, foundInfo *IdentInfo, err error) {
		if seenResult, ok := seenQuery[curIdent]; ok {
			return nil, seenResult, nil
		}

		nilIdentInfoResult := func() *IdentInfo {
			seenQuery[curIdent] = nil
			return nil
		}

		// Populate the basic IdentInfo fields.

		curIdentPkg, curIdentFileAst, _ := i.FindAstNode(curIdent)
		if curIdentPkg == nil {
			return nil, nilIdentInfoResult(), nil // e.g. built-in or from non-inspected package
		}

		typesObj := curIdentPkg.IdentTypesObj(curIdent)
		if typesObj == nil {
			return nil, nilIdentInfoResult(), nil // e.g. built-in or from non-inspected package
		}

		isGlobalRef := cage_types.IsObjectGlobalRef(typesObj)
		curIdInfo := IdentInfo{
			Name:     curIdent.Name,
			PkgName:  curIdentPkg.Name,
			PkgPath:  curIdentPkg.PkgPath,
			Position: i.FileSet.Position(curIdent.Pos()),
		}

		// Example breakpoint conditions which were useful in the past:
		//
		// - depth == 0 && curIdInfo.Name == ".." && curIdInfo.PkgName == ".." && curIdInfo.Position.Line == N

		var declPkgPath string

		// <import>.<type>.<curIdent> OR <import>.<curIdent>
		if cached := i.IdentContext(curIdentFileAst); cached != nil {
			declPkgPath = cached.ImportQuals[curIdent]
		}

		// Avoid searches for which there is insufficient information to continue.
		if !isGlobalRef && declPkgPath == "" {
			return nil, nilIdentInfoResult(), nil
		}

		if declPkgPath == "" {
			declPkgPath = curIdentPkg.PkgPath
		}

		// Use FindGlobalRef to determine the direct type dependencies.

		var globalRef *GlobalRef
		var globalRefErr error

		// The initial query looks for the declaration which curIdent is or refers to, starting with the current
		// package or one resolved a curIdent qualifier.
		if globalRef, globalRefErr = i.FindGlobalRef(curIdentPkg, curIdentFileAst, curIdent, declPkgPath); globalRefErr != nil {
			return nil, nil, errors.WithStack(globalRefErr)
		}

		// Search dot-imported packages (if any).
		if globalRef == nil {
			if importedPkg := i.SearchDotImportedGlobals(curIdentFileAst, curIdent.Name); importedPkg != nil {
				if globalRef, globalRefErr = i.FindGlobalRef(curIdentPkg, curIdentFileAst, curIdent, importedPkg.PkgPath); globalRefErr != nil {
					return nil, nil, errors.WithStack(globalRefErr)
				}
			}
		}

		if globalRef == nil {
			return nil, nilIdentInfoResult(), nil
		}

		curIdInfo.IsTypeDecl = globalRef.IsType

		if len(globalRef.Types) == 0 {
			return globalRef.Name, &curIdInfo, nil
		}

		// Convert the FindGlobalRef ast.Ident results to IdentInfo values, for the IdentInfo.Types slice,
		// with a recursive call.

		seenTypes := make(map[string]struct{}) // filter duplicates, keys are from IdentInfo.Id

		// Recursively query for the IdentInfo of each type dependency found by FindGlobalRef.
		for _, typeDep := range globalRef.Types {
			if typeDep == curIdent {
				continue
			}

			_, typeDepInfo, err := queryAllPkgs(typeDep, depth+1)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}

			if typeDepInfo == nil {
				continue
			}

			typeKey := typeDepInfo.Id()
			if _, ok := seenTypes[typeKey]; !ok {
				curIdInfo.Types = append(curIdInfo.Types, typeDepInfo)
				seenTypes[typeKey] = struct{}{}
			}
		}

		seenQuery[curIdent] = &curIdInfo

		return globalRef.Name, seenQuery[curIdent], nil
	}

	globalRef, idInfo, err := queryAllPkgs(ident, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if idInfo == nil {
		return nil, nil
	}

	// Finally, store IdentInfo about the global which the subject ast.Ident refers to
	// (which will be itself if it's a declaration).

	if globalRef != nil {
		// Collect the IdentInfo of the ast.Ident to which queryIdent refers.
		if globalRef == ident {
			idInfo.GlobalRef = idInfo
		} else {
			_, globalRef, globalRefErr := queryAllPkgs(globalRef, 0)
			if globalRefErr != nil {
				return nil, errors.WithStack(globalRefErr)
			}

			idInfo.GlobalRef = globalRef
		}
	}

	return idInfo, nil
}

// TypesString returns a tree-like string with all direct/transitive dependencies listed one per line.
//
// Each line is prefixed with a number of the input indentation string equal to the current recursion level.
func (i *IdentInfo) TypesString(indent string) string {
	var bld strings.Builder

	i.InspectTypes(func(info *IdentInfo, depth int) {
		bld.WriteString(strings.Repeat(indent, depth))
		bld.WriteString(info.PkgPath)
		bld.WriteString(".")
		bld.WriteString(info.Name)
		bld.WriteString(" ")
		bld.WriteString(cage_token.ShortPositionString(info.Position))
		bld.WriteString("\n")
	})

	return bld.String()
}

// TypesSet returns the node's unique direct/transitive type dependencies.
func (i *IdentInfo) TypesSet() (s []*IdentInfo) {
	seen := make(map[string]struct{})
	i.InspectTypes(func(info *IdentInfo, depth int) {
		key := info.Id()
		if _, ok := seen[key]; !ok {
			s = append(s, info)
			seen[key] = struct{}{}
		}
	})
	return s
}

// InspectTypes traverses the node's type dependencies in depth-first order.
func (i *IdentInfo) InspectTypes(fn func(info *IdentInfo, depth int)) {
	var walkTypeChain func(info *IdentInfo, depth int)
	walkTypeChain = func(info *IdentInfo, depth int) {
		for _, info := range info.Types {
			fn(info, depth)
			walkTypeChain(info, depth+1)
		}
	}
	walkTypeChain(i, 0)
}

func (i *IdentInfo) Id() string {
	return fmt.Sprintf(
		"%s.%s @ %s",
		i.PkgPath, i.Name, i.Position,
	)
}
