# Gospan

A native, local-first text expansion engine written in Go, inspired by Espanso.

Type a short trigger and it expands into longer text — Vietnamese shorthand,
code snippets, email templates, and reusable blocks.

```text
;dc    → được
;k     → không
;kt    → kiểm tra
;brse  → Bridge Software Engineer
;date  → 2026-10-18
```

## Status

Early development. The pure-Go matching engine, configuration system, and CLI
are functional. Windows keyboard capture and text insertion are implemented and
under active hardening.

See [`architect.md`](architect.md) for the full design, [`agents.md`](agents.md)
for roles, and [`skill.md`](skill.md) for conventions.

## Features

- Go-first, no CGO required for the core engine.
- YAML configuration with validated, atomic reload.
- Deterministic longest-match expansion with prefix-conflict handling.
- Word-boundary and case-insensitive matching.
- Escape character for literal triggers.
- Template variables: `{{date}}`, `{{time}}`, `{{datetime}}`, `{{uuid}}`.
- CLI for validation, diagnostics, and vocabulary management.
- Local-first storage; no account or network connection required.
- Persistent vocabulary in **JSONL** (default) or **SQLite** (pure-Go driver, no
  CGO).
- Opt-in vocabulary learning (disabled by default).

## Installation

Build from source (Go 1.26 or newer):

```powershell
go build -o dist/gospan.exe ./cmd/gospan
```

Or use the build script:

```powershell
./scripts/build.ps1 -Version 0.1.0
```

## Configuration

Gospan reads its configuration from, in order:

1. `--config <path>`
2. the `GOSPAN_CONFIG` environment variable
3. `%APPDATA%\gospan\config.yaml`

Copy the example to get started:

```powershell
New-Item -ItemType Directory -Force "$env:APPDATA\gospan" | Out-Null
Copy-Item configs/config.example.yaml "$env:APPDATA\gospan\config.yaml"
```

Or let Gospan create a starter for you:

```powershell
gospan config init
```

The GUI also creates a starter configuration automatically on first run if none
exists.

Then validate it:

```powershell
gospan config validate
```

A minimal configuration:

```yaml
engine:
  case_sensitive: false
  word_boundary: true

matches:
  - trigger: ";dc"
    replace: "được"
  - trigger: ";date"
    replace: "{{date:2006-01-02}}"
```

## CLI

```text
gospan start                     Run the engine with keyboard capture
gospan status                    Show configuration and engine status
gospan doctor                    Check the environment and configuration

gospan config validate           Validate the configuration
gospan config path               Print the resolved configuration path
gospan config init               Create a starter configuration

gospan match list                List configured triggers
gospan match test ";kt"          Test whether input ends with a trigger
gospan match add ";x" "example"  Add a rule to the YAML configuration
gospan match remove ";x"         Remove a rule from the YAML configuration

gospan vocab list                List vocabulary entries
gospan vocab add ";x" "example"  Add a vocabulary entry
gospan vocab export out.txt      Export vocabulary

gospan version
```

Exit codes: `0` success, `1` runtime error, `2` usage error, `3` configuration
validation error.

## GUI (optional)

A Fyne desktop front end acts as the on/off switch: expansion only runs while
the app is open and enabled.

```powershell
./scripts/build-gui.ps1
./dist/gospan-gui.exe
```

It opens **without a console window** (built with `-H=windowsgui`) and adds a
**system tray icon**: show the window, toggle expansion, or quit. Closing the
window hides it to the tray so capture keeps running in the background.

The GUI lives in its own Go module (`gui/fyne`) because Fyne requires CGO and a
C toolchain (`gcc`); the core engine stays pure-Go and builds with
`CGO_ENABLED=0`. See [`docs/gui.md`](docs/gui.md).

## Docker

A minimal container image for the core CLI:

```bash
docker build -t gospan:dev .
docker run --rm gospan:dev version
docker run --rm -v "$PWD/configs:/app/configs:ro" gospan:dev config validate --config /app/configs/config.example.yaml
```

Or with compose:

```bash
docker compose build
docker compose run --rm gospan version
```

The GUI is not containerized (it needs a display and CGO). See [`docs/gui.md`](docs/gui.md).

## Cross-platform

The core engine, config, CLI, and storage build on **Windows, macOS, and
Linux** with `CGO_ENABLED=0`. Keyboard capture and text insertion are currently
**Windows-only**; macOS/Linux ship best-effort clipboard and process helpers.
See [`docs/cross-platform.md`](docs/cross-platform.md).

## Development

```powershell
go test ./...
go vet ./...
go test -race ./...
go test -bench=. -benchmem ./internal/engine/...
```

`./scripts/test.ps1` runs all of the above.

## Project layout

```text
cmd/gospan/            CLI entrypoint
internal/config/       YAML schema, loader, validator, watcher
internal/engine/       platform-independent matching engine
internal/expansion/    template rendering and variables
internal/input/        bounded input buffer
internal/platform/     OS boundary (interface, Windows impl, host factory)
internal/vocabulary/   vocabulary service and suggestion collector
internal/storage/      persistence (JSONL)
internal/app/          application service and lifecycle
internal/cli/          command-line interface
configs/               example configuration and dictionaries
scripts/               build and test scripts
```

## Privacy

Gospan performs no telemetry and requires no network connection. Learning is
disabled by default and, when enabled, stores only phrase counts — never full
keystroke histories or complete sentences. Application exclusions are
best-effort, not a security boundary.

## License

To be determined.
