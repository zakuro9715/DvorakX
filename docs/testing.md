# Testing

## Example

[testing/example_test.go](../testing/example_test.go)

## API

```
go doc -all testing/env
```

### DSL

Shared DSL to describe key event for Methods.

```
# comment
a:down # down key
a:up   # up key
a      # same as `key:down key:up`
0x30   # keycode in hex

a b c  # events are separated by space
```
