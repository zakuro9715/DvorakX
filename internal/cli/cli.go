package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	urfavecli "github.com/urfave/cli/v3"

	"github.com/zakuro9715/DvorakX/internal/config"
	"github.com/zakuro9715/DvorakX/internal/keyboard"
)

// Runner starts the platform-specific remapping backend.
type Runner func(mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) error

// Run parses command-line arguments and runs the remapper.
func Run(ctx context.Context, args []string, runner Runner) error {
	command := &urfavecli.Command{
		Name:  "dvorakx",
		Usage: "remap keyboard keys",
		Flags: []urfavecli.Flag{
			&urfavecli.StringFlag{
				Name:  "config",
				Usage: "path to TOML configuration file",
			},
			&urfavecli.BoolFlag{
				Name:  "debug",
				Usage: "write remapping diagnostics to standard error",
			},
		},
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			return run(ctx, command.String("config"), command.Bool("debug"), runner)
		},
	}
	return command.Run(ctx, args)
}

func run(ctx context.Context, configPath string, debug bool, runner Runner) error {
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	configPath, err = config.ResolveConfigPath(configPath, executablePath)
	if err != nil {
		return err
	}
	loaded, err := config.LoadFile(configPath)
	if err != nil {
		return err
	}
	if debug {
		fmt.Fprintf(os.Stderr, "dvorakv: debug: loaded %d mapping(s) from %s\n", len(loaded.Remap), configPath)
	}

	stop := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	runContext, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-runContext.Done():
		case <-signals:
		}
		close(stop)
	}()

	mappings := keyboard.NewMappings(loaded.Remap)
	go watchConfig(configPath, mappings, stop, debug)

	fmt.Printf("DvorakX remapping %d key(s); press Ctrl+C to stop.\n", len(loaded.Remap))
	return runner(mappings, stop, debug)
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
