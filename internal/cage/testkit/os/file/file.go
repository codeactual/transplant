// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package file

import (
	"os"
	"path/filepath"
	"testing"

	tp_file "github.com/codeactual/transplant/internal/third_party/stackexchange/os/file"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	"github.com/codeactual/transplant/internal/cage/testkit"
	testkit_filepath "github.com/codeactual/transplant/internal/cage/testkit/path/filepath"
)

// dataDir defines the base directory for fixtures and test data.
const dataDir = "testdata"

func FixtureDataDir() string {
	return filepath.Join(dataDir, "fixture")
}

func DynamicDataDir() string {
	return filepath.Join(dataDir, "dynamic")
}

func DynamicDataDirAbs(t *testing.T) string {
	return testkit_filepath.Abs(t, filepath.Join(dataDir, "dynamic"))
}

// ResetTestdata removes all dynamic files/directories found in the conventional location.
//
// If the conventional location does not exist, it makes it.
func ResetTestdata(t *testing.T) {
	dir := DynamicDataDirAbs(t)
	exists, _, err := cage_file.Exists(dir)
	if err != nil {
		testkit.FatalErrf(t, err, "failed to reset test data dir")
	}
	if !exists {
		testkit.FatalErrf(t, os.MkdirAll(dir, 0700), "failed to make dir [%s]", dir)
	}
	testkit.FatalErrf(t, tp_file.RemoveContents(dir), "failed to remove dir contents [%s]", dir)
}

// CreatePath returns a path under DynamicDataDir() at a relative path built by joining the path parts.
func CreatePath(t *testing.T, pathPart ...string) (relPath string, absPath string) {
	pathPart = append([]string{DynamicDataDir()}, pathPart...)
	relPath = filepath.Join(pathPart...)

	absPath, err := filepath.Abs(relPath)
	testkit.FatalErrf(t, err, "failed to get absolute path [%s]", relPath)

	return relPath, absPath
}

// FixturePath returns a path under FixtureDataDir() at a relative path built by joining the path parts.
func FixturePath(t *testing.T, pathPart ...string) (relPath string, absPath string) {
	pathPart = append([]string{FixtureDataDir()}, pathPart...)
	relPath = filepath.Join(pathPart...)

	absPath, err := filepath.Abs(relPath)
	testkit.FatalErrf(t, err, "failed to get absolute path [%s]", relPath)

	return relPath, absPath
}

// CreateFile creates a file under DynamicDataDir() at a relative path built by joining the path parts.
//
// All parts except the final one are assumed to be ancestor directrories (which are created if needed).
//
// The file is created with 0600. Directories are created with 0700.
func CreateFile(t *testing.T, pathPart ...string) (relPath string, absPath string) {
	relPath, absPath = CreatePath(t, pathPart...)
	_, err := cage_file.CreateFileAll(absPath, 0, 0600, 0700)
	testkit.FatalErrf(t, err, "failed to create file [%s]", absPath)
	return relPath, absPath
}

func MustOpen(t *testing.T, filename string) *os.File {
	f, err := os.Open(filename) // #nosec G304
	testkit.FatalErrf(t, err, "failed to get open file [%s]", filename)
	return f
}

func MustOpenFixturePath(t *testing.T, pathPart ...string) *os.File {
	_, absPath := FixturePath(t, pathPart...)
	return MustOpen(t, absPath)
}
