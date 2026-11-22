package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
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
	displayPath := ""
	if configPath == "" {
		displayPath = "config.toml"
	}
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	configPath, err = config.ResolveConfigPath(configPath, executablePath)
	if err != nil {
		return err
	}
	loaded, err := loadConfig(configPath, displayPath)
	if err != nil {
		return err
	}
	mappings, err := keyboard.NewMappingsFromConfig(loaded)
	if err != nil {
		return fmt.Errorf("prepare config: %w", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "dvorakv: debug: loaded config from %s\n", configPath)
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

	go watchConfig(configPath, displayPath, mappings, stop, debug)

	//nolint:forbidigo
	fmt.Printf("DvorakX remapping configured keys; press Ctrl+C to stop.\n")
	return runner(mappings, stop, debug)
}

type configFileState struct {
	size    int64
	modTime time.Time
}

func watchConfig(path, displayPath string, mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) {
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

			loaded, err := loadConfig(path, displayPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, formatConfigError("reload", path, err, colorOutput(os.Stderr)))
				continue
			}
			if err := mappings.ReplaceConfig(loaded); err != nil {
				fmt.Fprintln(os.Stderr, formatConfigError("reload", path, err, colorOutput(os.Stderr)))
				continue
			}
			if debug {
				fmt.Fprintf(os.Stderr, "dvorakv: debug: reloaded config from %s\n", path)
			}
		}
	}
}

func loadConfig(path, displayPath string) (config.Config, error) {
	loaded, err := config.LoadFile(path)
	if diagnostics, ok := errors.AsType[*config.DiagnosticsError](err); ok && displayPath != "" {
		for index := range diagnostics.Items {
			diagnostics.Items[index].Path = displayPath
		}
	}
	return loaded, err
}

func formatConfigError(operation, path string, err error, colors ...bool) string {
	color := len(colors) > 0 && colors[0]
	formatter := config.PrettyFormatter{Color: color}
	var output strings.Builder
	if diagnostics, ok := errors.AsType[*config.DiagnosticsError](err); ok {
		fmt.Fprintf(&output, "dvorakv: %s config %q:\n", operation, path)
		_ = formatter.FormatTo(&output, diagnostics.Items)
		return output.String()
	}
	prefix := fmt.Sprintf("dvorakv: %s config %q", operation, path)
	lines := strings.Split(err.Error(), "\n")
	for index, line := range lines {
		lines[index] = prefix + ": " + line
	}
	return strings.Join(lines, "\n")
}

func colorOutput(file *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
