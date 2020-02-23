// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package strings

type Stack struct {
	items []string
}

// NewStack returns an initialized Stack.
func NewStack() *Stack {
	return &Stack{}
}

// Push appends one or more items.
func (s *Stack) Push(items ...string) {
	s.items = append(s.items, items...)
}

// Pop removes the most recently pushed item, if it exists, and returns it.
// If it does not exist, it returns nil.
func (s *Stack) Pop() *string {
	itemsLen := len(s.items)

	if itemsLen == 0 {
		return nil
	}

	var item string
	item, s.items = s.items[itemsLen-1], s.items[:itemsLen-1]

	return &item
}

// Contains returns true if the query is present.
func (s *Stack) Contains(query string) bool {
	for _, item := range s.items {
		if item == query {
			return true
		}
	}

	return false
}

// Items returns the current contents.
func (s *Stack) Items() (items []string) {
	for _, item := range s.items {
		items = append(items, item)
	}

	return items
}
