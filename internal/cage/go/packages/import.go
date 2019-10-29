// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages

import (
	"fmt"
	"sort"

	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

type Import struct {
	// Dir is the absolute path of the source directory.
	Dir string

	// DeclName is the package name from the package clause.
	DeclName string

	// UsedName is the package name used in the importer file.
	//
	// By default it will equal DeclName unless initialized by a New* function which
	// has that information, or updated via SetUsedName (e.g. with an ast.ImportSpec.Name.Name).
	UsedName string

	// Path is the import path.
	Path string
}

// NewImportFromPkg returns an Import initialized from a golang.org/x/tools/go/packages.Package.
func NewImportFromPkg(p *Package) Import {
	return Import{
		Path: p.PkgPath,
		Dir:  p.Dir,

		DeclName: p.Name,
		UsedName: p.Name,
	}
}

func (i *Import) SetUsedName(name string) {
	i.UsedName = name
}

func (i Import) String() string {
	return fmt.Sprintf("package [%s] at [%s] as [%s] from dir [%s]", i.DeclName, i.Path, i.UsedName, i.Dir)
}

type ImportList struct {
	list   []Import
	dedupe map[string]*Import
}

func NewImportList() *ImportList {
	return &ImportList{
		dedupe: make(map[string]*Import),
	}
}

func (l *ImportList) Add(i Import) {
	if _, ok := l.dedupe[i.Path]; ok {
		return
	}
	l.list = append(l.list, i)
	l.dedupe[i.Path] = &l.list[len(l.list)-1]
}

func (l *ImportList) Has(p string) bool {
	_, found := l.dedupe[p]
	return found
}

func (l *ImportList) Get(p string) *Import {
	return l.dedupe[p]
}

func (l *ImportList) SortedSlice() []Import {
	all := l.Copy()
	sort.Stable(all)
	return all.list
}

func (l *ImportList) Paths() (p []string) {
	for _, i := range l.list {
		p = append(p, i.Path)
	}
	cage_strings.SortStable(p)
	return p
}

// Swap implements sort.Interface.
func (l *ImportList) Swap(i, j int) {
	tmp := l.list[i]
	l.list[i] = l.list[j]
	l.list[j] = tmp
}

// Less implements sort.Interface.
func (l *ImportList) Less(i, j int) bool {
	return l.list[i].Path < l.list[j].Path
}

// Len implements sort.Interface.
func (l *ImportList) Len() int {
	return len(l.list)
}

func (l *ImportList) Pop() (i Import) {
	i, l.list = l.list[0], l.list[1:]
	delete(l.dedupe, i.Path)
	return i
}

// Copy returns a shallow copy.
func (l *ImportList) Copy() *ImportList {
	dst := NewImportList()
	for _, i := range l.list {
		dst.Add(i)
	}
	return dst
}
