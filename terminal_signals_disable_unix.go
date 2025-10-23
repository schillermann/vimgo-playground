//go:build !windows

package main

import (
	"golang.org/x/sys/unix"
)

// function disables generation of Ctrl-C (SIGINT), Ctrl-Z (SIGTSTP),
// and disables software flow control (Ctrl-S / Ctrl-Q) so those keys can be read normally.
func terminalSignalsDisable(fd int) error {
	// Get current terminal attributes
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}

	// Disable signal generation (Ctrl-C, Ctrl-Z)
	termios.Lflag &^= unix.ISIG

	// Disable software flow control (Ctrl-S, Ctrl-Q)
	termios.Iflag &^= unix.IXON

	// Apply updated settings immediately
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}
