/*
Minimal terminal-based text editor with basic features and no dependencies for Linux, macOS and Windows.
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const editorVersion = "0.1"

// ANSI escape sequences
const (
	ansiCursorHide                    = "\033[?25l"
	ansiCursorPositionMove            = "\033[%d;%dH"
	ansiCursorPositionMoveToOffScreen = "\033[999;999H"
	ansiCursorPositionRequest         = "\033[6n"
	ansiCursorPositionRestore         = "\0338"
	ansiCursorPositionSave            = "\0337"
	ansiCursorPositionToHome          = "\033[H"
	ansiCursorShow                    = "\033[?25h"
	ansiLineClear                     = "\033[K"
	ansiScreenAltOff                  = "\033[?1049l"
	ansiScreenAltOn                   = "\033[?1049h"
	ansiScreenClear                   = "\033[2J"
	ansiScrollbackClear               = "\033[3J"
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

var cursorIndexX, cursorIndexY int
var editorLines []string // current in-memory buffer lines

// readKeyBlocking reads from stdin (one or more bytes) and returns a KeyEvent.
// It assumes stdin is in raw mode.
func readKeyBlocking(inputReader *bufio.Reader) (KeyEvent, error) {
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

func getTerminalSize(fd int) (columns, rows int, err error) {
	columns, rows, err = term.GetSize(fd)
	if err == nil && columns > 0 && rows > 0 {
		return columns, rows, nil
	}

	// Fallback: use cursor position query (CSI 6n)
	fmt.Print(ansiCursorPositionSave)
	fmt.Print(ansiCursorPositionMoveToOffScreen)
	fmt.Print(ansiCursorPositionRequest)

	// Read the response: ESC [ rows ; cols R
	responceBuffer := make([]byte, 32)
	os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	responseSize, _ := os.Stdin.Read(responceBuffer)
	os.Stdin.SetReadDeadline(time.Time{}) // clear deadline

	// Restore cursor
	fmt.Print(ansiCursorPositionRestore)

	// Parse response if valid
	if responseSize > 0 {
		// Expected: ESC [ rows ; cols R
		esc := bytes.IndexByte(responceBuffer[:responseSize], '[')
		rowsAndCols := bytes.IndexByte(responceBuffer[:responseSize], 'R')
		if esc >= 0 && rowsAndCols > esc {
			var rows, cols int
			if _, perr := fmt.Sscanf(string(responceBuffer[esc+1:rowsAndCols]), "%d;%d", &rows, &cols); perr == nil {
				if cols > 0 && rows > 0 {
					return cols, rows, nil
				}
			}
		}
	}

	const defaultRows = 25
	const defaultCols = 80

	// Fallback to safe default
	return defaultRows, defaultCols, fmt.Errorf("could not determine terminal size, using defaults %dx%d", defaultRows, defaultCols)
}

func drawRows(buf *bytes.Buffer, terminalColumns, terminalRows int) {
	for i := 0; i < terminalRows; i++ {
		buf.WriteString(ansiLineClear)

		if len(editorLines) == 0 {
			welcomeRow := terminalRows / 3

			if i == welcomeRow {
				welcome := fmt.Sprintf("VimGo -- version %s", editorVersion)
				buf.WriteString("~")

				welcomeText := welcome
				if len(welcomeText) > terminalColumns {
					welcomeText = welcomeText[:terminalColumns]
				}
				padding := (terminalColumns - len(welcome)) / 2
				if padding > 0 {
					buf.WriteString(strings.Repeat(" ", padding))
				}
				buf.WriteString(welcomeText)
			} else {
				buf.WriteString("~")
			}
		} else {
			if i < len(editorLines) {
				line := editorLines[i]
				if len(line) > terminalColumns {
					line = line[:terminalColumns]
				}
				buf.WriteString(line)
			} else {
				buf.WriteString("~")
			}
		}

		if i < terminalRows-1 {
			buf.WriteString("\r\n")
		}
	}
}

func refreshTerminal(columns, rows int) error {
	var buf bytes.Buffer

	buf.WriteString(ansiCursorHide)

	// Fullscreen - Accumulate screen update in buffer
	buf.WriteString(ansiScrollbackClear)
	buf.WriteString(ansiCursorPositionToHome)

	drawRows(&buf, columns, rows)

	buf.WriteString(fmt.Sprintf(ansiCursorPositionMove, cursorIndexY+1, cursorIndexX+1))
	buf.WriteString(ansiCursorShow)

	// Single write
	_, writeErr := os.Stdout.Write(buf.Bytes())

	return writeErr
}

func editorMoveCursor(ev KeyEvent, terminalColumns, terminalRows int) {
	// Vim-style movement: h, j, k, l
	switch ev.Rune {
	case 'h':
		if cursorIndexX > 0 {
			cursorIndexX--
		}
	case 'l':
		if cursorIndexX < terminalColumns-1 {
			cursorIndexX++
		}
	case 'k':
		if cursorIndexY > 0 {
			cursorIndexY--
		}
	case 'j':
		if cursorIndexY < terminalRows-1 {
			cursorIndexY++
		}
	}

	// Page Up/Down and Home/End navigation.
	switch ev.KeyCode {
	case KeyPageUp:
		cursorIndexY = 0
	case KeyPageDown:
		if terminalRows > 0 {
			cursorIndexY = terminalRows - 1
		}
	case KeyHome:
		cursorIndexX = 0
	case KeyEnd:
		if terminalColumns > 0 {
			cursorIndexX = terminalColumns - 1
		}
	}
}

func editorOpen(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		editorLines = []string{}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		editorLines = append(editorLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func main() {
	// put stdin into raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Fatal error during setting raw mode: %v", err)
	}

	stdin := int(os.Stdin.Fd())
	stdout := int(os.Stdout.Fd())

	if err := terminalRawConfigure(stdin); err != nil {
		panic(err)
	}
	if err := terminalRawConfigure(stdout); err != nil {
		panic(err)
	}

	// enter new screen buffer
	fmt.Print(ansiScreenAltOn)
	// leave new screen buffer
	defer fmt.Print(ansiScreenAltOff)

	// restores the terminal settings after program exit or abort
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	reader := bufio.NewReader(os.Stdin)
	keyChannel := make(chan KeyEvent, 1)

	// Start a single goroutine that continuously reads key events.
	go func() {
		for {
			keyEvent, err := readKeyBlocking(reader)
			if err != nil {
				close(keyChannel)
				return
			}
			select {
			case keyChannel <- keyEvent:
			default:
				// drop this key press if main loop hasn't consumed the previous event
			}
		}
	}()

	if len(os.Args) > 1 {
		if err := openEditor(os.Args[1]); err != nil {
			log.Fatalf("Fatal error during opening the file %s: %v", os.Args[1], err)
		}
	}

	for {
		terminalColumns, terminalRows, err := getTerminalSize(int(os.Stdout.Fd()))
		if err != nil {
			log.Fatalf("Fatal error during reading the number of terminal columns and rows: %w", err)
		}
		if err := refreshTerminal(terminalColumns, terminalRows); err != nil {
			log.Fatalf("Fatal error during refreshing screen: %v", err)
		}

		select {
		case ev, ok := <-keyChannel:
			if !ok {
				return
			}

			editorMoveCursor(ev, terminalColumns, terminalRows)

			// Quit on Ctrl-Q
			if ev.Ctrl && ev.Rune == 'q' {
				fmt.Println("\nQuit (Ctrl-Q). Restoring terminal and exiting.")
				return
			}
		}
	}
}
