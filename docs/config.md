# Configuration Reference

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
#order=['qwerty', 'dvorak']

[global]
remaps=[{ caps = 'ctrl', "0" = "@dv" }]

[global.actions]
dv= { layer-switch = ['dvorak'] }

[src]
# define source layer. This layer is used as source of remapping.
def=[
    'grv', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '=', 'bspc',
    'tab', 'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '[', ']', '\',
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
_|'|,|.|p|y|f|g|c|r|l|/|=|\
_|a|o|e|u|i|d|h|t|n|s|-|`|_
_|;|q|j|k|x|b|m|w|v|z|_
_, _, _, _, _, _, _,
'''
```

## Actions

### Basics

Actions are defined in `[layer.actions]`. Defined action can be use by `@action-name` in remaps and def.

```
[src]
def = '1 2 3'
[layer]
def = '@abc 2 3'
remaps = { x = '@abc' }
[layer.actions]
# name = { action = [arg1, arg2, ...] }
abc = { send-keys = ['abc'] }
# single argument shorthand
ab = { send-keys = 'ab' }
# no-argument shorthand
disable-key = 'nope'
# multi
abc-efg = [ { send-keys = ['abc'] }, { send-keys = ['efg'] } ]
# delay
abc-100ms-efg = [ { send-keys = ['abc'] }, { delay = [100] }, { send-keys = ['efg'] } ]
# delay shorthand
abc-10ms-efg = [ { send-keys = ['abc'] }, 10, { send-keys = ['efg'] } ]
```

### Featuers

#### Multi

Call multiple actions in one action with array.

```
abcefg = [ send-keys = ['abc'], send-keys = ['efg'] ]
```

#### Interval

Insert interval between actions with `delay` action.

```
abc-100ms-efg = [ send-keys = ['abc'], delay = [100], send-keys = ['efg'] ]
```

### Pre-defined Actions

### noop

It does nothing. Useful for disabling key

```
# disable `a` key.
remaps = { a = '@disable-key' }
actions { disable-key = { noop = [] } }
```

#### send-keys

`send-keys` action send key sequence.

```
remaps = { a = '@send-abc' }
actions = { send-abc = { send-keys = ['abc'] } }
```

#### passthrough

`passthrough` action send key sequence as is. It is useful for define side effect without blocking.

```
remaps = { x = '@toggle' }
actions { toggle = [ { toggle-layer = ['layer2'] }, 'paththrough' ] }
```

You can use `_` alias for `passthrough`.

```
actions { toggle = [ { toggle-layer = ['layer2'] }, '_' ] }
```

#### layer-switch

switch layer.

```
switch-to-layer1 = { layer-switch = ['@layer1'] }
switch-to-layer2 = { layer-switch = ['layer2'] } # @ is optional
```

#### toggle-layer

toggle layer while pressing key. It is useful with `passthrough` action.

```
toggle-layer2 = [ { toggle-layer = ['layer2'] }, '_' ]
```

```
[src]
def = '1 2 3'
[global]
actions = { toggle-layer2 = [ { toggle-layer = ['layer2'] }, '_' ] }
remaps = { ctrl = '@toggle-layer2' }
[layer1]
def = '@src'
[layer2]
def = 'a b c'
```

In this example, `ctrl+1` on layer1 will be `ctrl+a`
