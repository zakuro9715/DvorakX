# Configuration Reference

> **Implementation status:** The initial Windows Go implementation currently
> supports only `[global].remap` direct mappings. The remaining configuration
> language described below is planned but is rejected until implemented.

## Format

DvorakX is configured by toml file.

## Summary

- One layer, One table.
- configure DvorakX on the `[meta]` table
- There are some special string.
- String literals with prefixes have special meanings.
    - `&layer-name` is reference to other layer.
    - `#keyword` is pre-defined [keyword](#Keyword).

## Example

```toml
# -- global config --
[meta]
# configure DvorakX itself

# table name of source definition. by default, `src`.
# To avoid key duplication, Don't write `src='src'`, just omit it.
# src='mysource'

# define default layer. it's startup layer.
# it can be referred as `#default`.
default="&qwerty"

# define layer order. This is used for `#next` and `#prev` keyword.
order=['qwerty', 'dvorak']

# Hotkeys. This is processed before remap.
hotkeys = [
    { 'lshift-rshift-esc' = '#exit' }
    { 'lshift-rshift-r' = 'rld' }
]

[global]
# define alias name.
# In this example, `&qw` is alias of `&qwerty`, and `&dv` is alias of `&dvorak`.
alias = { qw = '&qwerty', dv = '&dvorak', }

# Layer switch / toggle
# a|b is a or b.
# a-b is a and b.
# a+b is a and b.
layer-toggle = { 'ctrl|alt|shift|met' = '#default', }
# switch layer.
layer-switch = [
  { ctrl-alt-0='#default', ctrl-alt-1 = '&qw', ctrl-alt-2 = '&dv'},
  { ctrl-alt-right = '#next', ctrl-alt-left = '#prev'},
]

# direct remap.
# `caps` key will be remapped to `ctrl`
# This mapping is proceed after `def` remapping.
remap={ caps = 'ctrl' }

[src]
# define source layer. This layer is used as source of remapping.
def=[
    'grv', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '=', 'bspc',
    'tab', 'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '[', ']', '\\',
    'caps', 'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', ';', "{'}", 'ret',
    'lsft', 'z', 'x', 'c', 'v', 'b', 'n', 'm', ',', '.', '/', 'rsft',
    'lctl', 'lmet', 'lalt', 'spc', 'ralt', 'rmet', 'rctl',
]

[qwerty]
# use source layout.
def='#src'

[dvorak]
# define layer by string literal. separator is `|`, space or comma.
def='''
# separator is `|` or space.
_ 1 2 3 4 5 6 7 8 9 0 [ ] _
_|,|.|p|y|f|g|c|r|l|/|=|{|}|_
_|o|e|u|i|d|h|t|n|s|-|`|_
_|q|j|k|x|b|m|w|v|z|_
_, _, _, _, _, _, _,
'''
```

## Actions

|--|--|
|reload,rld|reload configuration|
|exit|exit DvorakX|
