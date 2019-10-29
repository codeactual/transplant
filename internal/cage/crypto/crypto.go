// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"hash"
	"io"

	"github.com/pkg/errors"
)

func RandBytes(c int) ([]byte, error) {
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		return []byte{}, errors.Wrapf(err, "failed to generate [%d] random bytes", c)
	}
	return b, nil
}

func RandHexString(bytesLen int) (string, error) {
	input, err := RandBytes(bytesLen)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return hex.EncodeToString(input[:]), nil
}

// ReaderHashSum returns the computed sum from the reader's bytes.
//
// The caller is responsible for reseting the input hasher if it reused.
func ReaderHashSum(h hash.Hash, r io.Reader) ([]byte, error) {
	if _, err := io.Copy(h, r); err != nil {
		return []byte{}, errors.WithStack(err)
	}
	sum := h.Sum(nil)[:]
	return sum, nil
}

// ReaderHashSumsEqual returns the hashes of one or more io.Reader contents and indicates if they were all equal.
//
// It will reset the input hasher prior to each use, but will not reset it before returning.
func ReaderHashSumsEqual(h hash.Hash, readers ...io.Reader) (same bool, sums [][]byte, _ error) {
	same = true
	var prevSum []byte

	for n, r := range readers {
		h.Reset()
		sum, err := ReaderHashSum(h, r)
		if err != nil {
			return false, [][]byte{}, errors.WithStack(err)
		}
		sums = append(sums, sum)
		if same && n > 0 {
			same = bytes.Equal(prevSum, sum)
		}
		prevSum = sum
	}

	return same, sums, nil
}
