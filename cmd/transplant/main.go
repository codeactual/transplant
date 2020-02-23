// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/codeactual/transplant/cmd/transplant/egress"
	"github.com/codeactual/transplant/cmd/transplant/ingress"
	"github.com/codeactual/transplant/internal/cage/cli/handler"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "transplant",
		Short: "Copy a Go project between a origin module and standalone module",
	}

	rootCmd.Version = handler.Version()
	rootCmd.AddCommand(egress.NewCommand())
	rootCmd.AddCommand(ingress.NewCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
