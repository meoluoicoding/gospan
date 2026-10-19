# Cross-platform Support

Gospan targets **Windows 10/11 x64 first**. macOS and Linux are supported at the
core level (engine, config, CLI, storage) and ship **best-effort** platform
helpers, but keyboard capture and text insertion are **not yet implemented**
there.

## What works everywhere

- Core matching engine (pure Go, no CGO).
- YAML configuration parsing and validation.
- CLI: `config`, `match`, `vocab`, `doctor`, `version`.
- Storage: JSONL and SQLite (pure-Go driver).
- Vocabulary management.

## What is Windows-only

- Low-level keyboard capture (`SetWindowsHookExW`).
- Text insertion (`SendInput`).
- Clipboard helpers (Windows API).
- Foreground-process detection.
- System tray and Fyne GUI (Fyne needs CGO; the GUI module builds on Windows).

## macOS

- Clipboard: via `pbpaste`/`pbcopy` (best effort).
- Foreground process: via `osascript`.
- Keyboard capture and insertion: **not implemented** (require CGEventTap, CGO).

## Linux

- Clipboard: via `xclip` or `xsel`.
- Foreground process: via `xdotool` + `/proc`.
- Keyboard capture and insertion: **not implemented** (require X11/Wayland).

## Building

```bash
# Cross-compile the core (no CGO required)
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -o dist/gospan-linux  ./cmd/gospan
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/gospan-macos  ./cmd/gospan
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/gospan.exe    ./cmd/gospan
```

The GUI (`gui/fyne`) builds only on Windows with CGO enabled. See
[`gui.md`](gui.md).

## Roadmap

Full macOS/Linux keyboard capture and insertion are tracked as Phase 13 in
[`../plan.md`](../plan.md). Contributions welcome.
