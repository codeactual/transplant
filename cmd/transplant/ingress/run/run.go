// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package run

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/codeactual/transplant/internal/cage/cli/handler"
	handler_cobra "github.com/codeactual/transplant/internal/cage/cli/handler/cobra"
	log_pprof "github.com/codeactual/transplant/internal/cage/cli/handler/mixin/log/pprof"
	log_zap "github.com/codeactual/transplant/internal/cage/cli/handler/mixin/log/zap"
	cage_errors "github.com/codeactual/transplant/internal/cage/errors"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_reflect "github.com/codeactual/transplant/internal/cage/reflect"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	"github.com/codeactual/transplant/internal/transplant"
)

// Handler defines the sub-command flags and logic.
type Handler struct {
	handler.Session

	ConfigFile string `usage:"configuration file (.json/.toml/.yaml/.yml)"`
	Op         string `usage:"Ops.Id value from the config file"`
	PlanFile   string `usage:"Dry-run mode, only write a plan file"`
	PlanField  string `usage:"(comma-separated) Include extra field(s) in the plan file: PruneGlobalIds,PruneGoFiles"`
	Progress   string `usage:"(comma-separated) Printed status message types: audit,copy"`

	Log     *log_zap.Mixin
	Profile *log_pprof.Mixin

	config transplant.Config

	progressTypes map[string]bool
	planFields    *cage_strings.Set
}

// Init defines the command, its environment variable prefix, etc.
//
// It implements cli/handler/cobra.Handler.
func (h *Handler) Init() handler_cobra.Init {
	h.Log = &log_zap.Mixin{}
	h.Profile = &log_pprof.Mixin{}
	h.progressTypes = make(map[string]bool)

	return handler_cobra.Init{
		Cmd: &cobra.Command{
			Use:   "run",
			Short: "Perform a copy",
		},
		EnvPrefix: "TRANSPLANT",
		Mixins: []handler.Mixin{
			h.Log,
			h.Profile,
		},
	}
}

// BindFlags binds the flags to Handler fields.
//
// It implements cli/handler/cobra.Handler.
func (h *Handler) BindFlags(cmd *cobra.Command) []string {
	cmd.Flags().StringVarP(&h.ConfigFile, "config", "", "", cage_reflect.GetFieldTag(*h, "ConfigFile", "usage"))
	cmd.Flags().StringVarP(&h.PlanFile, "plan", "", "", cage_reflect.GetFieldTag(*h, "PlanFile", "usage"))
	cmd.Flags().StringVarP(&h.PlanField, "plan-field", "", "", cage_reflect.GetFieldTag(*h, "PlanField", "usage"))
	cmd.Flags().StringVarP(&h.Op, "op", "", "", cage_reflect.GetFieldTag(*h, "Op", "usage"))
	cmd.Flags().StringVarP(&h.Progress, "progress", "", "audit,copy", cage_reflect.GetFieldTag(*h, "Op", "progress"))
	return []string{"op"}
}

// PreRun executes after flag parsing and before Run.
//
// If it returns an error, Run and PostRun are not executed.
//
// It implements cli/handler.PreRun
func (h *Handler) PreRun(ctx context.Context, args []string) error {
	for _, t := range strings.Split(h.Progress, ",") {
		h.progressTypes[strings.TrimSpace(t)] = true
	}

	h.planFields = cage_strings.NewSet()
	for _, f := range strings.Split(h.PlanField, ",") {
		h.planFields.Add(strings.TrimSpace(f))
	}

	return nil
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
		fmt.Fprintf(h.Err(), "available operations:%s\n", opList)
		h.Log.ExitOnErr(1, errors.Errorf("config file [%s] does not contain operation [%s]", h.ConfigFile, h.Op))
		return
	}

	if h.PlanFile != "" {
		op.DryRun = true
	}

	audit := transplant.NewIngressAudit(op)

	if h.progressTypes["audit"] {
		audit.Progress = h.Err()
	}

	errs = audit.Generate()
	h.Log.ExitOnErr(1, errs...)

	if len(audit.UnconfiguredDirs) > 0 {
		audit.PrintUnconfiguredDirs(h.Err())
		h.Log.ExitOnErr(1, errors.Errorf("operation [%s] config does not account for at least one dependency", h.Op))
	}

	copier, copyErr := transplant.NewCopier(ctx, audit)
	h.Log.ExitOnErr(1, copyErr)

	copier.OverwriteMin = true
	copier.Stderr = h.Err()

	if h.progressTypes["copy"] {
		copier.ProgressCore = h.Err()
	}

	plan, errs := copier.Run()
	if len(errs) > 0 {
		fmt.Fprintf(h.Err(), "(files staged for copy were saved here: %s)\n", plan.StagePath)
	}
	h.Log.ExitOnErr(1, errs...)

	if h.PlanFile != "" {
		h.Log.ExitOnErr(1, plan.WriteFile(h.PlanFile, h.planFields))
	}

	h.Log.ExitOnErr(1, cage_file.RemoveAllSafer(plan.StagePath))

	if h.Profile.CpuFile != "" {
		fmt.Fprintf(h.Out(), "go tool pprof -top -cum %s | head -20\n", h.Profile.CpuFile)
	}
}

// New returns a cobra command instance based on Handler.
func NewCommand() *cobra.Command {
	return handler_cobra.NewHandler(&Handler{
		Session: &handler.DefaultSession{},
	})
}

var _ handler_cobra.Handler = (*Handler)(nil)
var _ handler.PreRun = (*Handler)(nil)
