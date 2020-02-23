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
	"sort"
	"strings"

	"github.com/pkg/errors"

	cage_ast "github.com/codeactual/transplant/internal/cage/go/ast"
	cage_token "github.com/codeactual/transplant/internal/cage/go/token"
	cage_types "github.com/codeactual/transplant/internal/cage/go/types"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// IdentInfo provides details about an ast.Ident node loaded by an Inspector.
//
// If it is located in an IdentInfo.Types slice, then it describes the node which declares a type
// dependency of the subject IdentInfo.
//
// If it is located in an IdentInfo.Types slice, it may also represent the start of a type
// dependency cycle. In that case, a minimal subset of the fields will be populated in addition
// to IsCycle: Name, PkgName, and PkgPath. For example, in `type A struct { next *A }`, the
// Types slice for the declaration node will contain "itself" as a dependency due to the
// field of the same type. But to avoid infinite walks, the IdentInfo which represents
// the start of the cycle will have the traits described earlier and no Types elements
// of its own.
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
	// The IdentInfo.Types elements type chains to determine all underlying types which
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
	Types IdentInfoSlice

	// IsCycle is true if the IdentInfo represents the start of a type dependency cycle.
	IsCycle bool
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
	// typeStack records the dependency path traversed by recursive queryAllPkgs calls
	// in order to detect/prevent cycles.
	typeStack := cage_strings.NewStack()

	// newTypeStackId creates unique IDs for use as typeStack elements.
	newTypeStackId := func(pkgPath string, typeIdent *ast.Ident) string {
		return pkgPath + "." + cage_ast.FileSetNodeToStringShort(i.FileSet, typeIdent)
	}

	// queryAllPkgs searches all inspected packages for the identifier.
	//
	// It performs recursive calls, so depth is tracked for debugging.
	var queryAllPkgs func(curIdent *ast.Ident, depth int) (*GlobalRef, *IdentInfo, error)

	queryAllPkgs = func(curIdent *ast.Ident, depth int) (globalRef *GlobalRef, foundInfo *IdentInfo, err error) {
		// Populate the basic IdentInfo fields.

		curIdentPkg, curIdentFileAst, _ := i.FindAstNode(curIdent)
		if curIdentPkg == nil {
			return nil, nil, nil // e.g. built-in or from non-inspected package
		}

		typesObj := curIdentPkg.IdentTypesObj(curIdent)
		if typesObj == nil {
			return nil, nil, nil // e.g. built-in or from non-inspected package
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
			return nil, nil, nil
		}

		if declPkgPath == "" {
			declPkgPath = curIdentPkg.PkgPath
		}

		// Use ResolveGlobalRef to determine the direct type dependencies.

		var globalRefErr error

		// The initial query looks for the declaration which curIdent is or refers to, starting with the current
		// package or one resolved a curIdent qualifier.
		if globalRef, globalRefErr = i.ResolveGlobalRef(curIdentPkg, curIdentFileAst, curIdent, declPkgPath); globalRefErr != nil {
			return nil, nil, errors.WithStack(globalRefErr)
		}

		// Search dot-imported packages (if any).
		if globalRef == nil {
			if importedPkg := i.SearchDotImportedGlobals(curIdentFileAst, curIdent.Name); importedPkg != nil {
				if globalRef, globalRefErr = i.ResolveGlobalRef(curIdentPkg, curIdentFileAst, curIdent, importedPkg.PkgPath); globalRefErr != nil {
					return nil, nil, errors.WithStack(globalRefErr)
				}
			}
		}

		if globalRef == nil {
			return nil, nil, nil
		}

		curIdInfo.IsTypeDecl = globalRef.IsType

		if len(globalRef.Types) == 0 {
			return globalRef, &curIdInfo, nil
		}

		// Convert the ResolveGlobalRef ast.Ident results to IdentInfo values, for the IdentInfo.Types slice,
		// with a recursive call.

		// Recursively query for the IdentInfo of each type dependency found by ResolveGlobalRef.
		for _, typeDep := range globalRef.Types {
			typeDepPkg, _, _ := i.FindAstNode(typeDep)
			if typeDepPkg == nil {
				continue
			}

			typeDepId := newTypeStackId(typeDepPkg.PkgPath, typeDep)

			// If the type dependency matches an queryAllPkgs input node, past or the current one,
			// then append a IdentInfo representing the start of a dependency cycle and avoid
			// a recursive query into the cycle.

			if typeDep == curIdent || typeStack.Contains(typeDepId) {
				curIdInfo.Types = append(curIdInfo.Types, NewIdentInfoCycle(typeDepPkg.PkgPath, typeDepPkg.Name, typeDep.Name))
				continue
			}

			// Perform a recursive query for the current type dependency, updating the type stack in
			// order to avoid following infinite cycles.

			typeStack.Push(typeDepId)

			_, typeDepInfo, err := queryAllPkgs(typeDep, depth+1)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}

			typeStack.Pop()

			if typeDepInfo == nil {
				continue
			}

			curIdInfo.Types = append(curIdInfo.Types, typeDepInfo)
		}

		return globalRef, &curIdInfo, nil
	}

	identPkg, _, _ := i.FindAstNode(ident)
	if identPkg == nil {
		return nil, nil
	}

	typeStack.Push(newTypeStackId(identPkg.PkgPath, ident))

	globalRef, idInfo, err := queryAllPkgs(ident, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if idInfo == nil {
		return nil, nil
	}

	queryIdentIsDecl := globalRef != nil && globalRef.Name == ident

	// Finally, store IdentInfo about the global which the subject ast.Ident refers to
	// (which will be itself if it's a declaration).

	if globalRef != nil {
		// Collect the IdentInfo of the ast.Ident to which `ident` refers.
		if queryIdentIsDecl {
			idInfo.GlobalRef = idInfo
		} else {
			_, globalRefInfo, globalRefErr := queryAllPkgs(globalRef.Name, 0)
			if globalRefErr != nil {
				return nil, errors.WithStack(globalRefErr)
			}

			idInfo.GlobalRef = globalRefInfo
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
		if info.Position.IsValid() {
			bld.WriteString(" ")
			bld.WriteString(cage_token.ShortPositionString(info.Position))
		}
		if info.IsCycle {
			bld.WriteString(" (cycle)")
		}
		bld.WriteString("\n")
	})

	return bld.String()
}

// UniqueTypes returns the node's unique direct/transitive type dependencies.
func (i *IdentInfo) UniqueTypes() (s IdentInfoSlice) {
	seen := make(map[string]struct{})

	i.InspectTypes(func(info *IdentInfo, depth int) {
		// If a cycle is starting, then the set should already contain the type. However
		// in this case we cannot rely on IdShort below to see it as a duplicate because
		// the position info will be empty (the cycle node only contains the minimum values
		// to represent the type), leading to seen-check to consider the node as unique.
		if info.IsCycle {
			return
		}

		key := info.IdShort()

		if _, ok := seen[key]; !ok {
			s = append(s, info)
			seen[key] = struct{}{}
		}
	})

	return s
}

// InspectTypes traverses the node's type dependencies in depth-first order.
func (i *IdentInfo) InspectTypes(fn func(info *IdentInfo, depth int)) {
	// As the type chain is walked recursively, use a stack to store the current dependency
	// path to prevent cycles.
	typeStack := cage_strings.NewStack()

	var walkTypeChain func(info *IdentInfo, depth int)
	walkTypeChain = func(info *IdentInfo, depth int) {
		sortedTypes := info.Types.Copy()
		sort.Stable(&sortedTypes)

		for _, tInfo := range sortedTypes {
			fn(tInfo, depth)

			if tInfo.IsCycle {
				continue
			}

			typeId := tInfo.IdShort()

			if typeStack.Contains(typeId) {
				continue
			}

			typeStack.Push(typeId)
			walkTypeChain(tInfo, depth+1)
			typeStack.Pop()
		}
	}
	walkTypeChain(i, 0)
}

// Id returns a string which uniquely identifies the ast.Ident in a build.
//
// Use it instead of IdShort if the absolute path to the source file is necessary.
func (i *IdentInfo) Id() string {
	return fmt.Sprintf(
		"%s.%s @ %s",
		i.PkgPath, i.Name, i.Position,
	)
}

// Id returns a string which uniquely identifies the ast.Ident in a build.
//
// Use it instead of Id the absolute path to the source file is unnecessary.
func (i *IdentInfo) IdShort() string {
	return fmt.Sprintf(
		"%s.%s @ %s",
		i.PkgPath, i.Name, cage_ast.PositionStringShort(i.Position),
	)
}

// NewIdentInfoCycle creates a value for use in an IdentInfo.Types slice which
// represents the start of a type dependency cycle.
func NewIdentInfoCycle(pkgPath, pkgName, idName string) *IdentInfo {
	return &IdentInfo{
		IsCycle: true,
		Name:    idName,
		PkgName: pkgName,
		PkgPath: pkgPath,
	}
}
