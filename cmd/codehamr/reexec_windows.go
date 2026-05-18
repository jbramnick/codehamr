//go:build windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
)

// reExec brings the freshly-installed binary online when Unix's
// syscall.Exec isn't an option. Windows has no execve — syscall.Exec is
// a stub that returns EWINDOWS without doing any work — so we instead
// spawn execPath as a child that inherits this process's stdio, wait for
// it, and forward its exit code. The user sees one continuous codehamr
// session even though Task Manager briefly shows two PIDs (parent waiting,
// child running the new TUI); scripts checking codehamr's exit code see
// the same value the child returned. This mirrors the user-visible
// behavior of the Unix execve path in reexec_unix.go.
//
// signal.Ignore is the gotcha: without it the parent's default Ctrl+C
// handler would terminate the parent while the child kept running on
// the same console, leaving the shell prompt interleaved with the
// child's TUI output. Ignoring SIGINT in the parent funnels all
// console Ctrl+C events to the child, which has its own cancel
// handler — exactly the Unix execve experience where there is no
// parent left to receive signals.
func reExec(exe string, args []string, env []string) error {
	signal.Ignore(os.Interrupt)
	cmd := exec.Command(exe, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		// A child that exited non-zero is success from reExec's
		// perspective — the new binary ran, the user saw its output,
		// and we just need to propagate its code to our caller's
		// caller (the shell). Anything else (spawn failure, missing
		// binary) is a real error the caller surfaces and falls
		// through to the old in-memory binary.
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil
}
