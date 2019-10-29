// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package crypto_test

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_crypto "github.com/codeactual/transplant/internal/cage/crypto"
	testkit_file "github.com/codeactual/transplant/internal/cage/testkit/os/file"
)

type Suite struct {
	suite.Suite
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestCompareFileHashes() {
	t := s.T()

	fixtureDir := "compare_file_hashes"
	_, a0 := testkit_file.FixturePath(t, fixtureDir, "a0")
	a0Hash := "87428fc522803d31065e7bce3cf03fe475096631e5e07bbd7a0fde60c4cf25c7"

	_, a1 := testkit_file.FixturePath(t, fixtureDir, "a1")
	a1Hash := "87428fc522803d31065e7bce3cf03fe475096631e5e07bbd7a0fde60c4cf25c7"

	_, b0 := testkit_file.FixturePath(t, fixtureDir, "b0")
	b0Hash := "0263829989b6fd954f72baaf2fc64bc2e2f01d692d4de72986ea808f6e99813f"

	cases := []struct {
		files        []string
		expectSame   bool
		expectHashes []string
	}{
		{
			files:        []string{},
			expectSame:   true,
			expectHashes: []string{},
		},
		{
			files:      []string{a0},
			expectSame: true,
			expectHashes: []string{
				a0Hash,
			},
		},
		{
			files:      []string{a0, a1},
			expectSame: true,
			expectHashes: []string{
				a0Hash,
				a1Hash,
			},
		},
		{
			files:      []string{a0, b0},
			expectSame: false,
			expectHashes: []string{
				a0Hash,
				b0Hash,
			},
		},
	}

	for n, c := range cases {
		msg := fmt.Sprintf("case index %d", n)
		sha256 := sha256.New()

		var files []io.Reader
		for _, name := range c.files {
			file, err := os.Open(name) // #nosec G304
			require.NoError(t, err)
			files = append(files, file)
		}
		actualSame, actualHashes, err := cage_crypto.ReaderHashSumsEqual(sha256, files...)

		require.NoError(t, err, msg)
		require.Exactly(t, c.expectSame, actualSame, msg)
		require.Exactly(t, len(c.expectHashes), len(actualHashes), msg)

		for e, expectHash := range c.expectHashes {
			require.Exactly(
				t,
				expectHash,
				fmt.Sprintf("%x", actualHashes[e]),
				msg,
			)
		}
	}
}
