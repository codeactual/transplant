// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package types

import (
	std_types "go/types"
	"strings"
)

// IsObjectGlobalRef returns true if the object refers to a name in the global scope.
//
// For example, a file may declare a global constant "X" and inside a function the constant
// is assigned to a new local variable. In that assignment, globalRef would be true because
// when "X" is used inside the function, it refers to the global. However if a local constant
// "X" had been declared above the assignment, globalRef would be false because it refers
// to a local-scope name.
//
// We compute this by checking if the object's scope is the same as the package's.
func IsObjectGlobalRef(obj std_types.Object) bool {
	return obj != nil && obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope()
}

// ParseTypeString parses "simple," single-type go/types.Type.String() values in the form
// "<import path>.<type name>".
func ParseTypeString(s string) (typePkgPath, typeName string) {
	if strings.Contains(s, "func(") {
		return "", ""
	}

	sepIdx := strings.LastIndex(s, ".")
	if sepIdx == -1 {
		return "", ""
	}

	return s[:sepIdx], s[sepIdx+1:]
}

// ParseTypeObject parses a go/types.Type node for its package/type details.
func ParseTypeObject(t std_types.Object) (pkgPath, typeName string) {
	var pkg *std_types.Package

	// Attempt all cases where a go/types.Package is available and relevant to identifying
	// the origin of non-builtin types.
	switch objType := t.(type) {
	case *std_types.PkgName:
		pkg = objType.Pkg()
	case *std_types.Func:
		pkg = objType.Pkg()
	case *std_types.Var:
		pkg = objType.Pkg()
	case *std_types.TypeName:
		pkg = objType.Pkg()
		if objType.IsAlias() {
			pkgPath, typeName := ParseTypeString(objType.Id())
			if pkgPath == "" {
				pkgPath = pkg.Path()
				typeName = objType.Name()
			}

			return pkgPath, typeName
		}
	}

	if pkg == nil {
		return "", ""
	}

	return pkg.Path(), t.Name()
}
