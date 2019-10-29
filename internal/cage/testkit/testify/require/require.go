// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package require

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	std_require "github.com/stretchr/testify/require"

	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	testkit "github.com/codeactual/transplant/internal/cage/testkit"
)

// ReaderLineReplacer receives an "actual" line, from an expected vs. actual test comparison, and optionally
// returns a new "actual" line value.
//
// For example, if the actual values will change over time (e.g. "go <semver>" strings in new go.mod files),
// a ReaderLineReplacer can allow test fixtures to have out-dated semver values (which are not the SUT) by
// replacing the dynamic value with the expected static value.
//
// The expectedName and actualName values are the reader (e.g. file) names.
type ReaderLineReplacer func(expectedName, actualName, actualLine string) (replacement string)

func FilesMatch(t *testing.T, expectedName, actualName string, lrs ...ReaderLineReplacer) {
	expected, err := os.Open(expectedName) // #nosec G304
	std_require.NoError(t, errors.WithStack(err))

	actual, err := os.Open(actualName) // #nosec G304
	std_require.NoError(t, errors.WithStack(err))

	ReadersMatch(t, expectedName, expected, actualName, actual, lrs...)
}

func ReadersMatch(t *testing.T, expectedName string, expected io.Reader, actualName string, actual io.Reader, lrs ...ReaderLineReplacer) {
	expectedBytes, err := ioutil.ReadAll(expected)
	std_require.NoError(t, errors.WithStack(err))
	expectedLines := strings.Split(string(expectedBytes), "\n")

	actualBytes, err := ioutil.ReadAll(actual)
	std_require.NoError(t, errors.WithStack(err))
	actualLines := strings.Split(string(actualBytes), "\n")

	// Allow tests to work around lines which may change over time or differ between platforms.
	for _, lr := range lrs {
		for n := range actualLines {
			actualLines[n] = lr(expectedName, actualName, actualLines[n])
		}
	}

	// Add newlines for long paths
	std_require.Exactly(t, expectedLines, actualLines, "\nexpected [%s]\nactual [%s]", expectedName, actualName)
}

func DirsMatch(t *testing.T, expectedName, actualName string, lrs ...ReaderLineReplacer) {
	var expectedRelPaths, expectedAbsPaths, actualRelPaths, actualAbsPaths []string

	assertId := fmt.Sprintf("expect dir: %s\nactual dir: %s\n", expectedName, actualName)

	errs := cage_filepath.WalkAbs(expectedName, func(p string, fi os.FileInfo, walkErr error) []error {
		if walkErr != nil {
			return []error{errors.WithStack(walkErr)}
		}

		if !fi.IsDir() {
			expectedAbsPaths = append(expectedAbsPaths, p)

			relPath, err := filepath.Rel(expectedName, p)
			std_require.NoError(t, errors.WithStack(err))
			expectedRelPaths = append(expectedRelPaths, relPath)
		}

		return nil
	})
	testkit.RequireNoErrors(t, errs)

	errs = cage_filepath.WalkAbs(actualName, func(p string, fi os.FileInfo, walkErr error) []error {
		if walkErr != nil {
			return []error{errors.WithStack(walkErr)}
		}

		if !fi.IsDir() {
			actualAbsPaths = append(actualAbsPaths, p)

			relPath, err := filepath.Rel(actualName, p)
			std_require.NoError(t, errors.WithStack(err))
			actualRelPaths = append(actualRelPaths, relPath)
		}

		return nil
	})
	testkit.RequireNoErrors(t, errs)

	cage_strings.SortStable(expectedRelPaths)
	cage_strings.SortStable(actualRelPaths)
	std_require.Exactly(t, expectedRelPaths, actualRelPaths, assertId)

	for n := 0; n < len(expectedRelPaths); n++ {
		FilesMatch(t, expectedAbsPaths[n], actualAbsPaths[n], lrs...)
	}
}

func StringSortedSliceExactly(t *testing.T, expected []string, actual []string) {
	e := make([]string, len(expected))
	copy(e, expected[:])
	cage_strings.SortStable(e)

	a := make([]string, len(actual))
	copy(a, actual[:])
	cage_strings.SortStable(a)

	StringSliceExactly(t, e, a)
}

func StringSliceExactly(t *testing.T, expected []string, actual []string) {
	std_require.Exactly(t, expected, actual, fmt.Sprintf(
		"expect: %s\nactual: %s\n", spew.Sdump(expected), spew.Sdump(actual),
	))
}

func MatchRegexp(t *testing.T, subject string, expectedReStr ...string) {
	for _, reStr := range expectedReStr {
		std_require.True(
			t,
			regexp.MustCompile(reStr).MatchString(subject),
			fmt.Sprintf("subject [%s]\nregexp [%s]", subject, reStr),
		)
	}
}

func StringContains(t *testing.T, subject string, expected ...string) {
	for _, e := range expected {
		std_require.True(
			t,
			strings.Contains(subject, e),
			fmt.Sprintf("subject [%s]\nsubstring [%s]", subject, e),
		)
	}
}

func ReaderStringContains(t *testing.T, r io.Reader, expected ...string) {
	readBytes, err := ioutil.ReadAll(r)
	std_require.NoError(t, errors.WithStack(err))
	StringContains(t, string(readBytes), expected...)
}

func FileStringContains(t *testing.T, name string, expected ...string) {
	f, err := os.Open(name) // #nosec G304
	std_require.NoError(t, errors.WithStack(err))
	defer func() {
		std_require.NoError(t, f.Close(), "failed to close file: "+name)
	}()
	ReaderStringContains(t, f, expected...)
}
