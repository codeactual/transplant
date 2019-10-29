// Copyright (C) 2019 The testecho Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package testecho assists execution of the CLI from test cases and assertion of its result.
//
// Its provides some default inputs for the CLI to support use cases where details such as the exact
// standard output/error are asserted for equality but semantic content is unnecessary.
package testecho

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/codeactual/transplant/internal/cage/env/golang"
)

const (
	// DefaultStdout is the default value for the --stdout flag.
	DefaultStdout = "some stdout message"

	// DefaultStdoutFromStdin is the standard output emitted by testecho when it receives DefaultStdout via
	// standard input.
	//
	// It supports the "testecho | testecho" use case and asserting the final process's behavior.
	DefaultStdoutFromStdin = "stdin [" + DefaultStdout + "]"

	// DefaultStderr is the default value for the --stderr flag.
	DefaultStderr = "some stderr message"

	// DefaultStdoutFromStdinNested is the standard output emitted by testecho when it receives
	// DefaultStdoutFromStdin via standard input.
	//
	// It supports the "testecho | testecho | testecho" use case and asserting the final process's behavior.
	DefaultStdoutFromStdinNested = "stdin [" + DefaultStdoutFromStdin + "]"
)

// Input defines the CLI flags to use in a testecho invocation.
type Input struct {
	// Code is the value for the --code flag.
	Code int

	// Stderr is a value for the --sleep flag.
	Sleep int

	// Spawn is true if the command should spawn a child process and block forever (because the child does).
	Spawn bool

	// Stdin is true if the command is expected to receive stdin.
	Stdin bool

	// Stderr is a value for the --stderr flag.
	//
	// If empty, DefaultStderr is used.
	Stderr string

	// Stderr is a value for the --stdout flag.
	//
	// If empty, DefaultStdout is used.
	Stdout string
}

// Which returns the absolute path to where the CLI would be installed based on GOPATH.
func Which() string {
	return golang.BinPath("testecho")
}

// NewCmdArgs converts an Input value into argument strings for use in exec.Command.
//
// It no Input is passed, a zero-valued Input is used.
// If one Input is passed, it is converted.
// If multiple Input values are passed, it panics.
func NewCmdArgs(inputs ...Input) (args []string) {
	var i Input

	if len(inputs) > 1 {
		panic(errors.New("cannot pass multiple Echo"))
	} else if len(inputs) == 0 {
		i = Input{}
	} else {
		i = inputs[0]
	}

	args = []string{}

	if i.Code > 0 {
		args = append(args, "--code", fmt.Sprintf("%d", i.Code))
	}
	if i.Sleep > 0 {
		args = append(args, "--sleep", fmt.Sprintf("%d", i.Sleep))
	}

	// always request it, regardless of exit code, to allow tests to assert the
	// subject's handling of it
	if i.Stderr == "" {
		args = append(args, "--stderr", DefaultStderr)
	} else {
		args = append(args, "--stderr", i.Stderr)
	}

	// When it spawns a grandchild process, the child will print the former's PID
	// instead of what it receives via -stdout.
	if i.Spawn {
		args = append(args, "--spawn")
	} else {
		if !i.Stdin { // stdin will automatically be echoed back over stdout
			if i.Stdout == "" {
				args = append(args, "--stdout", DefaultStdout)
			} else {
				args = append(args, "--stdout", i.Stdout)
			}
		}
	}

	return args
}

// NewCmd converts an Input into a command.
//
// It no Input is passed, a zero-valued Input is used.
// If one Input is passed, it is converted.
// If multiple Input values are passed, it panics.
func NewCmd(ctx context.Context, i ...Input) *exec.Cmd {
	return exec.CommandContext(ctx, Which(), NewCmdArgs(i...)...)
}
