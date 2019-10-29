// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package os_test

import (
	cage_os "github.com/codeactual/transplant/internal/cage/os"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendEnv(t *testing.T) {
	key := "TestAppendEnv"
	require.NoError(t, os.Unsetenv(key))

	delimiter := " "

	origValue, newValue, err := cage_os.AppendEnv(key, "apple", delimiter)
	require.NoError(t, err)
	require.Exactly(t, "", origValue)
	require.Exactly(t, "apple", newValue)
	require.Exactly(t, "apple", os.Getenv(key))

	// redundant append has no effect

	origValue, newValue, err = cage_os.AppendEnv(key, "apple", delimiter)
	require.NoError(t, err)
	require.Exactly(t, "apple", origValue)
	require.Exactly(t, "apple", newValue)
	require.Exactly(t, "apple", os.Getenv(key))

	// new value added

	origValue, newValue, err = cage_os.AppendEnv(key, "bread", delimiter)
	require.NoError(t, err)
	require.Exactly(t, "apple", origValue)
	require.Exactly(t, "apple"+delimiter+"bread", newValue)
	require.Exactly(t, "apple"+delimiter+"bread", os.Getenv(key))

	// redundant append has no effect

	origValue, newValue, err = cage_os.AppendEnv(key, "apple", delimiter)
	require.NoError(t, err)
	require.Exactly(t, "apple"+delimiter+"bread", origValue)
	require.Exactly(t, "apple"+delimiter+"bread", newValue)
	require.Exactly(t, "apple"+delimiter+"bread", os.Getenv(key))

	// redundant append has no effect

	origValue, newValue, err = cage_os.AppendEnv(key, "bread", delimiter)
	require.NoError(t, err)
	require.Exactly(t, "apple"+delimiter+"bread", origValue)
	require.Exactly(t, "apple"+delimiter+"bread", newValue)
	require.Exactly(t, "apple"+delimiter+"bread", os.Getenv(key))
}
