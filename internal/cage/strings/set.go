// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package strings

import "sync"

type Set struct {
	sync.RWMutex

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
	s.Lock()
	defer s.Unlock()

	if _, ok := s.data[el]; ok {
		return false
	}

	s.data[el] = struct{}{}

	return true
}

func (s *Set) AddSet(sets ...*Set) *Set {
	s.Lock()
	defer s.Unlock()

	for _, set := range sets {
		for el := range set.data {
			s.data[el] = struct{}{}
		}
	}

	return s
}

func (s *Set) AddSlice(slices ...[]string) *Set {
	s.Lock()
	defer s.Unlock()

	for _, slice := range slices {
		for _, el := range slice {
			s.data[el] = struct{}{}
		}
	}

	return s
}

func (s *Set) Remove(el string) bool {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.data[el]; ok {
		delete(s.data, el)
		return true
	}

	return false
}

func (s *Set) Contains(el string) bool {
	s.RLock()
	defer s.RUnlock()

	_, ok := s.data[el]
	return ok
}

func (s *Set) Len() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.data)
}

func (s *Set) Slice() []string {
	s.RLock()
	defer s.RUnlock()

	all := []string{}
	for k := range s.data {
		all = append(all, k)
	}
	return all
}

func (s *Set) Copy() *Set {
	s.RLock()
	defer s.RUnlock()

	all := NewSet()
	for el := range s.data {
		all.Add(el)
	}
	return all
}

func (s *Set) Equals(other *Set) bool {
	// RUnlock this prior to SortedSlice to avoid double lock.
	s.RLock()

	if len(s.data) != len(other.data) {
		s.RUnlock()
		return false
	}
	s.RUnlock()

	otherSlice := other.SortedSlice()
	for n, v := range s.SortedSlice() {
		if v != otherSlice[n] {
			return false
		}
	}

	return true
}

func (s *Set) Clear() {
	s.Lock()
	defer s.Unlock()

	for k := range s.data {
		delete(s.data, k)
	}
}

func (s *Set) SortedSlice() []string {
	// Do not lock -- Slice locks internally.

	all := s.Slice()
	SortStable(all)
	return all
}

func (s *Set) SortedReverseSlice() []string {
	// Do not lock -- Slice locks internally.

	all := s.Slice()
	SortReverseStable(all)
	return all
}

// Diff returns the strings found in this Set but not in the other Set.
func (s *Set) Diff(other *Set) *Set {
	s.RLock()
	defer s.RUnlock()

	var unique []string
	d := NewSet()

	for el := range s.data {
		if !other.Contains(el) {
			unique = append(unique, el)
		}
	}

	d.AddSlice(unique)

	return d
}
