//go:build windows

package main

// function is a no-op on Windows because
// golang.org/x/term.MakeRaw already disables Ctrl-C handling.
// Software flow control and literal-next behavior do not apply.
func terminalSignalsDisable(fd int) error {
	return nil
}
