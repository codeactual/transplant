// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package list

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	cage_exec "github.com/codeactual/transplant/internal/cage/os/exec"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// Module holds the per-module fields unmarshaled from "go list" output.
type Module struct {
	// Path is an import path.
	Path string

	// Version is a semver string.
	Version string
}

// ModuleSet holds Module values with unique import paths.
type ModuleSet struct {
	// list is indexed by Module.Path values.
	list map[string]Module

	// paths holds Module.Path values.
	paths *cage_strings.Set
}

// NewModuleSet returns an initialized ModuleSet.
func NewModuleSet() *ModuleSet {
	return &ModuleSet{
		list:  make(map[string]Module),
		paths: cage_strings.NewSet(),
	}
}

// Add updates the set, either creating a new entry or overwriting an existing one with the same import path.
func (l *ModuleSet) Add(m Module) *ModuleSet {
	l.list[m.Path] = m
	l.paths.Add(m.Path)
	return l
}

// GetPaths returns all import paths.
func (l *ModuleSet) GetPaths() *cage_strings.Set {
	return l.paths.Copy()
}

// GetByPath returns the Module associated with the input import path, if found. Otherwise it returns nil.
func (l *ModuleSet) GetByPath(p string) *Module {
	m, ok := l.list[p]
	if ok {
		return &m
	}
	return nil
}

// Query stores options for, and executes, "go list" in order to produce ModuleSet values.
type Query struct {
	// dir is the working directory from which to run "go list".
	dir string

	// executor executes os/exec commands.
	executor cage_exec.Executor

	// modules holds the value of an "-m" option of "go list".
	modules string
}

// NewQuery returns an initialized Query.
func NewQuery(executor cage_exec.Executor, dir string) *Query {
	return &Query{
		executor: executor,
		dir:      dir,
	}
}

// Modules selects an "-m" option of "go list".
func (q *Query) Modules(optVal string) *Query {
	q.modules = optVal
	return q
}

// AllModules ia an alias for an "-m all" option of "go list".
func (q *Query) AllModules() *Query {
	return q.Modules("all")
}

// Run executes "go list" with the selected options.
func (q *Query) Run(ctx context.Context) (mods *ModuleSet, err error) {
	mods = NewModuleSet()
	args := []string{"list"}

	if q.modules != "" {
		args = append(args, "-m", q.modules)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Dir = q.dir
	stdout, stderr, _, err := q.executor.Buffered(ctx, cmd)

	ctxErr := ctx.Err()
	if ctxErr != nil {
		err = ctxErr
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to run 'go list' with query [%s] : %s", strings.Join(args, " "), stderr.String())
	}

	results := stdout.Bytes()
	lines := bytes.Split(results, []byte("\n"))

	for _, line := range lines {
		parts := strings.Split(string(line), " ")
		if len(parts) == 2 {
			mods.Add(Module{Path: parts[0], Version: parts[1]})
		}
	}

	return mods, nil
}
