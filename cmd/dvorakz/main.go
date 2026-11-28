package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zakuro9715/DvorakZ/internal/cli"
	"github.com/zakuro9715/DvorakZ/internal/config/parser"
)

func main() {
	if err := cli.Run(context.Background(), os.Args, run); err != nil {
		exitf("%s", formatError(err))
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func formatError(err error) string {
	formatter := parser.PrettyFormatter{Color: colorOutput(os.Stderr)}
	var output strings.Builder
	if diagnosticList, ok := errors.AsType[*parser.DiagnosticsError](err); ok {
		_ = formatter.FormatTo(&output, diagnosticList.Items)
		return output.String()
	}
	fmt.Fprintf(&output, "dvorakz: %s", err)
	return output.String()
}

func colorOutput(file *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
