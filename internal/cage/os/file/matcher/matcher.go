// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package matcher

import (
	"github.com/pkg/errors"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
)

func InvertDir(m cage_file.DirMatcher) cage_file.DirMatcher {
	return func(absName string, files cage_file.FinderMatchFiles) (bool, error) {
		match, err := m(absName, files)
		if err != nil {
			return false, nil
		}
		return !match, nil
	}
}

func InvertFile(m cage_file.FileMatcher) cage_file.FileMatcher {
	return func(f cage_file.FinderFile) (bool, error) {
		match, err := m(f)
		if err != nil {
			return false, nil
		}
		return !match, nil
	}
}

func GoDir(absName string, files cage_file.FinderMatchFiles) (bool, error) {
	for _, f := range files {
		if cage_filepath.IsGoFile(f.AbsPath) {
			return true, nil
		}
	}
	return false, nil
}

func PopulatedDir(absName string, files cage_file.FinderMatchFiles) (bool, error) {
	return len(files) > 0, nil
}

func DirWithFile(absName string, files cage_file.FinderMatchFiles) (bool, error) {
	for _, f := range files {
		if !f.Info.IsDir() {
			return true, nil
		}
	}
	return false, nil
}

func GoFile(f cage_file.FinderFile) (match bool, err error) {
	return cage_filepath.IsGoFile(f.AbsPath), nil
}

func MatchAnyDir(config cage_filepath.MatchAnyInput) cage_file.DirMatcher {
	return func(absName string, files cage_file.FinderMatchFiles) (bool, error) {
		res, err := cage_filepath.PathMatchAny(cage_filepath.MatchAnyInput{
			Name:    absName,
			Include: config.Include,
			Exclude: config.Exclude,
		})
		if err != nil {
			return false, errors.WithStack(err)
		}
		return res.Match, nil
	}
}

func MatchAnyFile(config cage_filepath.MatchAnyInput) cage_file.FileMatcher {
	return func(candidate cage_file.FinderFile) (bool, error) {
		res, err := cage_filepath.PathMatchAny(cage_filepath.MatchAnyInput{
			Name:    candidate.AbsPath,
			Include: config.Include,
			Exclude: config.Exclude,
		})
		if err != nil {
			return false, errors.WithStack(err)
		}
		return res.Match, nil
	}
}
