// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package exec_test

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testecho "github.com/codeactual/transplant/internal/cage/cmd/testecho"
	cage_os "github.com/codeactual/transplant/internal/cage/os"
	cage_exec "github.com/codeactual/transplant/internal/cage/os/exec"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

var onlyDigits *regexp.Regexp = regexp.MustCompile("^[1-9]+[0-9]*$")

const (
	badPath          = "/some/bad/path"
	startErrExitCode = -1
	startErrPid      = -1
	startErrPgid     = -1
	cancelExitCode   = -1
	sigKillMsg       = "signal: killed"
)

type stageResult struct {
	Code   int
	ErrStr string

	Stdout string
	Stderr string

	// to support cases prone to race conditions without a clear fix, take precedence over single-value fields
	Codes   []int
	ErrStrs []string
	Stderrs []string
}

func requireNonZeroPidPgid(t *testing.T, res cage_exec.PipelineResult) {
	for _, r := range res.Cmd {
		require.True(t, r.Pid > 0, "pid [%d]", r.Pid)
		require.True(t, r.Pgid > 0, "pgid [%d]", r.Pgid)
	}
}

func requirePipelineStages(t *testing.T, cmds []*exec.Cmd, res cage_exec.PipelineResult, expectedStage ...stageResult) {
	cmdsLen := len(cmds)

	// catch argument mismatches
	require.Len(t, res.Cmd, cmdsLen)
	require.Len(t, expectedStage, cmdsLen)

	for n, cmd := range cmds {
		id := fmt.Sprintf("cmd %d", n)

		if len(expectedStage[n].ErrStrs) > 0 {
			require.Contains(t, expectedStage[n].ErrStrs, res.Cmd[cmd].Err.Error())
		} else {
			if expectedStage[n].ErrStr == "" {
				require.NoError(t, res.Cmd[cmd].Err, id)
			} else {
				require.EqualError(t, res.Cmd[cmd].Err, expectedStage[n].ErrStr, id)
			}
		}

		actualStdout := res.Cmd[cmd].Stdout.String()
		actualStderr := res.Cmd[cmd].Stderr.String()

		if len(expectedStage[n].Codes) > 0 {
			require.Contains(t, expectedStage[n].Codes, res.Cmd[cmd].Code)
		} else {
			require.Exactly(t, expectedStage[n].Code, res.Cmd[cmd].Code, id)
		}

		require.Exactly(t, expectedStage[n].Stdout, actualStdout, id)

		if len(expectedStage[n].Stderrs) > 0 {
			require.Contains(t, expectedStage[n].Stderrs, actualStderr)
		} else {
			require.Exactly(t, expectedStage[n].Stderr, actualStderr, id)
		}
	}
}

func requireProcessKilled(t *testing.T, res cage_exec.PipelineResult, actualErr error, grandChildPid string, cmds ...*exec.Cmd) {
	var wg sync.WaitGroup
	wg.Add(len(cmds))

	for _, cmd := range cmds {
		go func(cmd *exec.Cmd) {
			require.Exactly(t, cancelExitCode, res.Cmd[cmd].Code)
			require.Contains(t, actualErr.Error(), sigKillMsg)

			time.Sleep(cage_exec.SigKillDelay)

			_, err := cage_os.FindProcess(res.Cmd[cmd].Pid)
			require.Error(t, err)

			if grandChildPid != "none" {
				_, err = cage_os.StringPidToProcess(grandChildPid)
				require.Error(t, err)
			}

			wg.Done()
		}(cmd)
	}

	wg.Wait()
}

func TestStandard(t *testing.T) {
	t.Run("should fail if no context specified", func(t *testing.T) {
		//lint:ignore SA1012 nil context is the SUT
		res, err := cage_exec.CommonExecutor{}.Standard(nil, nil, nil, nil, testecho.NewCmd(context.Background()))
		require.Exactly(t, cage_exec.PipelineResult{}, res)
		require.EqualError(t, err, "non-nil context is required")
	})

	t.Run("should fail if no commands specified", func(t *testing.T) {
		res, err := cage_exec.CommonExecutor{}.Standard(context.Background(), nil, nil, nil)
		require.Exactly(t, cage_exec.PipelineResult{}, res)
		require.EqualError(t, err, "pipeline contains 0 commands")
	})

	t.Run("should fail if nil command specified", func(t *testing.T) {
		res, err := cage_exec.CommonExecutor{}.Standard(context.Background(), nil, nil, nil, nil)
		require.Exactly(t, cage_exec.PipelineResult{}, res)
		require.EqualError(t, err, "pipeline contains a nil command")
	})
}

func TestBufferedOneCommand(t *testing.T) {
	t.Run("should handle success", func(t *testing.T) {
		ctx := context.Background()
		cmd := testecho.NewCmd(ctx)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmd)

		require.NoError(t, err)
		require.Exactly(t, testecho.DefaultStdout, stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		requireNonZeroPidPgid(t, res)

		require.Exactly(t, 0, res.Cmd[cmd].Code)
		require.NoError(t, res.Cmd[cmd].Err)
		require.Exactly(t, testecho.DefaultStdout, res.Cmd[cmd].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmd].Stderr.String())
	})

	t.Run("should handle Start failure", func(t *testing.T) {
		ctx := context.Background()
		cmd := exec.CommandContext(ctx, badPath)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmd)

		require.EqualError(t, err, "failed to start command: path=/some/bad/path args=[]string{} dir=: fork/exec /some/bad/path: no such file or directory")
		require.Exactly(t, "", stdout.String())
		require.Exactly(t, "", stderr.String())

		require.Exactly(t, startErrExitCode, res.Cmd[cmd].Code)
		require.Exactly(t, startErrPid, res.Cmd[cmd].Pid)
		require.Exactly(t, startErrPgid, res.Cmd[cmd].Pgid)
		require.EqualError(t, res.Cmd[cmd].Err, "fork/exec /some/bad/path: no such file or directory")
		require.Exactly(t, "", res.Cmd[cmd].Stdout.String())
		require.Exactly(t, "", res.Cmd[cmd].Stderr.String())
	})

	t.Run("should handle Wait failure", func(t *testing.T) {
		ctx := context.Background()
		expectedCode := 3
		cmd := testecho.NewCmd(ctx, testecho.Input{Code: expectedCode})

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmd)

		require.EqualError(t, err, "command failed: path="+testecho.Which()+` args=[]string{"--code", "3", "--stderr", "some stderr message", "--stdout", "some stdout message"} dir=: exit status 3`)
		require.Exactly(t, testecho.DefaultStdout, stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		require.Exactly(t, expectedCode, res.Cmd[cmd].Code)
		require.EqualError(t, res.Cmd[cmd].Err, "exit status 3")
		require.Exactly(t, testecho.DefaultStdout, res.Cmd[cmd].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmd].Stderr.String())
	})

	t.Run("should kill process via context cancel", func(t *testing.T) {
		var wg sync.WaitGroup

		ctx, cancel := context.WithCancel(context.Background())
		cmd := testecho.NewCmd(ctx, testecho.Input{Sleep: 1})

		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
			wg.Done()
		}()

		wg.Add(1)
		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmd)
		wg.Wait() // for cancel()

		require.EqualError(t, err, "command failed: path="+testecho.Which()+` args=[]string{"--sleep", "1", "--stderr", "some stderr message", "--stdout", "some stdout message"} dir=: `+sigKillMsg)
		require.Exactly(t, testecho.DefaultStdout, stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		require.Exactly(t, res.Cmd[cmd].Code, -1)
		require.EqualError(t, res.Cmd[cmd].Err, sigKillMsg)
		require.Exactly(t, testecho.DefaultStdout, res.Cmd[cmd].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmd].Stderr.String())

		requireProcessKilled(t, res, err, "none", cmd)
	})

	t.Run("should kill process via context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		cmd := testecho.NewCmd(ctx, testecho.Input{Sleep: 1})

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmd)

		require.EqualError(t, err, "command failed: path="+testecho.Which()+` args=[]string{"--sleep", "1", "--stderr", "some stderr message", "--stdout", "some stdout message"} dir=: `+sigKillMsg)
		require.Exactly(t, testecho.DefaultStdout, stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		require.Exactly(t, res.Cmd[cmd].Code, -1)
		require.EqualError(t, res.Cmd[cmd].Err, sigKillMsg)
		require.Exactly(t, testecho.DefaultStdout, res.Cmd[cmd].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmd].Stderr.String())

		requireProcessKilled(t, res, err, "none", cmd)
	})

	t.Run("should kill process group via context cancel", func(t *testing.T) {
		var wg sync.WaitGroup

		ctx, cancel := context.WithCancel(context.Background())
		cmd := testecho.NewCmd(ctx, testecho.Input{Sleep: 1, Spawn: true})

		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
			wg.Done()
		}()

		wg.Add(1)
		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmd)
		wg.Wait() // for cancel()

		require.EqualError(t, err, "command failed: path="+testecho.Which()+` args=[]string{"--sleep", "1", "--stderr", "some stderr message", "--spawn"} dir=: `+sigKillMsg)
		require.Regexp(t, onlyDigits, stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		require.Exactly(t, res.Cmd[cmd].Code, -1)
		require.EqualError(t, res.Cmd[cmd].Err, sigKillMsg)
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmd].Stderr.String())

		requireProcessKilled(t, res, err, stdout.String(), cmd)
	})

	t.Run("should kill process group via context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		cmd := testecho.NewCmd(ctx, testecho.Input{Sleep: 1, Spawn: true})

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmd)

		require.EqualError(t, err, "command failed: path="+testecho.Which()+` args=[]string{"--sleep", "1", "--stderr", "some stderr message", "--spawn"} dir=: `+sigKillMsg)
		require.Regexp(t, onlyDigits, stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		require.Exactly(t, res.Cmd[cmd].Code, -1)
		require.EqualError(t, res.Cmd[cmd].Err, sigKillMsg)
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmd].Stderr.String())

		requireProcessKilled(t, res, err, stdout.String(), cmd)
	})
}

func TestBufferedTwoCommands(t *testing.T) {
	t.Run("should handle success", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		require.NoError(t, err)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		requireNonZeroPidPgid(t, res)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdoutFromStdin,
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should handle xargs", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			exec.CommandContext(ctx, "xargs", "echo"),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		require.NoError(t, err)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout + "\n",
				Stderr: "",
			},
		)
	})

	t.Run("should handle first process Start failure", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			exec.CommandContext(ctx, badPath),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"failed to start command",
			`path=/some/bad/path args=\[\]string\{\} dir= |`,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir=:`,
			"fork/exec /some/bad/path: no such file or directory",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		require.Exactly(t, startErrPid, res.Cmd[cmds[0]].Pid)
		require.Exactly(t, startErrPgid, res.Cmd[cmds[0]].Pgid)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   startErrExitCode,
				ErrStr: "fork/exec /some/bad/path: no such file or directory",
				Stdout: "",
				Stderr: "",
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: "stdin []",
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should handle second process Start failure", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			exec.CommandContext(ctx, badPath),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"failed to start command",
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= `,
			`path=/some/bad/path args=\[\]string\{\} dir=:`,
			`fork/exec /some/bad/path: no such file or directory`,
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		require.Exactly(t, startErrPid, res.Cmd[cmds[1]].Pid)
		require.Exactly(t, startErrPgid, res.Cmd[cmds[1]].Pgid)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   1,
				ErrStr: "io: read/write on closed pipe",
				Stdout: "", // failed due to pipe error
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   startErrExitCode,
				ErrStr: "fork/exec /some/bad/path: no such file or directory",
				Stdout: "",
				Stderr: "",
			},
		)
	})

	t.Run("should handle first process Wait failure", func(t *testing.T) {
		ctx := context.Background()
		expectedCode := 3
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx, testecho.Input{Code: expectedCode}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--code", "3", "--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir=`,
			"exit status 3",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		require.Exactly(t, expectedCode, res.Cmd[cmds[0]].Code)
		require.EqualError(t, res.Cmd[cmds[0]].Err, "exit status 3")
		require.Exactly(t, testecho.DefaultStdout, res.Cmd[cmds[0]].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmds[0]].Stderr.String())

		// unclear how to resolve this race, still flaky after adding sleep on the first/second process
		require.True(t, 1 == res.Cmd[cmds[1]].Code || 0 == res.Cmd[cmds[1]].Code)
		require.True(t, res.Cmd[cmds[1]].Err == nil || res.Cmd[cmds[1]].Err.Error() == "io: read/write on closed pipe")

		require.Exactly(t, testecho.DefaultStdoutFromStdin, res.Cmd[cmds[1]].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmds[1]].Stderr.String())

	})

	t.Run("should handle second process Wait failure", func(t *testing.T) {
		ctx := context.Background()
		expectedCode := 3
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			testecho.NewCmd(ctx, testecho.Input{Code: expectedCode, Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--code", "3", "--stderr", "some stderr message"\} dir=:`,
			"exit status 3",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   expectedCode,
				ErrStr: "exit status 3",
				Stdout: testecho.DefaultStdoutFromStdin,
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should kill pipeline via context cancel", func(t *testing.T) {
		var wg sync.WaitGroup

		ctx, cancel := context.WithCancel(context.Background())
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx, testecho.Input{Sleep: 1}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true, Sleep: 1}),
		}
		cmdsLen := len(cmds)

		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
			wg.Done()
		}()

		wg.Add(1)
		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)
		wg.Wait() // for cancel()

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message"\} dir=:`,
			"signal: killed",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				// - Adding a sleep (1s, 5s, etc.) to the 2nd command for some reason causes testecho
				//   to only emit stdout from the first fmt.Print (see more in testecho).
				//   Since we already have coverage of the piping, we'll not verify here exactly
				//   what was piped and instead verify that both commands were cancelled with the same context.
				// - The same goes for stderr but the difference is that nothing is emitted
				//   because we don't currently prefix it with something like "stderr [". We already
				//   have coverage of stderr collection, so we'll avoid even more hacks/one-offs
				//   for edge cases.
				Stdout: "stdin [",
				Stderr: "",
			},
		)

		requireProcessKilled(t, res, err, "none", cmds...)
	})

	t.Run("should kill pipeline via context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx, testecho.Input{Sleep: 1}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true, Sleep: 1}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message"\} dir=:`,
			"signal: killed",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				// - Adding a sleep (1s, 5s, etc.) to the 2nd command for some reason causes testecho
				//   to only emit stdout from the first fmt.Print (see more in testecho).
				//   Since we already have coverage of the piping, we'll not verify here exactly
				//   what was piped and instead verify that both commands were cancelled with the same context.
				// - The same goes for stderr but the difference is that nothing is emitted
				//   because we don't currently prefix it with something like "stderr [". We already
				//   have coverage of stderr collection, so we'll avoid even more hacks/one-offs
				//   for edge cases.
				Stdout: "stdin [",
				Stderr: "",
			},
		)

		requireProcessKilled(t, res, err, "none", cmds...)
	})
}

func TestBufferedThreeCommands(t *testing.T) {
	t.Run("should handle success with non fixture programs", func(t *testing.T) {
		ctx := context.Background()
		printfInput := "one\ntwo pick me\nthree"
		cmds := []*exec.Cmd{
			exec.CommandContext(ctx, "printf", printfInput),
			exec.CommandContext(ctx, "grep", "two"),
			exec.CommandContext(ctx, "rev"),
		}
		expectedFinalStdout := "em kcip owt\n"

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		require.NoError(t, err)
		require.Exactly(t, expectedFinalStdout, stdout.String())
		require.Exactly(t, "", stderr.String())

		requireNonZeroPidPgid(t, res)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: printfInput,
				Stderr: "",
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: "two pick me\n",
				Stderr: "",
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: expectedFinalStdout,
				Stderr: "",
			},
		)
	})

	t.Run("should handle success", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		require.NoError(t, err)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		requireNonZeroPidPgid(t, res)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdoutFromStdin,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdoutFromStdinNested,
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should handle first process Start failure", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			exec.CommandContext(ctx, badPath),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"failed to start command",
			`path=/some/bad/path args=\[\]string\{\} dir= `,
			`| path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir= `,
			`| path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir=:`,
			"fork/exec /some/bad/path: no such file or directory",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		require.Exactly(t, startErrPid, res.Cmd[cmds[0]].Pid)
		require.Exactly(t, startErrPgid, res.Cmd[cmds[0]].Pgid)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   startErrExitCode,
				ErrStr: "fork/exec /some/bad/path: no such file or directory",
				Stdout: "",
				Stderr: "",
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: "stdin []",
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: "stdin [stdin []]",
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should handle second process Start failure", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			exec.CommandContext(ctx, badPath),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"failed to start command",
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= |`,
			`path=/some/bad/path args=\[\]string\{\} dir= | `,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir=:`,
			"fork/exec /some/bad/path: no such file or directory",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		require.Exactly(t, startErrPid, res.Cmd[cmds[1]].Pid)
		require.Exactly(t, startErrPgid, res.Cmd[cmds[1]].Pgid)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   1,
				ErrStr: "io: read/write on closed pipe",
				Stdout: "", // failed due to pipe error
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   startErrExitCode,
				ErrStr: "fork/exec /some/bad/path: no such file or directory",
				Stdout: "",
				Stderr: "",
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: "stdin []",
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should handle third process Start failure", func(t *testing.T) {
		ctx := context.Background()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
			exec.CommandContext(ctx, badPath),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"failed to start command",
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message", "--stdout", "some stdout message"} dir= |`,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir= | `,
			`path=/some/bad/path args=\[\]string\{\} dir=:`,
			"fork/exec /some/bad/path: no such file or directory",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.True(t, stderr.String() == testecho.DefaultStderr || stderr.String() == testecho.DefaultStderr+testecho.DefaultStderr, stderr.String()) // race condition

		require.Exactly(t, startErrPid, res.Cmd[cmds[2]].Pid)
		require.Exactly(t, startErrPgid, res.Cmd[cmds[2]].Pgid)

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Codes:   []int{1, startErrExitCode},                                       // race condition
				ErrStrs: []string{"io: read/write on closed pipe", "signal: broken pipe"}, // race condition
				Stdout:  "",                                                               // failed due to pipe error
				Stderrs: []string{testecho.DefaultStderr, ""},                             // race condition
			},
			stageResult{
				Code:   startErrExitCode,
				ErrStr: "fork/exec /some/bad/path: no such file or directory",
				Stdout: "",
				Stderr: "",
			},
		)
	})

	t.Run("should handle first process Wait failure", func(t *testing.T) {
		ctx := context.Background()
		expectedCode := 3
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx, testecho.Input{Code: expectedCode}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--code", "3", "--stderr", "some stderr message", "--stdout", "some stdout message"} dir= |`,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir=:`,
			"exit status 3",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		require.Exactly(t, expectedCode, res.Cmd[cmds[0]].Code)
		require.EqualError(t, res.Cmd[cmds[0]].Err, "exit status 3")
		require.Exactly(t, testecho.DefaultStdout, res.Cmd[cmds[0]].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmds[0]].Stderr.String())

		// unclear how to resolve this race, still flaky after adding sleep on the first/second process
		require.True(t, 1 == res.Cmd[cmds[1]].Code || 0 == res.Cmd[cmds[1]].Code)
		require.True(t, res.Cmd[cmds[1]].Err == nil || res.Cmd[cmds[1]].Err.Error() == "io: read/write on closed pipe")

		require.Exactly(t, testecho.DefaultStdoutFromStdin, res.Cmd[cmds[1]].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmds[1]].Stderr.String())

		require.Exactly(t, 0, res.Cmd[cmds[2]].Code)
		require.NoError(t, res.Cmd[cmds[2]].Err)
		require.Exactly(t, testecho.DefaultStdoutFromStdinNested, res.Cmd[cmds[2]].Stdout.String())
		require.Exactly(t, testecho.DefaultStderr, res.Cmd[cmds[2]].Stderr.String())
	})

	t.Run("should handle second process Wait failure", func(t *testing.T) {
		ctx := context.Background()
		expectedCode := 3
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			testecho.NewCmd(ctx, testecho.Input{Code: expectedCode, Stdin: true}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message", "--stdout", "some stdout message"} dir= |`,
			`path=.*testecho args=\[\]string\{"--code", "3", "--stderr", "some stderr message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir=:`,
			"exit status 3",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   expectedCode,
				ErrStr: "exit status 3",
				Stdout: testecho.DefaultStdoutFromStdin,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdoutFromStdinNested,
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should handle third process Wait failure", func(t *testing.T) {
		ctx := context.Background()
		expectedCode := 3
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true}),
			testecho.NewCmd(ctx, testecho.Input{Code: expectedCode, Stdin: true}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message", "--stdout", "some stdout message"} dir= |`,
			`path=.*testecho args=\[\]string\{"--stderr", "some stderr message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--code", "3", "--stderr", "some stderr message"\} dir=:`,
			"exit status 3",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr+testecho.DefaultStderr+testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   0,
				ErrStr: "",
				Stdout: testecho.DefaultStdoutFromStdin,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   expectedCode,
				ErrStr: "exit status 3",
				Stdout: testecho.DefaultStdoutFromStdinNested,
				Stderr: testecho.DefaultStderr,
			},
		)
	})

	t.Run("should kill pipeline via context cancel", func(t *testing.T) {
		var wg sync.WaitGroup

		ctx, cancel := context.WithCancel(context.Background())
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx, testecho.Input{Sleep: 1}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true, Sleep: 1}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true, Sleep: 1}),
		}
		cmdsLen := len(cmds)

		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
			wg.Done()
		}()

		wg.Add(1)
		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)
		wg.Wait() // for cancel()

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message"\} dir=:`,
			"signal: killed",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				// - Adding a sleep (1s, 5s, etc.) to the 2nd command for some reason causes testecho
				//   to only emit stdout from the first fmt.Print (see more in testecho).
				//   Since we already have coverage of the piping, we'll not verify here exactly
				//   what was piped and instead verify that both commands were cancelled with the same context.
				// - The same goes for stderr but the difference is that nothing is emitted
				//   because we don't currently prefix it with something like "stderr [". We already
				//   have coverage of stderr collection, so we'll avoid even more hacks/one-offs
				//   for edge cases.
				Stdout: "stdin [",
				Stderr: "",
			},
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				// See comments about 2nd command's expected Stdout/Stderr
				Stdout: "stdin [",
				Stderr: "",
			},
		)

		requireProcessKilled(t, res, err, "none", cmds...)
	})

	t.Run("should kill pipeline via context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
		defer cancel()
		cmds := []*exec.Cmd{
			testecho.NewCmd(ctx, testecho.Input{Sleep: 1}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true, Sleep: 1}),
			testecho.NewCmd(ctx, testecho.Input{Stdin: true, Sleep: 1}),
		}
		cmdsLen := len(cmds)

		stdout, stderr, res, err := cage_exec.CommonExecutor{}.Buffered(ctx, cmds...)

		testkit_require.MatchRegexp(
			t,
			err.Error(),
			"command failed",
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message", "--stdout", "some stdout message"\} dir= |`,
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message"} dir= |`,
			`path=.*testecho args=\[\]string\{"--sleep", "1", "--stderr", "some stderr message"\} dir=:`,
			"signal: killed",
		)
		require.Exactly(t, res.Cmd[cmds[cmdsLen-1]].Stdout.String(), stdout.String())
		require.Exactly(t, testecho.DefaultStderr, stderr.String())

		requirePipelineStages(
			t, cmds, res,
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				Stdout: testecho.DefaultStdout,
				Stderr: testecho.DefaultStderr,
			},
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				// - Adding a sleep (1s, 5s, etc.) to the 2nd command for some reason causes testecho
				//   to only emit stdout from the first fmt.Print (see more in testecho).
				//   Since we already have coverage of the piping, we'll not verify here exactly
				//   what was piped and instead verify that both commands were cancelled with the same context.
				// - The same goes for stderr but the difference is that nothing is emitted
				//   because we don't currently prefix it with something like "stderr [". We already
				//   have coverage of stderr collection, so we'll avoid even more hacks/one-offs
				//   for edge cases.
				Stdout: "stdin [",
				Stderr: "",
			},
			stageResult{
				Code:   cancelExitCode,
				ErrStr: sigKillMsg,
				// See comments about 2nd command's expected Stdout/Stderr
				Stdout: "stdin [",
				Stderr: "",
			},
		)

		requireProcessKilled(t, res, err, "none", cmds...)
	})
}

func TestArgToCmd(t *testing.T) {
	t.Run("should build commands", func(t *testing.T) {
		args := cage_strings.SliceOfSlice(
			[]string{
				"cmd0", "cmd0Arg0", "cmd0Arg1",
			},
			[]string{
				"cmd1", "cmd1Arg0", "cmd1Arg1",
			},
			[]string{
				"cmd2", "cmd2Arg0", "cmd2Arg1",
			},
		)
		cmds := cage_exec.ArgToCmd(context.Background(), args...)
		require.Exactly(t, len(args), len(cmds))
		for n, cmd := range cmds {
			require.Exactly(t, args[n][0], cmd.Path)
			require.Exactly(t, args[n], cmd.Args)
		}
	})

	t.Run("should share context", func(t *testing.T) {
		args := cage_strings.SliceOfSlice(
			[]string{
				"/bin/echo", "cmd0Arg0", "cmd0Arg1",
			},
			[]string{
				"/bin/echo", "cmd1Arg0", "cmd1Arg1",
			},
			[]string{
				"/bin/echo", "cmd2Arg0", "cmd2Arg1",
			},
		)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cmds := cage_exec.ArgToCmd(ctx, args...)
		for _, cmd := range cmds {
			require.EqualError(t, cmd.Run(), "context canceled")
		}
	})
}
