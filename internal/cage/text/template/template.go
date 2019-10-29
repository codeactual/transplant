// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package template

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"

	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	cage_structs "github.com/codeactual/transplant/internal/cage/structs"
)

func ExecuteBuffered(body string, data interface{}) (b bytes.Buffer, err error) {
	t, err := template.New("ExecuteBuffered").Parse(body)
	if err != nil {
		return bytes.Buffer{}, errors.Wrapf(err, "failed to parse template [%s]", body)
	}
	err = t.Execute(&b, data)
	if err != nil {
		return bytes.Buffer{}, errors.Wrapf(err, "failed to parse template [%s]", body)
	}
	return b, nil
}

// StringMapBuilder supports use cases where a given template string will be expanded
// multiple times with different sets of key/value pairs, and the data map is composed
// from combinations of map[string]string and struct values which are convertible to
// the former.
//
// An empty string value will be remain empty if its key is registered by SetExpectedKey.
// If the key "K" is not expected, the value will be replaced by "{{.K}}" to allow a later
// expansion to provide a value.
type StringMapBuilder struct {
	data         map[string]string
	expectedKeys *cage_strings.Set
}

func NewStringMapBuilder() *StringMapBuilder {
	return &StringMapBuilder{
		data:         map[string]string{},
		expectedKeys: cage_strings.NewSet(),
	}
}

// SetExpectedKey configures the builder to replace the input keys in Map
// instead of retaining them as placeholders for a future expanasion when
// they are expected.
func (d *StringMapBuilder) SetExpectedKey(k ...string) *StringMapBuilder {
	d.expectedKeys.AddSlice(k)
	return d
}

// Merge augments the key/value map with one or more map[string]map values.
func (d *StringMapBuilder) Merge(mode cage_structs.MergeMode, maps ...interface{}) *StringMapBuilder {
	var currentMap []interface{}
	currentMap = append(currentMap, d.data)
	d.data = cage_structs.MergeAsStringMap(mode, append(currentMap, maps...)...)
	return d
}

// Map returns the currently built key/value map.
func (d *StringMapBuilder) Map() map[string]string {
	m := map[string]string{}
	for k, v := range d.data {
		if v == "" && !d.expectedKeys.Contains(k) {
			m[k] = "{{." + k + "}}"
		} else {
			m[k] = v
		}
	}
	return m
}

// ExpandFromStringMap expands placeholders in each input string based on the input key/value map.
func ExpandFromStringMap(data map[string]string, toExpand ...*string) error {
	for _, s := range toExpand {
		buf, err := ExecuteBuffered(*s, data)
		if err != nil {
			return errors.Wrapf(err, "failed to expand template variables in string [%s]", *s)
		}
		*s = buf.String()
	}
	return nil
}
