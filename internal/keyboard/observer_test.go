package keyboard

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func TestEscapeHold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stops := 0
		esc := keys.Key{ScanCode: keys.Escape}
		hold := NewInputObserver(func() { stops++ })
		// Other keys and a repeat without an initial Esc press cannot arm the timer.
		hold.Observe(Event{Key: keys.Key{ScanCode: keys.A}}, false)
		hold.Observe(Event{Key: esc}, true)
		time.Sleep(3 * time.Second)
		synctest.Wait()
		if stops != 0 {
			t.Fatal("stopped without an initial Esc press")
		}
		// Reset cancels an active hold before the deadline.
		hold.Observe(Event{Key: esc}, false)
		hold.Reset()
		time.Sleep(3 * time.Second)
		synctest.Wait()
		if stops != 0 {
			t.Fatal("stopped after resetting the input source")
		}
		// Release a short press, then start a new hold before the old deadline.
		hold.Observe(Event{Key: esc}, false)
		time.Sleep(time.Second)
		hold.Observe(Event{Key: esc, KeyUp: true}, false)
		hold.Observe(Event{Key: esc}, false)
		time.Sleep(3*time.Second - time.Millisecond)
		synctest.Wait()
		if stops != 0 {
			t.Fatal("stopped before the second press reached three seconds")
		}
		// A repeat just before the deadline must not extend the hold.
		hold.Observe(Event{Key: esc}, true)
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if stops != 1 {
			t.Fatalf("got %d stops after three seconds, want 1", stops)
		}
		hold.Reset()
	})
}
