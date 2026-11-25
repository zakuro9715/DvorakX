package keyboard

import (
	"slices"

	"github.com/zakuro9715/DvorakZ/internal/keys"
)

// Event is an OS-independent scan code event.
type Event struct {
	Key   keys.Key
	KeyUp bool
}

// Output emits simulated key events.
type Output interface {
	Send(Event) error
}

// SimulatedKeyboard remaps key events and emits simulated replacements.
type SimulatedKeyboard struct {
	mappings *Mappings
	output   Output
	pressed  map[keys.Key]Mapping
}

// NewSimulatedKeyboard creates a simulated keyboard backed by mappings.
func NewSimulatedKeyboard(mappings *Mappings, output Output) *SimulatedKeyboard {
	return &SimulatedKeyboard{
		mappings: mappings,
		output:   output,
		pressed:  make(map[keys.Key]Mapping),
	}
}

// Handle remaps event and sends it to the output.
//
// It returns false when the event is not mapped or the output rejects it, so
// callers can pass the original event through to the operating system.
func (k *SimulatedKeyboard) Handle(event Event) (bool, error) {
	mapping, ok := k.mapping(event)
	if !ok {
		return false, nil
	}
	if mapping.Noop {
		return true, nil
	}

	if len(mapping.SendKeys) != 0 {
		if event.KeyUp {
			return true, nil
		}
		if err := k.sendKeys(mapping.SendKeys); err != nil {
			delete(k.pressed, event.Key)
			return false, err
		}
		return true, nil
	}

	if err := k.output.Send(Event{Key: mapping.Destination, KeyUp: event.KeyUp}); err != nil {
		return false, err
	}
	return true, nil
}

func (k *SimulatedKeyboard) mapping(event Event) (Mapping, bool) {
	if event.KeyUp {
		mapping, ok := k.pressed[event.Key]
		if !ok {
			return Mapping{}, false
		}
		delete(k.pressed, event.Key)
		return mapping, true
	}

	if mapping, ok := k.pressed[event.Key]; ok {
		return mapping, true
	}

	mapping, ok := k.mappings.lookup(event.Key)
	if stages, stagesOK := k.mappings.lookupStages(); stagesOK {
		mapping, ok = resolveStages(event.Key, stages)
	}
	if !ok {
		return Mapping{}, false
	}
	k.pressed[event.Key] = mapping
	return mapping, true
}

func resolveStages(key keys.Key, stages []MappingStage) (Mapping, bool) {
	current := key
	matched := false
	for _, stage := range stages {
		mapping, ok := stage.Mappings[current]
		if !ok || mapping.Passthrough {
			continue
		}
		matched = true
		if mapping.Noop {
			return Mapping{Noop: true}, true
		}
		if len(mapping.SendKeys) != 0 {
			return Mapping{SendKeys: slices.Clone(mapping.SendKeys)}, true
		}
		current = mapping.Destination
	}
	if !matched {
		return Mapping{}, false
	}
	return Mapping{Destination: current}, true
}

func (k *SimulatedKeyboard) sendKeys(sendKeys []keys.Key) error {
	held := make([]keys.Key, 0, len(sendKeys))
	for _, key := range sendKeys {
		if isModifier(key) {
			if err := k.output.Send(Event{Key: key}); err != nil {
				return err
			}
			held = append(held, key)
			continue
		}
		if err := k.output.Send(Event{Key: key}); err != nil {
			return err
		}
		if err := k.output.Send(Event{Key: key, KeyUp: true}); err != nil {
			return err
		}
	}
	for _, key := range slices.Backward(held) {
		if err := k.output.Send(Event{Key: key, KeyUp: true}); err != nil {
			return err
		}
	}
	return nil
}

func isModifier(key keys.Key) bool {
	switch key.ScanCode {
	case 0x1D, 0x2A, 0x36, 0x38, 0x5B, 0x5C: // Ctrl, Shift, Alt, Left/Right Win
		return true
	default:
		return false
	}
}
