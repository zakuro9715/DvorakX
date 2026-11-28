//go:build !windows && !(linux && (amd64 || arm64))

package main

import (
	"errors"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

func run(_ *keyboard.Mappings, _ <-chan struct{}, _ bool) error {
	return errors.New("dvorakz is supported on Windows and Linux (amd64, arm64)")
}
