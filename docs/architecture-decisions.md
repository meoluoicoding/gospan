# Architecture Decision Records

Short, dated records of decisions that shape Gospan. The full rationale lives in
[`../architect.md`](../architect.md) §19; this file tracks the decisions actually
implemented so far.

## ADR-001 — Go as the primary language
**Status:** Accepted (implemented).
Core logic, configuration, matching, storage, CLI, and future GUI are written in
Go. See `go.mod`.

## ADR-002 — YAML for user-defined matches
**Status:** Accepted (implemented).
`internal/config` parses YAML into Go structs and validates them before
activation. Third-party dependency: `gopkg.in/yaml.v3`.

## ADR-003 — Modular monolith
**Status:** Accepted (implemented).
One module (`github.com/snzpet/gospan`) with all packages under `internal/`
except `cmd/gospan`. No IPC, no services.

## ADR-004 — Windows API integration through Go
**Status:** Accepted (partially implemented).
`internal/platform/windows` calls Win32 through `syscall` lazy DLL bindings
with no CGO. Keyboard capture uses `SetWindowsHookExW`; insertion uses
`SendInput`.

## ADR-005 — GUI after engine stability
**Status:** Accepted.
No GUI package exists yet. CLI and engine come first.

## ADR-006 — Privacy-first vocabulary learning
**Status:** Accepted (implemented).
`internal/vocabulary` disables learning by default; candidates require explicit
acceptance and are never activated automatically.

## ADR-007 — Defer expansion for prefix-conflicting triggers
**Status:** Accepted (implemented).
When one trigger is a proper prefix of another (for example `;k` and `;kt`), the
shorter trigger is not expanded immediately. The engine waits for the next
keystroke: a longer match wins, otherwise the deferred trigger is flushed along
with any trailing runes. Implemented in `internal/engine`.

## ADR-008 — Fyne GUI in a separate module
**Status:** Accepted (implemented).
The optional desktop front end is built with Fyne and lives in its own Go module
(`gui/fyne`, module `github.com/snzpet/gospan/gui`). Fyne requires CGO and a C
toolchain, which architect.md §2.2 requires us to document and justify: the GUI
is explicitly optional, so isolating it keeps the core engine pure-Go and
buildable with `CGO_ENABLED=0`. The core module never imports `gui/`; the GUI
consumes `internal/app` and adds no matching logic (architect.md §11.2).

## ADR-009 — The window is the on/off switch
**Status:** Accepted (implemented).
Keyboard capture is not a background service. It is installed only when the user
enables it in the GUI (`Service.Start`) and removed on disable or window close
(`Service.Stop`). Nothing listens to the keyboard while the app is off.

## ADR-010 — SQLite backend via a pure-Go driver
**Status:** Accepted (implemented).
Vocabulary can be stored as JSONL (default) or SQLite. The SQLite backend uses
`modernc.org/sqlite`, a pure-Go driver, so the core keeps building with
`CGO_ENABLED=0` and no C toolchain (architect.md §2.2, §13.2). A single database
connection is used to avoid `SQLITE_BUSY`; the vocabulary workload is low volume.
All callers open storage through `storage.Open(backend, path)` so the configured
backend is actually honoured — previously `backend: sqlite` was accepted by
validation but silently fell back to JSONL.

## ADR-011 — Snippet markers over magic strings
**Status:** Planned.
Cursor position and tab stops are expressed with explicit markers (`$|$`,
`$0`/`$1`) rendered into a structured result (text + cursor offset + tab
stops), not embedded as invisible characters. See architect.md ADR-007.

## ADR-012 — Regex triggers are opt-in and bounded
**Status:** Planned.
Regex triggers require `trigger_type: regex` and are executed with a timeout and
complexity limits. See architect.md ADR-008.

## ADR-013 — App filtering by process name only
**Status:** Planned.
App-specific matching keys off the lower-cased foreground process base name.
See architect.md ADR-009.

## ADR-014 — Clipboard insertion is a strategy, not a default
**Status:** Planned.
Short replacements use backspace + typing; clipboard insertion is used only
above a configurable threshold or for multiline blocks, with safe save/restore.
See architect.md ADR-010.

## ADR-015 — Search bar is a frontend concern
**Status:** Planned.
The search bar is implemented in the GUI layer and calls application services.
See architect.md ADR-011.

## ADR-016 — Cross-platform via the existing interface
**Status:** Planned.
macOS and Linux implement `platform.Platform`; no changes to the core engine.
See architect.md ADR-012.


