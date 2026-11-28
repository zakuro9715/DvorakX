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
0x30   # scan code in hex

a b c  # events are separated by space
```

## HTML Coverage

```
make test.cov
make open-cov
```

## E2E Test

```sh
go test -tags=e2e ./e2e/...
```

### Linux

Run in an isolated Linux VM (amd64 or arm64) with udev running and uinput
available. Load the module with `sudo modprobe uinput` if needed. The test user
needs write access to `/dev/uinput` (or `/dev/input/uinput`) and read access to
the evdev nodes for `DvorakZ E2E source` and `DvorakZ virtual keyboard`.
DvorakZ can grab other keyboards it has permission to read during the test.
