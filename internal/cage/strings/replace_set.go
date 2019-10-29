// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package strings

import "strings"

// Replace defines strings.Replace parameters.
type Replace struct {
	Limit int
	New   string
	Old   string
}

// ReplaceSet holds replacement definitons indexed by Replace.Old values.
type ReplaceSet map[string]Replace

// Add creates or overwrites a replacement definiton for the given old/target string.
func (s *ReplaceSet) Add(old, new string, limit int) *ReplaceSet {
	(*s)[old] = Replace{Limit: limit, New: new, Old: old}
	return s
}

// InString returns the subject string with all replacements performed in length-descending order
// of Replace.Old values.
func (s *ReplaceSet) InString(subject string) string {
	for _, r := range s.sortedSlice() {
		subject = strings.Replace(subject, r.Old, r.New, r.Limit)
	}
	return subject
}

// InByte returns the subject byte slice with all replacements performed in length-descending order
// of Replace.Old values.
func (s *ReplaceSet) InByte(subject []byte) []byte {
	str := string(subject)
	for _, r := range s.sortedSlice() {
		str = strings.Replace(str, r.Old, r.New, r.Limit)
	}
	return []byte(str)
}

// sortedSlice returns the replacement definitions sorted by Replace.Old in descending order
// so longer values are consumed before shorter.
func (s *ReplaceSet) sortedSlice() (r []Replace) {
	var old []string
	for o := range *s {
		old = append(old, o)
	}

	SortReverseStable(old)

	for _, o := range old {
		r = append(r, (*s)[o])
	}

	return r
}
