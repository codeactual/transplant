// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"go/ast"
)

// IdentDeclKind describes the kind of identifier an ast.Ident represents.
type IdentDeclKind string

const (
	// IdentDeclFunc labels an ast.Ident found in a function declaration.
	IdentDeclFunc IdentDeclKind = "function"

	// IdentDeclValue labels an ast.Ident found in a const/var declaration.
	IdentDeclValue IdentDeclKind = "value"

	// IdentDeclType labels an ast.Ident found in a type declaration.
	IdentDeclType IdentDeclKind = "type"

	// IdentDeclStructField labels an ast.Ident found in a struct field declaration.
	IdentDeclStructField IdentDeclKind = "field"
)

// IdentInfo provides additional information about an ast.Ident node's declaration.
type IdentDecl struct {
	// Name is the subject identifier.
	Name *ast.Ident

	// Kind indicates what was declared: const, function, type, or var.
	Kind IdentDeclKind

	// Parent is the parent ast.GenDecl/ast.FuncDecl of the subject identifier.
	Parent ast.Node

	// SpecType is the type expression explicitly selected syntactically to the right
	// of the identifier, inferred from its const iota group, or inferred from the
	// value assigned to it.
	SpecType ast.Expr
}

// NewIdentDecl creates a new declaration instance and initializes the SpecTypeName
// and SpecTypeQualifier fields from the input (const/type/var) type expressions.
func NewIdentDecl(parent ast.Node, name *ast.Ident, specType ast.Expr, declKind IdentDeclKind) *IdentDecl {
	return &IdentDecl{Parent: parent, Name: name, SpecType: specType, Kind: declKind}
}
