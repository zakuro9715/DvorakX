package keyboard

import (
	"sync"
	"time"

	"github.com/zakuro9715/DvorakZ/internal/keys"
)

// InputObserver monitors pre-remap events from one input source.
type InputObserver struct {
	mu    sync.Mutex
	timer *time.Timer
	stop  func()
}

// NewInputObserver calls stop when Escape has been held for three seconds.
func NewInputObserver(stop func()) *InputObserver {
	return &InputObserver{stop: stop}
}

// Observe checks an input event before it is remapped.
func (h *InputObserver) Observe(event Event, repeat bool) {
	if event.Key != (keys.Key{ScanCode: keys.Escape}) {
		return
	}
	if event.KeyUp {
		h.Reset()
	} else if !repeat {
		h.press()
	}
}

func (h *InputObserver) press() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.timer != nil {
		return
	}
	var timer *time.Timer
	timer = time.AfterFunc(3*time.Second, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.timer == timer {
			h.stop()
		}
	})
	h.timer = timer
}

// Reset cancels the current deadline.
func (h *InputObserver) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
}
