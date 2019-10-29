// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package list_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_go_list "github.com/codeactual/transplant/internal/cage/go/list"
	cage_exec "github.com/codeactual/transplant/internal/cage/os/exec"
	cage_testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
)

type ListSuite struct {
	suite.Suite

	ctx      context.Context
	executor cage_exec.Executor
}

func TestListSuite(t *testing.T) {
	suite.Run(t, new(ListSuite))
}

func (s *ListSuite) SetupTest() {
	s.ctx = context.Background()
	s.executor = cage_exec.CommonExecutor{}
}

func (s *ListSuite) TestAllModules() {
	t := s.T()

	_, dir := cage_testkit_file.FixturePath(t, "all_mods")
	mods, err := cage_go_list.NewQuery(s.executor, dir).AllModules().Run(s.ctx)
	require.NoError(t, err)

	actualPaths := mods.GetPaths().SortedSlice()

	if strings.Contains(os.Getenv("GOFLAGS"), "-mod=vendor") {
		require.Exactly(
			t,
			[]string{
				"code.cloudfoundry.org/bytefmt",
				"github.com/cloudfoundry/bytefmt",
				"github.com/onsi/ginkgo",
				"github.com/onsi/gomega",
				"github.com/pkg/errors",
			},
			actualPaths,
		)
	} else {
		require.Exactly(
			t,
			[]string{
				"code.cloudfoundry.org/bytefmt",
				"github.com/cloudfoundry/bytefmt",
				"github.com/fsnotify/fsnotify",
				"github.com/golang/protobuf",
				"github.com/hpcloud/tail",
				"github.com/onsi/ginkgo",
				"github.com/onsi/gomega",
				"github.com/pkg/errors",
				"golang.org/x/net",
				"golang.org/x/sync",
				"golang.org/x/sys",
				"golang.org/x/text",
				"gopkg.in/check.v1",
				"gopkg.in/fsnotify.v1",
				"gopkg.in/tomb.v1",
				"gopkg.in/yaml.v2",
			},
			actualPaths,
		)
	}

	require.Exactly(
		t,
		cage_go_list.Module{
			Path:    "code.cloudfoundry.org/bytefmt",
			Version: "v0.0.0-20180906201452-2aa6f33b730c",
		},
		*mods.GetByPath("code.cloudfoundry.org/bytefmt"),
	)

	require.Exactly(
		t,
		cage_go_list.Module{
			Path:    "github.com/cloudfoundry/bytefmt",
			Version: "v0.0.0-20180906201452-2aa6f33b730c",
		},
		*mods.GetByPath("github.com/cloudfoundry/bytefmt"),
	)

	require.Exactly(
		t,
		cage_go_list.Module{
			Path:    "github.com/onsi/ginkgo",
			Version: "v1.8.0",
		},
		*mods.GetByPath("github.com/onsi/ginkgo"),
	)

	require.Exactly(
		t,
		cage_go_list.Module{
			Path:    "github.com/onsi/gomega",
			Version: "v1.5.0",
		},
		*mods.GetByPath("github.com/onsi/gomega"),
	)

	require.Exactly(
		t,
		cage_go_list.Module{
			Path:    "github.com/pkg/errors",
			Version: "v0.8.1",
		},
		*mods.GetByPath("github.com/pkg/errors"),
	)
}
