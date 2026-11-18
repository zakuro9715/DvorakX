package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zakuro9715/DvorakX/internal/cli"
	"github.com/zakuro9715/DvorakX/internal/config"
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
	formatter := config.PrettyFormatter{Color: colorOutput(os.Stderr)}
	var output strings.Builder
	if diagnosticList, ok := err.(*config.Diagnostics); ok {
		_ = formatter.FormatTo(&output, diagnosticList.Items)
		return output.String()
	}
	fmt.Fprintf(&output, "dvorakv: %s", err)
	return output.String()
}

func colorOutput(file *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
