package exec

import (
	"fmt"
	"io"
	"os"
	std_exec "os/exec"
	"os/signal"
	"syscall"

	"github.com/kr/pty"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

// Pty runs the command in a pseudo-terminal and returns an error only if the command fails to start.
//
// It implements an Executor behavior.
//
// Origin:
//   https://github.com/kr/pty/blob/fa756f09eeb418bf1cc6268c66ceaad9bb98f598/README.md
//   MIT: https://github.com/kr/pty/blob/fa756f09eeb418bf1cc6268c66ceaad9bb98f598/License
//
// Changes:
//   - Migrate to github.com/pkg/errors
func Pty(cmd *std_exec.Cmd) error {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return errors.WithStack(err)
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if sizeErr := pty.InheritSize(os.Stdin, ptmx); err != nil {
				fmt.Fprintf(os.Stderr, "error resizing pty: %s", sizeErr)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize.

	// Set stdin in raw mode.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Copy stdin to the pty and the pty to stdout.
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, ptmx)

	return nil
}
