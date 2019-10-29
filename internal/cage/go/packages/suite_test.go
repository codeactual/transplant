// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages_test

import (
	"github.com/stretchr/testify/suite"
	std_packages "golang.org/x/tools/go/packages"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	testkit "github.com/codeactual/transplant/internal/cage/testkit"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
)

type BaseInspectorSuite struct {
	suite.Suite
}

func (s *BaseInspectorSuite) FixturePath(parts ...string) string {
	_, p := testkit_file.FixturePath(s.T(), parts...)
	return p
}

func (s *BaseInspectorSuite) Inspect(wd string, mode std_packages.LoadMode, dirs ...string) (*cage_pkgs.Inspector, []error) {
	i := cage_pkgs.NewInspector(
		cage_pkgs.NewConfig(&std_packages.Config{
			Dir:  wd,
			Mode: mode,
		}),
		dirs...,
	)
	return i, i.Inspect()
}

func (s *BaseInspectorSuite) MustInspect(wd string, mode std_packages.LoadMode, dirs ...string) *cage_pkgs.Inspector {
	i, errs := s.Inspect(wd, mode, dirs...)
	testkit.RequireNoErrors(s.T(), errs)
	return i
}

func (s *BaseInspectorSuite) InitId(filename string) string {
	return "init." + filename
}
