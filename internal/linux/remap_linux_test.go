//go:build amd64 || arm64

package linux

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

type keyEvent struct {
	code  uint16
	value int32
}

func TestHeldMappings(t *testing.T) {
	a, _ := keys.Lookup("a")
	b, _ := keys.Lookup("b")
	c, _ := keys.Lookup("c")
	mappings := keyboard.NewMappings(map[keys.Key]keys.Key{a: b})
	var events []keyEvent
	r := newRemapper(mappings, func(code uint16, value int32) error {
		events = append(events, keyEvent{code, value})
		return nil
	})
	// Hold two inputs that both emit B, then reload while they are held.
	require.NoError(t, r.handle(keys.A, 1, false))
	require.NoError(t, r.handle(keys.B, 1, false))
	mappings.Replace(map[keys.Key]keys.Key{a: c, b: c})
	require.NoError(t, r.handle(keys.A, 2, false))
	require.NoError(t, r.handle(keys.A, 0, false))
	require.NoError(t, r.handle(keys.B, 0, false))
	// The next press uses the newly loaded destination.
	require.NoError(t, r.handle(keys.A, 1, false))
	require.NoError(t, r.handle(keys.A, 0, false))
	require.Equal(t, []keyEvent{{keys.B, 1}, {keys.B, 2}, {keys.B, 0}, {keys.C, 1}, {keys.C, 0}}, events)
}

func TestRecoverAndRepeatActions(t *testing.T) {
	a, _ := keys.Lookup("a")
	b, _ := keys.Lookup("b")
	ctrl, _ := keys.Lookup("ctrl")
	c, _ := keys.Lookup("c")
	mappings := keyboard.NewMappingsWithActions(nil, map[keys.Key]bool{b: true}, map[keys.Key][]keys.Key{a: {ctrl, c}})
	var events []keyEvent
	r := newRemapper(mappings, func(code uint16, value int32) error {
		events = append(events, keyEvent{code, value})
		return nil
	})
	// Recover a missed action press while releasing an unknown passthrough key.
	const unknown = 120 // KEY_SCALE
	require.NoError(t, r.handle(unknown, 1, false))
	var state keyBits
	state[keys.A/8] = 1 << (keys.A % 8)
	require.NoError(t, r.sync(state))
	require.Equal(t, []keyEvent{{unknown, 1}, {unknown, 0}}, events)
	events = nil
	// A real repeat runs the action, while a noop press and release stay silent.
	require.NoError(t, r.handle(keys.A, 2, false))
	require.NoError(t, r.handle(keys.A, 0, false))
	require.NoError(t, r.handle(keys.B, 1, false))
	require.NoError(t, r.handle(keys.B, 0, false))
	require.Equal(t, []keyEvent{{keys.LeftCtrl, 1}, {keys.C, 1}, {keys.C, 0}, {keys.LeftCtrl, 0}}, events)
}

func TestSpecialCodesRoundTrip(t *testing.T) {
	for name, code := range specialCodes {
		key, ok := keys.Lookup(name)
		require.True(t, ok, name)
		require.Equal(t, code, keyToCode[key], name)
		require.Equal(t, key, codeToKey[code], name)
	}
}

func TestDisconnectAndReconnect(t *testing.T) {
	mappings := keyboard.NewMappings(nil)
	devices := make(map[string]*device)
	const path = "/dev/input/event0"
	for range 2 {
		// Reuse the same event path after the old device disconnects.
		var input [2]int
		var output [2]int
		require.NoError(t, unix.Pipe(input[:]))
		require.NoError(t, unix.Pipe(output[:]))
		r := newRemapper(mappings, func(code uint16, value int32) error {
			return writeKey(output[1], code, value)
		})
		devices[path] = &device{input: input[0], output: output[1], remap: r, grabbed: true, observer: keyboard.NewInputObserver(func() {})}
		// Unplug while a key is held and verify its release is emitted.
		require.NoError(t, r.handle(keys.A, 1, false))
		require.NoError(t, unix.Close(input[1]))
		require.NoError(t, pollDevices(devices, nil))
		require.Empty(t, devices)
		var events [4 * eventSize]byte
		n, err := unix.Read(output[0], events[:])
		require.NoError(t, err)
		require.Equal(t, len(events), n)
		require.Zero(t, events[2*eventSize+20])
		require.NoError(t, unix.Close(output[0]))
	}
}

func TestEmergencyEscape(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		esc, a, b := keys.Key{ScanCode: keys.Escape}, keys.Key{ScanCode: keys.A}, keys.Key{ScanCode: keys.B}
		mappings := keyboard.NewMappings(map[keys.Key]keys.Key{esc: a, b: esc})
		stops := 0
		var events []keyEvent
		newDevice := func() *device {
			return &device{
				observer: keyboard.NewInputObserver(func() { stops++ }),
				remap: newRemapper(mappings, func(code uint16, value int32) error {
					events = append(events, keyEvent{code, value})
					return nil
				}),
			}
		}
		first, second := newDevice(), newDevice()
		// A remapped Esc output does not arm the deadline.
		require.NoError(t, first.process(evKey, keys.B, 1))
		time.Sleep(3 * time.Second)
		synctest.Wait()
		require.Zero(t, stops)
		require.NoError(t, first.process(evKey, keys.B, 0))
		// Event loss cancels a physical Esc hold and discards subsequent events.
		require.NoError(t, second.process(evKey, keys.Escape, 1))
		require.NoError(t, second.process(evSyn, synDropped, 0))
		time.Sleep(3 * time.Second)
		synctest.Wait()
		require.Zero(t, stops)
		// The physical Esc still maps to A and arms the deadline on its own device.
		require.NoError(t, first.process(evKey, keys.Escape, 1))
		require.NoError(t, newDevice().process(evKey, keys.Escape, 0))
		time.Sleep(3 * time.Second)
		synctest.Wait()
		require.Equal(t, 1, stops)
		require.Equal(t, []keyEvent{{keys.Escape, 1}, {keys.Escape, 0}, {keys.A, 1}, {keys.A, 1}}, events)
		first.observer.Reset()
	})
}
