//go:build darwin && cgo

package main

import (
	"io"
	"os"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/macos"
)

func run(mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) error {
	var output io.Writer
	if debug {
		output = os.Stderr
	}
	return macos.Run(mappings, stop, output)
}
