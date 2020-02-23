// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:generate mockery -all
//go:generate mockgen -copyright_file=$LICENSE_HEADER -package=mock -destination=$GODIR/mock/exec.go -source=$GODIR/$GOFILE
package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	std_exec "os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	tp_exec "github.com/codeactual/transplant/internal/third_party/github.com/os/exec"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	tp_bytes "github.com/codeactual/transplant/internal/third_party/gist.github.com/bytes"
)

const (
	SigIntDelay  = 2 * time.Second
	SigKillDelay = 5 * time.Second
)

type Result struct {
	Code int

	// Pid supports concerns like verifying the process exited.
	Pid int

	// Pgid supports concerns like verifying the process group exited.
	//
	// Values are NOT pre-negated, e.g. for syscall.Kill.
	Pgid int

	Stdout *bytes.Buffer
	Stderr *bytes.Buffer

	Err error
}

type PipelineResult struct {
	// Cmd stores more detail about each process executed in the pipeline.
	Cmd map[*std_exec.Cmd]Result
}

// executeInput and executeOutput were added to further generalize the shared CommonExecutor
// behavior into the execute method by supporting additional input/output fields by expanding
// struct types instead of argument/return lists.
type executeInput struct {
	ctx    context.Context
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
	cmds   []*std_exec.Cmd
}

type executeOutput struct {
	pipelineResult PipelineResult
}

// Executor implementations handle os/exec.Cmd execution in various ways, e.g. buffered output.
//
// The main intent is to allow tests of client code to use mock implementations.
type Executor interface {
	// Command replaces the standard library os/exec method, allowing mock implementations to
	// control Cmd creation in addition to Cmd execution.
	//
	// Tests may need to use it when asserting mocking a method's (e.g. Buffered) return values
	// and also asserting PipelineResult.Cmd contents. This is because PipelineResult.Cmd is a map
	// indexed by the executed os/exec.Cmd. A mock Command will allow the method (e.g. Buffered)
	// to execute, and also index PipelineResult.Cmd, will the exact same pointer.
	Command(name string, arg ...string) *std_exec.Cmd

	// CommandContext replaces the standard library os/exec method, allowing mock implementations to
	// control Cmd creation in addition to Cmd execution.
	//
	// Tests may need to use it when asserting mocking a method's (e.g. Buffered) return values
	// and also asserting PipelineResult.Cmd contents. This is because PipelineResult.Cmd is a map
	// indexed by the executed os/exec.Cmd. A mock Command will allow the method (e.g. Buffered)
	// to execute, and also index PipelineResult.Cmd, will the exact same pointer.
	CommandContext(ctx context.Context, name string, arg ...string) *std_exec.Cmd

	// Buffered returns standard output/error in addition to the exit code.
	//
	// - Assumes all commands in the pipeline share the passed context.
	// - Return error should be used to determine overall pipeline success.
	//   Per-process results are stored in the return PipelineResult.Cmd map.
	// - If multiple commands are passed, standard out/error between them will be piped.
	// - Returned stdout is only from cmds[len(cmds)-1].
	// - Returned stderr is from all commands. Its ordering is not guaranteed, but per-command stderr
	//   is available in PipelineResult.Cmd Result values.
	Buffered(ctx context.Context, cmds ...*std_exec.Cmd) (stdout *bytes.Buffer, stderr *bytes.Buffer, res PipelineResult, err error)

	// Standard allows standard in, out, and error to be customized.
	//
	// - Assumes all commands in the pipeline share the passed context.
	// - Return error should be used to determine overall pipeline success.
	//   Per-process results are stored in the return PipelineResult.Cmd map.
	// - If multiple commands are passed, standard out/error between them will be piped.
	// - Input stdin will be piped to the first input exec.Cmd.
	// - Input stdout receives only from cmds[len(cmds)-1].
	// - Input stderr receives from all commands. Its ordering is not guaranteed, but per-command stderr
	//   is available in PipelineResult.Cmd Result values.
	Standard(ctx context.Context, stdout io.Writer, stderr io.Writer, stdin io.Reader, cmds ...*std_exec.Cmd) (res PipelineResult, err error)

	// Pty runs the command in a pseudo-terminal and returns an error only if the command fails to start.
	Pty(cmd *std_exec.Cmd) error
}

// CommonExecutor provides a general case Executor implementation.
type CommonExecutor struct{}

// Command completely delegates to the os/exec method.
//
// It implements an Executor behavior.
func (c CommonExecutor) Command(name string, arg ...string) *std_exec.Cmd {
	return std_exec.Command(name, arg...)
}

// CommandContext completely delegates to the os/exec method.
//
// It implements an Executor behavior.
func (c CommonExecutor) CommandContext(ctx context.Context, name string, arg ...string) *std_exec.Cmd {
	return std_exec.CommandContext(ctx, name, arg...)
}

// Buffered returns standard output/error in addition to the exit code.
//
// It implements an Executor behavior.
func (c CommonExecutor) Buffered(ctx context.Context, cmds ...*std_exec.Cmd) (stdout *bytes.Buffer, stderr *bytes.Buffer, res PipelineResult, err error) {
	// Use a goroutine-safe versions because all commands in the pipeline will write to them.
	sharedStdout := tp_bytes.NewSharedBuffer()
	sharedStderr := tp_bytes.NewSharedBuffer()

	out, err := c.execute(executeInput{
		ctx:    ctx,
		stdout: sharedStdout,
		stderr: sharedStderr,
		stdin:  nil,
		cmds:   cmds,
	})

	return sharedStdout.Unshared(), sharedStderr.Unshared(), out.pipelineResult, errors.WithStack(err)
}

// Standard allows standard in, out, and error to be customized.
//
// It implements an Executor behavior.
func (c CommonExecutor) Standard(ctx context.Context, stdout io.Writer, stderr io.Writer, stdin io.Reader, cmds ...*std_exec.Cmd) (res PipelineResult, err error) {
	out, err := c.execute(executeInput{
		ctx:    ctx,
		stdout: stdout,
		stderr: stderr,
		stdin:  stdin,
		cmds:   cmds,
	})
	return out.pipelineResult, err
}

// execute performs common logic for Buffered, Standard, and Pty.
func (c CommonExecutor) execute(input executeInput) (executeOutput, error) {
	var output executeOutput

	if input.ctx == nil {
		return executeOutput{}, errors.New("non-nil context is required")
	}

	cmdsLen := len(input.cmds)
	if cmdsLen == 0 {
		return executeOutput{}, errors.New("pipeline contains 0 commands")
	}

	output.pipelineResult.Cmd = make(map[*std_exec.Cmd]Result)

	// Avoid "res" map data races by processing the result of a pipeline stages, and the related *exec.Cmd
	// keys, one at a time.
	//
	// This approach was selected over sync.Map only based on a preference for consolidating the
	// synchronization logic in one place.
	var stageResWg sync.WaitGroup

	var mu sync.Mutex

	// io.Pipe pairs initialized during the iteration when the (stdout) writer is processed in the first pass.
	reader := map[*std_exec.Cmd]*io.PipeReader{}
	writer := map[*std_exec.Cmd]*io.PipeWriter{}

	// first pass: connect standard in/out/error of all commands
	for n, cmd := range input.cmds {
		if cmd == nil {
			return executeOutput{}, errors.New("pipeline contains a nil command")
		}

		output.pipelineResult.Cmd[cmd] = Result{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)}

		if n == 0 {
			cmd.Stdin = input.stdin
		}

		if n == cmdsLen-1 {
			cmd.Stdout = io.MultiWriter(input.stdout, output.pipelineResult.Cmd[cmd].Stdout)
			cmd.Stderr = io.MultiWriter(input.stderr, output.pipelineResult.Cmd[cmd].Stderr)

		} else {
			cmd.Stderr = io.MultiWriter(input.stderr, output.pipelineResult.Cmd[cmd].Stderr)

			r, w := io.Pipe()

			input.cmds[n].Stdout = io.MultiWriter(w, output.pipelineResult.Cmd[cmd].Stdout)

			// Prepare the following command to read the stdout from the current command.
			input.cmds[n+1].Stdin = r

			// Store the ends of the pipe so they can be closed at the correct times in the second pass:
			// - Close the writer when this command is done.
			// - Close the reader when the following command is done.
			writer[cmd] = w
			reader[input.cmds[n+1]] = r
		}
	}

	g, gCtx := errgroup.WithContext(input.ctx)
	stageResWg.Add(cmdsLen)

	// second pass: run the pre-connected commands
	//
	// - Start all commands "at the same time".
	// - Also start them in reverse to avoid races with symptoms including empty stdin or "broken pipe" error.
	//   Quick experiments in 1.10.2 found that this approach seems reliable even if the first (stdout-sending)
	//   process is started 10s after the second (stdin-receicing) process.
	for n := cmdsLen - 1; n >= 0; n-- {
		cmd := input.cmds[n]

		// Update the command-indexed Result returned to the Standard caller.
		//
		// - Must be called before the function, passed to g.Go below, returns for any reason.
		// - Standardized to reduce copy/paste errors and document intent.
		updatePipelineRes := func(r Result) {
			mu.Lock()
			defer mu.Unlock()
			tmp := output.pipelineResult.Cmd[cmd]
			tmp.Code = r.Code
			tmp.Pid = r.Pid
			tmp.Pgid = r.Pgid
			tmp.Err = r.Err
			output.pipelineResult.Cmd[cmd] = tmp
			stageResWg.Done()
		}

		// The first loop pass stored a writer and/or reader for the given command if it needed
		// to close part(s) of the io.Pipe after the process ended.
		//
		// This must be called after the execution attempt ends for any reason (Start/Wait error,
		// context cancelled, etc.). Symptoms that a call was missing: Result.Stdout/Result.Stderr
		// unexpectedly empty, etc.
		closePipe := func() {
			if writer[cmd] != nil {
				if err := writer[cmd].Close(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to close writer: %+v\n", errors.WithStack(err))
				}
			}
			if reader[cmd] != nil {
				if err := reader[cmd].Close(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to close reader: %+v\n", errors.WithStack(err))
				}
			}
		}

		g.Go(func() error {
			r := Result{} // language workaround for updating the "res" map via updatePipelineRes

			// Start process in a group so the context can kill the group as a whole.
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

			// for all error cases that happen before Wait
			r.Code = -1
			r.Pid = -1
			r.Pgid = -1

			if startErr := cmd.Start(); startErr != nil {
				r.Err = startErr
				closePipe()
				updatePipelineRes(r)
				return errors.Wrapf(startErr, "failed to start command: %s", CmdToString(input.cmds...))
			}

			r.Pid = cmd.Process.Pid

			if r.Pid == 0 {
				return errors.Errorf("got process ID 0: %s", CmdToString(input.cmds...))
			}

			var pgidErr error
			r.Pgid, pgidErr = syscall.Getpgid(r.Pid)
			if pgidErr != nil {
				closePipe()
				updatePipelineRes(r)
				return errors.Wrapf(pgidErr, "failed to get process group ID: %s", CmdToString(input.cmds...))
			}

			if r.Pgid == 0 {
				return errors.Errorf("got process group ID 0: %s", CmdToString(input.cmds...))
			}

			// Work around issue (in 1.10.1) where commands which use cmd.Wait/cmd.Process.Wait
			// cannot be cancelled as long as standard out/error is being collected.
			//
			// Here we can rely on the cancellation channel but not that the runtime actually
			// kills the process.
			//
			// We also use the opportunity to kill the process group as a whole, so even
			// after the runtime issue is fixed we will still want to use this goroutine
			// for that purpose which is not currently supported in the API.
			//
			// See https://github.com/golang/go/issues/23019 and its linked/related issues.
			//
			// Another workaround not used but may provide more background:
			//
			//     cmd.Stderr = struct{ io.Writer }{os.Stderr}
			//
			//     https://github.com/smola/ci-tricks/commit/a0e4714fd033df1f6a3469ce469085af29e06b7f
			//     https://go-review.googlesource.com/c/go/+/42271/3/misc/android/go_android_exec.go#36
			go func(ctx context.Context, pgid int) {
				<-ctx.Done()

				go func() {
					time.Sleep(SigIntDelay)
					if err := syscall.Kill(pgid, syscall.SIGINT); err != nil {
						if err.Error() != "no such process" {
							fmt.Fprintf(os.Stderr, "failed to SIGINT process group %d: %+v\n", pgid, errors.WithStack(err))
						}
					}
				}()
				go func() {
					time.Sleep(SigKillDelay)
					if err := syscall.Kill(pgid, syscall.SIGKILL); err != nil {
						if err.Error() != "no such process" {
							fmt.Fprintf(os.Stderr, "failed to SIGKILL process group %d: %+v\n", pgid, errors.WithStack(err))
						}
					}
				}()
			}(gCtx, -r.Pgid) // syscall.Kill requires a negative value to denote a process group

			waitErr := cmd.Wait()

			closePipe()

			if waitErr == nil {
				r.Code = 0
				updatePipelineRes(r)
				return nil
			}

			r.Err = waitErr
			r.Code = 1

			// Try to get a more specific exit code (e.g. on Linux where its supported).
			if exitErr, ok := waitErr.(*std_exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					r.Code = status.ExitStatus()
				}
			}

			updatePipelineRes(r)

			return errors.Wrapf(waitErr, "command failed: %s", CmdToString(input.cmds...))
		})
	}

	groupWaitErr := g.Wait() // for all commands to either fail to start or stop running
	stageResWg.Wait()        // for the results of all commands to have been saved

	return output, errors.WithStack(groupWaitErr)
}

// Pty runs the command in a pseudo-terminal and returns an error only if the command fails to start.
//
// It implements an Executor behavior.
func (c CommonExecutor) Pty(cmd *std_exec.Cmd) error {
	return tp_exec.Pty(cmd)
}

var _ Executor = (*CommonExecutor)(nil)

// CmdToString stringifies an os/exec.Cmd.
func CmdToString(cmds ...*std_exec.Cmd) string {
	var parts []string
	for _, c := range cmds {
		// - omit c.Args[0] which usually (always?) equals c.Path
		parts = append(parts, fmt.Sprintf("path=%s args=%#v dir=%s", c.Path, c.Args[1:], c.Dir))
	}
	return strings.Join(parts, " | ")
}

func ArgToCmd(ctx context.Context, args ...[]string) (cmds []*std_exec.Cmd) {
	for _, a := range args {
		cmds = append(cmds, std_exec.CommandContext(ctx, a[0], a[1:]...))
	}
	return cmds
}
