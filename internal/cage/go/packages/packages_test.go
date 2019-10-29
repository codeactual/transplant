// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package packages_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	std_packages "golang.org/x/tools/go/packages"

	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
)

func TestTrimVendorPathPrefix(t *testing.T) {
	require.Exactly(t, "github.com/pkg/errors", cage_pkgs.TrimVendorPathPrefix("path/to/vendor/github.com/pkg/errors"))
	require.Exactly(t, "path/to/vendor/", cage_pkgs.TrimVendorPathPrefix("path/to/vendor/"))
	require.Exactly(t, "path/to/vendor", cage_pkgs.TrimVendorPathPrefix("path/to/vendor"))
	require.Exactly(t, "path/to/vendo", cage_pkgs.TrimVendorPathPrefix("path/to/vendo"))
}

func TestNeedSatisfied(t *testing.T) {
	require.False(t, cage_pkgs.NeedSatisfied(std_packages.NeedName, 0)) // missing NeedName

	require.True(t, cage_pkgs.NeedSatisfied(std_packages.NeedName, std_packages.NeedName)) // exact
	require.True(t, cage_pkgs.NeedSatisfied(                                               // superset
		std_packages.NeedName,
		std_packages.NeedName|std_packages.NeedFiles,
	))

	require.False(t, cage_pkgs.NeedSatisfied( // missing NeedFiles
		std_packages.NeedName|std_packages.NeedFiles,
		std_packages.NeedName,
	))
	require.False(t, cage_pkgs.NeedSatisfied( // missing NeedName
		std_packages.NeedName|std_packages.NeedFiles,
		std_packages.NeedFiles,
	))
}

func TestConfig(t *testing.T) {
	t.Run("should Copy embedded packages.Config by value", func(t *testing.T) {
		orig := cage_pkgs.NewConfig(&std_packages.Config{
			Mode: std_packages.NeedFiles,
		})

		cpy := orig.Copy()

		require.False(t, orig.Config == cpy.Config)
		require.Exactly(t, orig.Config.Mode, cpy.Config.Mode)
	})
}
