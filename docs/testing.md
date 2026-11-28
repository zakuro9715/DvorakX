# Testing

## Simulate

### Example

[testing/example_test.go](../testing/example_test.go)

### API

```
go doc -all testing/env
```

#### DSL

Shared DSL to describe key event for Methods.

```
# comment
a:down # down key
a:up   # up key
a      # same as `key:down key:up`
0x30   # scan code in hex

a b c  # events are separated by space
```

## Config Error Tests

Add a `.toml` file to [`internal/config/tests`](../internal/config/tests).
Place the exact CLI output after `# Output:`, with each line
prefixed by `# `. Preserve the output's indentation.

```toml
[global]
remaps = { "1" = 2 }

# Output:
# [sample.toml:2, 18]: invalid remap destination: expected string, got number
#    2 | remaps = { "1" = 2 }
#      |                  ^
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

### macOS

Run in an isolated graphical login session on a Mac or macOS VM with the Xcode
Command Line Tools. Grant Input Monitoring and Accessibility under System
Settings → Privacy & Security to the terminal or binaries identified by macOS.
Restart the affected processes after granting access. The test sends keys into
the login session, so keep other keyboard activity out of that session.

```sh
CGO_ENABLED=1 go test -tags=e2e ./e2e/macos/...
```
