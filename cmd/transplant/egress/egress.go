// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package egress

import (
	"github.com/spf13/cobra"

	"github.com/codeactual/transplant/cmd/transplant/egress/run"
	"github.com/codeactual/transplant/cmd/transplant/egress/why"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Commands for copying a project from the origin module to a standalone module",
	}
	cmd.AddCommand(run.NewCommand())
	cmd.AddCommand(why.NewCommand())
	return cmd
}
