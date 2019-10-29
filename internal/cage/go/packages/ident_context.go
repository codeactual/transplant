// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"go/ast"

	cage_ast "github.com/codeactual/transplant/internal/cage/go/ast"
)

// IdentContext holds contextual details about ast.Ident nodes in the file.
//
// It complement details provided by go/types, such as the package path of
// an ast.Ident's import name qualifier.
type IdentContext struct {
	// ImportQuals holds import paths, indexed by the ast.Ident of the field
	// in a "<import name>.<field>" ast.SelectorExpr.
	ImportQuals map[*ast.Ident]string
}

// NewIdentContext returns contextual details about ast.Ident nodes in the file.
func NewIdentContext(f *ast.File) *IdentContext {
	fi := IdentContext{
		ImportQuals: make(map[*ast.Ident]string),
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		switch nodeType := n.(type) {

		case *ast.SelectorExpr:

			switch seXType := nodeType.X.(type) {

			case *ast.Ident: // <type or import>.<input ident>
				if p := cage_ast.ImportNameToPath(f, seXType.Name); p != "" {
					fi.ImportQuals[nodeType.Sel] = p
				}

			case *ast.CompositeLit: // <type>{}.<input ident>
				switch compType := seXType.Type.(type) {

				case *ast.SelectorExpr: // <import>.<type>{}
					switch importName := compType.X.(type) {

					case *ast.Ident:

						if p := cage_ast.ImportNameToPath(f, importName.Name); p != "" {
							fi.ImportQuals[nodeType.Sel] = p
						}

					} // importName

				} // compType

			} // seXType

		} // nodeType

		return true
	})

	return &fi
}
