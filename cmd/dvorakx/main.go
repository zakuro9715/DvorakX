package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zakuro9715/DvorakX/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args, run); err != nil {
		exitf("%v", err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dvorakv: "+format+"\n", args...)
	os.Exit(1)
}
