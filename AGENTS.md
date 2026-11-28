# AGENTS.md

DvorakZ is keyboard remaper.

## Rules

- Don't edit README.md without instruction.
- Write doc if necessary.
- Do not prioritize compatibility with existing code; breaking changes are acceptable for any change when appropriate.

## Core Concept

- Keep Simple.

## Examples

When implement feature, write a all available configuration example in `examples/all.toml`

## files

- `docs/config.md` for config reference and plan.

## Code style

### Linting

run `golintci-lint run` to check code style.

### Testing

- When testing keyboard behavior, write comments that explain what the test is doing.

#### Coverage

- Aim for at least 80% overall Go statement coverage, measured on Linux with `make test.cov`.
- Aim for at least 90% Go statement coverage on added or modified code in each pull requesat.

### Comments

- Do not write unnecessary comments.
- Add brief comments that explain intent in complex code.
- Add documentation comments to exported APIs.
- Write the key name next to hard-coded key codes.

## Check list

- Run `make lint`
- Run `make test.cov`
- RUN `make test.win.cov` when change windows code.
