// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package filepath

import (
	"fmt"
	std_filepath "path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/pkg/errors"
)

// MatchAnyInput defines a path-matching operation which considers whether a single candidate path
// matches at least one inclusion pattern while matching zero exclusion patterns.
type MatchAnyInput struct {
	// Name is the candidate path used in evaluation against Include and Exclude globs.
	Name string

	// Include holds patterns which must match against Name.
	//
	// It can be combined with Exclude patterns.
	Include []string

	// Exclude holds patterns which cannot match against Name.
	//
	// It can be combined with Include patterns.
	Exclude []string
}

// MatchAnyOutput describes the result of a MatchAnyInput evalation of a candidate path.
type MatchAnyOutput struct {
	// Match is true if MatchAnyInput.Name is considered a match.
	Match bool

	// Include is the first MatchAnyInput.Include pattern that led to a match.
	Include string

	// Exclude is the first MatchAnyExput.Exclude pattern that prevented a match.
	Exclude string
}

// PathMatchAny evaluates whether a single candidate path matches at least one inclusion pattern
// while matching zero exclusion patterns.
func PathMatchAny(in MatchAnyInput) (out MatchAnyOutput, err error) {
	if in.Name == "" || len(in.Include) == 0 {
		return out, nil
	}

	for _, pattern := range in.Exclude {
		match, matchErr := doublestar.PathMatch(pattern, in.Name)
		if matchErr != nil {
			return out, errors.Wrapf(matchErr, "bad exclude pattern [%s]", pattern)
		}

		if match {
			out.Exclude = pattern
			return out, nil
		}
	}

	for _, pattern := range in.Include {
		match, matchErr := doublestar.PathMatch(pattern, in.Name)
		if matchErr != nil {
			return out, errors.Wrapf(matchErr, "bad include pattern [%s]", pattern)
		}

		if match {
			out.Match = true
			out.Include = pattern
			break
		}
	}

	return out, nil
}

// Glob defines an inclusion pattern for GlobAny searches.
type Glob struct {
	// Pattern is a pattern to match against a candidate path.
	Pattern string

	// Root is an optional prefix prepended to Glob in case the latter is a relative path.
	Root string
}

func (i Glob) String() string {
	return fmt.Sprintf("glob [%s] root [%s]", i.Pattern, i.Root)
}

// GlobAnyInput defines a path-matching operation which considers whether candidate paths
// match at least one inclusion pattern while matching zero exclusion patterns.
type GlobAnyInput struct {
	// Include selects the patterns used to search for candidate paths.
	Include []Glob

	// Exclude selects patterns used to disqualify candidate paths.
	Exclude []Glob
}

// GlobAnyOutput describes the result of a GlobAnyInput evalation of candidate paths.
type GlobAnyOutput struct {
	// Include holds absolute paths indexed by the patterns that matched them.
	//
	// If multiple patterns match, the value is the first encountered.
	Include map[string]Glob

	// Exclude holds absolute paths indexed by the patterns that matched them.
	//
	// If multiple patterns match, the value is the first encountered.
	Exclude map[string]Glob
}

func (o GlobAnyOutput) String() (s string) {
	for p, include := range o.Include {
		s += fmt.Sprintf("include: path [%s] -> %s\n", p, include)
	}
	for p, exclude := range o.Exclude {
		s += fmt.Sprintf("exclude: path [%s] -> %s\n", p, exclude)
	}
	return s
}

// GlobAny evaluates whether candidate paths match at least one inclusion pattern
// while matching zero exclusion patterns.
//
// All GlobAnyInput.Include patterns will be used to discover candidate paths.
// All GlobAnyInput.Include patterns will be used to disqualify every candidate path.
func GlobAny(in GlobAnyInput) (out GlobAnyOutput, err error) {
	var absErr error

	out.Include = make(map[string]Glob)
	out.Exclude = make(map[string]Glob)

	if len(in.Include) == 0 {
		return GlobAnyOutput{}, nil
	}

	for _, include := range in.Include {
		var includePattern string

		// Apply the configured prefix or working directory if the pattern is relative.
		if std_filepath.IsAbs(include.Pattern) {
			includePattern = include.Pattern
		} else if include.Root == "" {
			if includePattern, absErr = std_filepath.Abs(include.Pattern); absErr != nil {
				return GlobAnyOutput{}, errors.Wrapf(absErr, "failed to get absolute path [%s]", include.Pattern)
			}
		} else {
			includePattern = std_filepath.Join(include.Root, include.Pattern)
		}

		matches, globErr := doublestar.Glob(includePattern)
		if globErr != nil {
			return GlobAnyOutput{}, errors.Wrapf(globErr, "bad include pattern [%s]", includePattern)
		}

		for _, name := range matches {
			if name, absErr = std_filepath.Abs(name); absErr != nil {
				return GlobAnyOutput{}, errors.Wrapf(absErr, "failed to get absolute path [%s]", name)
			}

			// Only collect the first inclusion/exclusion patterns confirmed to be effective.
			if _, found := out.Include[name]; found {
				continue
			}
			if _, found := out.Exclude[name]; found {
				continue
			}

			// Verify the candidate match does not match against an exclusion pattern.

			var accepted int

			for _, exclude := range in.Exclude {
				// Apply the configured prefix or working directory if the pattern is relative.
				var excludePattern string
				if std_filepath.IsAbs(exclude.Pattern) {
					excludePattern = exclude.Pattern
				} else if exclude.Root == "" {
					if excludePattern, absErr = std_filepath.Abs(exclude.Pattern); absErr != nil {
						return GlobAnyOutput{}, errors.Wrapf(absErr, "failed to get absolute path [%s]", exclude.Pattern)
					}
				} else {
					excludePattern = std_filepath.Join(exclude.Root, exclude.Pattern)
				}

				excludeMatch, matchErr := PathMatchAny(MatchAnyInput{Name: name, Include: []string{excludePattern}})
				if matchErr != nil {
					return GlobAnyOutput{}, errors.Wrapf(matchErr, "bad exclude pattern [%s]", includePattern)
				}

				if excludeMatch.Match { // return the pattern that caused the exclusion
					out.Exclude[name] = exclude
					break
				}

				accepted++
			}

			if accepted == len(in.Exclude) {
				out.Include[name] = include // return the pattern that caused the inclusion
			}
		}
	}

	return out, nil
}
