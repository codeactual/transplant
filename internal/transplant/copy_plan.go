// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// CopyFileError describes an error encountered while processing a specific file, but the error
// is collected for inclusion in CopyPlan instead of propagating down the stack.
type CopyFileError struct {
	// Name is an absolute path.
	Name string

	// Err is an Error() string.
	Err string
}

// CopyPlan is written to a file selected by the --plan CLI flag.
type CopyPlan struct {
	// Add holds the absolute paths of all files to be added to Ops.To.FilePath.
	Add []string `json:",omitempty" toml:",omitempty" yaml:"Add,omitempty" `

	// Overwrite holds the absolute paths of all files to be overwritten in Ops.To.FilePath.
	Overwrite []string `json:",omitempty" toml:",omitempty" yaml:"Overwrite,omitempty"`

	// Remove holds the absolute paths of all files to be removed from Ops.To.FilePath.
	Remove []string `json:",omitempty" toml:",omitempty" yaml:"Remove,omitempty"`

	// PruneGlobalIds describes the global identifiers omitted from Ops.Dep packages because they were not
	// direct/transitive dependencies of packages under Ops.From.FilePath.
	//
	// Global identifier name format: <absolute path to package directory>.<package name>.<identifier name>
	// Files are identified by their absolute paths.
	PruneGlobalIds []string `json:",omitempty" toml:",omitempty" yaml:"PruneGlobalIds,omitempty"`

	// PruneGoFiles holds files which were omitted from the copy because they did not contain a
	// direct/transitive dependency of packages under Ops.From.FilePath.
	PruneGoFiles []string `json:",omitempty" toml:",omitempty" yaml:"PruneGoFiles,omitempty"`

	// GoFormatErr describes Ops.From.CopyOnlyFilePath Go files which could not be automatically
	// formatted by go/format.Source, e.g. due to a syntax error.
	//
	// Those errors are collected here, instead of causing the overall copy operation to fail, to support
	// fixtures with intentional defects. It also supports the general distinction between CopyOnlyFilePath
	// patterns and others like GoFilePath which require error-free parsing.
	GoFormatErr []CopyFileError `json:",omitempty" toml:",omitempty" yaml:"GoFormatErr,omitempty"`

	// RenameNotFound holds absolute paths of RenameFilePath targets which were not found.
	//
	// Paths are collected instead of causing the copy operation to fail to support cases where the
	// target file is not always present.
	RenameNotFound []string `json:",omitempty" toml:",omitempty" yaml:"RenameNotFound,omitempty"`

	// GoModVendor is true if `go mod vendor` is called.
	GoModVendor bool `yaml:"GoModVendor"`

	// Env holds environment details which affect Copier behavior.
	Env map[string]string `yaml:"Env"`

	// StagePath holds the absolute path to the staging file tree where the copy is generated as a whole and then
	// copied at the end of the process.
	StagePath string `yaml:"StagePath"`

	// OverwriteSkip holds the absolute paths of all files not overwritten in Ops.To.FilePath
	// because the destination's content is the same.
	OverwriteSkip []string `json:"-" toml:"-" yaml:"-"`
}

func (p *CopyPlan) String() string {
	var b strings.Builder

	unusedActions := cage_strings.NewSet()

	writeSection := func(title, adverb string, items []string) {
		_, _ = b.WriteString("---\n")
		if len(items) > 0 {
			_, _ = b.WriteString(title + ":\n")
			for _, v := range items {
				_, _ = b.WriteString("\t" + v + "\n")
			}
		} else {
			unusedActions.Add(adverb)
		}
	}

	writeSection("Add", "added", p.Add)
	writeSection("Overwrite", "overwritten", p.Overwrite)
	writeSection("Remove", "removed", p.Remove)
	writeSection("PruneGlobalIds", "pruned", p.PruneGlobalIds)

	if unusedActions.Len() > 0 {
		_, _ = b.WriteString("---\n")
		b.WriteString(fmt.Sprintf("No files will be: %s\n", strings.Join(unusedActions.SortedSlice(), ", ")))
	}

	return b.String()
}

func (p *CopyPlan) WriteFile(name string, optFields *cage_strings.Set) (err error) {
	source := *p

	// Omit these by default.
	if optFields == nil || !optFields.Contains("PruneGlobalIds") {
		source.PruneGlobalIds = nil
	}
	if optFields == nil || !optFields.Contains("PruneGoFiles") {
		source.PruneGoFiles = nil
	}

	var fileBytes []byte

	switch filepath.Ext(name) {
	case ".json":
		fileBytes, err = json.MarshalIndent(source, "", "  ")
	case ".toml":
		fileBytes, err = toml.Marshal(source)
	default:
		fileBytes, err = yaml.Marshal(source)
	}

	if err != nil {
		return errors.Wrap(err, "failed to marshal CopyPlan")
	}

	if err = ioutil.WriteFile(name, fileBytes, newFileMode); err != nil {
		return errors.Wrapf(err, "failed to write CopyPlan to file [%s]", name)
	}

	return nil
}
