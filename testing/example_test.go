package testing

import (
	"github.com/zakuro9715/DvorakX/testing/env"
)

const config = `
# config file content
[global]
remaps = { a = 'b' }
    `

func Example_simulateExpect() {
	env.Config(t, config).
		Simulate("a a:down a:up").
		Expect("b b:down b:up")
}

func Example_test() {
	env.Config(t, config).
		Test("a :down a:up", "b b:down b:up")
}

func Example_loadConfig() {
	env.LoadConfig(t, "example.toml").
		Test("a b c", "a b c")
}
