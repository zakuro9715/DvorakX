//go:build !windows && !(linux && (amd64 || arm64)) && !darwin

package main

import (
	"errors"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

func run(_ *keyboard.Mappings, _ <-chan struct{}, _ bool) error {
	return errors.New("dvorakz is supported on Windows, Linux (amd64, arm64), and macOS (with cgo)")
}
