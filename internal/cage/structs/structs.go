// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package structs

import (
	vendor_structs "github.com/fatih/structs"
)

type MergeMode int

const (
	MergeModeCombine = iota
	MergeModeOverwrite
)

func ToStringMap(s interface{}) map[string]string {
	m := map[string]string{}
	for k, v := range vendor_structs.Map(s) {
		m[k] = v.(string) //nolint:errcheck
	}
	return m
}

func MergeStringMap(mode MergeMode, maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			var write bool
			if _, ok := merged[k]; ok {
				if mode == MergeModeOverwrite {
					write = true
				}
			} else {
				write = true
			}
			if write {
				merged[k] = v
			}
		}
	}
	return merged
}

func MergeAsStringMap(mode MergeMode, maps ...interface{}) map[string]string {
	var converted []map[string]string
	for _, m := range maps {
		switch t := m.(type) {
		case map[string]string: // ToStringMap requires structs
			converted = append(converted, t)
			continue
		}
		converted = append(converted, ToStringMap(m))
	}
	return MergeStringMap(mode, converted...)
}
