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
	if !ok {
		return Mapping{}, false
	}
	k.pressed[event.VirtualKey] = mapping
	return mapping, true
}
