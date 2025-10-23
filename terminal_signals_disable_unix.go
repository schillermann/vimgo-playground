//go:build !windows

/*
File: terminal_signals_disable_unix.go

Disables default terminal signal and flow-control keys:
  Ctrl-C  - normally sends SIGINT (interrupt)
  Ctrl-Z  - normally sends SIGTSTP (suspend)
  Ctrl-S  - pauses output (XOFF)
  Ctrl-Q  - resumes output (XON)
  Ctrl-V  - quotes next character literally

These behaviors are disabled by clearing the following termios flags:
  ISIG   – disables signal generation (Ctrl-C, Ctrl-Z, Ctrl-\)
  IXON   – disables software flow control (Ctrl-S, Ctrl-Q)
  IEXTEN – disables extended input processing (Ctrl-V “literal next”)
*/

package main

import (
	"golang.org/x/sys/unix"
)

// function disables:
// - Ctrl-C (SIGINT) and Ctrl-Z (SIGTSTP) via ISIG
// - Ctrl-S / Ctrl-Q software flow control via IXON
// - Ctrl-V "literal next" quoting behavior via IEXTEN
func terminalSignalsDisable(fd int) error {
	// Get current terminal attributes
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}

	// Disable signal generation (Ctrl-C, Ctrl-Z)
	termios.Lflag &^= unix.ISIG

	// Disable software flow control (Ctrl-S / Ctrl-Q)
	termios.Iflag &^= unix.IXON

	// Disable special extended input processing (Ctrl-V)
	termios.Lflag &^= unix.IEXTEN

	// Apply updated settings immediately
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}
