// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

// PackageUsedByNode describes an package found by PackagesUsedByNode
//
// Other details, like the dependent nodes, may be added later as needed.
type PackageUsedByNode struct {
	Name string
	Path string
}

// PackageUsedByNodeWalkFunc receives one PackageUsedByNode per discovered dependency.
type PackageUsedByNodeWalkFunc func(PackageUsedByNode)
