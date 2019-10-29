// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package testkit

import (
	"testing"
)

func FatalErrf(t *testing.T, err error, f string, v ...interface{}) {
	if err != nil {
		f = f + ": %+v"
		v = append(v, err)
		t.Fatalf(f, v...)
	}
}

// RequireNoErrors is an alternative to methods like require.Exactly which will only display the
// "%s" of each unexpected error. This function outputs the "%+v" string to include the stack trace.
func RequireNoErrors(t *testing.T, errs []error) {
	errsLen := len(errs)
	if errsLen == 0 {
		return
	}
	for n, err := range errs {
		t.Errorf("unexpected error %d of %d: %+v", n+1, errsLen, err)
	}
	t.FailNow()
}
