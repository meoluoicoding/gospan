# Skills — Gospan

**Project:** Gospan (Go Text Expansion Engine)
**Companion documents:** `architect.md` (source of truth), `agents.md` (roles)
**Status:** Living document — update when new tools or workflows are validated.

This document lists the concrete skills, conventions, and workflows required to
build, test, debug, and ship Gospan. An agent should read this together with
`architect.md` before making changes.

---

## 1. Repository Workflow

```text
architect.md   → architecture source of truth (do not contradict)
agents.md      → who does what
skill.md       → how to do it (this file)
configs/       → user-facing YAML (examples + match dictionaries)
internal/      → all application packages (unexported by policy)
cmd/gospan/    → CLI entrypoint
```

Rules:

- Keep all application packages under `internal/` until a public library API is
  genuinely required.
- Never import `gui/` from the core engine.
- Never import Windows-specific packages from `internal/engine/`.
- One package = one clear responsibility. If a file grows beyond ~400 lines,
  split it along behavior, not by accident.

---

## 2. Go Skills

| Skill | Practice |
|---|---|
| Module setup | `go mod init github.com/snzpet/gospan`, pin Go version in `go.mod` |
| Formatting | `gofmt -s` / `go fmt ./...` — always before commit |
| Static checks | `go vet ./...` must be clean |
| Tests | `go test ./...`, table-driven where practical |
| Race safety | `go test -race ./...` for any concurrent code |
| Benchmarks | `go test -bench=. -benchmem ./internal/engine/...` |
| Build | `go build ./cmd/gospan` |
| Errors | Wrap with `fmt.Errorf("...: %w", err)`, never log-and-return |
| Logging | `log/slog`, structured key/value, no typed content by default |
| Concurrency | Bounded channels, explicit cancellation via `context.Context` |
| Immutability | Publish immutable snapshots; never mutate a live index |

### 2.1 Go Style Conventions

- Exported identifiers need doc comments beginning with the identifier name.
- Prefer small interfaces defined at the consumer, not the producer.
- Return concrete types from constructors (`*Engine`, not `Matcher`).
- Avoid package-level mutable state.
- No `panic` in libraries; return errors. `main` may exit with a code.
- Use `errors.Is` / `errors.As` for error classification.

---

## 3. Configuration Skill (YAML)

- Schema lives in `internal/config/schema.go` as Go structs with `yaml:"..."` tags.
- Loader in `internal/config/loader.go` reads bytes → structs.
- Validation in `internal/config/validate.go` enforces semantic rules.
- Watcher in `internal/config/watcher.go` debounces filesystem events.
- Reload is atomic: parse → validate → build index → publish. Never publish a
  partially constructed dictionary.
- On failure, keep the last valid configuration and report an actionable error
  with file, line, and field where possible.
- Reject: empty triggers, duplicate triggers (without explicit override),
  triggers/replacements exceeding configured length limits.

Validation error message shape:

```text
configs/matches/vietnamese.yaml:12: duplicate trigger ";kt"
```

---

## 4. Matching Engine Skill

- Trigger index is a `map[string]Match` for direct suffix lookup.
- Input buffer is a bounded ring/suffix buffer of runes (not bytes) to keep
  Unicode and Vietnamese diacritics correct.
- On each rune, evaluate only plausible suffix candidates — never scan the whole
  dictionary per keystroke.
- Deterministic precedence: longest matching trigger wins; ties are resolved by
  document order, then alphabetically.
- Boundary rules are configurable (`word_boundary`).
- Case sensitivity is configurable (`case_sensitive`).
- Escaping: a configured escape character emits the literal trigger text.
- Reentrancy guard: injected replacement events must never re-trigger expansion.
- The engine must be fully testable with synthetic event sequences — no OS
  dependency.

Benchmark discipline:

- Distinguish matcher latency from end-to-end capture/insertion latency.
- Use realistic Unicode/Vietnamese inputs, not ASCII-only synthetic data.
- Do not add a trie / Aho–Corasick unless benchmarks justify it.

---

## 5. Platform Integration Skill (Windows)

- Platform code is isolated behind `platform.Platform` and guarded with
  `//go:build windows`.
- Keyboard capture uses a low-level keyboard hook; the callback must be short:
  push the event to a bounded channel; never parse YAML or touch storage inside
  the callback.
- Text insertion uses `SendInput`; support both strategies:
  1. backspace over the matched trigger then type the replacement;
  2. clipboard-assisted insertion for large/multiline replacements (save and
     restore the clipboard safely).
- Filter injected events (`LLKHF_INJECTED`) to prevent loops.
- Handle: layout changes, IME composition, surrogate pairs, elevated apps,
  RDP sessions, secure desktop, hook lifecycle/thread affinity.
- Never attempt to bypass OS security boundaries. Document limitations.

---

## 6. CLI Skill

- CLI is the primary management and debugging surface, built on the standard
  library (Cobra optional).
- Stable exit codes; human-readable by default; `--json` for scripts.
- Flags: `--config`, `--verbose`, `--quiet`.
- No interactive prompts in automation mode.
- Paths with spaces and Unicode must work.
- `start`/`stop` use an explicit lifecycle mechanism (named pipe / lock file /
  signal) — never fragile PID-only control.

Commands (provisional):

```text
gospan start | stop | status
gospan config validate | path | reload
gospan match list | test <trigger> | add <trigger> <replacement> | remove <trigger>
gospan vocab list | add <trigger> <replacement> | export <file>
gospan doctor | version
```

---

## 7. Vocabulary & Privacy Skill

- Learning is **disabled by default**; requires explicit opt-in.
- MVP: manual CRUD via CLI/GUI (`vocab add/list/export`).
- Optional learning collects only counts/timestamps/last-seen — never full
  keystroke histories, never full sentences.
- Candidates must be user-reviewed and explicitly accepted before activation.
- Exclude configured sensitive processes; treat exclusion as best-effort, not a
  security boundary.
- Provide commands to disable learning and delete learned statistics.
- No telemetry, no network, no secrets in the dictionary.

---

## 8. Storage Skill

- Human-maintained rules → YAML (source of truth).
- Learned vocabulary/statistics → local app data. Two backends are implemented:
  JSONL (`internal/storage/jsonl.go`) and SQLite via the pure-Go
  `modernc.org/sqlite` driver (`internal/storage/sqlite.go`). Select with
  `storage.backend`; always open through `storage.Open(backend, path)`.
- Generated indexes are disposable runtime state.
- File safety: write temp file → fsync → atomic rename; keep backups before
  destructive migration.
- Never store user data next to the executable; use the OS app-data directory.

Default paths:

```text
%APPDATA%\gospan\config.yaml
%APPDATA%\gospan\data\vocabulary.jsonl
%APPDATA%\gospan\logs\gospan.log
```

---

## 9. Testing Skill

| Layer | Location | Focus |
|---|---|---|
| Unit | `internal/**/*_test.go` | matcher, validation, templates, Unicode |
| Benchmark | `internal/engine/benchmark_test.go` | lookup p95, load time |
| Integration | `tests/integration/` | lifecycle, reload, persistence, CLI |
| Manual matrix | `docs/` | Notepad, Terminal, browsers, VS Code, Word, VN IME |

Validation commands:

```powershell
go test ./...
go vet ./...
go test -race ./...
go test -bench=. -benchmem ./internal/engine/...
go build -o dist/gospan.exe ./cmd/gospan
```

Golden-file tests for YAML validation errors are encouraged: assert the exact
error string so error quality does not regress.

---

## 10. Build & Release Skill

```powershell
# Development
go test ./...; go build ./cmd/gospan

# Windows release
$env:CGO_ENABLED="0"; $env:GOOS="windows"; $env:GOARCH="amd64"
go build -trimpath -ldflags "-s -w -X main.version=$version" -o dist/gospan.exe ./cmd/gospan
```

- Prefer a CGO-free build. Any CGO dependency must be documented and justified.
- Release artifacts: exe, version info, example config, README, SHA-256 checksum,
  license/third-party notices, install/uninstall instructions.
- `scripts/build.ps1` and `scripts/test.ps1` wrap the above.

---

## 11. Definition of "Ready" for Any Change

A change is ready when:

1. It respects the architectural rules in `architect.md` §3.1.
2. `gofmt`, `go vet`, `go test ./...` all pass.
3. `go test -race ./...` passes for concurrent code.
4. New behavior has tests; bug fixes have a regression test.
5. Public surface has doc comments.
6. Logging contains no typed user content.
7. Limits (trigger/expansion length, buffer size, channel size) are enforced.

---

## 12. Working Order (Do Not Skip)

```text
Go matcher → Windows keyboard integration → safe text insertion
→ configuration management → vocabulary tools → optional GUI
```

Correctness and predictable typing beat feature count. Build the smallest
reliable engine first. Do not start with GUI, ML, cloud sync, or premature
optimization.
