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
	pressed  map[uint16]uint16
}

// NewSimulatedKeyboard creates a simulated keyboard backed by mappings.
func NewSimulatedKeyboard(mappings *Mappings, output Output) *SimulatedKeyboard {
	return &SimulatedKeyboard{
		mappings: mappings,
		output:   output,
		pressed:  make(map[uint16]uint16),
	}
}

// Handle remaps event and sends it to the output.
//
// It returns false when the event is not mapped or the output rejects it, so
// callers can pass the original event through to the operating system.
func (k *SimulatedKeyboard) Handle(event Event) (bool, error) {
	destination, ok := k.destination(event)
	if !ok {
		return false, nil
	}

	if err := k.output.Send(Event{VirtualKey: destination, KeyUp: event.KeyUp}); err != nil {
		return false, err
	}
	return true, nil
}

func (k *SimulatedKeyboard) destination(event Event) (uint16, bool) {
	if event.KeyUp {
		destination, ok := k.pressed[event.VirtualKey]
		if !ok {
			return 0, false
		}
		delete(k.pressed, event.VirtualKey)
		return destination, true
	}

	if destination, ok := k.pressed[event.VirtualKey]; ok {
		return destination, true
	}

	destination, ok := k.mappings.Load()[event.VirtualKey]
	if !ok {
		return 0, false
	}
	k.pressed[event.VirtualKey] = destination
	return destination, true
}
