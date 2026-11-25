package env

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestEnvironmentSimulatesRemapsAndDSL(t *testing.T) {
	New(t).
		Config(`
[global]
remaps = { a = "b", "0" = "c" }
`).
		Simulate(`
# ordinary keys expand to down and up
a 0x0B:down 0x0B:up
`).
		Expect("b c:down c:up")
}

func TestEnvironmentSimulatesActions(t *testing.T) {
	New(t).
		Config(`
[global]
remaps = { a = "@copy", b = "@disabled" }

[global.actions]
copy = { send-keys = ["ctrl", "c"] }
disabled = { noop = [] }
`).
		Simulate("a b").
		Expect("ctrl:down c ctrl:up")
}

func TestEnvironmentLoadsConfigRelativeToTestSource(t *testing.T) {
	New(t).LoadConfig("testdata/config.toml").Simulate("a").Expect("b")
}

func TestEnvironmentTestCombinesSimulateAndExpect(t *testing.T) {
	New(t).
		Config(`[global]
remaps = { a = "b" }
`).
		Test("a", "b")
}

func TestEnvironmentAppliesDefinitionsInOrder(t *testing.T) {
	New(t).
		Config(`
[global]
remaps = { a = "b" }

[meta]
default = "&layer"

[src]
def = ["a", "b", "c"]

[layer]
def = ["e", "f", "g"]
remaps = { f = "x" }
`).
		Test("a b", "x x")
}

func TestEnvironmentTestReportsMismatch(t *testing.T) {
	tester := new(mockTestingT)
	tester.On("Helper").Maybe()
	tester.On("Fatalf", mock.Anything, mock.Anything).Maybe()

	var got string
	tester.On("Errorf", mock.Anything, mock.Anything).Once().
		Run(func(call mock.Arguments) {
			got = fmt.Sprintf(call.String(0), call.Get(1).([]any)...)
		})

	New(tester).
		DisableColor().
		Config(`[global]
remaps = { a = "b" }
`).
		Test("a", "c")

	want := "output events mismatch:\n- c\n+ b"
	if got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	tester.AssertExpectations(t)
}

func TestEnvironmentMismatchPreservesSimulateTokens(t *testing.T) {
	tester := new(mockTestingT)
	tester.On("Helper").Maybe()
	tester.On("Errorf", mock.Anything, mock.Anything).Once().
		Run(func(call mock.Arguments) {
			got := fmt.Sprintf(call.String(0), call.Get(1).([]any)...)
			if got != "output events mismatch:\n- 0x42\n+ 0x43" {
				t.Errorf("error = %q", got)
			}
		})

	New(tester).
		DisableColor().
		Simulate("0x43").
		Expect("0x42")
	tester.AssertExpectations(t)
}

type mockTestingT struct {
	mock.Mock
}

func (tester *mockTestingT) Helper() {
	tester.Called()
}

func (tester *mockTestingT) Errorf(format string, args ...any) {
	tester.Called(format, args)
}

func (tester *mockTestingT) Fatalf(format string, args ...any) {
	tester.Called(format, args)
}
