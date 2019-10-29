// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"

	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
)

// GlobalIdName identifies global entities.
//
// Identifier names take these forms:
//
//   <const/var/func ast.Ident.Name>
//   <type name>.<field/method ast.Ident.Name>
//   init.<source file absolute path>
type GlobalIdName = string

type GlobalId struct {
	// Filename is the absolute path to the source file which declared the identifier.
	Filename string

	// PkgName is the package name as declared in the source file which declared the identifier.
	PkgName string

	// PkgPath is the package's import path.
	PkgPath string

	// Name is the global's name.
	Name GlobalIdName
}

func NewGlobalId(pkgPath, pkgName, filename, idName GlobalIdName) GlobalId {
	return GlobalId{
		Filename: filename,
		PkgName:  pkgName,
		PkgPath:  pkgPath,
		Name:     idName,
	}
}

// BaseName returns the right-most section of a Name value, e.g. variable or method name.
//
// It will return an empty string if used on identifiers of init functions and blank identifiers.
func (i GlobalId) BaseName() string {
	parts := strings.Split(i.Name, GlobalIdSeparator)
	partsLen := len(parts)

	if partsLen == 1 {
		return parts[0]
	} else if partsLen >= 2 {
		return parts[partsLen-1]
	}

	return ""
}

func (i GlobalId) MethodType() string {
	parts := strings.Split(i.Name, GlobalIdSeparator)
	switch len(parts) {
	case 1:
		break
	case 2:
		if parts[0] == InitIdNamePrefix {
			break
		}
		return parts[0]
	}
	return ""
}

func (i GlobalId) Dir() string {
	if cage_filepath.IsGoFile(i.Filename) {
		return filepath.Dir(i.Filename)
	}
	return i.Filename // e.g. dir vertex in transplant
}

func (i GlobalId) String() string {
	filename, pkgName, globalIdName := "<empty filename>", "<empty package name>", "<empty global ID>"
	if i.Filename != "" {
		filename = i.Filename
	}
	if i.PkgName != "" {
		pkgName = i.PkgName
	}
	if i.Name != "" {
		globalIdName = i.Name
	}
	return filename + GlobalIdSeparator + pkgName + GlobalIdSeparator + globalIdName
}

// globalIdDeclType describes the AST origin of a globalIdDecl value.
type globalIdDeclType int

// These globalIdDeclType constants reflect the granularity at which they're currently collected.
// Variable/constant/type disambiguation can be added later if needed.
const (
	nonFuncOrMethodDeclType globalIdDeclType = iota
	funcDeclType
	fieldDeclType
	methodDeclType
	methodRecvDeclType
	funcOrMethodParamDeclType
)

func (t globalIdDeclType) String() string {
	switch t {
	case nonFuncOrMethodDeclType:
		return "non-func/method"
	case funcDeclType:
		return "func"
	case fieldDeclType:
		return "field"
	case methodDeclType:
		return "method"
	case methodRecvDeclType:
		return "method recv"
	case funcOrMethodParamDeclType:
		return "func/method param"
	default:
		return fmt.Sprintf("invalid type: %d", t)
	}
}

// globalIdDecl describes a potential global identifier declaration (or shadow declaration).
type globalIdDecl struct {
	// Name is the name of the declared global.
	//
	// If ImportPath is non-empty, Name will be the package name used in the file to reference ImportPath,
	// i.e. it may match the package clause or not match it if "import <custom name> <path>" syntax is used.
	Name GlobalIdName

	// Ident is the identifier's node, e.g. within a larger ast.GenDecl declaration group.
	Ident *ast.Ident

	// StructPropType is the type's identifier name.
	StructPropType string

	// StructPropName is the method's identifier name.
	StructPropName string

	// DeclType is the declaration's AST origin.
	DeclType globalIdDeclType

	// ImportPath is the path of the shadowed package (if applicable).
	ImportPath string
}

func (s globalIdDecl) String() string {
	return fmt.Sprintf(
		"name [%s] field/method (name [%s] type [%s]) type [%s] import [%s]",
		s.Name, s.StructPropName, s.StructPropType, s.DeclType, s.ImportPath,
	)
}

func GlobalIdNameBase(idName GlobalIdName) string {
	sepIdx := strings.LastIndex(idName, GlobalIdSeparator)
	if sepIdx == -1 {
		return idName
	}
	return idName[sepIdx+1:]
}
