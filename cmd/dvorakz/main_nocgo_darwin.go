//go:build darwin && !cgo

package main

import (
	"errors"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

func run(_ *keyboard.Mappings, _ <-chan struct{}, _ bool) error {
	return errors.New("macOS support requires a build with CGO_ENABLED=1 and the macOS SDK")
}
