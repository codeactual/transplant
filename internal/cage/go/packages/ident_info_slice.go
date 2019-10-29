// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

type IdentInfoSlice []*IdentInfo

func (t *IdentInfoSlice) Swap(i, j int) {
	tmp := (*t)[i]
	(*t)[i] = (*t)[j]
	(*t)[j] = tmp
}

func (t *IdentInfoSlice) Less(i, j int) bool {
	if (*t)[i].PkgPath < (*t)[j].PkgPath {
		return true
	} else if (*t)[i].PkgPath > (*t)[j].PkgPath {
		return false
	}
	return (*t)[i].Name < (*t)[j].Name
}

func (t *IdentInfoSlice) Len() int {
	return len(*t)
}
