// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package io

import (
	"fmt"
	std_io "io"
	"os"

	"github.com/pkg/errors"
)

type SyncCloser interface {
	Sync() error
	Close() error
}

// CloseOrStderr attempts to close a io.Close implementation and outputs to
// standard error on failure.
func CloseOrStderr(c std_io.Closer, id string) {
	err := c.Close()
	if err != nil {
		// Use %+v to support extended output from packages like github.com/pkg/errors
		fmt.Fprintf(os.Stderr, "failed to close io.Closer [%s]: %+v\n", id, err)
	}
}

func SyncClose(sc SyncCloser, id string) error {
	if err := sc.Sync(); err != nil {
		return errors.Wrapf(err, "failed to sync before closing [%s]", id)
	}
	if err := sc.Close(); err != nil {
		return errors.Wrapf(err, "failed to close [%s]", id)
	}
	return nil
}

func Rewind(s std_io.Seeker) error {
	_, err := s.Seek(0, std_io.SeekStart)
	return errors.WithStack(err)
}
