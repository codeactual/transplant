// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	// BlankIdNamePrefix is used to build names for blank identifiers.
	//
	// Format: "<BlankIdNamePrefix><per-file zero-indexed pos #><GlobalIdSeparator><per-GenDecl zero-indexed pos #><GlobalIdSeparator><file absolute path>"
	//
	// It supports cases such as indexing the types used by interface assertions.
	BlankIdNamePrefix = "_"
)

// BlankId describes the location of a blank identifier ("_").
type BlankId struct {
	// Filename identifies the source file of the node.
	Filename string

	// FilePos uniquely identifes the node within its file.
	//
	// It is 0-indexed and based on a counter of the occurence of blank identifiers within the file.
	FilePos int

	// GenDeclPos uniquely identifes the node within its parent ast.GenDecl node.
	//
	// It is 0-indexed and based on a counter of the occurence of blank identifiers within the ast.GenDecl.
	GenDeclPos int
}

func NewBlankId(filename string, filePos, genDeclPos int) *BlankId {
	return &BlankId{
		Filename:   filename,
		FilePos:    filePos,
		GenDeclPos: genDeclPos,
	}
}

func (i *BlankId) String() string {
	return BlankIdNamePrefix + strconv.Itoa(i.FilePos) +
		GlobalIdSeparator +
		strconv.Itoa(i.GenDeclPos) +
		GlobalIdSeparator +
		i.Filename
}

func NewBlankIdFromString(s string) (id *BlankId, err error) {
	if !strings.HasPrefix(s, BlankIdNamePrefix) {
		return nil, errors.Errorf("malformed blank ID [%s]: missing [%s] prefix", s, BlankIdNamePrefix)
	}
	parts := strings.Split(s[len(BlankIdNamePrefix):], GlobalIdSeparator)
	if len(parts) < 3 {
		return nil, errors.Errorf("malformed blank ID [%s]: expected 3 [%s]-separated parts", s, GlobalIdSeparator)
	}

	filePos, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"malformed blankID [%s]: failed to parse first/FilePos [%s]-separated part [%s] to int",
			s, parts[0], GlobalIdSeparator,
		)
	}

	genDeclPos, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"malformed blankID [%s]: failed to parse second/GenDeclPos [%s]-separated part [%s] to int",
			s, parts[1], GlobalIdSeparator,
		)
	}

	return NewBlankId(parts[2], filePos, genDeclPos), nil
}
