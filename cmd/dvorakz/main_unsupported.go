//go:build !windows

package main

import (
	"errors"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

func run(_ *keyboard.Mappings, _ <-chan struct{}, _ bool) error {
	return errors.New("dvorakz is supported only on Windows")
}
