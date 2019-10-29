// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
	"github.com/codeactual/transplant/internal/transplant"
)

// Per-case comments may refer to configuration file sections such as Ops.From and Ops.Dep.
// The file is located in ./fixture/config/transplant.yml.

type ConfigSuite struct {
	Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) TestExpandEnv() {
	t := s.T()

	opId := "expand_env"

	s.Env = map[string]string{
		"copy_module_filepath":   testkit_file.DynamicDataDirAbs(t),
		"copy_module_importpath": "copy.tld/user/proj",
		"origin_module_filepath": s.FixturePath("config"),
		"inline_edit":            "edit",
	}
	for k, v := range s.Env {
		require.NoError(t, os.Setenv(k, v))
	}

	fixture := s.MustLoadFixture("egress", "config", "ConfigSuite", "yml", opId)

	expectedEgress := transplant.Op{
		Id: opId,
		From: transplant.RootFrom{
			ModuleFilePath:   s.FixturePath("config", opId, "origin"),
			LocalFilePath:    s.Env["inline_edit"] + "_local",
			ModuleImportPath: "origin.tld/user/proj",
			LocalImportPath:  "origin.tld/user/proj/" + s.Env["inline_edit"] + "_local",
			GoFilePath: transplant.FilePathQuery{
				Include: []string{
					"**/*",
				},
				Exclude: nil,
			},
			CopyOnlyFilePath: transplant.FilePathQuery{
				Include: nil,
				Exclude: nil,
			},
			GoDescendantFilePath: transplant.FilePathQuery{
				Include: nil,
				Exclude: nil,
			},
			RenameFilePath: []transplant.RenameSpec{
				{Old: s.Env["inline_edit"] + "_old", New: s.Env["inline_edit"] + "_new"},
			},
			ModuleSum: false,
			Tests:     false,
			Vendor:    false,
		},
		To: transplant.RootTo{
			ModuleFilePath:   filepath.Join(testkit_file.DynamicDataDirAbs(t)),
			LocalFilePath:    s.Env["inline_edit"] + "_copy",
			ModuleImportPath: s.Env["copy_module_importpath"],
			LocalImportPath:  s.Env["copy_module_importpath"] + "/" + s.Env["inline_edit"] + "_copy",
		},
		Dep: []transplant.Dep{
			{
				From: transplant.DepFrom{
					FilePath:   s.Env["inline_edit"] + "_dep1",
					ImportPath: "origin.tld/user/proj/" + s.Env["inline_edit"] + "_dep1",
					GoFilePath: transplant.FilePathQuery{
						Include: []string{
							"**/*",
						},
						Exclude: nil,
					},
					CopyOnlyFilePath: transplant.FilePathQuery{
						Include: []string{
							s.Env["inline_edit"] + "_copy_only",
						},
						Exclude: nil,
					},
					GoDescendantFilePath: transplant.FilePathQuery{
						Include: nil,
						Exclude: nil,
					},
					Tests: false,
				},
				To: transplant.DepTo{
					FilePath:   "internal/" + s.Env["inline_edit"] + "_dep1",
					ImportPath: s.Env["copy_module_importpath"] + "/internal/" + s.Env["inline_edit"] + "_dep1",
				},
			},
		},
		DryRun:  false,
		Ingress: false,
	}
	require.Exactly(t, expectedEgress, fixture.Audit.Op())
}

func (s *ConfigSuite) TestIngressFinalizer() {
	t := s.T()

	opId := "ingress_finalizer"

	expectedIngress := transplant.Op{
		Id: opId,
		From: transplant.RootFrom{
			ModuleFilePath:   s.FixturePath("config", opId, "copy"),
			ModuleImportPath: "copy.tld/user/proj",
			LocalImportPath:  "copy.tld/user/proj",
			GoFilePath: transplant.FilePathQuery{
				Include: []string{
					"**/*",
				},
				Exclude: nil,
			},
			CopyOnlyFilePath: transplant.FilePathQuery{
				Include: []string{
					"bin/*",
				},
				Exclude: nil,
			},
			GoDescendantFilePath: transplant.FilePathQuery{
				Include: nil,
				Exclude: nil,
			},
			RenameFilePath: []transplant.RenameSpec{
				{Old: "new1", New: "old1"},
				{Old: "new2", New: "old2"},
			},
			ModuleSum: false,
			Tests:     false,
			Vendor:    false,
		},
		To: transplant.RootTo{
			ModuleFilePath:   filepath.Join(testkit_file.DynamicDataDirAbs(t)),
			LocalFilePath:    "local",
			ModuleImportPath: "origin.tld/user/proj",
			LocalImportPath:  "origin.tld/user/proj/local",
		},
		Dep: []transplant.Dep{
			{
				From: transplant.DepFrom{
					FilePath:   filepath.Join("internal", "dep1"),
					ImportPath: "copy.tld/user/proj/internal/dep1",
					GoFilePath: transplant.FilePathQuery{
						Include: []string{
							"**/*",
						},
						Exclude: nil,
					},
					CopyOnlyFilePath: transplant.FilePathQuery{
						Include: []string{
							"bin/*",
						},
						Exclude: nil,
					},
					GoDescendantFilePath: transplant.FilePathQuery{
						Include: nil,
						Exclude: nil,
					},
					Tests: false,
				},
				To: transplant.DepTo{
					FilePath:   filepath.Join("dep1"),
					ImportPath: "origin.tld/user/proj/dep1",
				},
			},
		},
		DryRun:  false,
		Ingress: true,
	}

	fixture, errs := s.LoadFixture("ingress", "config", "ConfigSuite", "yml", opId)
	for _, err := range errs {
		if strings.Contains(err.Error(), "From.FilePath not found") && strings.Contains(err.Error(), "testdata") {
			continue
		}
		require.NoError(t, err)
	}
	require.Exactly(t, expectedIngress, fixture.Audit.Op())
}

func (s *ConfigSuite) TestJson() {
	t := s.T()

	opId := "operation_id"

	fixture, errs := s.LoadFixture("egress", "config", "ConfigSuite", "json", opId)
	for _, err := range errs {
		if strings.Contains(err.Error(), "From.FilePath not found") && strings.Contains(err.Error(), "testdata") {
			continue
		}
		require.NoError(t, err)
	}

	expectedEgress := transplant.Op{
		Id: opId,
		From: transplant.RootFrom{
			ModuleFilePath:   s.FixturePath("config", opId, "origin"),
			LocalFilePath:    "local",
			ModuleImportPath: "origin.tld/user/proj",
			LocalImportPath:  "origin.tld/user/proj/local",
			GoFilePath: transplant.FilePathQuery{
				Include: []string{
					"**/*",
				},
				Exclude: nil,
			},
			CopyOnlyFilePath: transplant.FilePathQuery{
				Include: []string{
					"bin/*",
				},
				Exclude: nil,
			},
			GoDescendantFilePath: transplant.FilePathQuery{
				Include: nil,
				Exclude: nil,
			},
			RenameFilePath: []transplant.RenameSpec{
				{Old: "old1", New: "new1"},
				{Old: "old2", New: "new2"},
			},
			ModuleSum: false,
			Tests:     false,
			Vendor:    false,
		},
		To: transplant.RootTo{
			ModuleFilePath:   filepath.Join(testkit_file.DynamicDataDirAbs(t)),
			ModuleImportPath: "copy.tld/user/proj",
			LocalImportPath:  "copy.tld/user/proj",
		},
		Dep: []transplant.Dep{
			{
				From: transplant.DepFrom{
					FilePath:   "dep1",
					ImportPath: "origin.tld/user/proj/dep1",
					GoFilePath: transplant.FilePathQuery{
						Include: []string{
							"**/*",
						},
						Exclude: nil,
					},
					CopyOnlyFilePath: transplant.FilePathQuery{
						Include: []string{
							"bin/*",
						},
						Exclude: nil,
					},
					GoDescendantFilePath: transplant.FilePathQuery{
						Include: nil,
						Exclude: nil,
					},
					Tests: false,
				},
				To: transplant.DepTo{
					FilePath:   filepath.Join("internal", "dep1"),
					ImportPath: "copy.tld/user/proj/internal/dep1",
				},
			},
		},
		DryRun:  false,
		Ingress: false,
	}
	require.Exactly(t, expectedEgress, fixture.Audit.Op())
}

func (s *ConfigSuite) TestToml() {
	t := s.T()

	opId := "operation_id"

	fixture, errs := s.LoadFixture("egress", "config", "ConfigSuite", "toml", opId)
	for _, err := range errs {
		if strings.Contains(err.Error(), "From.FilePath not found") && strings.Contains(err.Error(), "testdata") {
			continue
		}
		require.NoError(t, err)
	}

	expectedEgress := transplant.Op{
		Id: opId,
		From: transplant.RootFrom{
			ModuleFilePath:   s.FixturePath("config", opId, "origin"),
			LocalFilePath:    "local",
			ModuleImportPath: "origin.tld/user/proj",
			LocalImportPath:  "origin.tld/user/proj/local",
			GoFilePath: transplant.FilePathQuery{
				Include: []string{
					"**/*",
				},
				Exclude: nil,
			},
			CopyOnlyFilePath: transplant.FilePathQuery{
				Include: []string{
					"bin/*",
				},
				Exclude: nil,
			},
			GoDescendantFilePath: transplant.FilePathQuery{
				Include: nil,
				Exclude: nil,
			},
			RenameFilePath: []transplant.RenameSpec{
				{Old: "old1", New: "new1"},
				{Old: "old2", New: "new2"},
			},
			ModuleSum: false,
			Tests:     false,
			Vendor:    false,
		},
		To: transplant.RootTo{
			ModuleFilePath:   filepath.Join(testkit_file.DynamicDataDirAbs(t)),
			ModuleImportPath: "copy.tld/user/proj",
			LocalImportPath:  "copy.tld/user/proj",
		},
		Dep: []transplant.Dep{
			{
				From: transplant.DepFrom{
					FilePath:   "dep1",
					ImportPath: "origin.tld/user/proj/dep1",
					GoFilePath: transplant.FilePathQuery{
						Include: []string{
							"**/*",
						},
						Exclude: nil,
					},
					CopyOnlyFilePath: transplant.FilePathQuery{
						Include: []string{
							"bin/*",
						},
						Exclude: nil,
					},
					GoDescendantFilePath: transplant.FilePathQuery{
						Include: nil,
						Exclude: nil,
					},
					Tests: false,
				},
				To: transplant.DepTo{
					FilePath:   filepath.Join("internal", "dep1"),
					ImportPath: "copy.tld/user/proj/internal/dep1",
				},
			},
		},
		DryRun:  false,
		Ingress: false,
	}
	require.Exactly(t, expectedEgress, fixture.Audit.Op())
}
