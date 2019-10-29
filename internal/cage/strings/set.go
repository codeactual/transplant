// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package strings

type Set struct {
	data map[string]struct{}
}

// NewSet returns an initialized Set.
//
// It returns a pointer to support use as a map's value type and avoid the "cannot call pointer method" error.
func NewSet() *Set {
	return &Set{
		data: make(map[string]struct{}),
	}
}

func (s *Set) Add(el string) bool {
	if _, ok := s.data[el]; ok {
		return false
	}
	s.data[el] = struct{}{}
	return true
}

func (s *Set) AddSet(sets ...*Set) *Set {
	for _, set := range sets {
		for el := range set.data {
			s.data[el] = struct{}{}
		}
	}
	return s
}

func (s *Set) AddSlice(slices ...[]string) *Set {
	for _, slice := range slices {
		for _, el := range slice {
			s.data[el] = struct{}{}
		}
	}
	return s
}

func (s *Set) Remove(el string) bool {
	if _, ok := s.data[el]; ok {
		delete(s.data, el)
		return true
	}
	return false
}

func (s *Set) Contains(el string) bool {
	_, ok := s.data[el]
	return ok
}

func (s *Set) Len() int {
	return len(s.data)
}

func (s *Set) Slice() []string {
	all := []string{}
	for k := range s.data {
		all = append(all, k)
	}
	return all
}

func (s *Set) Copy() *Set {
	all := NewSet()
	for el := range s.data {
		all.Add(el)
	}
	return all
}

func (s *Set) Equals(other *Set) bool {
	if len(s.data) != len(other.data) {
		return false
	}
	otherSlice := other.SortedSlice()
	for n, v := range s.SortedSlice() {
		if v != otherSlice[n] {
			return false
		}
	}
	return true
}

func (s *Set) Clear() {
	for k := range s.data {
		delete(s.data, k)
	}
}

func (s *Set) SortedSlice() []string {
	all := s.Slice()
	SortStable(all)
	return all
}

func (s *Set) SortedReverseSlice() []string {
	all := s.Slice()
	SortReverseStable(all)
	return all
}

// Diff returns the strings found in this Set but not in the other Set.
func (s *Set) Diff(other *Set) *Set {
	d := NewSet()
	for el := range s.data {
		if !other.Contains(el) {
			d.Add(el)
		}
	}
	return d
}
