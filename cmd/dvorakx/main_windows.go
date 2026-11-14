//go:build windows

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zakuro9715/DvorakX/internal/cli"
	"github.com/zakuro9715/DvorakX/internal/keyboard"
	"github.com/zakuro9715/DvorakX/internal/windows"
)

func main() {
	if err := cli.Run(context.Background(), os.Args, run); err != nil {
		exitf("%v", err)
	}
}

func run(mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) error {
	var debugOutput *os.File
	if debug {
		debugOutput = os.Stderr
	}
	return windows.Run(mappings, stop, debugOutput)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dvorakv: "+format+"\n", args...)
	os.Exit(1)
}
