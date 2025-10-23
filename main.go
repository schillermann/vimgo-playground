/*
File: main.go

This program reads raw keypresses from the terminal and reports them.
*/

package main

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/term"
)

// KeyCode represents special non-printable keys.
type KeyCode int

const (
	KeyUnknown KeyCode = iota
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyDelete
	KeyEnter
	KeyCtrl // meta for ctrl combos, printable rune will be passed too
	KeyEsc
	KeyRune // a normal printable rune
)

// KeyEvent is the parsed result of a keypress.
type KeyEvent struct {
	KeyCode KeyCode // the key code kind
	Rune    rune    // if printable (KeyRune) or ctrl printable char
	Ctrl    bool    // whether ctrl was pressed (for printable letters)
}

// readKey reads from stdin (one or more bytes) and returns a KeyEvent.
// It assumes stdin is in raw mode.
func readKey(inputReader *bufio.Reader) (KeyEvent, error) {
	var ev KeyEvent

	inputByte, err := inputReader.ReadByte()
	if err != nil {
		return ev, err
	}

	// handle Ctrl-keys: ASCII 1..26 => Ctrl-A..Ctrl-Z
	if inputByte >= 1 && inputByte <= 26 {
		ev.KeyCode = KeyRune
		ev.Rune = rune('a' - 1 + int(inputByte)) // ctrl code maps to lowercase letter
		ev.Ctrl = true
		// Special-case Ctrl-M and Ctrl-J?
		// Ctrl-M (13) is carriage return (Enter) often; we'll normalize:
		if inputByte == 13 {
			ev.KeyCode = KeyEnter
			ev.Rune = '\r'
		}
		return ev, nil
	}

	// printable ordinary characters (including space, digits, letters)
	if inputByte >= 32 && inputByte <= 126 {
		if inputByte == 13 || inputByte == '\n' {
			ev.KeyCode = KeyEnter
			ev.Rune = '\r'
		} else {
			ev.KeyCode = KeyRune
			ev.Rune = rune(inputByte)
		}
		return ev, nil
	}

	// handle escape sequences (starting with 27)
	if inputByte == 27 {
		ev.KeyCode = KeyEsc
		// try to read next byte(s) to interpret escape sequences common in terminals
		// Many sequences start with ESC [ or ESC O
		inputSecondByte, err := inputReader.ReadByte()
		if err != nil {
			// lone ESC
			return ev, nil
		}

		if inputSecondByte == '[' {
			// CSI sequences
			inputThirdByte, err := inputReader.ReadByte()
			if err != nil {
				return ev, nil
			}
			switch inputThirdByte {
			case 'A':
				ev.KeyCode = KeyArrowUp
				return ev, nil
			case 'B':
				ev.KeyCode = KeyArrowDown
				return ev, nil
			case 'C':
				ev.KeyCode = KeyArrowRight
				return ev, nil
			case 'D':
				ev.KeyCode = KeyArrowLeft
				return ev, nil
			case 'H':
				ev.KeyCode = KeyHome
				return ev, nil
			case 'F':
				ev.KeyCode = KeyEnd
				return ev, nil
			default:
				// sequences like ESC [ 1 ~  , ESC [ 3 ~  (delete), or ESC [5~ pageup
				// if inputThirdByte is a digit, read until '~'
				if inputThirdByte >= '0' && inputThirdByte <= '9' {
					// inputThirdByte is first digit; read the rest until we hit '~'
					num := []byte{inputThirdByte}
					for {
						c, err := inputReader.ReadByte()
						if err != nil {
							return ev, nil
						}
						if c == '~' {
							// interpret num
							switch string(num) {
							case "1", "7":
								ev.KeyCode = KeyHome
								return ev, nil
							case "4", "8":
								ev.KeyCode = KeyEnd
								return ev, nil
							case "3":
								ev.KeyCode = KeyDelete
								return ev, nil
							case "5":
								ev.KeyCode = KeyPageUp
								return ev, nil
							case "6":
								ev.KeyCode = KeyPageDown
								return ev, nil
							}
							return ev, nil
						}
						num = append(num, c)
					}
				}
				// unknown ESC [ sequence
				return ev, nil
			}
		} else if inputSecondByte == 'O' {
			// ESC O H = Home, ESC O F = End on some terminals
			inputThirdByte, err := inputReader.ReadByte()
			if err != nil {
				return ev, nil
			}
			if inputThirdByte == 'H' {
				ev.KeyCode = KeyHome
				return ev, nil
			}
			if inputThirdByte == 'F' {
				ev.KeyCode = KeyEnd
				return ev, nil
			}
			return ev, nil
		}

		// If neither '[' nor 'O' followed ESC, it's likely a lone ESC or alt-key prefix.
		// For now, return KeyEsc but we consumed an extra byte -> if user wants to handle Alt+key,
		// they can treat inputSecondByte + following bytes as the printable char.
		// If inputSecondByte is printable, expose it as Rune with Alt semantics (not implemented here).
		return ev, nil
	}

	// Anything else -> unknown control code (e.g. 0)
	return ev, nil
}

func main() {
	// put stdin into raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to set raw mode:", err)
		os.Exit(1)
	}

	// Disable Ctrl-C and Ctrl-Z signal generation
	if err := terminalSignalsDisable(int(os.Stdin.Fd())); err != nil {
		fmt.Fprintln(os.Stderr, "failed to disable signals:", err)
	}

	// restores the terminal settings after program exit or abort
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Println("Raw mode enabled. Press keys (Ctrl-Q to quit).")
	reader := bufio.NewReader(os.Stdin)

	for {
		ev, err := readKey(reader)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read error:", err)
			break
		}

		// Quit on Ctrl-Q (ASCII 17)
		if ev.Ctrl && ev.Rune == 'q' {
			fmt.Println("\nQuit (Ctrl-Q). Restoring terminal and exiting.")
			break
		}

		// print what we saw
		switch ev.KeyCode {
		case KeyRune:
			if ev.Ctrl {
				fmt.Printf("Ctrl + %c\n", ev.Rune)
			} else {
				fmt.Printf("Rune: %q\n", ev.Rune)
			}
		case KeyEnter:
			fmt.Println("Enter")
		case KeyArrowUp:
			fmt.Println("ArrowUp")
		case KeyArrowDown:
			fmt.Println("ArrowDown")
		case KeyArrowLeft:
			fmt.Println("ArrowLeft")
		case KeyArrowRight:
			fmt.Println("ArrowRight")
		case KeyHome:
			fmt.Println("Home")
		case KeyEnd:
			fmt.Println("End")
		case KeyPageUp:
			fmt.Println("PageUp")
		case KeyPageDown:
			fmt.Println("PageDown")
		case KeyDelete:
			fmt.Println("Delete")
		case KeyEsc:
			fmt.Println("Escape")
		default:
			fmt.Println("Unknown key")
		}
	}
}
