// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package why

import (
	"fmt"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	"io"
	"path/filepath"
	"strings"
)

// Log holds file/dir activity messages which support `{egress,ingress} why` queries.
//
// The messages are indexed by file/dir absolute paths.
type Log map[string][]string

func FileQuery(log Log, absPath string) (messages []string) {
	if _, ok := log[absPath]; ok {
		messages = make([]string, len(log[absPath]))
		copy(messages, log[absPath][:])
		return messages
	}
	return []string{}
}

func FileSiblingQuery(log Log, absPath string) []string {
	var siblings []string
	for k := range log {
		if strings.HasPrefix(k, absPath) {
			siblings = append(siblings, k)
		}
	}
	cage_strings.SortStable(siblings)
	return siblings
}

func PrintLog(w io.Writer, log Log, absPath string) {
	fileLogs := FileQuery(log, absPath)
	if len(fileLogs) > 0 {
		fmt.Fprintf(w, "Activity for %s\n", absPath)
		for _, l := range fileLogs {
			fmt.Fprintln(w, "\t * "+l)
		}
	} else {
		dir := filepath.Dir(absPath)
		if siblings := FileSiblingQuery(log, dir); len(siblings) > 0 {
			fmt.Fprintf(w, "directory [%s] inspection did not include that file, but did include these files:\n", dir)
			for _, f := range siblings {
				fmt.Fprintln(w, "\t * "+f)
			}
		} else {
			fmt.Fprintf(w, "Directory [%s] was never inspected.\n", dir)
		}
	}
}
