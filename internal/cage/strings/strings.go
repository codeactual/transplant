// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package strings

import (
	"sort"

	"github.com/pkg/errors"
)

// Truncate returns the string with a maximum length enforced and optional
// appended hint (e.g. "...").
//
// The truncate string and hint will fit inside the maximum length if possible.
func TruncateAppendHint(s string, max int, hint string) string {
	return truncate(s, max, hint, true)
}

// Truncate returns the string with a maximum length enforced and optional
// prepended hint (e.g. "...").
//
// The truncate string and hint will fit inside the maximum length if possible.
func TruncatePrependHint(s string, max int, hint string) string {
	return truncate(s, max, hint, false)
}

// Assumptions:
//
// - If there's not enough room for the hint, it's preferrable to show at least one character
//   and the hint to indicate a longer version exists, rather than e.g. just the one character,
//   otherwise there's no reason to use this function over plain truncation.
// - If there's not enough room for the hint, it's okay to exceed the max anyway because
//   the maximum overflow is predictable to the caller since all lengths are known
//   ahead and again, if strict truncation is the priority than a hint-less alternative
//   should be used anyway.
func truncate(s string, max int, hint string, append bool) string {
	sLen := len(s)
	hintLen := len(hint)

	if sLen <= max {
		return s
	}

	if hint == "" {
		return s[:max]
	}

	if max < hintLen+1 {
		if append {
			return s[:1] + hint
		}
		return hint + s[sLen-1:]
	}

	if append {
		return s[:max-hintLen] + hint
	}

	return hint + s[max-hintLen+1:]
}

func SliceOfSlice(slices ...[]string) (sos [][]string) {
	sos = make([][]string, len(slices))
	for n, s := range slices {
		sos[n] = make([]string, len(s))
		copy(sos[n], s)
	}
	return sos
}

// Copy returns a copy of the source slice.
func Copy(src []string) (dst []string) {
	dst = make([]string, len(src))
	copy(dst, src)
	return dst
}

func SortStable(s []string) {
	sort.Stable(sort.StringSlice(s))
}

func SortReverseStable(s []string) {
	sort.Stable(sort.Reverse(sort.StringSlice(s)))
}

// StringMapPtr returns string pointer which can be used to update a map[string]string value,
// working around the lack of support for syntax like Fn(&myMap[key]) which leads to the
// error "cannot take the address of myMap[key]".
//
// The function it returns must be called in order to save each update.
// An error is returned if the target key is not found in the map.
func StringKeyPtr(m *map[string]string, targetKey string) (*string, func(), error) {
	var found bool

	for k := range *m {
		if k == targetKey {
			found = true
			break
		}
	}

	if !found {
		return nil, nil, errors.Errorf("key [%s] not found in map", targetKey)
	}

	targetVal := (*m)[targetKey]

	return &targetVal, func() {
		(*m)[targetKey] = targetVal
	}, nil
}
