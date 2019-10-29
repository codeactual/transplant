// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"path/filepath"
)

// AssignPos describes where an identifier is used in an assignment.
type AssignPos int

const (
	// NonAssignUsage indicates the identifier was not used in an assignment.
	NonAssignUsage AssignPos = iota

	// LhsAssignUsage indicates the identifier was used on the left-hand side of an assignment.
	LhsAssignUsage

	// HhsAssignUsage indicates the identifier was used on the right-hand side of an assignment.
	RhsAssignUsage
)

// IdUsedByNode describes an identifier found by WalkIdsUsedByNode.
type IdUsedByNode struct {
	// BlankIdAssignPos describes where the identifier was used in a blank identifier assignment.
	// If it was not found in such an assignment, the value will be NonAssignUsage.
	//
	// It is only populated by full-node queries. Otherwise the value is always NonAssignUsage.
	BlankIdAssignPos AssignPos

	// Name identifies a global ast.Ident.
	Name GlobalIdName

	// IdentInfo describes the identifier itself, the global identifier to which the former refers
	// (which in declarations will be the same node, and the direct/transitive type dependencies.
	IdentInfo *IdentInfo
}

func (i IdUsedByNode) Dir() string {
	return filepath.Dir(i.IdentInfo.Position.Filename)
}

func (i IdUsedByNode) GlobalId() GlobalId {
	return NewGlobalId(
		i.IdentInfo.PkgPath,
		i.IdentInfo.PkgName,
		i.IdentInfo.Position.Filename,
		i.Name,
	)
}

// IdUsedByNodeWalkFunc defines the input function to WalkIdsUsedByNode.
type IdUsedByNodeWalkFunc func(IdUsedByNode)
