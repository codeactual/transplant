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
	"strings"

	"github.com/pkg/errors"

	cage_ast "github.com/codeactual/transplant/internal/cage/go/ast"
	cage_types "github.com/codeactual/transplant/internal/cage/go/types"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// GlobalRef is the return type of FindGlobalRef.
type GlobalRef struct {
	// GlobalRef is the name from the declaration to which the subject identifier refers,
	// which may be equal to the latter.
	Name *ast.Ident

	// Parent is the ast.GenDecl/ast.FuncDecl which contains the subject identifier.
	Parent ast.Node

	// IsType is true if the subject identifier is the name in type declaration.
	IsType bool

	// Types holds the names from declarations of types on which the subject identifier depends.
	Types []*ast.Ident
}

// GlobalIdNode returns the associated Node.
func (i *Inspector) GlobalIdNode(dir, pkgName, idName GlobalIdName) (Node, error) {
	dirIdNodes, ok := i.GlobalIdNodes[dir]
	if !ok {
		return Node{}, errors.Errorf("inspector did not collect nodes from dir [%s]", dir)
	}

	pkgIdNodes, ok := dirIdNodes[pkgName]
	if !ok {
		return Node{}, errors.Errorf("inspector did not collect nodes from pkg [%s] in dir [%s]", pkgName, dir)
	}

	node, ok := pkgIdNodes[idName]
	if !ok {
		return Node{}, errors.Errorf("inspector did not collect node [%s] from pkg [%s] in dir [%s]", idName, pkgName, dir)
	}

	return node, nil
}

func (i *Inspector) FindGlobalInAstNode(node ast.Node, idName GlobalIdName) (queue []GlobalInAstNode, _ error) {
	switch astNodeType := node.(type) {
	case *ast.GenDecl:
		var queryBlankId *BlankId

		if strings.HasPrefix(idName, BlankIdNamePrefix) {
			var blankIdErr error
			queryBlankId, blankIdErr = NewBlankIdFromString(idName)
			if blankIdErr != nil {
				return []GlobalInAstNode{}, errors.WithStack(blankIdErr)
			}
		}

		var prevNonNilType ast.Expr // to apply to non-first iota-valued const declarations
		var blankIdGenDeclPos int   // to know when we've found the query blank ID in a ast.ValueSpec

		for _, spec := range astNodeType.Specs {
			if len(queue) > 0 { // we've already found a match
				break
			}

			switch s := spec.(type) {

			case *ast.TypeSpec:
				// collect the type itself

				if s.Name.Name == idName {
					queue = append(queue, GlobalInAstNode{Ast: s})
				}

				// if it's an interface, collect its methods

				switch ifaceSpec := s.Type.(type) {
				case *ast.InterfaceType:
					if ifaceSpec.Methods == nil {
						break
					}

					for _, field := range ifaceSpec.Methods.List {
						switch funcType := field.Type.(type) {
						case *ast.FuncType:
							queue = append(queue, GlobalInAstNode{Ast: funcType})
						}
					}
				}

			// var/const

			case *ast.ValueSpec:
				valuesLen := len(s.Values)

				if astNodeType.Tok == token.CONST && valuesLen > 0 && prevNonNilType != nil {
					// if a `const (...)` block contains multiple iota declarations with different types,
					// reset the saved type to prevent the next untyped identifier from using the previous iota's type
					prevNonNilType = nil
				}

				for n, ident := range s.Names {
					isQueryBlankId := queryBlankId != nil && queryBlankId.GenDeclPos == blankIdGenDeclPos

					if ident.Name == idName || isQueryBlankId {
						typeExpr := s.Type

						// const declaration with no type or value: assume type is the previous one used (at iota declaration)
						if astNodeType.Tok == token.CONST && typeExpr == nil && (valuesLen == 0 || s.Values[n] == nil) && prevNonNilType != nil {
							typeExpr = prevNonNilType
						}

						if typeExpr != nil { // nil if an inferred type, e.g. `const C = ""`
							queue = append(queue, GlobalInAstNode{BlankIdAssignPos: LhsAssignUsage, Ast: typeExpr}) // inspect its type
						}

						if valuesLen > 0 && n < valuesLen { // e.g. valuesLen will be 0 for `var x, y, z int`
							queue = append(queue, GlobalInAstNode{BlankIdAssignPos: RhsAssignUsage, Ast: s.Values[n]}) // inspect what's assigned to it
						}
					}

					if astNodeType.Tok == token.CONST && s.Type != nil {
						prevNonNilType = s.Type
					}

					if ident.Name == "_" {
						blankIdGenDeclPos++
					}
				}
			}
		}

	default:
		queue = append(queue, GlobalInAstNode{Ast: node})
	}

	return queue, nil
}

// FindGlobalRef returns an ast.Ident for each direct type dependency of the input identifier.
//
// It also determines whether the identifier is itself the name created in a type declaration.
func (i *Inspector) FindGlobalRef(identPkg *Package, identFile *ast.File, ident *ast.Ident, declPkgPath string) (*GlobalRef, error) {
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
		if queryIsDecl { // effectively IdentTypeDeps.IsType
			declDeps = append(declDeps, identDecl.Name)
		} else {
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
	// FindGlobalRef provides.
	//
	// However when the ast.Ident is only the usage of T1 (inside the T2 declaration), the only
	// direct dependency FindGlobalRef should return is T1's declaration (the transitive "..."
	// dependencies will be still be available in IdentInfo.Types chains). If we did not make
	// this distinction in the latter case, FindGlobalRef would return a mix of direct and
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
		var depTypeObj types.Object

		depTypeObj, depTypePkgPath, depTypeName = i.IdentObjectOf(identPkg.PkgPath, identFile, depIdent)

		depTypeIsCurQuery := depTypeObj != nil &&
			depTypeObj.Pos() == ident.Pos() &&
			depTypePkgPath == declPkgPath &&
			depTypeName == ident.Name

		// Avoid a stack overflow after returning the query ast.Ident and starting a queryAllPkgs->FindGlobalRef cycle.
		if depTypeIsCurQuery {
			continue
		}

		// Add the dependency as a candidate.
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

// GlobalRefsInNode returns all ast.Ident nodes, found in the subject node's AST, which refer
// to another identifier in the global scope.
func (i *Inspector) GlobalRefsInNode(subject ast.Node) (idents []*ast.Ident) {
	ast.Inspect(subject, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		switch ident := n.(type) {
		case *ast.Ident:
			if pkg, _, _ := i.FindAstNode(ident); pkg != nil {
				if typesObj := pkg.IdentTypesObj(ident); typesObj != nil && cage_types.IsObjectGlobalRef(typesObj) {
					idents = append(idents, ident)
				}
			}
		}

		return true
	})

	return idents
}

// FindAstNode returns location details about the package/file which contains the node.
func (i *Inspector) FindAstNode(n ast.Node) (_ *Package, _ *ast.File, filename string) {
	pos := n.Pos()
	if pkg, position := i.FindPos(pos); pkg != nil {
		for _, f := range pkg.Syntax {
			if pkg.FileToName[f] == position.Filename {
				return pkg, f, position.Filename
			}
		}
	}

	return nil, nil, ""
}

// SearchDotImportedGlobals returns the dot-imported package which exports an identifier
// with the input name.
//
// If the identifier is not found in any of the packages, both return values will be nil.
func (i *Inspector) SearchDotImportedGlobals(file *ast.File, identName string) (match *Package) {
	for _, importedPath := range cage_ast.DotImportPaths(file) {
		importedPkg := i.ImportPathToPkg[importedPath]
		if importedPkg == nil { // non-inspected package, e.g. third-party
			continue
		}

		for _, importedAst := range importedPkg.Syntax {
			if cage_ast.IsGlobalDeclName(importedAst, identName) {
				match = importedPkg
				break
			}
		}

		if match != nil {
			break
		}
	}

	return match
}

// FindPkgGlobal returns the ast.FuncDecl or ast.GenDecl which contains the global's declaration
// and an IdentDecl which further describes the latter.
func (i *Inspector) FindPkgGlobal(pkgPath, idName GlobalIdName) (identDecl *IdentDecl) {
	var declPkg *Package

	for p, pkg := range i.ImportPathToPkg {
		if p == pkgPath {
			declPkg = pkg
			break
		}
	}

	if declPkg == nil {
		return nil
	}

	// Favor GlobalIdNodes over a switch/case-heavy ast.File.Decls scan because the latter was
	// already performed to build the former.
	dirIdNodes, ok := i.GlobalIdNodes[declPkg.Dir]
	if !ok {
		return nil
	}

	pkgIdNodes, ok := dirIdNodes[declPkg.Name]
	if !ok {
		return nil
	}

	node, ok := pkgIdNodes[idName]
	if !ok {
		return nil
	}

	switch astNode := node.Ast.(type) {

	// function and method

	case *ast.FuncDecl:
		return NewIdentDecl(astNode, astNode.Name, nil, IdentDeclFunc)

		// const, type, and regular variable

	case *ast.GenDecl:
		// While GlobalIdNodes uses "<custom qualifier>.<ident name>" to identify globals, FindGenDeclIdent queries
		// for the identifier name as it appears in the ast.GenDecl.
		if idNameBase := GlobalIdNameBase(idName); idNameBase != "" {
			if identDecl = i.FindGenDeclIdent(astNode, idNameBase); identDecl != nil {
				return identDecl
			}
		}
	}

	return nil
}

// GlobalImportPaths returns all import paths of packages whose globals were imported
// into the file via dot- or non-blank-named import. It returns the path count.
func (i *Inspector) GlobalImportPaths(f *ast.File) []string {
	if hit, ok := i.globalImportPaths[f]; ok {
		return hit
	}
	i.globalImportPaths[f] = cage_ast.GlobalImportPaths(f)
	return i.globalImportPaths[f]
}

// IdentObjectOf searches for the inspected-type details of an ast.Ident, evaluating its own package/file
// and also the packages imported into file.
func (i *Inspector) IdentObjectOf(pkgPath string, file *ast.File, ident *ast.Ident) (_ types.Object, typePkgPath, typeName string) {
	cage_ast.GlobalImportPaths(file)
	// Limit the scope to packages whose globals (qualified or non-qualified) exist in the file's namespace.
	for _, importedPath := range append(i.GlobalImportPaths(file), pkgPath) {
		pkg := i.ImportPathToPkg[importedPath]
		if pkg == nil {
			continue
		}

		typesObj := pkg.IdentTypesObj(ident)
		if typesObj == nil {
			continue
		}

		// Check for the simplest "<import name>.<type name>" case.
		typePkgPath, typeName := cage_types.ParseTypeString(typesObj.Type().String())
		if typePkgPath != "" {
			return typesObj, typePkgPath, typeName
		}

		// Check for a concrete type, e.g. go/types.Var, which expose package/name methods.
		typePkgPath, typeName = cage_types.ParseTypeObject(typesObj)
		if typePkgPath != "" {
			return typesObj, typePkgPath, typeName
		}
	}

	return nil, "", "" // E.g. expression is a built-in type or literal.
}

// FindGenDeclIdent searches for an ast.Ident declaration with the input name.
//
// If the name is not found, the returned ast.Ident pointer is nil.
func (i *Inspector) FindGenDeclIdent(g *ast.GenDecl, identName string) *IdentDecl {
	// Track the first spec's type to cover scenarios such as a iota-based const() declarations
	// where the type is only specified for the first identifier in the group.
	//
	// In some cases it will always be nil, such as var/const declarations where the type
	// can only be inferred from the value.
	var firstType ast.Expr

	for _, spec := range g.Specs {
		switch s := spec.(type) {

		case *ast.TypeSpec:
			// The ast.Ident is declared in this ast.GenDecl as a new type.
			if s.Name.Name == identName {
				return NewIdentDecl(g, s.Name, s.Type, IdentDeclType)
			}

			switch sType := s.Type.(type) {
			case *ast.StructType:
				for _, field := range sType.Fields.List {
					for _, fieldIdent := range field.Names {
						if fieldIdent.Name == identName {
							return NewIdentDecl(g, fieldIdent, field.Type, IdentDeclStructField)
						}
					}
				}
			}

		// The ast.Ident is declared in this ast.GenDecl as a new value, e.g. var/const.

		case *ast.ValueSpec:
			if s.Type != nil {
				if firstType == nil {
					firstType = s.Type
				}
			}

			for n := range s.Names {
				if s.Names[n].Name == identName {

					if s.Type == nil {
						// Both will be nil in var/const declarations where the type can only be inferred from the value.
						if firstType == nil {
							if n < len(s.Values) {
								return NewIdentDecl(g, s.Names[n], s.Values[n], IdentDeclValue)
							} else {
								return NewIdentDecl(g, s.Names[n], nil, IdentDeclValue)
							}
						}
						return NewIdentDecl(g, s.Names[n], firstType, IdentDeclValue)
					} else {
						return NewIdentDecl(g, s.Names[n], s.Type, IdentDeclValue)
					}
				}
			}
		}
	}

	return nil
}

// IdentContext returns the IdentContext of the input file.
func (i *Inspector) IdentContext(f *ast.File) *IdentContext {
	if hit := i.identContexts[f]; hit != nil {
		return hit
	}
	i.identContexts[f] = NewIdentContext(f)
	return i.identContexts[f]
}

func (i *Inspector) BlankImportsInFile(dir, pkgName, file string) *cage_strings.Set {
	if i.BlankImports[dir] == nil {
		return nil
	}

	if i.BlankImports[dir][pkgName] == nil {
		return nil
	}

	return i.BlankImports[dir][pkgName][file]
}

func (i *Inspector) DotImportsInFile(dir, pkgName, file string) *cage_strings.Set {
	if i.DotImports[dir] == nil {
		return nil
	}

	if i.DotImports[dir][pkgName] == nil {
		return nil
	}

	return i.DotImports[dir][pkgName][file]
}

// FindPos returns location details of the token.Pos.
func (i *Inspector) FindPos(query token.Pos) (*Package, token.Position) {
	if p := i.FileSet.Position(query); p.IsValid() {
		return i.FilePkgs[p.Filename], p
	}
	return nil, token.Position{}
}
