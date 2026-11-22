# DvorakX

A keyboard remaper, not only dvorak.

Spiritual Successor of [DvorakJ](https://web.archive.org/web/20230609021050/https://blechmusik.xii.jp/dvorakj/)

Supports windows only for now. Linux and MacOS support is planned.

## Usage

```powershell
go build -o dvorakx.exe ./cmd/dvorakx
```

By default, it loads `config.toml`. Pass a different file with `--config`:

```powershell
.\dvorakv.exe --config C:\path\to\config.toml
```

Pass `--debug` to output configuration, hook lifecycle, and remapped virtual-key
events:

```powershell
.\dvorakv.exe --debug
```

## Development

- go tool lefthook install
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/local/)

### Windows E2E test

The keyboard-hook E2E test builds the executable, temporarily installs its
global hook, and verifies that `a` and `b` are swapped in a test window. Run it
only from an interactive Windows desktop:

```powershell
$env:DVORAKX_E2E = "1"
go test ./e2e
```

Additional cases can reuse `e2e/testenv`: call `testenv.Start(t, config)` and
then `environment.AssertKey(source, destination)` or
`environment.AssertShiftedKey(source, destination)`. The shifted number-key
tests expect the US keyboard layout.

## Features

### Configuration

- Easy config file.
- multiple layers supported. Toggle or hold to switch.
- Global configuration. Affects to all layers.
- Source layout as a layer.

## About

### Why DvorakX?

I'm using DvorakJ for a long time. I like it.

But DvorakJ announced "no longer available" and closed download link.

I' was trying kmonad and kanata. But they are too complex for me.
And There were some problems on my environment. So I decided to create a new one.

### What is DvorakJ?

DvorakJ is a keyboard remapper for Windows, famous in japan.

| DvorakJ is a portable freeware that remaps your keyboard arrangement in Windows Vista, Windows 7, Windows 8 and Windows 10 With DvorakJ, you can use various keyboard arrangements, such as QWERTY, Dvorak and Colemak layouts. Also, you can make use of keyboard layouts of your own creation.

Now, DvorakJ is no longer available. Website was closed.

- [Official Website(Weyback Machine)](https://web.archive.org/web/20230609021050/https://blechmusik.xii.jp/dvorakj/)
- [Source Code on GitHub (Unofficial)](https://github.com/k-ayaki/dvorakj_2023)
- [Download Page (Unofficial)](https://freesoft-100.com/review/dvorakj.html

## Similar Projects

- [Kanata](https://github.com/jtroo/kanata)
- [Kmonad](https://github.com/kmonad/kmonad)
- and others. See [kanata's README](https://github.com/jtroo/kanata#similar-projects)

### Pros/Cons

- DvorakX is easy to use.
    - Simple configuration file.
    - GUI
- DvorakX is not complex
    - No support for complex key combinations.
- DvorakX is a new project.
    - Not stable yet.
- DvorakX is not cross-platform for now.
    - Only Windows is supported.
    - I'm planning to support Linux and MacOS in the future.

## Acknowledgements

- [DvorakJ](https://web.archive.org/web/20230609021050/https://blechmusik.xii.jp/dvorakj/)
