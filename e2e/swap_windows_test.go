//go:build windows && (amd64 || arm64)

package e2e_test

import (
	"testing"

	"github.com/zakuro9715/DvorakX/e2e/testenv"
)

func TestSwapAAndB(t *testing.T) {
	environment := testenv.Start(t, "[global]\nremap = { a = 'b', b = 'a' }\n")

	environment.AssertKey("a", "b")
	environment.AssertKey("b", "a")
}

func TestSwapOneAndTwoWithShift(t *testing.T) {
	environment := testenv.Start(t, "[global]\nremap = { '1' = '2', '2' = '1' }\n")

	environment.AssertShiftedKey("1", "@")
	environment.AssertShiftedKey("2", "!")
}
