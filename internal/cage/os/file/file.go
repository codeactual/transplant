// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package file

import (
	"os"
	"path/filepath"
	"strings"

	tp_file "github.com/codeactual/transplant/internal/third_party/github.com/os/file"
	"github.com/pkg/errors"

	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// Exists checks if a file/directory exists.
func Exists(name string) (bool, os.FileInfo, error) {
	fi, err := os.Stat(name)
	if err == nil {
		return true, fi, nil
	}
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	return false, nil, errors.Wrapf(err, "failed to stat [%s]", name)
}

// CreateFileAll calls MkdirAll to ensure all intermediate directories exist prior to creation.
func CreateFileAll(name string, fileFlag int, filePerm, dirPerm os.FileMode) (*os.File, error) {
	dirPath := filepath.Dir(name)
	if err := os.MkdirAll(dirPath, dirPerm); err != nil {
		return nil, errors.Wrapf(err, "failed to make dir [%s] for new file [%s]", dirPath, name)
	}

	f, err := os.OpenFile(name, os.O_CREATE|fileFlag, filePerm)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create file [%s]", name)
	}

	return f, nil
}

func Readdir(dir string, max int) (files []os.FileInfo, err error) {
	f, err := os.Open(dir) // #nosec G304
	if err != nil {
		return []os.FileInfo{}, errors.Wrapf(err, "failed to open file [%s]", dir)
	}
	files, err = f.Readdir(max)
	if err != nil {
		return []os.FileInfo{}, errors.Wrapf(err, "failed to read dir contents [%s]", dir)
	}
	return files, err
}

func Readdirnames(dir string, max int) (names []string, err error) {
	f, err := os.Open(dir) // #nosec G304
	if err != nil {
		return []string{}, errors.Wrapf(err, "failed to open file [%s]", dir)
	}
	names, err = f.Readdirnames(max)
	if err != nil {
		return []string{}, errors.Wrapf(err, "failed to read dir names [%s]", dir)
	}
	cage_strings.SortStable(names)
	return names, err
}

func ValidateRemove(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return errors.New("cannot remove empty path")
	}
	p = filepath.Clean(p)
	if !filepath.IsAbs(p) {
		return errors.New("removing a relative path is not allowed")
	}
	if len(p) == 1 { // e.g. only `os.PathSeparator`
		return errors.Errorf("removing a root path [%c] is not allowed", p[0])
	}
	return nil
}

func RemoveAllSafer(p string) error {
	if err := ValidateRemove(p); err != nil {
		return errors.WithStack(err)
	}
	return tp_file.UnsafeRemoveAll(p)
}

func RemoveSafer(p string) error {
	if err := ValidateRemove(p); err != nil {
		return errors.WithStack(err)
	}
	return os.Remove(p)
}
