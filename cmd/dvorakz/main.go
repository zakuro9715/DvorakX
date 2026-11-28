package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zakuro9715/DvorakZ/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args, run); err != nil {
		exitf("%s", cli.FormatError(err, colorOutput(os.Stderr)))
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func colorOutput(file *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
