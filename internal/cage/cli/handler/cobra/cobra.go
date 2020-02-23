// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cobra

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	std_cobra "github.com/spf13/cobra"

	"github.com/codeactual/transplant/internal/cage/cli/handler"
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
	handler.Session

	// BindFlags optionally defines CLI flags.
	BindFlags(*std_cobra.Command) (requiredFlags []string)

	// Init defines the cobra command object, prefix for environment variable configs, etc.
	Init() Init

	// Run is called when all bound flags are available.
	Run(context.Context, handler.Input)
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
		// Allow Config.PreRun to indicate when the error should be propagated or handled by
		// simply printing the usage content. This retains compatibility with
		// programs whose main function panics on the root command's Execute call,
		// but avoids panics when the cause appears to be invalid user input.
		if showUsage, err := config.PreRun(); err != nil {
			if showUsage {
				fmt.Fprintln(h.Err(), cmd.UsageString()+"\n"+err.Error())
				os.Exit(1)
			} else {
				return errors.Wrap(err, "failed to configure the command")
			}
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
		input := handler.Input{
			Args: args,
		}

		argsLenAtDash := cmd.ArgsLenAtDash() // -1 if dash not present

		if argsLenAtDash == 0 {
			input.ArgsAfterDash = make([]string, len(args)-argsLenAtDash)
			copy(input.ArgsAfterDash, args[argsLenAtDash:])
		} else if argsLenAtDash == -1 {
			input.ArgsBeforeDash = make([]string, len(args))
			copy(input.ArgsBeforeDash, args)
		} else {
			input.ArgsBeforeDash = make([]string, argsLenAtDash)
			copy(input.ArgsBeforeDash, args[:argsLenAtDash])

			input.ArgsAfterDash = make([]string, len(args)-argsLenAtDash)
			copy(input.ArgsAfterDash, args[argsLenAtDash:])
		}

		h.Run(init.Ctx, input)
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

		if m, ok := mixin.(handler.Session); ok {
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

	// Ideally we would use a SetUsageString but SetUsageTemplate has the same effect in this case.
	usageTmpl := init.Cmd.UsageString()
	if init.Cmd.HasAvailableFlags() {
		usageTmpl += fmt.Sprintf("\nSet flags via command line or environment variable (%s_KEY_NAME).\n", init.EnvPrefix)

		reqKeys := config.MissingRequiredKeyStrings()
		if len(reqKeys) > 0 {
			usageTmpl += "\n" + "Required:"
			for _, s := range reqKeys {
				usageTmpl += "\n\t" + s
			}
		}
	}
	init.Cmd.SetUsageTemplate(usageTmpl + "\n")

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
		init.Cmd.Version = handler.Version()
	}
	return NewCommand(h, init)
}
