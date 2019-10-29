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

func TestTruncateAppendHint(t *testing.T) {
	t.Run("should empty string", func(t *testing.T) {
		require.Exactly(t, "", cage_strings.TruncateAppendHint("", 3, "..."))
	})

	t.Run("should handle empty hint", func(t *testing.T) {
		require.Exactly(t, "hel", cage_strings.TruncateAppendHint("hello world", 3, ""))
	})

	t.Run("should handle final string shorter than max", func(t *testing.T) {
		require.Exactly(t, "h...", cage_strings.TruncateAppendHint("hello world", 3, "..."))
	})

	t.Run("should handle final string longer than max", func(t *testing.T) {
		require.Exactly(t, "hello...", cage_strings.TruncateAppendHint("hello world", 8, "..."))
	})
}

func TestTruncatePrepend(t *testing.T) {
	t.Run("should empty string", func(t *testing.T) {
		require.Exactly(t, "", cage_strings.TruncatePrependHint("", 3, "..."))
	})

	t.Run("should handle empty hint", func(t *testing.T) {
		require.Exactly(t, "hel", cage_strings.TruncatePrependHint("hello world", 3, ""))
	})

	t.Run("should handle final string shorter than max", func(t *testing.T) {
		require.Exactly(t, "...d", cage_strings.TruncatePrependHint("hello world", 3, "..."))
	})

	t.Run("should handle final string longer than max", func(t *testing.T) {
		require.Exactly(t, "...world", cage_strings.TruncatePrependHint("hello world", 8, "..."))
	})
}

func TestCopy(t *testing.T) {
	t.Run("should return a shallow copy", func(t *testing.T) {
		src := []string{"3", "1", "4", "0", "2"}
		dst := cage_strings.Copy(src)

		require.Exactly(t, []string{"3", "1", "4", "0", "2"}, dst)

		// If dst were simply assigned the value of src, the sort would affect both.
		cage_strings.SortStable(src)
		require.Exactly(t, []string{"3", "1", "4", "0", "2"}, dst)
	})
}

func TestStringMapPtr(t *testing.T) {
	t.Run("should error on invalid key", func(t *testing.T) {
		ptr, save, err := cage_strings.StringKeyPtr(&map[string]string{}, "does not exist")
		require.Error(t, err, "key [does not exist] not found in mapy")

		// cannot use Equal/Exactly due to type mismatch
		require.True(t, ptr == nil)
		require.True(t, save == nil)
	})

	t.Run("should update map", func(t *testing.T) {
		subject := map[string]string{
			"a": "apple",
			"b": "bear",
		}

		ptr, save, err := cage_strings.StringKeyPtr(&subject, "a")
		require.NoError(t, err)
		*ptr = "amber"
		save()
		require.Exactly(
			t,
			map[string]string{"a": "amber", "b": "bear"},
			subject,
		)

		ptr, save, err = cage_strings.StringKeyPtr(&subject, "b")
		require.NoError(t, err)
		*ptr = "baseball"
		save()
		require.Exactly(
			t,
			map[string]string{"a": "amber", "b": "baseball"},
			subject,
		)
	})
}
