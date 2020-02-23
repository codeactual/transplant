// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:generate mockgen -copyright_file=$LICENSE_HEADER -package=mock -destination=$GODIR/mock/handler.go -source=$GODIR/$GOFILE
package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	cage_debug "github.com/codeactual/transplant/internal/cage/runtime/debug"
	"github.com/codeactual/transplant/internal/ldflags"
	tp_os "github.com/codeactual/transplant/internal/third_party/stackexchange/os"
)

// Input defines the framework-agnostic user input passed to handler's Run methods.
type Input struct {
	// Args holds all unbound arguments except for the first "--" if present.
	Args []string

	// ArgsBeforeDash holds the subset of Args located before the first "--" if present,
	// or the same elements as Args if absent.
	ArgsBeforeDash []string

	// ArgsAfterDash holds the subset of Args located after the first "--" if present.
	ArgsAfterDash []string
}

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

// Session defines CLI handler behaviors which are useful to replace with test doubles.
//
// Behaviors related to terminal I/O are based on the interface approach in docker's CLI:
//   https://github.com/docker/cli/blob/f1b116179f2a95d0ea45780dfb8be51c2825b9c0/cli/command/cli.go
//   https://www.apache.org/licenses/LICENSE-2.0.html
type Session interface {
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

	// ExitOnError exits the program if the error is non-nil and prints the error in full format (%+v).
	ExitOnErr(err error, msg string, code int)

	// ExitOnErrorShort exits the program if the error is non-nil and prints the error's string value.
	ExitOnErrShort(err error, msg string, code int)

	// ExitOnErrsShort exits the program if the slice is non-empty and prints each error's string value.
	ExitOnErrsShort(errs []error, code int)

	// Exitf exits the program with a formatted message.
	Exitf(code int, format string, a ...interface{})
}

// DefaultSession can be embedded in all mixins and sub-command handlers and provide a default implementation
// of the Session interface which uses os.Stdout, os.Stderr, and os.Stdin by default.
type DefaultSession struct {
	err io.Writer
	out io.Writer
	in  io.Reader
}

func (s *DefaultSession) Err() io.Writer {
	if s.err == nil {
		return os.Stderr
	}
	return s.err
}

func (s *DefaultSession) Out() io.Writer {
	if s.out == nil {
		return os.Stdout
	}
	return s.out
}

func (s *DefaultSession) In() io.Reader {
	if s.in != nil && s.in != os.Stdin {
		return s.in
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

func (s *DefaultSession) SetErr(w io.Writer) {
	s.err = w
}

func (s *DefaultSession) SetOut(w io.Writer) {
	s.out = w
}

func (s *DefaultSession) SetIn(r io.Reader) {
	s.in = r
}

func (s *DefaultSession) OnSignal(sig os.Signal, f func(os.Signal)) (cancel func()) {
	doneCh := make(chan struct{}, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sig)
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case received := <-sigCh:
				f(received)
			}
		}
	}()

	return func() {
		doneCh <- struct{}{}
	}
}

func (s *DefaultSession) ExitOnErr(err error, msg string, code int) {
	if err == nil {
		return
	}

	if msg == "" {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Fprintf(os.Stderr, "%s: %+v\n", msg, err)
	}

	os.Exit(code)
}

func (s *DefaultSession) ExitOnErrShort(err error, msg string, code int) {
	if err == nil {
		return
	}

	if msg == "" {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
	}

	os.Exit(code)
}

func (s *DefaultSession) ExitOnErrsShort(errs []error, code int) {
	errsLen := len(errs)

	if errsLen == 0 {
		return
	}

	for n, err := range errs {
		errId := fmt.Sprintf("error %d/%d", n+1, errsLen)
		fmt.Fprintf(s.Err(), "(%s): %s\n", errId, err.Error())
	}

	os.Exit(code)
}

func (s *DefaultSession) Exitf(code int, format string, a ...interface{}) {
	w := os.Stdout
	if code != 0 {
		w = os.Stderr
	}
	fmt.Fprintf(w, format+"\n", a...)
	os.Exit(code)
}

var _ Session = (*DefaultSession)(nil)

func Version() string {
	return ldflags.Version + "\n" + cage_debug.BuildInfoString()
}
