// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package trace

import (
	"fmt"

	"github.com/go-stack/stack"
)

// ThisFile returns the absolute path of the caller's file.
func ThisFile() string {
	c := stack.Caller(1)
	return fmt.Sprintf("%#s", c)
}
