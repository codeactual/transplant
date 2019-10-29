// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package reflect

import std_reflect "reflect"

// GetFieldTag returns the value of a struct field's tag.
func GetFieldTag(val interface{}, field string, key string) string {
	t := std_reflect.TypeOf(val)
	f, found := t.FieldByName(field)
	if found {
		return f.Tag.Get(key)
	}
	return ""
}
