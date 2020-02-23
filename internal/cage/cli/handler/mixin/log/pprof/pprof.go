// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pprof

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/codeactual/transplant/internal/cage/cli/handler"
	cage_crypto "github.com/codeactual/transplant/internal/cage/crypto"
	cage_io "github.com/codeactual/transplant/internal/cage/io"
	cage_reflect "github.com/codeactual/transplant/internal/cage/reflect"
)

type Mixin struct {
	handler.DefaultSession

	CpuFile string `usage:"File to receive pprof CPU profile"`
	MemFile string `usage:"File to receive pprof memory profile"`
}

// Implements cage/cli/handler.Mixin
func (m *Mixin) BindCobraFlags(cmd *cobra.Command) []string {
	cmd.Flags().StringVarP(&m.CpuFile, "pprof-cpu-file", "", "", cage_reflect.GetFieldTag(*m, "CpuFile", "usage"))
	cmd.Flags().StringVarP(&m.MemFile, "pprof-mem-file", "", "", cage_reflect.GetFieldTag(*m, "MemFile", "usage"))
	return []string{}
}

// Implements cage/cli/handler/Mixin
func (m *Mixin) Name() string {
	return "cage/cli/handler/mixin/log/pprof"
}

// Implements cage/cli/handler.PreRun
func (m *Mixin) PreRun(ctx context.Context, args []string) error {
	if m.CpuFile == "" {
		return nil
	}

	f, err := os.Create(m.CpuFile)
	if err != nil {
		return errors.Wrapf(err, "failed to create --pprof-cpu-file [%s]", m.CpuFile)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		return errors.Wrap(err, "failed to start CPU profiling")
	}

	return nil
}

// Implements cage/cli/handler.PostRun
func (m *Mixin) PostRun(ctx context.Context) {
	if m.CpuFile != "" {
		pprof.StopCPUProfile()
	}
	if m.MemFile != "" {
		f, err := os.Create(m.MemFile)
		if err != nil {
			fmt.Fprintf(m.Err(), "could not create memory profile: %+v", err)
			return
		}
		defer cage_io.CloseOrStderr(f, m.MemFile)
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(m.Err(), "could not write memory profile: %+v", err)
		}
	}
}

func (m *Mixin) Do(ctx context.Context, f func(context.Context)) (tagKey, tagVal string, err error) {
	if m.CpuFile == "" {
		return "", "", nil
	}

	tagKey = strings.Replace(m.Name(), "/", "_", -1) // more regex friendly

	tagVal, err = cage_crypto.RandHexString(4)
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	pprof.Do(ctx, pprof.Labels(tagKey, tagVal), f)
	return tagKey, tagVal, nil
}

var _ handler.Mixin = (*Mixin)(nil)
var _ handler.PreRun = (*Mixin)(nil)
var _ handler.PostRun = (*Mixin)(nil)
