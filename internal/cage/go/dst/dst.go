// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package dst

import "github.com/dave/dst"

// SpecsTrimEmptyLine removes extra line spacing from each spec.
//
// It covers cases go/format.Source neglects, such as:
//
// import (
//
//   // format.Source retains the empty line above
//   "sort"
// )
func SpecsTrimEmptyLine(specs []dst.Spec) {
	specsLen := len(specs)

	trim := func(d *dst.NodeDecs, idx int) {
		if idx == 0 && d.Before == dst.EmptyLine { // empty line above
			d.Before = dst.NewLine
		}
		if idx == specsLen-1 && d.After == dst.EmptyLine { // empty line below
			d.After = dst.NewLine
		}
	}

	for n, spec := range specs {
		switch s := spec.(type) {
		case *dst.ImportSpec:
			trim(&s.Decs.NodeDecs, n)
			specs[n] = s
		case *dst.TypeSpec:
			trim(&s.Decs.NodeDecs, n)
			specs[n] = s
		case *dst.ValueSpec:
			trim(&s.Decs.NodeDecs, n)
			specs[n] = s
		}
	}
}
