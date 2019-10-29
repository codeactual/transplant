// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cobra

import (
	"context"
	"os"

	"github.com/pkg/errors"
	std_cobra "github.com/spf13/cobra"

	"github.com/codeactual/transplant/internal/cage/cli/handler"
	"github.com/codeactual/transplant/internal/ldflags"
)

type Init struct {
	Cmd *std_cobra.Command

	Ctx context.Context

	EnvPrefix string

	// Mixins defines all Mixin implementations for automatic integration into stages
	// of the command run, e.g. binding flags.
	Mixins []handler.Mixin
}

// Cobra defines the handler behaviors implemented by each the Handler structs
// exported by each sub-command package.
//
// It has several intents:
//
// * Minimize boilerplate in sub-command packages.
// * Remove cobra knowledge from all Run methods for testability.
type Handler interface {
	handler.Responder

	// BindFlags optionally defines CLI flags.
	BindFlags(cmd *std_cobra.Command) (requiredFlags []string)

	// Init defines the cobra command object, prefix for environment variable configs, etc.
	Init() Init

	// Run is called when all bound flags are available.
	Run(ctx context.Context, args []string)
}

// NewCommand accepts an initial command object, created in a sub-commands Init method,
// and finishes preparation of the object.
//
// For example, it automatically calls the handler's BindFlags method.
func NewCommand(h Handler, init Init) *std_cobra.Command {
	if h.Out() == nil {
		h.SetOut(os.Stdout)
	}
	if h.Err() == nil {
		h.SetErr(os.Stderr)
	}
	// h.SetIn is not called because h.In will automatically return it if
	// needed to handle a pipe (and h.SetIn was not already used to select something
	// other than os.Stdin, e.g. for a test).

	config := Config{}
	config.Init(init.EnvPrefix, init.Cmd)

	init.Cmd.PreRunE = func(cmd *std_cobra.Command, args []string) error {
		if err := config.PreRun(); err != nil {
			return errors.Wrap(err, "failed to configure the command")
		}

		for _, mixin := range init.Mixins {
			if preRunner, ok := mixin.(handler.PreRun); ok {
				if err := preRunner.PreRun(init.Ctx, args); err != nil {
					return errors.WithStack(err)
				}
			}
		}

		if preRunner, ok := h.(handler.PreRun); ok {
			if err := preRunner.PreRun(init.Ctx, args); err != nil {
				return errors.WithStack(err)
			}
		}

		return nil
	}

	// Define a thin adapter to allow Run methods to have no knowledge of cobra.
	init.Cmd.Run = func(cmd *std_cobra.Command, args []string) {
		h.Run(init.Ctx, args)
	}

	init.Cmd.PostRun = func(cmd *std_cobra.Command, args []string) {
		for _, mixin := range init.Mixins {
			if postRunner, ok := mixin.(handler.PostRun); ok {
				postRunner.PostRun(init.Ctx)
			}
		}

		if postRunner, ok := h.(handler.PostRun); ok {
			postRunner.PostRun(init.Ctx)
		}
	}

	requiredFlags := h.BindFlags(init.Cmd)

	for _, mixin := range init.Mixins {
		requiredFlags = append(requiredFlags, mixin.BindCobraFlags(init.Cmd)...)

		if m, ok := mixin.(handler.Responder); ok {
			if m.Out() == nil {
				m.SetOut(os.Stdout)
			}
			if m.Err() == nil {
				m.SetErr(os.Stderr)
			}
			// m.SetIn is not called for the same reason handler.SetIn is not called below.
			// See the comments about the latter.
		}
	}

	// If needed later, this behavior could be disabled via new Init field.
	config.BindEnvToAllFlags(init.Cmd)

	config.SetRequired(requiredFlags...)

	// Don't always display the error returned by handler.Run and the usage info.
	// Let handlers/mixins control that class of output.
	init.Cmd.SilenceErrors = true
	init.Cmd.SilenceUsage = true

	return init.Cmd
}

// NewHandler is called by parent commands in order to create a new sub-command "defined" by
// the given handler.
//
// The process of defining the sub-command relies on the handler implementing the
// Cobra interface from this package, e.g. Init method that provides the initial
// configuration elements like the command name, description, etc. within the cobra object.
//
//     subCmd := handler.New(&gc.Handler{})
//     parent.AddCommand(subCmd)
func NewHandler(h Handler) *std_cobra.Command {
	init := h.Init()
	if init.Ctx == nil {
		init.Ctx = context.Background()
	}
	if init.Cmd.Version == "" {
		init.Cmd.Version = ldflags.Version
	}
	return NewCommand(h, init)
}
