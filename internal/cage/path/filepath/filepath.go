// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package filepath

import (
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	std_filepath "path/filepath"
)

var unsafePathPartRe *regexp.Regexp

func init() {
	unsafePathPartRe = regexp.MustCompile(`[\s\\\/]`)
}

// WalkFunc matches std_filepath.WalkFunc except it returns an error slice.
type WalkFunc func(string, os.FileInfo, error) []error

// Append behaves like Join except that as each element must add at least one
// level to the path.
//
// For example, an element with "../../some_path" would effectively replace
// a level instead of adding one.
//
// An error is returned after the first illegal element is reached.
func Append(paths ...string) (joined string, err error) {
	var prev string

	for _, p := range paths {
		joined = std_filepath.Join(joined, p)

		if prev != "" && !strings.HasPrefix(joined, prev) {
			return "", errors.Errorf("path [%s] cannot be appended, escapes prefix [%s]", p, prev)
		}

		prev = joined
	}

	return joined, nil
}

func Abs(p *string) error {
	abs, err := std_filepath.Abs(*p)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve absolute path [%s]", *p)
	}
	*p = abs
	return nil
}

// BaseWithoutExt returns "abc" from "/path/to/abc.txt".
func BaseWithoutExt(filename string) string {
	base := std_filepath.Base(filename)
	ext := std_filepath.Ext(base)
	return base[0 : len(base)-len(ext)]
}

func IsGoFile(name string) bool {
	return std_filepath.Ext(name) == ".go"
}

func IsGoTestFile(name string) bool {
	return strings.HasSuffix(name, "_test.go")
}

// AncestorDirs returns all absolute paths between a starting descendant absolute path and
// an ancestor absolute path.
//
// The end path is included in the results (if it is encountered).
//
// If the end path is not encountered, all ancestors will be returned up to the root.
func AncestorDirs(start, end string) (dirs []string) {
	start = std_filepath.Clean(start)
	end = std_filepath.Clean(end)
	current := start
	for {
		if current == end {
			break
		}
		prev := current
		current = std_filepath.Dir(current)
		if prev == current { // cannot ascend any farther
			break
		}
		dirs = append(dirs, current)
	}
	return dirs
}

// WalkAbs wraps the standard Walk to make several adjustments: an absolute path is
// passed to the standard Walk, absolute paths are passed to the input WalkFunc,
// and the WalkFunc is a new type defined by this package.
func WalkAbs(root string, walkFn WalkFunc) (errs []error) {
	root, rootErr := std_filepath.Abs(root)
	if rootErr != nil {
		return []error{errors.Wrapf(rootErr, "failed to walk file tree: unable to get absolute path [%s]", root)}
	}

	_ = std_filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		pathAbs, pathErr := std_filepath.Abs(path)
		if pathErr != nil {
			errs = append(errs, errors.Wrapf(pathErr, "failed to walk file tree: unable to get absolute path [%s]", path))
			return errors.New("") // just cancel the walk
		}

		if walkFnErrs := walkFn(pathAbs, info, walkErr); len(walkFnErrs) > 0 {
			if walkFnErrs[0] == std_filepath.SkipDir {
				return walkFnErrs[0]
			}
			for _, err := range walkFnErrs {
				errs = append(errs, errors.WithStack(err))
			}
			return errors.New("") // just cancel the walk
		}

		return nil
	})
	return errs
}

// FileAncestor returns the intermediate directories between a root directory and a descendant
// file/directory.
//
// For example, consider the behavior of os.MkdirAll. Just as it creates all intermediate
// directories as necessary to fulfill the request, this function returns the names of those
// directories instead of creating them. And instead of using / or the working directory as a
// root or starting point based on the name, this function requires the root to be chosen
// and applies it globally.
//
// It will return an error if the file/directory is not a descendant of the root.
//
// Every input and output path is resolved with filepath.Abs.
func FileAncestor(descendant, root string) (a []string, err error) {
	root, err = std_filepath.Abs(root)
	if err != nil {
		return a, errors.Wrapf(err, "failed to get absolute path of root [%s]", root)
	}

	if descendant == root {
		return a, nil
	}

	descendant, err = std_filepath.Abs(descendant)
	if err != nil {
		return a, errors.Wrapf(err, "failed to get absolute path of descendant [%s]", descendant)
	}

	if !strings.HasPrefix(descendant, root) {
		return a, errors.Errorf("[%s] is not a descendant of [%s]", descendant, root)
	}

	for {
		base := descendant
		descendant = std_filepath.Dir(descendant)

		if descendant == base { // reached filesystem root
			break
		}

		a = append(a, descendant)

		if descendant == root {
			break
		}
	}

	return a, nil
}

// PathToSafeFilename converts a filepath to a string usable as a filename.
func PathToSafeFilename(p string) string {
	return strings.Trim(unsafePathPartRe.ReplaceAllString(p, "-"), "-")
}
