// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package strings_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

func TestReplaceSet(t *testing.T) {
	t.Run("should apply limit", func(t *testing.T) {
		subject := "filename with spaces"
		set := &cage_strings.ReplaceSet{}

		set.Add(" ", "_", -1)
		require.Exactly(
			t,
			"filename_with_spaces",
			set.InString(subject),
		)

		set.Add(" ", "_", 1)
		require.Exactly(
			t,
			"filename_with spaces",
			set.InString(subject),
		)
	})

	t.Run("should apply to string in length-descending order", func(t *testing.T) {
		subject := "path: github.com/user/project/path/to/pkg"
		set := &cage_strings.ReplaceSet{}

		set.Add("g..to/pkg", "<path>", 1)                      // third
		set.Add("github.com/user/project/path/to", "g..to", 1) // first
		set.Add("g..to", "pkg", 1)                             // second

		require.Exactly(
			t,
			"path: <path>",
			set.InString(subject),
		)
	})

	t.Run("should apply to byte slice in length-descending order", func(t *testing.T) {
		subject := []byte("path: github.com/user/project/path/to/pkg")
		set := &cage_strings.ReplaceSet{}

		set.Add("g..to/pkg", "<path>", 1)                      // third
		set.Add("github.com/user/project/path/to", "g..to", 1) // first
		set.Add("g..to", "pkg", 1)                             // second

		require.Exactly(
			t,
			[]byte("path: <path>"),
			set.InByte(subject),
		)
		require.Exactly(t, subject, []byte("path: github.com/user/project/path/to/pkg")) // unchanged
	})
}
