//go:build !windows

package main

import (
	"golang.org/x/sys/unix"
)

// function disables the generation of SIGINT (Ctrl-C), SIGTSTP (Ctrl-Z),
// and SIGQUIT (Ctrl-\) by clearing the ISIG flag in the terminal settings.
func terminalSignalsDisable(fd int) error {
	// Get current terminal attributes
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}

	// Clear the ISIG flag (disable signal generation)
	termios.Lflag &^= unix.ISIG

	// Apply updated settings immediately
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}
