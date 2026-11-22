package env

import (
	"strconv"
	"strings"

	"github.com/zakuro9715/DvorakX/internal/keyboard"
	"github.com/zakuro9715/DvorakX/internal/keys"
)

func parseEvents(t testingT, text string) []keyboard.Event {
	t.Helper()
	var events []keyboard.Event
	for _, token := range strings.Fields(stripComments(text)) {
		name, action, hasAction := strings.Cut(token, ":")
		if name == "" || (hasAction && action != "down" && action != "up") {
			t.Fatalf("invalid key event %q", token)
		}
		key, ok := parseKey(name)
		if !ok {
			t.Fatalf("unsupported key %q", name)
		}
		if !hasAction {
			events = append(events, keyboard.Event{VirtualKey: key}, keyboard.Event{VirtualKey: key, KeyUp: true})
			continue
		}
		events = append(events, keyboard.Event{VirtualKey: key, KeyUp: action == "up"})
	}
	return events
}

func stripComments(text string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		if comment := strings.IndexByte(line, '#'); comment >= 0 {
			lines[index] = line[:comment]
		}
	}
	return strings.Join(lines, "\n")
}

func parseKey(name string) (uint16, bool) {
	if strings.HasPrefix(strings.ToLower(name), "0x") {
		value, err := strconv.ParseUint(name[2:], 16, 16)
		return uint16(value), err == nil
	}
	if key, ok := keys.Lookup(name); ok {
		return key, true
	}
	return 0, false
}
