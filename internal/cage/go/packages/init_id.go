// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import "strings"

const (
	// InitIdNamePrefix is used to build identifier names for init functions.
	//
	// Format: "<InitIdNamePrefix><GlobalIdSeparator><file absolute path>"
	//
	// It supports cases such as indexing the nodes used by a given init function.
	InitIdNamePrefix = "init"
)

func NewInitIdName(filename string) string {
	return InitIdNamePrefix + GlobalIdSeparator + filename
}

func ParseInitIdName(name string) (filename string) {
	if !strings.HasPrefix(name, InitIdNamePrefix+GlobalIdSeparator) {
		return ""
	}
	return name[len(InitIdNamePrefix+GlobalIdSeparator):]
}
