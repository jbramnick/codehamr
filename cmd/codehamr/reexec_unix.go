//go:build unix

package main

import "syscall"

// reExec replaces the current process image with execPath via execve(2).
// Same-PID transition: the parent shell that launched codehamr keeps
// waiting on one process the whole time, and the user sees one continuous
// session. A successful exec never returns; only an exec failure (missing
// binary, wrong arch, no exec bit) surfaces as the returned error.
//
// Windows lacks execve and uses reexec_windows.go's spawn-and-wait
// fallback to achieve the same user-visible behavior.
func reExec(exe string, args []string, env []string) error {
	return syscall.Exec(exe, args, env)
}
