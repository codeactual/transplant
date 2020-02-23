// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"go/ast"

	"github.com/pkg/errors"
)

// GlobalRef describes a global node which is referred to by another, identically named node.
//
// Example:
//
//   // ...
//
//   type A struct {
//     b B
//     c C
//   }
//   func main() { _ = A{} }
//
// The GlobalRef for the "A" in main would describe the global it refers to, the "A" from `type struct A`.
// GlobalRef.Types would include the B and C types. IsType would equal true. And Parent would be the
// ast.GenDecl which contains the type declaration in the syntax tree.
//
type GlobalRef struct {
	// Name is from the global's declaration node.
	Name *ast.Ident

	// Parent is the ast.GenDecl/ast.FuncDecl which contains the Name node.
	Parent ast.Node

	// IsType is true if the global defines/declares a type.
	IsType bool

	// Types holds the declaration-name nodes of inspected types on which the global directly depends.
	//
	// For example, if the global defines a struct, then it will include the inspected types
	// of the fields.
	Types []*ast.Ident
}

// ResolveGlobalRef returns details about the global to which the input ast.Ident refers.
func (i *Inspector) ResolveGlobalRef(identPkg *Package, identFile *ast.File, ident *ast.Ident, declPkgPath string) (*GlobalRef, error) {
	if hit := i.globalRefs[ident]; hit != nil {
		return hit, nil
	}

	// Find the ast.FuncDecl or ast.GenDecl node which contains an name ast.Ident matching the input name.

	identDecl := i.FindPkgGlobal(declPkgPath, ident.Name)

	// declPkgPath is not an inspected package or it did not export ident.Name
	if identDecl == nil {
		return nil, nil
	}

	res := GlobalRef{
		Name:   identDecl.Name,
		Parent: identDecl.Parent,
	}

	if identDecl.Kind != IdentDeclFunc && identDecl.SpecType == nil {
		return nil, errors.Errorf(
			"failed to collect ast.GenDecl spec type for ident [%s] in pkg [%s]",
			ident.Name, declPkgPath,
		)
	}

	queryIsDecl := ident.Pos() == identDecl.Name.Pos()
	foundIdentIsType := identDecl.Kind == IdentDeclType
	res.IsType = queryIsDecl && foundIdentIsType

	// In the found ast.FuncDecl/ast.GenDecl, collect the ast.Ident of every type name in
	// the declaration body, e.g. function/method parameter and return types.

	var declDeps []*ast.Ident

	// Add the found declaration ast.Ident to declDeps so we can find out its type info when:
	// it declares a type and is the same as the query ast.Ident, or the ast.Ident declares a const/var.
	//
	// Exclude type name ast.Ident nodes when they differ from the query ast.Ident because identDecl will
	// already provide type info in that case during declDeps iteration,, so we need to avoid
	// discovering duplicate dependencies and adding them to the returned list.
	//
	// Exclude function name identifiers because the only type dependency information we'll find
	// will come from the results of FuncDeclTypeNames in a later step.
	if foundIdentIsType {
		// Only include the type node if it differs from the query node. For example, if `X` is
		// queried from source `type X int`, don't effectively treat `X` as a type dependency of itself.
		if !queryIsDecl {
			// FindPkgGlobal returned a type declaration that was referenced by the query ast.Ident,
			// so it's a direct dependency we can immediately add to the returned list.
			res.Types = append(res.Types, identDecl.Name)
		}
	} else if identDecl.Kind == IdentDeclValue { // exclude function declarations from declDeps
		declDeps = append(declDeps, identDecl.Name)
	}

	// type T1 {
	//   ...
	// }
	// type T2 struct {
	//   F1 T1
	// }
	// func F() {}
	//
	// queryIsDecl:
	//
	// When the query ast.Ident is the declaration of T1 (queryIsDecl == true), we need to find
	// the type dependencies in the "..." code because they're exactly the direct dependencies
	// ResolveGlobalRef provides.
	//
	// However when the ast.Ident is only the usage of T1 (inside the T2 declaration), the only
	// direct dependency ResolveGlobalRef should return is T1's declaration (the transitive "..."
	// dependencies will be still be available in IdentInfo.Types chains). If we did not make
	// this distinction in the latter case, ResolveGlobalRef would return a mix of direct and
	// transitive dependencies (the T1 declaration and dependencies found in "...").
	//
	// identDecl.Kind == IdentDeclFunc:
	//
	// When the query ast.Ident is either the declaration or use of F, we need to find the type
	// dependencies of the declaration.
	//
	if queryIsDecl || identDecl.Kind == IdentDeclFunc {
		switch parentType := identDecl.Parent.(type) {
		case *ast.FuncDecl: // function and method
			for _, funcDep := range i.FuncDeclTypeNames(parentType) {
				declDeps = append(declDeps, funcDep)
			}
		case *ast.GenDecl: // const, type, and regular variable
			for _, specDep := range i.GenDeclSpecTypeNames(identDecl.SpecType) {
				declDeps = append(declDeps, specDep)
			}
		}
	}

	// Collect an IdentInfo for each type dependency of the found ast.Ident declaration.

	// E.g. if a type from an inspected package is used more than once in the declaration
	// body, e.g. as both a parameter and return type in a FuncDecl, only return it once.
	seenDep := make(map[*ast.Ident]struct{})

	type candidate struct {
		Name    string
		PkgPath string
	}
	candidateKey := func(c candidate) string {
		return c.PkgPath + "." + c.Name
	}
	var candidates []candidate
	seenCandidate := make(map[string]struct{})

	// Collect the names and package paths of discovered type dependencies.

	for _, depIdent := range declDeps {
		var depTypeName, depTypePkgPath string
		_, depTypePkgPath, depTypeName = i.IdentObjectOf(identPkg.PkgPath, identFile, depIdent)

		// Add the dependency as a candidate for resolution via FindPkgGlobal below.
		cand := candidate{PkgPath: depTypePkgPath, Name: depTypeName}
		if _, ok := seenCandidate[candidateKey(cand)]; !ok {
			candidates = append(candidates, cand)
			seenCandidate[candidateKey(cand)] = struct{}{}
		}
	}

	// Collect the ast.Ident nodes of the type dependency candidates.

	for _, cand := range candidates {
		if depTypeDecl := i.FindPkgGlobal(cand.PkgPath, cand.Name); depTypeDecl != nil {
			if _, ok := seenDep[depTypeDecl.Name]; !ok && depTypeDecl.Kind == IdentDeclType {
				res.Types = append(res.Types, depTypeDecl.Name)
				seenDep[depTypeDecl.Name] = struct{}{}
			}
		}
	}

	i.globalRefs[ident] = &res

	return i.globalRefs[ident], nil
}
