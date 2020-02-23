// Copyright (C) 2020 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package debug

import (
	"fmt"
	std_debug "runtime/debug"
	"strings"
)

// BuildInfoString returns the build information, embedded in the binary,
// as a human-readable string intended for cases like CLI version commands/flags.
func BuildInfoString() string {
	info, ok := std_debug.ReadBuildInfo()
	if !ok {
		return "<build information from runtime/debug not available>"
	}

	var bld strings.Builder

	fmt.Fprintf(&bld, "paths: build=%s main=%s\n", info.Path, info.Main.Path)

	if len(info.Deps) > 0 {
		fmt.Fprintln(&bld, "dependencies:")
		for _, d := range info.Deps {
			fmt.Fprintf(&bld, "\t%s@%s %s\n", d.Path, d.Version, d.Sum)
			if d.Replace != nil {
				fmt.Fprintf(&bld, "\t\treplacement: %s@%s %s\n", d.Replace.Path, d.Replace.Version, d.Replace.Sum)
			}
		}
	}

	return bld.String()
}
