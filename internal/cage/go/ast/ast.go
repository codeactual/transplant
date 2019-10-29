// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package ast

import (
	"bytes"
	"fmt"
	std_ast "go/ast"
	"go/printer"
	"go/token"
	"path"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

func nodeToString(n std_ast.Node) string {
	var s string
	var buf bytes.Buffer

	if err := printer.Fprint(&buf, token.NewFileSet(), n); err == nil {
		s = buf.String()
	} else {
		s = spew.Sdump(n)
	}

	return strings.TrimSpace(s)
}

func FileSetNodeToString(fset *token.FileSet, n std_ast.Node) string {
	s := nodeToString(n)

	if strings.Count(s, "\n") > 0 {
		return fmt.Sprintf("\n------\n%s\n%s\n------", fset.Position(n.Pos()), s)
	} else {
		return fmt.Sprintf("%s: %s", fset.Position(n.Pos()), s)
	}
}

// IsGlobalDeclName returns true if a const/function/type/var's name matches the input.
func IsGlobalDeclName(f *std_ast.File, name string) bool {
	for _, decl := range f.Decls {
		switch declType := decl.(type) {

		case *std_ast.FuncDecl:
			if declType.Name.Name == name {
				return true
			}

		case *std_ast.GenDecl:
			for _, spec := range declType.Specs {
				switch specType := spec.(type) {
				case *std_ast.ValueSpec:
					for _, valueName := range specType.Names {
						if valueName.Name == name {
							return true
						}
					}
				case *std_ast.TypeSpec:
					if specType.Name.Name == name {
						return true
					}
				}
			}

		}
	}

	return false
}

// ImportNameToPath returns the path by a non-dot, non-blank import name.
func ImportNameToPath(f *std_ast.File, target string) string {
	if target != "." && target != "_" {
		for _, i := range f.Imports {
			p := i.Path.Value[1 : len(i.Path.Value)-1]

			var name string
			if i.Name == nil {
				name = path.Base(p)
			} else {
				name = i.Name.Name
			}

			if name == target {
				return p
			}
		}
	}
	return ""
}

func DotImportPaths(f *std_ast.File) (paths []string) {
	for _, i := range f.Imports {
		if i.Name != nil && i.Name.Name == "." {
			paths = append(paths, i.Path.Value[1:len(i.Path.Value)-1])
		}
	}
	return paths
}

// GlobalImportPaths returns all import paths of packages whose globals were imported
// into the file via dot- or non-blank-named import. It returns the path count.
func GlobalImportPaths(f *std_ast.File) (paths []string) {
	paths = DotImportPaths(f)
	for _, i := range f.Imports {
		if i.Name != nil && (i.Name.Name == "_" || i.Name.Name == ".") {
			continue
		}
		paths = append(paths, i.Path.Value[1:len(i.Path.Value)-1])
	}
	return paths
}
