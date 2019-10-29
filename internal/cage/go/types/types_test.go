// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cage_types "github.com/codeactual/transplant/internal/cage/go/types"
)

func TestParseTypeString(t *testing.T) {
	expects := []struct {
		s    string
		path string
		name string
	}{
		{s: "", path: "", name: ""},
		{s: "invalid", path: "", name: ""},
		{s: "path.type", path: "path", name: "type"},
		{s: "domain.tld/pkg.type", path: "domain.tld/pkg", name: "type"},
		{s: "sub.domain.tld/path/to/pkg.type", path: "sub.domain.tld/path/to/pkg", name: "type"},
	}

	for _, expect := range expects {
		actualPath, actualName := cage_types.ParseTypeString(expect.s)
		require.Exactly(t, expect.path, actualPath, expect.s)
		require.Exactly(t, expect.name, actualName, expect.s)
	}
}
