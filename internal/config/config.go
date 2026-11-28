package config

import "github.com/zakuro9715/DvorakZ/internal/keys"

// Config contains the resolved bindings for every configured layer.
type Config struct {
	DefaultLayer string
	Global       Bindings
	Layers       map[string]Layer
}

// Layer contains the definition and remap stages for a named layer.
type Layer struct {
	Definition Bindings
	Remaps     Bindings
}

// Bindings associates input keys with their transformations in one stage.
type Bindings map[keys.Key]Binding

// BindingKind identifies the payload of a binding.
type BindingKind uint8

const (
	_ BindingKind = iota
	RemapKey
	NoopAction
	SendKeysAction
)

// Binding describes a key transformation. Kind selects the applicable payload.
type Binding struct {
	Kind        BindingKind
	Destination keys.Key
	Keys        []keys.Key
}
