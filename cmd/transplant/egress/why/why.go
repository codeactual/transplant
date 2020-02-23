// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package why

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/codeactual/transplant/cmd/transplant/why"
	"github.com/codeactual/transplant/internal/cage/cli/handler"
	handler_cobra "github.com/codeactual/transplant/internal/cage/cli/handler/cobra"
	log_zap "github.com/codeactual/transplant/internal/cage/cli/handler/mixin/log/zap"
	cage_errors "github.com/codeactual/transplant/internal/cage/errors"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_reflect "github.com/codeactual/transplant/internal/cage/reflect"
	"github.com/codeactual/transplant/internal/transplant"
)

const exampleText = "file /path/to/file"

// Handler defines the sub-command flags and logic.
type Handler struct {
	handler.Session

	ConfigFile string `usage:"YAML configuration file"`
	Op         string `usage:"Ops.Id value from the config file"`

	Log *log_zap.Mixin

	config transplant.Config
}

// Init defines the command, its environment variable prefix, etc.
//
// It implements cli/handler/cobra.Handler.
func (h *Handler) Init() handler_cobra.Init {
	h.Log = &log_zap.Mixin{}

	return handler_cobra.Init{
		Cmd: &cobra.Command{
			Use:     "why",
			Short:   "Display a log of audit/copy activity related to the file/dir path",
			Example: exampleText,
		},
		EnvPrefix: "TRANSPLANT",
		Mixins: []handler.Mixin{
			h.Log,
		},
	}
}

// BindFlags binds the flags to Handler fields.
//
// It implements cli/handler/cobra.Handler.
func (h *Handler) BindFlags(cmd *cobra.Command) []string {
	cmd.Flags().StringVarP(&h.ConfigFile, "config", "", "", cage_reflect.GetFieldTag(*h, "ConfigFile", "usage"))
	cmd.Flags().StringVarP(&h.Op, "op", "", "", cage_reflect.GetFieldTag(*h, "Op", "usage"))
	return []string{"op"}
}

// Run performs the sub-command logic.
//
// It implements cli/handler/cobra.Handler.
func (h *Handler) Run(ctx context.Context, input handler.Input) {
	errs := h.config.ReadFile(h.ConfigFile, h.Op)
	errsLen := len(errs)
	if errsLen > 0 {
		errs = append(errs, errors.Errorf("config file contains %d issue(s), canceled [%s] operation", errsLen, h.Op))
		cage_errors.WriteErrList(h.Err(), errs...)
		h.Log.ErrToFile(errs...)
		os.Exit(1)
	}

	if len(h.config.Ops) == 0 {
		h.Log.ExitOnErr(1, errors.Errorf("config file [%s] does not contain any operations", h.ConfigFile))
		return
	}

	op, ok := h.config.Ops[h.Op]
	if !ok {
		var opList string
		for id := range h.config.Ops {
			opList += "\n\t" + id
		}
		fmt.Fprintf(h.Err(), "Available operations:%s\n", opList)
		h.Log.ExitOnErr(1, errors.Errorf("config file [%s] does not contain operation [%s]", h.ConfigFile, h.Op))
		return
	}

	if input.Args[0] == "" {
		h.Exitf(1, "missing argument, example: "+exampleText)
	}

	if absErr := cage_filepath.Abs(&input.Args[0]); absErr != nil {
		h.Log.ExitOnErr(1, absErr)
	}

	whyLog := make(why.Log)

	op.DryRun = true
	fmt.Fprintln(h.Err(), "Starting dry-run to collect file/dir activity logs ...")

	audit := transplant.NewEgressAudit(op)

	audit.Progress = h.Err() //  dry-run takes almost as long as full runs, explain the delay
	audit.WhyLog = whyLog

	errs = audit.Generate()
	h.Log.ExitOnErr(1, errs...)

	if len(audit.UnconfiguredDirs) > 0 {
		audit.PrintUnconfiguredDirs(h.Err())
		h.Log.ExitOnErr(1, errors.Errorf("operation [%s] config does not account for at least one dependency", h.Op))
	}

	copier, copyErr := transplant.NewCopier(ctx, audit)
	h.Log.ExitOnErr(1, copyErr)

	copier.Stderr = h.Err()
	copier.ProgressCore = h.Err() //  dry-run takes almost as long as full runs, explain the delay
	copier.WhyLog = whyLog

	plan, errs := copier.Run()
	if len(errs) > 0 {
		fmt.Fprintf(h.Err(), "(files staged for copy were saved here: %s)\n", plan.StagePath)
	}
	h.Log.ExitOnErr(1, errs...)

	why.PrintLog(h.Out(), whyLog, input.Args[0])

	h.Log.ExitOnErr(1, cage_file.RemoveAllSafer(plan.StagePath))
}

// New returns a cobra command instance based on Handler.
func NewCommand() *cobra.Command {
	return handler_cobra.NewHandler(&Handler{
		Session: &handler.DefaultSession{},
	})
}

var _ handler_cobra.Handler = (*Handler)(nil)
