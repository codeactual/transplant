// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	tp_os "github.com/codeactual/transplant/internal/third_party/stackexchange/os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type Mixin interface {
	// BindCobraFlags gives each mixin the opportunity to define its own flags.
	BindCobraFlags(cmd *cobra.Command) (requiredFlags []string)

	// Name should identify the mixin for use in error/log messages.
	//
	// Ideally it is short, e.g. like a package name.
	Name() string
}

// PreRun is optionally implemented by handlers/mixins to perform tasks after flags are parsed
// but before Handler.Run.
//
// If it returns an error, Handler.Run and Handler.PostRun do not execute.
type PreRun interface {
	PreRun(ctx context.Context, args []string) error
}

// PostRun is optionally implemented by handlers/mixins to perform tasks after Handler.Run finishes.
type PostRun interface {
	PostRun(ctx context.Context)
}

// Responder defines the common response behavior expected from each sub-command implementation.
//
// Behaviors related to terminal I/O are based on the interface approach in docker's CLI:
//   https://github.com/docker/cli/blob/f1b116179f2a95d0ea45780dfb8be51c2825b9c0/cli/command/cli.go
//   https://www.apache.org/licenses/LICENSE-2.0.html
//
// Specifically, Out and Err methods are intended to improve testability by expecting that terminal
// messages can be captured for assertions about their content (as they are in docker's own CLI tests).
type Responder interface {
	// Err returns the standard error destination.
	Err() io.Writer

	// In returns the standard input source.
	In() io.Reader

	// OnSignal starts a goroutine which runs the provided function every time the signal
	// is received. It returns a function to end the goroutine.
	OnSignal(s os.Signal, do func(os.Signal)) (cancel func())

	// Out returns the standard output destination.
	Out() io.Writer

	// SetErr assigns the standard error destination.
	SetErr(io.Writer)

	// SetIn assigns the standard input source.
	SetIn(io.Reader)

	// SetOut assigns the standard error destination.
	SetOut(io.Writer)
}

// IO can be embedded in all mixins and sub-command handlers and provide a default implementation
// of the Responder interface.
type IO struct {
	err io.Writer
	out io.Writer
	in  io.Reader
}

func (h *IO) Err() io.Writer {
	return h.err
}

func (h *IO) Out() io.Writer {
	return h.out
}

func (h *IO) In() io.Reader {
	if h.in != nil && h.in != os.Stdin {
		return h.in
	}

	pipeStdin, err := tp_os.IsPipeStdin()
	if err != nil {
		// Trade this panic for a more ergonomic use of In() without the error check.
		// The only cause of an error here is a failed os.Stdin.Stat(), which we will
		// assume for now is panic worthy.
		panic(errors.Wrap(err, "failed to check if stdin is a pipe"))
	}

	if pipeStdin {
		return os.Stdin
	}

	return nil
}

func (h *IO) SetErr(w io.Writer) {
	h.err = w
}

func (h *IO) SetOut(w io.Writer) {
	h.out = w
}

func (h *IO) SetIn(r io.Reader) {
	h.in = r
}

func (h *IO) OnSignal(s os.Signal, f func(os.Signal)) (cancel func()) {
	doneCh := make(chan struct{}, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, s)
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case sig := <-sigCh:
				f(sig)
			}
		}
	}()

	return func() {
		doneCh <- struct{}{}
	}
}

func (h *IO) ExitOnErr(err error, msg string, code int) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %+v\n", msg, err)
	os.Exit(code)
}

func (h *IO) ExitOnErrShort(err error, msg string, code int) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
	os.Exit(code)
}

func (h *IO) ExitOnErrsShort(errs []error, code int) {
	errsLen := len(errs)
	for n, err := range errs {
		errId := fmt.Sprintf("error %d/%d", n+1, errsLen)
		fmt.Fprintf(h.Err(), "(%s): %s\n", errId, err.Error())
	}
}

func (h *IO) Exitf(code int, format string, a ...interface{}) {
	w := os.Stdout
	if code != 0 {
		w = os.Stderr
	}
	fmt.Fprintf(w, format+"\n", a...)
	os.Exit(code)
}

func (h *IO) Exit(code int, a ...interface{}) {
	w := os.Stdout
	if code != 0 {
		w = os.Stderr
	}
	fmt.Fprintln(w, a...)
	os.Exit(code)
}

var _ Responder = (*IO)(nil)
