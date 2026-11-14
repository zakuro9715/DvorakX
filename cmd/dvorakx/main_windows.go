//go:build windows

package main

import (
	"os"

	"github.com/zakuro9715/DvorakX/internal/keyboard"
	"github.com/zakuro9715/DvorakX/internal/windows"
)

func run(mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) error {
	var debugOutput *os.File
	if debug {
		debugOutput = os.Stderr
	}
	return windows.Run(mappings, stop, debugOutput)
}
