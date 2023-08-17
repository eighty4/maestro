package main

import (
	"golang.org/x/term"
	"os"
)

type KeyCmd int

const (
	NotCmd KeyCmd = -1
	Space  KeyCmd = 0
	Enter  KeyCmd = 1
	Up     KeyCmd = 2
	Down   KeyCmd = 3
)

func readKeyCmd(acceptKeyCmds []KeyCmd) (KeyCmd, error) {
	for {
		if keyCmd, err := readKeyCmdInput(); err != nil {
			return NotCmd, err
		} else {
			for _, accept := range acceptKeyCmds {
				if keyCmd == accept {
					return keyCmd, nil
				}
			}
		}
	}
}

func readKeyCmdInput() (KeyCmd, error) {
	termFd := int(os.Stdin.Fd())
	termState, err := term.MakeRaw(termFd)
	if err != nil {
		return NotCmd, err
	}
	bytes := make([]byte, 3)
	bytesRead, err := os.Stdin.Read(bytes)
	_ = term.Restore(termFd, termState)
	if err != nil {
		return NotCmd, err
	}
	if bytesRead == 1 {
		if bytes[0] == 32 {
			return Space, nil
		} else if bytes[0] == 13 {
			return Enter, nil
		} else if bytes[0] == 3 {
			os.Exit(0)
		}
	} else if bytesRead == 3 {
		if bytes[0] == 27 && bytes[1] == 91 {
			if bytes[2] == 65 {
				return Up, nil
			} else if bytes[2] == 66 {
				return Down, nil
			}
		}
	}
	return NotCmd, nil
}
