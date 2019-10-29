// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package os

import (
	std_os "os"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

func StringPidToProcess(s string) (p *std_os.Process, err error) {
	pid, err := strconv.Atoi(s)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert string PID [%s] to int", s)
	}
	return FindProcess(pid)
}

func FindProcess(pid int) (p *std_os.Process, err error) {
	p, err = std_os.FindProcess(pid)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find process [%d]", pid)
	}

	// "On Unix systems, FindProcess always succeeds and returns a Process for the given pid, regardless of whether the process exists." (1.10.1)
	// https://golang.org/pkg/os/#FindProcess
	// https://stackoverflow.com/questions/15204162/check-if-a-process-exists-in-go-way?
	err = p.Signal(syscall.Signal(0))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return p, nil
}

// AppendEnv appends a string to an environment variable value if it is not already present.
func AppendEnv(key, valuePart, delimiter string) (origValue, newValue string, err error) {
	origValue = std_os.Getenv(key)
	newValue = origValue

	if strings.HasPrefix(origValue, valuePart) || strings.Contains(origValue, delimiter+valuePart) {
		return origValue, origValue, nil
	}

	if origValue != "" {
		newValue += delimiter
	}
	newValue += valuePart

	if updateErr := std_os.Setenv(key, newValue); updateErr != nil {
		return "", "", errors.Wrapf(updateErr, "failed to append [%s] to environment variable [%s]", valuePart, key)
	}

	return origValue, newValue, nil
}
