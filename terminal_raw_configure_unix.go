//go:build !windows

package main

import (
	"golang.org/x/sys/unix"
)

/*
Puts the terminal into a “clean” raw-like state by disabling various input, output, and signal processing features that interfere with reading raw keypresses.

Specifically, it clears or sets the following termios flags:

Input flags (Iflag):
  - BRKINT – disable break condition generating SIGINT
  - INPCK  – disable parity checking
  - ISTRIP – prevent stripping the 8th bit of input bytes
  - IXON   – disable software flow control (Ctrl-S / Ctrl-Q)
  - ICRNL  – disable carriage return → newline translation (fix Ctrl-M)

Local flags (Lflag):
  - ISIG   – disable signal generation (Ctrl-C, Ctrl-Z, Ctrl-\)
  - IEXTEN – disable special character handling (Ctrl-V “literal next”)

Control flags (Cflag):
  - CS8    – set 8-bit characters (no parity)

Output flags (Oflag):
  - OPOST  – disable all output processing ("\n" → "\r\n" translation)

This effectively produces a “raw input” mode suitable for implementing terminal-based text editors or REPL interfaces.
*/
func terminalRawConfigure(fd int) error {
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

	// Disable carriage return to newline translation (fix Ctrl-M)
	termios.Iflag &^= unix.ICRNL

	// Disable miscellaneous legacy input flags
	termios.Iflag &^= unix.BRKINT | unix.INPCK | unix.ISTRIP
	termios.Cflag |= unix.CS8

	// Turn off all output processing (disable "\n" → "\r\n" translation)
	termios.Oflag &^= unix.OPOST

	// Apply updated settings immediately
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}
