package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	urfavecli "github.com/urfave/cli/v3"

	"github.com/zakuro9715/DvorakZ/internal/config"
	"github.com/zakuro9715/DvorakZ/internal/config/parser"
	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

// Runner starts the platform-specific remapping backend.
type Runner func(mappings *keyboard.Mappings, stop <-chan struct{}, newObserver func() *keyboard.InputObserver, debug bool) error

// Run parses command-line arguments and runs the remapper.
func Run(ctx context.Context, args []string, runner Runner) error {
	command := &urfavecli.Command{
		Name:  "dvorakz",
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
	configPath, err = resolveConfigPath(configPath, executablePath)
	if err != nil {
		return err
	}
	loaded, err := loadConfig(configPath, displayPath)
	if err != nil {
		return err
	}
	mappings := keyboard.NewMappingsFromConfig(loaded)
	if debug {
		fmt.Fprintf(os.Stderr, "dvorakz: debug: loaded config from %s\n", configPath)
	}

	signalContext, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	runContext, cancel := context.WithCancel(signalContext)
	defer cancel()
	stop := runContext.Done()
	newObserver := func() *keyboard.InputObserver { return keyboard.NewInputObserver(cancel) }

	go watchConfig(configPath, displayPath, mappings, stop, debug)

	//nolint:forbidigo
	fmt.Printf("DvorakZ remapping configured keys; hold Esc for 3 seconds to stop.\n")
	return runner(mappings, stop, newObserver, debug)
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
		fmt.Fprintf(os.Stderr, "dvorakz: watch config %q: %v\n", path, err)
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
					fmt.Fprintf(os.Stderr, "dvorakz: reload config %q: %v\n", path, err)
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

			if err := reloadConfig(path, displayPath, mappings); err != nil {
				fmt.Fprintln(os.Stderr, formatReloadError(path, err, colorOutput(os.Stderr)))
				continue
			}
			if debug {
				fmt.Fprintf(os.Stderr, "dvorakz: debug: reloaded config from %s\n", path)
			}
		}
	}
}

func reloadConfig(path, displayPath string, mappings *keyboard.Mappings) error {
	loaded, err := loadConfig(path, displayPath)
	if err != nil {
		return err
	}
	mappings.ReplaceConfig(loaded)
	return nil
}

func loadConfig(path, displayPath string) (config.Config, error) {
	loaded, err := parser.LoadFile(path)
	if err != nil && displayPath != "" {
		return loaded, &configDisplayError{error: err, path: displayPath}
	}
	return loaded, err
}

type configDisplayError struct {
	error
	path string
}

func (e *configDisplayError) Unwrap() error { return e.error }

// FormatError renders a command failure for display on standard error.
func FormatError(err error, color bool) string {
	if message, ok := formatLocatedErrors(err, color); ok {
		return message
	}
	return fmt.Sprintf("dvorakz: %s", err)
}

type locatedError interface {
	error
	Location() parser.Location
}

func formatLocatedErrors(err error, color bool) (string, bool) {
	var issues []error
	if validation, ok := errors.AsType[*parser.ValidationError](err); ok {
		issues = validation.Unwrap()
	} else if syntax, ok := errors.AsType[*parser.SyntaxError](err); ok {
		issues = []error{syntax}
	}
	if len(issues) == 0 {
		return "", false
	}
	display, hasDisplay := errors.AsType[*configDisplayError](err)
	var output strings.Builder
	for _, issue := range issues {
		located, ok := errors.AsType[locatedError](issue)
		if !ok {
			continue
		}
		at := located.Location()
		if hasDisplay {
			at.Path = display.path
		}
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		writeLocatedError(&output, at, located.Error(), color)
	}
	return output.String(), true
}

func writeLocatedError(output *strings.Builder, at parser.Location, message string, color bool) {
	heading := fmt.Sprintf("[%s:%d, %d]: %s", at.Path, at.Line, at.Column, message)
	offset := min(max(0, at.Column-1), len(at.Text))
	prefix := strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		return ' '
	}, at.Text[:offset])
	end := offset + min(max(0, at.Length), len(at.Text)-offset)
	marker := prefix + strings.Repeat("^", max(1, utf8.RuneCountInString(at.Text[offset:end])))
	if color {
		heading = "\x1b[31m" + heading + "\x1b[0m"
		marker = "\x1b[31m" + marker + "\x1b[0m"
	}
	fmt.Fprintf(output, "%s\n%4d | %s\n     | %s", heading, at.Line, at.Text, marker)
}

func formatReloadError(path string, err error, colors ...bool) string {
	color := len(colors) > 0 && colors[0]
	if message, ok := formatLocatedErrors(err, color); ok {
		return fmt.Sprintf("dvorakz: reload config %q:\n%s", path, message)
	}
	prefix := fmt.Sprintf("dvorakz: reload config %q", path)
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

func resolveConfigPath(explicitPath, executablePath string) (string, error) {
	if explicitPath != "" {
		return filepath.Clean(explicitPath), nil
	}
	if executablePath == "" {
		return "", errors.New("resolve default config path: executable path is empty")
	}
	return filepath.Join(filepath.Dir(executablePath), "config.toml"), nil
}
