// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package file

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

type FinderFile struct {
	AbsPath string
	Info    os.FileInfo
}

type DirMatcher func(absName string, files FinderMatchFiles) (bool, error)
type FileMatcher func(candidate FinderFile) (bool, error)

// FinderMatchFiles indexes FinderFile results by file absolute path.
type FinderMatchFiles map[string]FinderFile

// DirToFinderMatchFiles indexes os.Readdir results by directory absolute path.
type DirToFinderMatchFiles map[string]FinderMatchFiles

type Finder struct {
	dirs          *cage_strings.Set
	dirMatchers   []DirMatcher
	walkHiddenDir bool
	readdirCache  DirToFinderMatchFiles
}

func NewFinder() *Finder {
	f := &Finder{}
	return f.Clear()
}

func (f *Finder) readdir(dir string) (FinderMatchFiles, error) {
	if cached, ok := f.readdirCache[dir]; ok {
		return cached, nil
	}
	files, err := Readdir(dir, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	results := make(FinderMatchFiles)
	for _, file := range files {
		absName := filepath.Join(dir, file.Name())
		results[absName] = FinderFile{AbsPath: absName, Info: file}
	}
	f.readdirCache[dir] = results

	return f.readdirCache[dir], nil
}

func (f *Finder) Clear() *Finder {
	f.dirs = cage_strings.NewSet()
	f.dirMatchers = []DirMatcher{}
	f.readdirCache = make(DirToFinderMatchFiles)
	return f
}

func (f *Finder) WalkHiddenDir(b bool) *Finder {
	f.walkHiddenDir = b
	return f
}

func (f *Finder) Dir(dirs ...string) *Finder {
	f.dirs.AddSlice(dirs)
	return f
}

func (f *Finder) DirMatcher(matchers ...DirMatcher) *Finder {
	f.dirMatchers = append(f.dirMatchers, matchers...)
	return f
}

// GetDirMatches walks the registered directories and returns a FinderFile element for each
// sub-directory which satisfies the matcher.
//
// Each registered directory will be included if it also satisfies the matcher.
func (f *Finder) GetDirMatches() (dirs []FinderFile, err error) {
	rootedWalkFunc := func(root string) func(dir string, fi os.FileInfo, walkErr error) []error {
		return func(dir string, fi os.FileInfo, walkErr error) []error {
			if walkErr != nil {
				return []error{errors.WithStack(walkErr)}
			}

			if !fi.IsDir() {
				return []error{}
			}

			if filepath.Base(dir)[0] == '.' && !f.walkHiddenDir {
				return []error{filepath.SkipDir}
			}

			files, readErr := f.readdir(dir)
			if readErr != nil {
				return []error{errors.WithStack(readErr)}
			}

			matchAll := true
			for _, matcher := range f.dirMatchers {
				match, matchErr := matcher(dir, files)
				if matchErr != nil {
					return []error{errors.WithStack(matchErr)}
				}
				if !match {
					matchAll = false
					break
				}
			}
			if matchAll {
				dirs = append(dirs, FinderFile{AbsPath: dir, Info: fi})
			}

			return []error{}
		}
	}

	for _, root := range f.dirs.SortedSlice() {
		errs := cage_filepath.WalkAbs(root, rootedWalkFunc(root)) // create function to avoid closure-in-loop var issue

		if len(errs) > 0 {
			return nil, errors.WithStack(errs[0]) // only return the first one encountered by WalkAbs
		}
	}

	return dirs, nil
}

// GetDirnameMatches walks the registered directories and returns the absolute paths to all
// sub-directories which satisfy the matcher.
//
// Each registered directory will be included if it also satisfies the matcher.
func (f *Finder) GetDirnameMatches() (*cage_strings.Set, error) {
	files, err := f.GetDirMatches()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	names := cage_strings.NewSet()
	for _, file := range files {
		names.Add(file.AbsPath)
	}
	return names, nil
}

func (f *Finder) GetFileMatches(fileMatchers ...FileMatcher) (DirToFinderMatchFiles, error) {
	dirs, dirsErr := f.GetDirMatches()
	if dirsErr != nil {
		return nil, errors.WithStack(dirsErr)
	}

	matches := make(DirToFinderMatchFiles)

	for _, dir := range dirs {
		matches[dir.AbsPath] = make(FinderMatchFiles)

		files, readErr := f.readdir(dir.AbsPath)
		if readErr != nil {
			return nil, errors.WithStack(readErr)
		}

		for _, file := range files {
			if file.Info.IsDir() {
				continue
			}

			matchAll := true

			for _, matcher := range fileMatchers {
				match, matchErr := matcher(file)
				if matchErr != nil {
					return nil, errors.WithStack(matchErr)
				}
				if !match {
					matchAll = false
					break
				}
			}

			if matchAll {
				matches[dir.AbsPath][file.AbsPath] = file
			}
		}
	}

	return matches, nil
}

func (f *Finder) GetFilenameMatches(fileMatchers ...FileMatcher) (*cage_strings.Set, error) {
	dirFileMatches, err := f.GetFileMatches(fileMatchers...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	names := cage_strings.NewSet()
	for _, fileMatches := range dirFileMatches {
		for absName := range fileMatches {
			names.Add(absName)
		}
	}
	return names, nil
}

func (f *Finder) GetFileObjectMatches(fileMatchers ...FileMatcher) (objs []FinderFile, err error) {
	dirFileMatches, err := f.GetFileMatches(fileMatchers...)
	if err != nil {
		return []FinderFile{}, errors.WithStack(err)
	}
	for _, fileMatches := range dirFileMatches {
		for _, obj := range fileMatches {
			objs = append(objs, obj)
		}
	}
	return objs, nil
}
