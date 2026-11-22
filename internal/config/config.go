package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
)

// Config contains the validated configuration file structure.
type Config struct {
	Meta   MetaConfig
	Global LayerConfig
	Source LayerConfig
	Layers map[string]LayerConfig

	// Remap, Noop, and SendKeys are the resolved mappings consumed by the
	// keyboard backend.
	Remap    map[uint16]uint16
	Noop     map[uint16]bool
	SendKeys map[uint16][]uint16
}

// MetaConfig contains settings from the [meta] table.
type MetaConfig struct {
	Default string
}

// LayerConfig contains a layer definition, actions, and direct remaps.
type LayerConfig struct {
	Definition Definition
	Actions    map[string]ActionSpec
	Remaps     []map[string]string
}

// Definition preserves the source form of a layer definition.
type Definition struct {
	Text string
	Keys []string
}

type definitionToken struct {
	value   string
	escaped bool
	action  ActionSpec
}

// Action identifies a configured action.
type Action uint8

const (
	noAction Action = iota
	NoopAction
	SendKeysAction
)

// ActionSpec contains an action and its resolved key arguments.
type ActionSpec struct {
	Kind Action
	Keys []uint16
}

type sourcePosition struct {
	line        int
	column      int
	length      int
	sourceLine  string
	sourceLines []string
}

func parseActions(value any, path string, positions sourcePositions, diagnostics *Diagnostics) map[string]ActionSpec {
	actions := make(map[string]ActionSpec)
	if value == nil {
		return actions
	}
	table, ok := value.(map[string]any)
	if !ok {
		addDiagnostic(diagnostics, positions, path, "%s must be a TOML table", path)
		return actions
	}
	for _, name := range sortedKeys(table) {
		body, ok := table[name].(map[string]any)
		if !ok || len(body) != 1 {
			addDiagnostic(diagnostics, positions, path+"."+name, "action %q must contain exactly one action", name)
			continue
		}
		switch {
		case body["noop"] != nil:
			arguments, ok := body["noop"].([]any)
			if !ok || len(arguments) != 0 {
				addDiagnostic(diagnostics, positions, path+"."+name+".noop", "noop action does not accept arguments")
				continue
			}
			actions[name] = ActionSpec{Kind: NoopAction}
		case body["send-keys"] != nil:
			keys, ok := parseSendKeys(body["send-keys"], path+"."+name+".send-keys", positions, diagnostics)
			if !ok {
				continue
			}
			actions[name] = ActionSpec{Kind: SendKeysAction, Keys: keys}
		default:
			addDiagnostic(diagnostics, positions, path+"."+name, "unsupported action in %q", name)
		}
	}
	return actions
}

func parseSendKeys(value any, path string, positions sourcePositions, diagnostics *Diagnostics) ([]uint16, bool) {
	var names []string
	switch value := value.(type) {
	case string:
		names = strings.Fields(value)
	case []any:
		names = make([]string, len(value))
		for index, item := range value {
			name, ok := item.(string)
			if !ok {
				addDiagnostic(diagnostics, positions, fmt.Sprintf("%s[%d]", path, index), "send-keys argument at position %d must be a string", index)
				return nil, false
			}
			names[index] = name
		}
	default:
		addDiagnostic(diagnostics, positions, path, "send-keys must be a string or string array")
		return nil, false
	}
	if len(names) == 0 {
		addDiagnostic(diagnostics, positions, path, "send-keys must contain at least one key")
		return nil, false
	}
	keys := make([]uint16, len(names))
	for index, name := range names {
		key, ok := virtualKey(name)
		if !ok {
			addDiagnostic(diagnostics, positions, fmt.Sprintf("%s[%d]", path, index), "unsupported send-keys key %q", name)
			return nil, false
		}
		keys[index] = key
	}
	return keys, true
}

func resolveAction(value string, actions map[string]ActionSpec) (ActionSpec, bool) {
	if !strings.HasPrefix(value, "@") || len(value) == 1 {
		return ActionSpec{}, false
	}
	action, ok := actions[value[1:]]
	return action, ok
}

func resolveDefinitionActions(tokens []definitionToken, actions map[string]ActionSpec, path string, positions sourcePositions, diagnostics *Diagnostics) ([]definitionToken, bool) {
	valid := true
	for index := range tokens {
		if tokens[index].escaped || !strings.HasPrefix(tokens[index].value, "@") {
			continue
		}
		action, ok := resolveAction(tokens[index].value, actions)
		if !ok {
			addDiagnostic(diagnostics, positions, fmt.Sprintf("%s[%d]", path, index), "undefined action %q", strings.TrimPrefix(tokens[index].value, "@"))
			valid = false
			continue
		}
		tokens[index].action = action
	}
	return tokens, valid
}

type sourcePositions map[string]sourcePosition

// ResolveConfigPath returns the explicit path, or config.toml next to the executable.
func ResolveConfigPath(explicitPath, executablePath string) (string, error) {
	if explicitPath != "" {
		return filepath.Clean(explicitPath), nil
	}
	if executablePath == "" {
		return "", fmt.Errorf("resolve default config path: executable path is empty")
	}
	return filepath.Join(filepath.Dir(executablePath), "config.toml"), nil
}

func collectSourcePositions(data []byte) (sourcePositions, error) {
	positions := make(sourcePositions)
	sourceLines := splitSourceLines(data)
	parser := &unstable.Parser{}
	parser.Reset(data)
	tablePath := ""
	for parser.NextExpression() {
		expression := parser.Expression()
		switch expression.Kind {
		case unstable.Table, unstable.ArrayTable:
			tablePath = nodeKey(expression)
			if tablePath != "" {
				positions[tablePath] = nodePosition(parser, expression.Child(), sourceLines)
			}
		case unstable.KeyValue:
			recordNodePositions(parser, expression, tablePath, positions, sourceLines)
		}
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}
	return positions, nil
}

func recordNodePositions(parser *unstable.Parser, expression *unstable.Node, tablePath string, positions sourcePositions, sourceLines []string) {
	key := expression.Key()
	if !key.Next() {
		return
	}
	keyNode := key.Node()
	valueNode := expression.Value()
	path := joinPath(tablePath, string(keyNode.Data))
	positions[path] = nodePosition(parser, keyNode, sourceLines)
	recordValuePositions(parser, valueNode, path, positions, sourceLines)
}

func recordValuePositions(parser *unstable.Parser, node *unstable.Node, path string, positions sourcePositions, sourceLines []string) {
	if node == nil {
		return
	}
	switch node.Kind {
	case unstable.Array:
		index := 0
		for child := node.Child(); child != nil; child = child.Next() {
			childPath := fmt.Sprintf("%s[%d]", path, index)
			positions[childPath] = nodePosition(parser, child, sourceLines)
			recordValuePositions(parser, child, childPath, positions, sourceLines)
			index++
		}
	case unstable.InlineTable:
		for child := node.Child(); child != nil; child = child.Next() {
			if child.Kind != unstable.KeyValue {
				continue
			}
			key := child.Key()
			if !key.Next() {
				continue
			}
			keyNode := key.Node()
			childPath := joinPath(path, string(keyNode.Data))
			positions[childPath] = nodePosition(parser, keyNode, sourceLines)
			recordValuePositions(parser, child.Value(), childPath, positions, sourceLines)
		}
	}
}

func nodeKey(node *unstable.Node) string {
	var parts []string
	for key := node.Key(); key.Next(); {
		parts = append(parts, string(key.Node().Data))
	}
	return strings.Join(parts, ".")
}

func nodePosition(parser *unstable.Parser, node *unstable.Node, sourceLines []string) sourcePosition {
	shape := parser.Shape(node.Raw)
	data := parser.Data()
	lineStart := shape.Start.Offset
	for lineStart > 0 && data[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd := shape.Start.Offset
	for lineEnd < len(data) && data[lineEnd] != '\n' {
		lineEnd++
	}
	return sourcePosition{
		line:        shape.Start.Line,
		column:      shape.Start.Column,
		length:      int(node.Raw.Length),
		sourceLine:  strings.TrimSuffix(string(data[lineStart:lineEnd]), "\r"),
		sourceLines: sourceLines,
	}
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func formatParseError(data []byte, err error) error {
	var decodeError *toml.DecodeError
	if errors.As(err, &decodeError) {
		line, column := decodeError.Position()
		sourceLine := sourceLine(data, line)
		diagnostics := NewDiagnostics()
		diagnostics.Add(Error{
			Line:        line,
			Column:      column,
			Length:      1,
			SourceLine:  sourceLine,
			SourceLines: splitSourceLines(data),
			Message:     strings.TrimPrefix(decodeError.Error(), "toml: "),
		})
		return diagnostics
	}
	return fmt.Errorf("parse config: %w", err)
}

func sourceLine(data []byte, line int) string {
	if line < 1 {
		return ""
	}
	start := 0
	for currentLine := 1; currentLine < line; currentLine++ {
		next := strings.IndexByte(string(data[start:]), '\n')
		if next < 0 {
			return ""
		}
		start += next + 1
	}
	end := start
	for end < len(data) && data[end] != '\n' {
		end++
	}
	return strings.TrimSuffix(string(data[start:end]), "\r")
}

func splitSourceLines(data []byte) []string {
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

// LoadFile reads, parses, and validates a configuration file.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	loaded, err := Parse(data)
	if diagnostics, ok := err.(*Diagnostics); ok {
		for index := range diagnostics.Items {
			diagnostics.Items[index].Path = path
		}
	}
	return loaded, err
}

// Parse parses and validates direct remaps and static layer mappings.
func Parse(data []byte) (Config, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, formatParseError(data, err)
	}
	positions, err := collectSourcePositions(data)
	if err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	diagnostics := NewDiagnostics()
	globalRemap, globalNoop, globalSendKeys := parseGlobal(raw, positions, diagnostics)

	remap := make(map[uint16]uint16)
	noop := make(map[uint16]bool)
	sendKeys := make(map[uint16][]uint16)
	hasLayers := resolveLayers(raw, remap, noop, sendKeys, positions, diagnostics)
	if len(diagnostics.Items) != 0 {
		return Config{}, diagnostics
	}
	if !hasLayers && len(globalRemap) == 0 && len(globalNoop) == 0 && len(globalSendKeys) == 0 {
		addDiagnostic(diagnostics, positions, "global", "config must define at least one [global].remaps entry")
	}
	for source, destination := range globalRemap {
		remap[source] = destination
		delete(noop, source)
		delete(sendKeys, source)
	}
	for source := range globalNoop {
		delete(remap, source)
		delete(sendKeys, source)
		noop[source] = true
	}
	for source, keys := range globalSendKeys {
		delete(remap, source)
		delete(noop, source)
		sendKeys[source] = append([]uint16(nil), keys...)
	}
	if len(remap) == 0 && len(noop) == 0 && len(sendKeys) == 0 {
		addDiagnostic(diagnostics, positions, "global.remaps", "config must define at least one mapping")
	}
	if len(diagnostics.Items) != 0 {
		return Config{}, diagnostics
	}
	config := Config{
		Meta:     captureMeta(raw),
		Global:   captureLayer(raw["global"]),
		Source:   captureLayer(raw["src"]),
		Layers:   make(map[string]LayerConfig),
		Remap:    remap,
		Noop:     noop,
		SendKeys: sendKeys,
	}
	for name, value := range raw {
		if name == "meta" || name == "global" || name == "src" {
			continue
		}
		config.Layers[name] = captureLayer(value)
	}
	return config, nil
}

func captureMeta(raw map[string]any) MetaConfig {
	table, _ := raw["meta"].(map[string]any)
	defaultValue, _ := table["default"].(string)
	return MetaConfig{Default: defaultValue}
}

func captureLayer(value any) LayerConfig {
	table, _ := value.(map[string]any)
	layer := LayerConfig{
		Actions: make(map[string]ActionSpec),
		Remaps:  make([]map[string]string, 0),
	}
	if table == nil {
		return layer
	}
	layer.Definition = captureDefinition(table["def"])
	layer.Actions = captureActions(table["actions"])
	layer.Remaps = captureRemaps(table["remaps"])
	return layer
}

func captureDefinition(value any) Definition {
	switch definition := value.(type) {
	case string:
		return Definition{Text: definition}
	case []any:
		keys := make([]string, 0, len(definition))
		for _, value := range definition {
			if key, ok := value.(string); ok {
				keys = append(keys, key)
			}
		}
		return Definition{Keys: keys}
	default:
		return Definition{}
	}
}

func captureActions(value any) map[string]ActionSpec {
	actions := make(map[string]ActionSpec)
	table, _ := value.(map[string]any)
	for name, value := range table {
		body, _ := value.(map[string]any)
		if arguments, ok := body["noop"].([]any); ok && len(arguments) == 0 {
			actions[name] = ActionSpec{Kind: NoopAction}
			continue
		}
		if keys, ok := parseSendKeysForCapture(body["send-keys"]); ok {
			actions[name] = ActionSpec{Kind: SendKeysAction, Keys: keys}
		}
	}
	return actions
}

func parseSendKeysForCapture(value any) ([]uint16, bool) {
	var names []string
	switch value := value.(type) {
	case string:
		names = strings.Fields(value)
	case []any:
		names = make([]string, len(value))
		for index, item := range value {
			name, ok := item.(string)
			if !ok {
				return nil, false
			}
			names[index] = name
		}
	default:
		return nil, false
	}
	if len(names) == 0 {
		return nil, false
	}
	keys := make([]uint16, len(names))
	for index, name := range names {
		key, ok := virtualKey(name)
		if !ok {
			return nil, false
		}
		keys[index] = key
	}
	return keys, true
}

func captureRemaps(value any) []map[string]string {
	var tables []map[string]any
	switch value := value.(type) {
	case map[string]any:
		tables = []map[string]any{value}
	case []any:
		for _, entry := range value {
			if table, ok := entry.(map[string]any); ok {
				tables = append(tables, table)
			}
		}
	}
	remaps := make([]map[string]string, 0, len(tables))
	for _, table := range tables {
		entries := make(map[string]string, len(table))
		for source, destination := range table {
			if destination, ok := destination.(string); ok {
				entries[source] = destination
			}
		}
		remaps = append(remaps, entries)
	}
	return remaps
}

// ResolveMappings returns the mappings consumed by the keyboard backend.
func (c Config) ResolveMappings() (map[uint16]uint16, map[uint16]bool, map[uint16][]uint16) {
	remap := make(map[uint16]uint16, len(c.Remap))
	for source, destination := range c.Remap {
		remap[source] = destination
	}
	noop := make(map[uint16]bool, len(c.Noop))
	for source, enabled := range c.Noop {
		noop[source] = enabled
	}
	sendKeys := make(map[uint16][]uint16, len(c.SendKeys))
	for source, keys := range c.SendKeys {
		sendKeys[source] = append([]uint16(nil), keys...)
	}
	return remap, noop, sendKeys
}

func addDiagnostic(e *Diagnostics, positions sourcePositions, path, format string, args ...any) {
	position := positions[path]
	if position.line == 0 {
		position = sourcePosition{line: 1, column: 1}
	}
	e.Add(Error{
		Line:        position.line,
		Column:      position.column,
		Length:      position.length,
		SourceLine:  position.sourceLine,
		SourceLines: position.sourceLines,
		Message:     fmt.Sprintf(format, args...),
	})
}

func parseGlobal(raw map[string]any, positions sourcePositions, diagnostics *Diagnostics) (map[uint16]uint16, map[uint16]bool, map[uint16][]uint16) {
	value, ok := raw["global"]
	if !ok {
		return map[uint16]uint16{}, map[uint16]bool{}, map[uint16][]uint16{}
	}
	table, ok := value.(map[string]any)
	if !ok {
		addDiagnostic(diagnostics, positions, "global", "[global] must be a TOML table")
		return map[uint16]uint16{}, map[uint16]bool{}, map[uint16][]uint16{}
	}
	remap := make(map[uint16]uint16)
	noop := make(map[uint16]bool)
	sendKeys := make(map[uint16][]uint16)
	actions := parseActions(table["actions"], "global.actions", positions, diagnostics)
	for _, name := range sortedKeys(table) {
		if name != "remaps" && name != "actions" {
			addDiagnostic(diagnostics, positions, "global."+name, "unsupported [global] field %q", name)
		}
	}

	remapValue, ok := table["remaps"]
	if !ok {
		return remap, noop, sendKeys
	}
	layerRemap, layerNoop, layerSendKeys := parseRemaps(remapValue, "global.remaps", actions, positions, diagnostics)
	for source, destination := range layerRemap {
		remap[source] = destination
		delete(noop, source)
		delete(sendKeys, source)
	}
	for source := range layerNoop {
		delete(remap, source)
		delete(sendKeys, source)
		noop[source] = true
	}
	for source, keys := range layerSendKeys {
		delete(remap, source)
		delete(noop, source)
		sendKeys[source] = keys
	}
	return remap, noop, sendKeys
}

func parseRemaps(value any, path string, actions map[string]ActionSpec, positions sourcePositions, diagnostics *Diagnostics) (map[uint16]uint16, map[uint16]bool, map[uint16][]uint16) {
	remap := make(map[uint16]uint16)
	noop := make(map[uint16]bool)
	sendKeys := make(map[uint16][]uint16)
	displayPath := remapDisplayPath(path)
	switch remaps := value.(type) {
	case map[string]any:
		mergeRemap(remap, noop, sendKeys, remaps, actions, positions, diagnostics, path)
	case []any:
		for index, value := range remaps {
			entryPath := fmt.Sprintf("%s[%d]", path, index)
			entries, ok := value.(map[string]any)
			if !ok {
				addDiagnostic(diagnostics, positions, entryPath, "%s[%d] must be a TOML table", displayPath, index)
				continue
			}
			mergeRemap(remap, noop, sendKeys, entries, actions, positions, diagnostics, entryPath)
		}
	default:
		addDiagnostic(diagnostics, positions, path, "%s must be a TOML table or table array", displayPath)
	}
	return remap, noop, sendKeys
}

func remapDisplayPath(path string) string {
	separator := strings.IndexByte(path, '.')
	if separator < 0 {
		return path
	}
	return "[" + path[:separator] + "]" + path[separator:]
}

func mergeRemap(remap map[uint16]uint16, noop map[uint16]bool, sendKeys map[uint16][]uint16, entries map[string]any, actions map[string]ActionSpec, positions sourcePositions, diagnostics *Diagnostics, path string) {
	for _, sourceName := range sortedKeysByPosition(entries, positions, path) {
		value := entries[sourceName]
		destinationName, ok := value.(string)
		if !ok {
			addDiagnostic(diagnostics, positions, path+"."+sourceName, "invalid remap destination for %q: must be a string", sourceName)
			continue
		}
		unescapedSourceName, err := unescapeKeyName(sourceName)
		if err != nil {
			addDiagnostic(diagnostics, positions, path+"."+sourceName, "invalid remap source key: %v", err)
			continue
		}
		if strings.HasPrefix(destinationName, "@") {
			action, ok := resolveAction(destinationName, actions)
			if !ok {
				addDiagnostic(diagnostics, positions, path+"."+sourceName, "undefined action %q", destinationName[1:])
				continue
			}
			source, ok := virtualKey(unescapedSourceName)
			if !ok {
				addDiagnostic(diagnostics, positions, path+"."+sourceName, "unsupported remap source key %q", unescapedSourceName)
				continue
			}
			switch action.Kind {
			case NoopAction:
				noop[source] = true
			case SendKeysAction:
				sendKeys[source] = append([]uint16(nil), action.Keys...)
			}
			continue
		}
		destinationName, err = unescapeKeyName(destinationName)
		if err != nil {
			addDiagnostic(diagnostics, positions, path+"."+sourceName, "invalid remap destination key: %v", err)
			continue
		}
		source, ok := virtualKey(unescapedSourceName)
		if !ok {
			addDiagnostic(diagnostics, positions, path+"."+sourceName, "unsupported remap source key %q", unescapedSourceName)
			continue
		}
		destination, ok := virtualKey(destinationName)
		if !ok {
			addDiagnostic(diagnostics, positions, path+"."+sourceName, "unsupported remap destination key %q", destinationName)
			continue
		}
		remap[source] = destination
	}
}

func resolveLayers(raw map[string]any, remap map[uint16]uint16, noop map[uint16]bool, sendKeys map[uint16][]uint16, positions sourcePositions, diagnostics *Diagnostics) bool {
	_, hasMeta := raw["meta"]
	sourceValue, hasSource := raw["src"]
	var layerNames []string
	for name := range raw {
		switch name {
		case "global", "meta", "src":
		default:
			layerNames = append(layerNames, name)
		}
	}
	sort.SliceStable(layerNames, func(left, right int) bool {
		leftPosition := positions[layerNames[left]]
		rightPosition := positions[layerNames[right]]
		if leftPosition.line != rightPosition.line {
			return leftPosition.line < rightPosition.line
		}
		if leftPosition.column != rightPosition.column {
			return leftPosition.column < rightPosition.column
		}
		return layerNames[left] < layerNames[right]
	})
	if !hasMeta && !hasSource {
		if len(layerNames) != 0 {
			for _, layerName := range layerNames {
				addDiagnostic(diagnostics, positions, layerName, "layer table %q requires [meta] and [src]", layerName)
				parseLayer(raw[layerName], layerName, positions, diagnostics)
			}
		}
		return false
	}
	if !hasMeta {
		addDiagnostic(diagnostics, positions, "meta", "layer configuration requires [meta]")
	}
	if !hasSource {
		addDiagnostic(diagnostics, positions, "src", "layer configuration requires [src]")
	}
	if !hasMeta || !hasSource {
		for _, layerName := range layerNames {
			parseLayer(raw[layerName], layerName, positions, diagnostics)
		}
		return false
	}

	meta, ok := raw["meta"].(map[string]any)
	if !ok {
		addDiagnostic(diagnostics, positions, "meta", "[meta] must be a TOML table")
		return false
	}
	for _, name := range sortedKeys(meta) {
		if name != "default" {
			addDiagnostic(diagnostics, positions, "meta."+name, "unsupported [meta] field %q", name)
		}
	}
	defaultValue, ok := meta["default"].(string)
	if !ok {
		addDiagnostic(diagnostics, positions, "meta.default", "[meta].default must be a string layer reference")
		return false
	}
	defaultLayer, ok := strings.CutPrefix(defaultValue, "&")
	if !ok || defaultLayer == "" {
		addDiagnostic(diagnostics, positions, "meta.default", "[meta].default must reference a layer with &")
		return false
	}

	sourceActions := parseActions(nil, "src.actions", positions, diagnostics)
	if sourceTable, ok := sourceValue.(map[string]any); ok {
		sourceActions = parseActions(sourceTable["actions"], "src.actions", positions, diagnostics)
	}
	source, sourceOK := parseDefinition(sourceValue, "[src]", sourceActions, positions, diagnostics)
	layerValue, ok := raw[defaultLayer]
	if !ok {
		addDiagnostic(diagnostics, positions, "meta.default", "[meta].default references undefined layer %q", defaultLayer)
		return false
	}
	for _, layerName := range layerNames {
		if layerName != defaultLayer {
			parseLayer(raw[layerName], layerName, positions, diagnostics)
		}
	}
	layerTable, _ := layerValue.(map[string]any)
	actions := parseActions(layerTable["actions"], defaultLayer+".actions", positions, diagnostics)
	layerRemap, layerNoop, layerSendKeys := parseLayerRemaps(layerTable, defaultLayer, actions, positions, diagnostics)
	destination, destinationOK := parseDefinition(layerValue, "["+defaultLayer+"]", actions, positions, diagnostics)
	if !sourceOK || !destinationOK {
		return false
	}
	if len(destination) == 1 && !destination[0].escaped && destination[0].value == "#src" {
		destination = source
	}
	if len(source) != len(destination) {
		addDiagnostic(diagnostics, positions, defaultLayer+".def", "source and default layer definitions have different lengths: %d and %d", len(source), len(destination))
		return false
	}

	for index, sourceToken := range source {
		sourceKey, sourceOK := virtualKey(sourceToken.value)
		if !sourceOK {
			addDiagnostic(diagnostics, positions, fmt.Sprintf("src.def[%d]", index), "unsupported source layer key %q at position %d", sourceToken.value, index)
			continue
		}
		if !destination[index].escaped && destination[index].value == "_" {
			continue
		}
		switch destination[index].action.Kind {
		case NoopAction:
			noop[sourceKey] = true
			continue
		case SendKeysAction:
			sendKeys[sourceKey] = append([]uint16(nil), destination[index].action.Keys...)
			continue
		}
		destinationKey, destinationOK := virtualKey(destination[index].value)
		if !destinationOK {
			addDiagnostic(diagnostics, positions, fmt.Sprintf("%s.def[%d]", defaultLayer, index), "unsupported default layer key %q at position %d", destination[index].value, index)
			continue
		}
		remap[sourceKey] = destinationKey
	}
	for source, destination := range layerRemap {
		remap[source] = destination
		delete(noop, source)
		delete(sendKeys, source)
	}
	for source := range layerNoop {
		delete(remap, source)
		delete(sendKeys, source)
		noop[source] = true
	}
	for source, keys := range layerSendKeys {
		delete(remap, source)
		delete(noop, source)
		sendKeys[source] = keys
	}
	return true
}

func parseLayer(value any, layerName string, positions sourcePositions, diagnostics *Diagnostics) {
	table, ok := value.(map[string]any)
	if !ok {
		parseDefinition(value, "["+layerName+"]", nil, positions, diagnostics)
		return
	}
	actions := parseActions(table["actions"], layerName+".actions", positions, diagnostics)
	parseDefinition(value, "["+layerName+"]", actions, positions, diagnostics)
	parseLayerRemaps(table, layerName, actions, positions, diagnostics)
}

func parseLayerRemaps(table map[string]any, layerName string, actions map[string]ActionSpec, positions sourcePositions, diagnostics *Diagnostics) (map[uint16]uint16, map[uint16]bool, map[uint16][]uint16) {
	value, ok := table["remaps"]
	if !ok {
		return map[uint16]uint16{}, map[uint16]bool{}, map[uint16][]uint16{}
	}
	return parseRemaps(value, layerName+".remaps", actions, positions, diagnostics)
}

func parseDefinition(value any, path string, actions map[string]ActionSpec, positions sourcePositions, diagnostics *Diagnostics) ([]definitionToken, bool) {
	table, ok := value.(map[string]any)
	if !ok {
		addDiagnostic(diagnostics, positions, strings.Trim(path, "[]"), "%s must be a TOML table", path)
		return nil, false
	}
	for _, name := range sortedKeys(table) {
		if name != "def" && name != "actions" && name != "remaps" {
			addDiagnostic(diagnostics, positions, strings.Trim(path, "[]")+"."+name, "unsupported %s field %q", path, name)
		}
	}
	definition, ok := table["def"]
	if !ok {
		addDiagnostic(diagnostics, positions, strings.Trim(path, "[]")+".def", "%s.def is required", path)
		return nil, false
	}
	definitionPath := strings.Trim(path, "[]") + ".def"
	switch definition := definition.(type) {
	case string:
		if strings.TrimSpace(definition) == "#src" {
			return []definitionToken{{value: "#src"}}, true
		}
		tokens, err := splitDefinition(definition)
		if err != nil {
			addDiagnostic(diagnostics, positions, definitionPath, "%s.def: %v", path, err)
			return nil, false
		}
		return resolveDefinitionActions(tokens, actions, definitionPath, positions, diagnostics)
	case []any:
		keys := make([]definitionToken, len(definition))
		valid := true
		for index, value := range definition {
			key, ok := value.(string)
			if !ok {
				addDiagnostic(diagnostics, positions, fmt.Sprintf("%s[%d]", definitionPath, index), "%s.def entry at position %d must be a string", path, index)
				valid = false
				continue
			}
			unescaped, err := unescapeKeyName(key)
			if err != nil {
				addDiagnostic(diagnostics, positions, fmt.Sprintf("%s[%d]", definitionPath, index), "%s.def entry at position %d: %v", path, index, err)
				valid = false
				continue
			}
			keys[index] = definitionToken{value: unescaped, escaped: unescaped != key}
		}
		if !valid {
			return nil, false
		}
		return resolveDefinitionActions(keys, actions, definitionPath, positions, diagnostics)
	default:
		addDiagnostic(diagnostics, positions, definitionPath, "%s.def must be a string or string array", path)
		return nil, false
	}
}

func sortedKeys(table map[string]any) []string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysByPosition(table map[string]any, positions sourcePositions, path string) []string {
	keys := sortedKeys(table)
	sort.SliceStable(keys, func(left, right int) bool {
		leftPosition := positions[path+"."+keys[left]]
		rightPosition := positions[path+"."+keys[right]]
		if leftPosition.line != rightPosition.line {
			return leftPosition.line < rightPosition.line
		}
		if leftPosition.column != rightPosition.column {
			return leftPosition.column < rightPosition.column
		}
		return keys[left] < keys[right]
	})
	return keys
}

func splitDefinition(definition string) ([]definitionToken, error) {
	var lines []string
	for _, line := range strings.Split(definition, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	var tokens []definitionToken
	var token strings.Builder
	escaped := false
	for _, character := range strings.Join(lines, " ") {
		switch {
		case escaped:
			switch character {
			case '{':
				return nil, fmt.Errorf("nested { is not allowed")
			case '}':
				if token.Len() == 0 {
					return nil, fmt.Errorf("{} is not allowed")
				}
				tokens = append(tokens, definitionToken{value: token.String(), escaped: true})
				token.Reset()
				escaped = false
			default:
				token.WriteRune(character)
			}
		case character == '{':
			if token.Len() != 0 {
				return nil, fmt.Errorf("{ must start a key definition")
			}
			escaped = true
		case character == '}':
			return nil, fmt.Errorf("unexpected }")
		case unicode.IsSpace(character) || character == ',' || character == '|':
			if token.Len() != 0 {
				tokens = append(tokens, definitionToken{value: token.String()})
				token.Reset()
			}
		default:
			token.WriteRune(character)
		}
	}
	if escaped {
		return nil, fmt.Errorf("unclosed {")
	}
	if token.Len() != 0 {
		tokens = append(tokens, definitionToken{value: token.String()})
	}
	return tokens, nil
}

func unescapeKeyName(name string) (string, error) {
	if !strings.ContainsAny(name, "{}") {
		return name, nil
	}
	if len(name) < 3 || name[0] != '{' || name[len(name)-1] != '}' {
		return "", fmt.Errorf("%q must be enclosed by one pair of braces", name)
	}
	unescaped := name[1 : len(name)-1]
	if unescaped == "" || strings.ContainsAny(unescaped, "{}") {
		return "", fmt.Errorf("%q contains invalid braces", name)
	}
	return unescaped, nil
}
