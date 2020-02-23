// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package token

import (
	"fmt"
	std_token "go/token"
	"path/filepath"
)

func ShortPositionString(p std_token.Position) string {
	s := filepath.Base(p.Filename)
	if p.Line > 0 {
		s += fmt.Sprintf(" L%d", p.Line)
	}
	if p.Column > 0 {
		s += fmt.Sprintf(" C%d", p.Column)
	}
	return s
}
