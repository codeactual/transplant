// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package trace

import (
	"os"
	"regexp"
)

var stackTraceFrameRe *regexp.Regexp
var stackTraceOmitRe *regexp.Regexp

// fullStackTrace will include frames like runtime.main if CAGO_FULL_STACK_TRACE is set to 1.
var fullStackTrace bool

func init() {
	stackTraceFrameRe = regexp.MustCompile(`([^\s]+)\s+([^:]+):([^\s]+)`)
	stackTraceOmitRe = regexp.MustCompile(`runtime\.(main|goexit)`)

	if os.Getenv("CAGE_FULL_STACK_TRACE") == "1" {
		fullStackTrace = true
	}
}
