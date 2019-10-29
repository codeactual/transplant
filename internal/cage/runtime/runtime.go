// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runtime

import (
	"regexp"
	std_runtime "runtime"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

// versionSemverRe matches the semver part of runtime.Version() output.
// Requires FindString or other function that only returns the leftmost match.
var versionSemverRe *regexp.Regexp

func init() {
	versionSemverRe = regexp.MustCompile("[0-9.]+")
}

func VersionAtLeast(min string) (bool, error) {
	cStr := ">= " + min
	c, err := semver.NewConstraint(cStr)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create new constraint from string [%s]", cStr)
	}
	vStr, err := VersionSemver()
	if err != nil {
		return false, errors.WithStack(err)
	}
	v, err := semver.NewVersion(vStr)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse runtime version [%s]", vStr)
	}
	return c.Check(v), nil
}

func VersionSemver() (string, error) {
	full := std_runtime.Version()
	v := versionSemverRe.FindString(full)
	if v == "" {
		return "", errors.Errorf("failed to find runtime version in [%s]", full)
	}
	return v, nil
}
