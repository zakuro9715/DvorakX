//go:build !windows

package main

import (
	"errors"

	"github.com/zakuro9715/DvorakX/internal/keyboard"
)

func run(_ *keyboard.Mappings, _ <-chan struct{}, _ bool) error {
	return errors.New("dvorakx is supported only on Windows")
}
