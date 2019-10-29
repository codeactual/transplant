// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package ingress

import (
	"github.com/spf13/cobra"

	"github.com/codeactual/transplant/cmd/transplant/ingress/run"
	"github.com/codeactual/transplant/cmd/transplant/ingress/why"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Commands copying a project from an egress-generated copy back to its origin",
	}
	cmd.AddCommand(run.New())
	cmd.AddCommand(why.New())
	return cmd
}
