// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package mod_test

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cage_mod "github.com/codeactual/transplant/internal/cage/go/mod"
	cage_testkit "github.com/codeactual/transplant/internal/cage/testkit"
	cage_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

func TestModPath(t *testing.T) {
	srcModFile := cage_file.MustOpenFixturePath(t, "src.mod")
	srcMod, err := cage_mod.NewMod(srcModFile)
	require.NoError(t, err)
	require.Exactly(t, "domain.com/path/to/src", srcMod.Path)
}

func TestModReplace(t *testing.T) {
	srcModFile := cage_file.MustOpenFixturePath(t, "src.mod")
	srcMod, err := cage_mod.NewMod(srcModFile)
	require.NoError(t, err)

	require.Exactly(
		t,
		[]cage_mod.ModReplace{
			{
				Old:     "replace0-old-domain.com/user/proj",
				New:     "replace0-new-domain.com/user/proj",
				Version: "",
			},
			{
				Old:     "replace1-old-domain.com/user/proj",
				New:     "replace1-new-domain.com/user/proj",
				Version: "replace1-version",
			},
			{
				Old:     "replace2-old-domain.com/user/proj",
				New:     "replace2-new-domain.com/user/proj",
				Version: "replace2-version",
			},
			{
				Old:     "replace3-old-domain.com/user/proj",
				New:     "replace3-new-domain.com/user/proj",
				Version: "",
			},
		},
		srcMod.Replaces(),
	)
}

func TestModGo(t *testing.T) {
	srcModFile := cage_file.MustOpenFixturePath(t, "src.mod")
	srcMod, err := cage_mod.NewMod(srcModFile)
	require.NoError(t, err)
	require.Exactly(t, "1.12", srcMod.Go)
}

func TestModString(t *testing.T) {
	srcModFile := cage_file.MustOpenFixturePath(t, "src.mod")
	goldenFile := cage_file.MustOpenFixturePath(t, "src-string.mod")

	srcMod, err := cage_mod.NewMod(srcModFile)
	require.NoError(t, err)

	testkit_require.ReadersMatch(t, "expected", goldenFile, "actual", strings.NewReader(srcMod.String()))
}

func TestGetHash(t *testing.T) {
	srcSumFile := cage_file.MustOpenFixturePath(t, "src.sum")

	srcSum, err := cage_mod.NewSum(srcSumFile)
	require.NoError(t, err)
	srcSumFile.Seek(0, io.SeekStart)

	srcSumFileBytes, err := ioutil.ReadAll(srcSumFile)
	require.NoError(t, err)
	srcSumFileLines := strings.Split(strings.TrimSpace(string(srcSumFileBytes)), "\n")

	parsedLines := srcSum.GetLines()
	require.Exactly(t, len(srcSumFileLines), len(parsedLines))

	for n, line := range parsedLines {
		require.Exactly(t, srcSumFileLines[n], line.Path+" "+line.Version+" "+line.Hash)
		require.Exactly(t, line.Hash, srcSum.GetHash(line.Path, line.Version))
	}
}

func TestSyncRequire(t *testing.T) {
	goldenModFile := cage_file.MustOpenFixturePath(t, "sync_golden.mod")

	srcModFile := cage_file.MustOpenFixturePath(t, "src.mod")
	srcMod, err := cage_mod.NewMod(srcModFile)
	require.NoError(t, err)

	dstModFile := cage_file.MustOpenFixturePath(t, "dst.mod")
	dstMod, err := cage_mod.NewMod(dstModFile)
	require.NoError(t, err)

	cage_testkit.RequireNoErrors(t, cage_mod.SyncRequire(srcMod, dstMod))
	testkit_require.ReadersMatch(t, "golden", goldenModFile, "actual", strings.NewReader(dstMod.String()))
}

func TestCompareSum(t *testing.T) {
	srcSumFile := cage_file.MustOpenFixturePath(t, "src.sum")
	srcSum, err := cage_mod.NewSum(srcSumFile)
	require.NoError(t, err)

	passSumFile := cage_file.MustOpenFixturePath(t, "compare_pass.sum")
	passSum, err := cage_mod.NewSum(passSumFile)
	require.NoError(t, err)

	cage_testkit.RequireNoErrors(t, cage_mod.CompareSum(srcSum, passSum))

	failSumFile := cage_file.MustOpenFixturePath(t, "compare_fail.sum")
	failSum, err := cage_mod.NewSum(failSumFile)
	require.NoError(t, err)

	errs := cage_mod.CompareSum(srcSum, failSum)
	//	require.Len(t, errs, 5)
	require.Exactly(t, errs[0].Error(), "expected hash for [github.com/inconshreveable/mousetrap] version [v1.0.0/go.mod] to be [h1:PxqpIevigyE2G7u3NXJIT2ANytuPF1OarO4DADm73n8=], found [h1:PxqpIev/gyE2G7u3NXJIT2ANytuPF1OarO4DADm73n8=]")
	require.Exactly(t, errs[1].Error(), "expected hash for [github.com/segmentio/ksuid] version [v1.0.2] to be [h1:9yBfKyw4ECGTdALaF09Snw3sLJmYIX6AbPJrAy6MrDc=], found [h1:9yBfKyw4ECGTdALaF09SNw3sLJmYIX6AbPJrAy6MrDc=]")
	require.Exactly(t, errs[2].Error(), "expected hash for [github.com/spf13/pflag] version [v1.0.3] to be [h1:zPAT6CGy6wXeQ7NtTnaTerfKOsV6V6F8agHXFiazDkg=], found [h1:zPAT6CGy6wXeQ7NtTnaTerfKksV6V6F8agHXFiazDkg=]")

}
