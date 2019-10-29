// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import "sort"

type GlobalIdList struct {
	list []GlobalId
}

func NewGlobalIdList() *GlobalIdList {
	return &GlobalIdList{}
}

// Swap implements sort.Interface.
func (l *GlobalIdList) Swap(i, j int) {
	tmp := l.list[i]
	l.list[i] = l.list[j]
	l.list[j] = tmp
}

func (l *GlobalIdList) Add(i ...GlobalId) *GlobalIdList {
	l.list = append(l.list, i...)
	return l
}

func (l *GlobalIdList) SortedSlice() []GlobalId {
	all := l.Copy()
	sort.Stable(all)
	return all.list
}

// Copy returns a shallow copy.
func (l *GlobalIdList) Copy() *GlobalIdList {
	dst := &GlobalIdList{}
	for _, id := range l.list {
		dst.Add(id)
	}
	return dst
}

// Less implements sort.Interface.
func (l *GlobalIdList) Less(i, j int) bool {
	if l.list[i].Dir() < l.list[j].Dir() {
		return true
	} else if l.list[i].Dir() > l.list[j].Dir() {
		return false
	}
	if l.list[i].PkgName < l.list[j].PkgName {
		return true
	} else if l.list[i].PkgName > l.list[j].PkgName {
		return false
	}
	if l.list[i].Filename < l.list[j].Filename {
		return true
	} else if l.list[i].Filename > l.list[j].Filename {
		return false
	}
	return l.list[i].Name < l.list[j].Name
}

// Len implements sort.Interface.
func (l *GlobalIdList) Len() int {
	return len(l.list)
}
