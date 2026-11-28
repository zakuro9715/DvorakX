package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/cover"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gocov2lcov:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: gocov2lcov in_gocov out_lcov")
	}
	input, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

	module, root := localModule()
	var output bytes.Buffer
	if err := convert(input, &output, module, root); err != nil {
		return err
	}
	return os.WriteFile(args[1], output.Bytes(), 0o666)
}

// localModule finds the module containing the converter's working directory.
func localModule() (module, root string) {
	dir, err := os.Getwd()
	if err != nil {
		return "", ""
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			return modfile.ModulePath(data), dir
		}
		parent := filepath.Dir(dir)
		if !errors.Is(err, os.ErrNotExist) || parent == dir {
			return "", ""
		}
		dir = parent
	}
}

func convert(input io.Reader, output io.Writer, prefix, root string) error {
	reader := bufio.NewReader(input)
	header, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read coverage header: %w", err)
	}
	switch strings.TrimRight(header, "\r\n") {
	case "mode: set", "mode: count", "mode: atomic":
	default:
		return fmt.Errorf("invalid coverage header %q", strings.TrimRight(header, "\r\n"))
	}
	profiles, err := cover.ParseProfilesFromReader(io.MultiReader(strings.NewReader(header), reader))
	if err != nil {
		return fmt.Errorf("parse coverage profile: %w", err)
	}

	files := make(map[string]map[int]int, len(profiles))
	prefix = strings.TrimRight(prefix, "/")
	for _, profile := range profiles {
		name := profile.FileName
		if prefix != "" {
			if relative, ok := strings.CutPrefix(name, prefix+"/"); ok {
				name = relative
				if root != "" {
					name = filepath.ToSlash(filepath.Join(root, filepath.FromSlash(relative)))
				}
			}
		}
		counts := files[name]
		if counts == nil {
			counts = make(map[int]int)
			files[name] = counts
		}
		if err := addBlocks(counts, profile.Blocks); err != nil {
			return fmt.Errorf("%s: %w", profile.FileName, err)
		}
	}

	// bufio.Writer retains the first write error and returns it from Flush.
	writer := bufio.NewWriter(output)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if counts := files[name]; len(counts) != 0 {
			writeRecord(writer, name, counts)
		}
	}
	return writer.Flush()
}

func addBlocks(counts map[int]int, blocks []cover.ProfileBlock) error {
	for _, block := range blocks {
		if block.StartLine < 1 || block.StartCol < 1 || block.EndCol < 1 ||
			block.EndLine < block.StartLine || (block.EndLine == block.StartLine && block.EndCol < block.StartCol) {
			return fmt.Errorf("invalid block position %d.%d,%d.%d", block.StartLine, block.StartCol, block.EndLine, block.EndCol)
		}
		if block.NumStmt == 0 || (block.StartLine == block.EndLine && block.StartCol == block.EndCol) {
			continue
		}
		end := block.EndLine
		if block.EndCol == 1 {
			end--
		}
		for line := range end - block.StartLine + 1 {
			number := block.StartLine + line
			counts[number] = max(counts[number], block.Count)
		}
	}
	return nil
}

func writeRecord(writer *bufio.Writer, name string, counts map[int]int) {
	_, _ = fmt.Fprintf(writer, "SF:%s\n", name)
	hit := 0
	for _, line := range slices.Sorted(maps.Keys(counts)) {
		count := counts[line]
		_, _ = fmt.Fprintf(writer, "DA:%d,%d\n", line, count)
		if count > 0 {
			hit++
		}
	}
	_, _ = fmt.Fprintf(writer, "LF:%d\nLH:%d\nend_of_record\n", len(counts), hit)
}
