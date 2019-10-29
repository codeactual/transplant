// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import cage_strings "github.com/codeactual/transplant/internal/cage/strings"

type NodeInspectInfo struct {
	Dirname     string
	Filename    string
	GlobalScope bool

	// GlobalFuncName is name of the node's parent function or method.
	GlobalFuncName string

	// GlobalTypeName is name of the node's parent `type` declration.
	//
	// It differs from GlobalMethodTypeName which is only for method body nodes.
	GlobalTypeName string

	// InitFuncPos is -1 if the node is not an init function, otherwise it is the 0-indexed position
	// in the file among the other init functions (if any).
	InitFuncPos int

	PkgName string
	PkgPath string

	// IotaValuedNames holds the names of identifers in an ast.GenDecl which are iota-valued constants.
	IotaValuedNames *cage_strings.Set

	// NonStdImports is a copy of Inspector.NonStdImports[NodeInspectInfo.Dirname][NodeInspectInfo.PkgName].
	//
	// It is indexed by import paths.
	NonStdImports map[string]Import

	// MethodType holds the name of the parent type for all nodes inside a method body.
	//
	// If differs from the GlobalTypeName which is only for nodes in a `type` declaration.
	GlobalMethodTypeName string
}

func NewNodeInspectInfo() NodeInspectInfo {
	return NodeInspectInfo{
		InitFuncPos:     -1,
		IotaValuedNames: cage_strings.NewSet(),
	}
}
