package keyboard

// Event is an OS-independent virtual key event.
type Event struct {
	VirtualKey uint16
	KeyUp      bool
}

// Output emits simulated key events.
type Output interface {
	Send(Event) error
}

// SimulatedKeyboard remaps key events and emits simulated replacements.
type SimulatedKeyboard struct {
	mappings *Mappings
	output   Output
	pressed  map[uint16]Mapping
}

// NewSimulatedKeyboard creates a simulated keyboard backed by mappings.
func NewSimulatedKeyboard(mappings *Mappings, output Output) *SimulatedKeyboard {
	return &SimulatedKeyboard{
		mappings: mappings,
		output:   output,
		pressed:  make(map[uint16]Mapping),
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
			delete(k.pressed, event.VirtualKey)
			return false, err
		}
		return true, nil
	}

	if err := k.output.Send(Event{VirtualKey: mapping.Destination, KeyUp: event.KeyUp}); err != nil {
		return false, err
	}
	return true, nil
}

func (k *SimulatedKeyboard) mapping(event Event) (Mapping, bool) {
	if event.KeyUp {
		mapping, ok := k.pressed[event.VirtualKey]
		if !ok {
			return Mapping{}, false
		}
		delete(k.pressed, event.VirtualKey)
		return mapping, true
	}

	if mapping, ok := k.pressed[event.VirtualKey]; ok {
		return mapping, true
	}

	mapping, ok := k.mappings.lookup(event.VirtualKey)
	if stages, stagesOK := k.mappings.lookupStages(); stagesOK {
		mapping, ok = resolveStages(event.VirtualKey, stages)
	}
	if !ok {
		return Mapping{}, false
	}
	k.pressed[event.VirtualKey] = mapping
	return mapping, true
}

func resolveStages(key uint16, stages []MappingStage) (Mapping, bool) {
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
			return Mapping{SendKeys: append([]uint16(nil), mapping.SendKeys...)}, true
		}
		current = mapping.Destination
	}
	if !matched {
		return Mapping{}, false
	}
	return Mapping{Destination: current}, true
}

func (k *SimulatedKeyboard) sendKeys(keys []uint16) error {
	held := make([]uint16, 0, len(keys))
	for _, key := range keys {
		if isModifier(key) {
			if err := k.output.Send(Event{VirtualKey: key}); err != nil {
				return err
			}
			held = append(held, key)
			continue
		}
		if err := k.output.Send(Event{VirtualKey: key}); err != nil {
			return err
		}
		if err := k.output.Send(Event{VirtualKey: key, KeyUp: true}); err != nil {
			return err
		}
	}
	for index := len(held) - 1; index >= 0; index-- {
		if err := k.output.Send(Event{VirtualKey: held[index], KeyUp: true}); err != nil {
			return err
		}
	}
	return nil
}

func isModifier(key uint16) bool {
	switch key {
	case 0x10, 0x11, 0x12, 0x5B, 0x5C, // Shift, Ctrl, Alt, Left/Right Win
		0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5: // Left/Right Shift/Ctrl/Alt
		return true
	default:
		return false
	}
}
