//go:build windows

package main

import (
	"os"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/windows"
)

func run(mappings *keyboard.Mappings, stop <-chan struct{}, newObserver func() *keyboard.InputObserver, debug bool) error {
	var debugOutput *os.File
	if debug {
		debugOutput = os.Stderr
	}
	return windows.Run(mappings, stop, newObserver, debugOutput)
}
