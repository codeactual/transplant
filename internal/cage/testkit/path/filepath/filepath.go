// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package filepath

import (
	"testing"

	"github.com/codeactual/transplant/internal/cage/testkit"
	std_filepath "path/filepath"
)

func Abs(t *testing.T, name string) string {
	abs, err := std_filepath.Abs(name)
	testkit.FatalErrf(t, err, "failed to get absolute path [%s]\n", name)
	return abs
}
