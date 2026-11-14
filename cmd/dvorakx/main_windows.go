//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zakuro9715/DvorakX/internal/config"
	"github.com/zakuro9715/DvorakX/internal/keyboard"
	"github.com/zakuro9715/DvorakX/internal/windows"
)

func main() {
	var configPath string
	var debug bool
	flag.StringVar(&configPath, "config", "", "path to TOML configuration file")
	flag.BoolVar(&debug, "debug", false, "write remapping diagnostics to standard error")
	flag.Parse()

	executablePath, err := os.Executable()
	if err != nil {
		exitf("resolve executable path: %v", err)
	}
	configPath, err = config.ResolveConfigPath(configPath, executablePath)
	if err != nil {
		exitf("%v", err)
	}
	loaded, err := config.LoadFile(configPath)
	if err != nil {
		exitf("%v", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "dvorakv: debug: loaded %d mapping(s) from %s\n", len(loaded.Remap), configPath)
	}

	stop := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		close(stop)
	}()

	mappings := keyboard.NewMappings(loaded.Remap)
	go watchConfig(configPath, mappings, stop, debug)

	fmt.Printf("DvorakX remapping %d key(s); press Ctrl+C to stop.\n", len(loaded.Remap))
	var debugOutput *os.File
	if debug {
		debugOutput = os.Stderr
	}
	if err := windows.Run(mappings, stop, debugOutput); err != nil {
		exitf("%v", err)
	}
}

type configFileState struct {
	size    int64
	modTime time.Time
}

func watchConfig(path string, mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dvorakv: watch config %q: %v\n", path, err)
		return
	}
	previous := configFileState{size: info.Size(), modTime: info.ModTime()}
	lastFailure := ""

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			info, err := os.Stat(path)
			if err != nil {
				if message := err.Error(); message != lastFailure {
					fmt.Fprintf(os.Stderr, "dvorakv: reload config %q: %v\n", path, err)
					lastFailure = message
				}
				continue
			}
			lastFailure = ""
			current := configFileState{size: info.Size(), modTime: info.ModTime()}
			if current == previous {
				continue
			}
			previous = current

			loaded, err := config.LoadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "dvorakv: reload config %q: %v\n", path, err)
				continue
			}
			mappings.Replace(loaded.Remap)
			if debug {
				fmt.Fprintf(os.Stderr, "dvorakv: debug: reloaded %d mapping(s) from %s\n", len(loaded.Remap), path)
			}
		}
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dvorakv: "+format+"\n", args...)
	os.Exit(1)
}
