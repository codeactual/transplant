// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package file_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
)

type FileSuite struct {
	suite.Suite
}

func (s *FileSuite) SetupTest() {
	t := s.T()

	testkit_file.ResetTestdata(t)
}

func (s *FileSuite) TestCreateFileAll() {
	t := s.T()

	flag := os.O_APPEND
	expectedDirPerm := os.ModeDir | os.FileMode(0700)
	expectedFilePerm := os.FileMode(0600)

	dirs := []string{"path", "to", "file"}
	_, name := testkit_file.CreatePath(t, dirs...)

	f, err := cage_file.CreateFileAll(name, flag, expectedFilePerm, expectedDirPerm)
	require.Exactly(t, name, f.Name())
	require.NoError(t, err)

	_, name = testkit_file.CreatePath(t, dirs...)
	exists, fi, err := cage_file.Exists(name)
	require.NoError(t, err)
	require.True(t, exists)
	require.False(t, fi.IsDir())
	require.Exactly(t, expectedFilePerm, fi.Mode(), fi.Mode().String())

	_, name = testkit_file.CreatePath(t, dirs[0:len(dirs)-1]...)
	exists, fi, err = cage_file.Exists(name)
	require.NoError(t, err)
	require.True(t, exists)
	require.True(t, fi.IsDir())
	require.Exactly(t, expectedDirPerm, fi.Mode(), fi.Mode().String())

	_, name = testkit_file.CreatePath(t, dirs[0:len(dirs)-2]...)
	exists, fi, err = cage_file.Exists(name)
	require.NoError(t, err)
	require.True(t, exists)
	require.True(t, fi.IsDir())
	require.Exactly(t, expectedDirPerm, fi.Mode(), fi.Mode().String())
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(FileSuite))
}
