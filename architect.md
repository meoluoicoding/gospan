# Architecture — Go Text Expansion Engine

**Project codename:** Gospan  
**Document:** `architect.md`  
**Version:** 1.0.0  
**Status:** Architecture Draft  
**Primary language:** Go  
**Initial platform:** Windows 10/11 x64  
**Product category:** Native text expander, abbreviation engine, vocabulary assistant

---

## 1. Product Vision

Build a native desktop text-expansion application inspired by Espanso, implemented primarily in Go.

The application allows users to define short triggers that expand into longer text, templates, code snippets, Vietnamese phrases, multilingual vocabulary, and reusable text blocks.

The product should be fast, predictable, configurable, privacy-friendly, and extensible without requiring cloud services.

### 1.1 Core Example

Typing:

- `;dc` → `được`
- `;k` → `không`
- `;kt` → `kiểm tra`
- `;brse` → `Bridge Software Engineer`
- `;mail` → a predefined email template
- `;date` → the current date

A vocabulary suggestion feature may detect frequently used phrases and offer to create new dictionary entries.

### 1.2 Product Principles

- Go-first implementation.
- Native operating-system integration.
- CLI-first development and debugging.
- YAML-based configuration.
- Local-first data storage.
- No mandatory account or cloud connection.
- Deterministic expansion behavior.
- Small runtime footprint.
- Explicit permission and safety boundaries.
- GUI must not be required for core functionality.

### 1.3 Non-goals for MVP

- Full predictive AI keyboard.
- Automatic rewriting of arbitrary sentences.
- Cloud synchronization.
- Cross-device accounts.
- Complex plugin marketplace.
- OCR, speech recognition, or LLM inference.
- A custom keyboard driver.
- Guaranteed support for every application or elevated security desktop.

### 1.4 In scope for v2 (toward Espanso parity)

The following are **in scope** for post-MVP releases and are detailed in
`plan.md`:

- Clipboard-assisted text insertion for large/multiline replacements.
- Snippet engine: cursor position marker (`$|$`) and tab stops (`$0`/`$1`).
- `{{clipboard}}` template variable.
- Regex triggers (with ReDoS protection).
- App-specific match sets (per foreground process).
- Search bar (global hotkey → fuzzy picker → insert).
- macOS and Linux platform backends.

These do not contradict the non-goals above: no cloud, no accounts, no
arbitrary shell execution, no AI/OCR/speech, no custom keyboard driver.

---

## 2. Technology Stack

| Concern | Technology |
|---|---|
| Core language | Go |
| Windows integration | Windows API bindings from Go |
| Configuration | YAML |
| Configuration validation | Go structs and explicit validation |
| CLI | Go standard library; Cobra optional |
| GUI | Fyne, evaluated as an optional frontend |
| Clipboard | OS clipboard via Go bindings (Windows/macOS/Linux) |
| Regular expressions | Go `regexp` standard library |
| Search UI | Fyne popup (optional frontend) |
| Cross-platform | `//go:build` per OS behind `platform.Platform` |
| Logging | `log/slog` |
| Local dictionary | In-memory indexes |
| Persistent vocabulary | JSONL or SQLite |
| SQLite driver | Pure-Go driver preferred |
| Testing | `go test`, race detector |
| Benchmarks | Go testing benchmarks |
| Build automation | `go test`, `go vet`, Makefile or justfile |
| Packaging | Native Windows executable and installer |
| Distribution | Signed executable when feasible |

### 2.1 Dependency Policy

Use the standard library by default.

Third-party dependencies must have a clear technical purpose and be evaluated for:

- Maintenance status.
- License compatibility.
- Security history.
- Windows support.
- CGO requirements.
- Binary size.
- Build reproducibility.

Avoid unnecessary frameworks and abstractions.

### 2.2 Pure-Go Requirement

The project targets Go for all application logic, matching, configuration, persistence, and GUI code.

Operating-system integration may call native Windows APIs through Go bindings. Such calls do not imply that the core engine must be written in C or C++.

If a dependency introduces native C/C++ code or CGO, it must be documented and justified. The preferred release build should not require a separate C/C++ application component.

---

## 3. System Architecture

The system follows a modular monolith architecture.

The core engine is independent of the CLI, GUI, configuration format, and operating-system event source.

```text
                    +----------------------+
                    | CLI / Optional GUI   |
                    +----------+-----------+
                               |
                    +----------v-----------+
                    | Application Service  |
                    +----------+-----------+
                               |
             +-----------------+-----------------+
             |                 |                 |
     +-------v-------+ +-------v-------+ +-------v-------+
     | Config Module | | Match Engine  | | Vocabulary    |
     | YAML / Reload | | Index / Rules | | Suggestions   |
     +---------------+ +-------+-------+ +---------------+
                               |
                    +----------v-----------+
                    | Expansion Pipeline   |
                    +----------+-----------+
                               |
                    +----------v-----------+
                    | Platform Interface   |
                    +----------+-----------+
                               |
                    +----------v-----------+
                    | Windows Integration  |
                    | Keyboard / Insertion |
                    +----------------------+
```

### 3.1 Architectural Rules

- The matching engine must not import Windows-specific packages.
- The configuration layer must not perform keyboard I/O.
- The platform layer must not interpret business rules.
- The GUI must call application services rather than manipulate engine internals.
- The vocabulary subsystem must not block the keyboard event loop.
- Configuration updates must be validated before becoming active.
- The active dictionary must be immutable or safely published to concurrent readers.
- Platform-specific code must be isolated behind interfaces.
- Regex triggers must be validated at load time and executed with a timeout.
- Snippet rendering must produce an explicit cursor offset, not a magic string.
- App-specific matching must depend on the foreground process name, never on window titles or keystrokes.
- The search bar must call application services, not engine internals.
- Clipboard reads must be size-capped and opt-in.

---

## 4. Repository Structure

```text
gospan/
├── architect.md
├── README.md
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
│
├── cmd/
│   └── gospan/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── service.go
│   │   ├── lifecycle.go
│   │   └── filter.go
│   │
│   ├── config/
│   │   ├── schema.go
│   │   ├── loader.go
│   │   ├── validate.go
│   │   └── watcher.go
│   │
│   ├── engine/
│   │   ├── engine.go
│   │   ├── matcher.go
│   │   ├── index.go
│   │   ├── trigger.go
│   │   ├── regex.go
│   │   └── benchmark_test.go
│   │
│   ├── expansion/
│   │   ├── pipeline.go
│   │   ├── template.go
│   │   ├── variables.go
│   │   └── snippet.go
│   │
│   ├── input/
│   │   ├── events.go
│   │   └── buffer.go
│   │
│   ├── platform/
│   │   ├── platform.go
│   │   ├── clipboard.go
│   │   ├── host/
│   │   └── windows/
│   │       ├── keyboard.go
│   │       ├── insert.go
│   │       ├── clipboard.go
│   │       └── process.go
│   │   ├── darwin/
│   │   └── linux/
│   │
│   ├── vocabulary/
│   │   ├── service.go
│   │   ├── suggestion.go
│   │   └── candidate.go
│   │
│   ├── storage/
│   │   ├── interface.go
│   │   ├── jsonl.go
│   │   └── sqlite.go
│   │
│   ├── logging/
│   │   └── logging.go
│   │
│   └── tray/
│       └── tray.go
│
├── gui/
│   └── fyne/
│       ├── main.go
│       ├── theme.go
│       ├── icon.go
│       └── search.go
│
├── configs/
│   ├── config.example.yaml
│   └── matches/
│       ├── vietnamese.yaml
│       ├── english.yaml
│       └── programming.yaml
│
├── tests/
│   ├── integration/
│   └── fixtures/
│
├── scripts/
│   ├── build.ps1
│   └── test.ps1
│
└── docs/
    ├── architecture-decisions.md
    ├── configuration.md
    ├── windows-integration.md
    └── security.md
```

All packages remain under `internal/` until a stable public library API is genuinely needed.

The `gui/` directory is optional and must not be imported by the core engine.

---

## 5. Core Interfaces

### 5.1 Platform Interface

```go
type KeyboardEvent struct {
    Rune      rune
    KeyCode   uint32
    IsDown    bool
    Modifiers uint32
    Timestamp int64
}

type Platform interface {
    Start() error
    Stop() error
    Events() <-chan KeyboardEvent
    ReplacePreviousText(typed string, replacement string) error
}
```

This is an architectural sketch. The actual implementation must define cancellation, buffer overflow behavior, event filtering, and replacement semantics.

### 5.2 Matcher Interface

```go
type Match struct {
    Trigger     string
    Replacement string
}

type Matcher interface {
    MatchSuffix(text string) (Match, bool)
    Reload(entries []Match) error
}
```

The matching engine is independent of Windows and can be tested with synthetic input.

### 5.3 Storage Interface

```go
type VocabularyEntry struct {
    Trigger    string
    Expansion  string
    Language   string
    Source     string
    CreatedAt  string
}

type VocabularyStore interface {
    Add(entry VocabularyEntry) error
    List() ([]VocabularyEntry, error)
    Close() error
}
```

Production code may use `time.Time` instead of a string for `CreatedAt`.

### 5.4 Application Service

The application service coordinates:

- Startup and shutdown.
- Configuration loading.
- Dictionary construction.
- Platform event consumption.
- Expansion execution.
- Reload and diagnostics.
- Vocabulary suggestion delivery.

It must not expose internal mutable indexes to GUI components.

### 5.5 Clipboard Interface

```go
type Clipboard interface {
    ReadText() (string, error)   // capped; returns ErrTooLarge when exceeded
    WriteText(string) error
}
```

### 5.6 Snippet Renderer

```go
type SnippetResult struct {
    Text         string
    CursorOffset  int      // -1 means "leave cursor at end"
    TabStops     []int    // byte offsets of $1..$0
}

type SnippetRenderer interface {
    Render(template string, ctx expansion.Context) (SnippetResult, error)
}
```

### 5.7 App Filter

```go
type MatchContext struct {
    ProcessName string // lower-cased base name of the foreground process
}

type AppFilter interface {
    Allowed(ctx MatchContext, rule config.MatchConfig) bool
}
```

### 5.8 Search Provider

```go
type SearchCandidate struct {
    Trigger     string
    Replacement string
    Score       float64
}

type SearchProvider interface {
    Search(query string, limit int) []SearchCandidate
}

---

## 6. Configuration Design

Use YAML as the human-editable configuration format.

Example:

```yaml
app:
  name: gospan
  log_level: info
  start_with_windows: false

engine:
  enabled: true
  case_sensitive: false
  word_boundary: true
  max_trigger_length: 64
  max_expansion_length: 16384

input:
  minimum_interval_ms: 0
  ignore_injected_events: true
  excluded_processes:
    - keepass.exe

suggestions:
  enabled: true
  learn_from_typing: false
  minimum_occurrences: 5
  minimum_phrase_length: 2
  maximum_candidates: 5

storage:
  backend: jsonl
  path: data/vocabulary.jsonl

matches:
  - trigger: ";dc"
    replace: "được"

  - trigger: ";k"
    replace: "không"

  - trigger: ";kt"
    replace: "kiểm tra"

  - trigger: ";brse"
    replace: "Bridge Software Engineer"
```

### 6.1 Configuration Requirements

- Validate all entries before activating them.
- Reject duplicate triggers unless an explicit override rule exists.
- Reject empty triggers.
- Set limits for trigger and replacement length.
- Support UTF-8 and Vietnamese diacritics.
- Define deterministic matching precedence.
- Provide useful line and field information for validation errors.
- Support atomic configuration reload.
- Preserve the last valid configuration if reload fails.
- Never silently overwrite a user's configuration.

### 6.2 Reload Strategy

1. Read configuration files.
2. Parse YAML.
3. Validate schema and semantic rules.
4. Build a new matcher index.
5. Publish the new immutable state.
6. Report success or failure.

Never expose a partially constructed dictionary to concurrent readers.

### 6.3 Snippet Markers

Replacements may contain cursor and tab-stop markers:

```yaml
matches:
  - trigger: ";sig"
    replace: "Trân trọng,\\nKhoa$|$"
  - trigger: ";func"
    replace: "func $1() {\\n\\t$0\\n}"
```

- `$|$` places the caret after insertion.
- `$1`, `$2`, … are tab stops; `$0` is the final stop.
- Markers are stripped from the inserted text.
- Snippets require a platform that supports cursor movement.

### 6.4 Regex Triggers

```yaml
matches:
  - trigger: "^;d(\\d+)$"
    trigger_type: regex
    replace: "Section $1"
```

- `trigger_type` defaults to `exact`.
- Capture groups are available as `$1`… in the replacement.
- Regexes are compiled and complexity-checked at load time.
- Execution is bounded by a timeout; pathological patterns never block the hook.

### 6.5 App Filters

```yaml
matches:
  - trigger: ";code"
    replace: "console.log()"
    apps:
      include: ["code.exe", "windowsterminal.exe"]
      exclude: ["keepass.exe"]
```

- `apps` is optional; absent means "all applications".
- Matching is by lower-cased process base name (glob supported).
- Exclusion in `input.excluded_processes` still takes precedence.

### 6.6 Clipboard

```yaml
clipboard:
  enabled: false          # opt-in
  max_length: 4096
```

- `{{clipboard}}` reads the current clipboard (text only).
- Reads are size-capped; oversized content is truncated with a warning.
- Clipboard insertion (strategy) is configured under `insertion`.

### 6.7 Insertion Strategy

```yaml
insertion:
  strategy: auto          # auto | backspace | clipboard
  clipboard_threshold: 256 # use clipboard when replacement exceeds this (runes)
```

- `auto` uses backspace for short replacements and clipboard for long/multiline.
- Clipboard strategy saves and restores the user's clipboard safely.

---

## 7. Text Expansion Engine

### 7.1 Processing Pipeline

```text
Keyboard event
    |
    v
Event filtering
    |
    v
Input buffer update
    |
    v
Trigger candidate lookup
    |
    v
Boundary and rule validation
    |
    v
Expansion rendering
    |
    v
Text replacement
    |
    v
Cooldown / reentrancy guard
```

### 7.2 Trigger Rules

The engine must support:

- Exact trigger matching.
- Configurable word-boundary requirements.
- Case-sensitive or case-insensitive matching.
- Prefix conflicts.
- Trigger escaping.
- Maximum trigger length.
- Multiple dictionaries.
- Per-rule enable and disable.
- Plain-text replacements.
- Regex triggers (with capture groups).
- App-specific activation.
- Snippet markers (cursor + tab stops).
- Clipboard variable.

Example:

```yaml
matches:
  - trigger: ";sig"
    replace: |
      Trân trọng,
      Khoa

  - trigger: ";date"
    replace: "{{date:2006-01-02}}"
```

The template syntax above is illustrative. The implementation must define and test the supported variables explicitly.

### 7.3 Matching Strategy

MVP:

- Use a suffix buffer with a bounded size.
- Index triggers in a map for direct lookup.
- Evaluate only plausible suffix candidates.
- Avoid scanning every dictionary entry on every keystroke.

Later optimization:

- Use a trie or another prefix-oriented index if benchmarks justify it.
- Consider Aho–Corasick only if the product expands into large-scale multi-pattern scanning.

Do not introduce advanced matching structures without measured evidence.

### 7.4 Replacement Safety

- Detect injected events where supported.
- Prevent expansion loops.
- Ensure the engine does not process its own replacement as new user input.
- Preserve normal typing when no trigger matches.
- Recover from platform insertion failures.
- Avoid destructive replacement when focus or input state is uncertain.
- Do not claim universal compatibility with every application.

---

## 8. Windows Integration

The initial implementation targets Windows 10/11.

### 8.1 Keyboard Event Capture

Evaluate Windows low-level keyboard hooks using Go bindings to the Windows API.

Requirements:

- Keep the hook callback short.
- Do not perform YAML parsing or database writes inside the callback.
- Forward relevant events to a bounded processing pipeline.
- Define behavior for queue saturation.
- Handle application shutdown and hook cleanup.
- Filter injected events where feasible.
- Respect secure desktop and process-integrity boundaries.

A Windows keyboard hook does not guarantee access to every protected or elevated application.

### 8.2 Text Insertion

Evaluate `SendInput` for simulated keyboard input.

Two possible strategies:

1. Backspace over the matched trigger, then insert the replacement.
2. Use clipboard-assisted insertion for large or multiline replacements.

The first strategy is suitable for simple triggers but may interact badly with input method editors, dead keys, Unicode composition, or applications that process keyboard input unusually.

Clipboard-assisted insertion can overwrite user clipboard contents unless the application saves and restores the clipboard safely.

The insertion layer must be configurable and tested against common applications.

### 8.3 Windows-Specific Concerns

- Keyboard layout changes.
- Vietnamese Telex/VNI input methods.
- Unicode and surrogate pairs.
- IME composition.
- Elevated applications.
- Remote desktop sessions.
- Secure desktop.
- Hook lifecycle and thread requirements.
- Antivirus false positives.
- Installer and startup registration.
- Windows API error reporting.

### 8.4 Clipboard Integration

- Clipboard reads for `{{clipboard}}` are text-only and size-capped.
- Clipboard-assisted insertion saves the current clipboard, writes the
  replacement, sends paste, and restores the original clipboard within a timeout.
- If the clipboard changes during the operation, insertion aborts rather than
  corrupting user data.

Use `//go:build windows` for platform-specific source files and provide stubs only where they are genuinely useful.

---

## 9. Vocabulary Suggestions

The vocabulary subsystem is optional and must be disabled by default until the user explicitly enables learning.

### 9.1 MVP Behavior

The user can manually add a vocabulary entry from the CLI or GUI.

Example:

```text
gospan vocab add ";kt" "kiểm tra"
gospan vocab list
gospan vocab export vocabulary.jsonl
```

### 9.2 Optional Local Learning

When enabled, the application may collect limited local statistics such as:

- Repeated words or phrases.
- Occurrence counts.
- Last-seen timestamps.
- Whether a candidate was accepted or dismissed.

It must not persist full keystroke histories by default.

### 9.3 Candidate Generation

A candidate is eligible only if:

- It exceeds a configurable occurrence threshold.
- It meets the minimum length.
- It is not already represented by a trigger.
- It is not on the exclusion list.
- It is not detected inside a configured sensitive context.

Suggestions must be reviewed and explicitly accepted by the user.

Do not automatically convert observed typing into active replacements.

### 9.4 Privacy

- No telemetry by default.
- No network dependency.
- No storage of complete typed sentences by default.
- Exclude password managers and other configured applications.
- Provide commands to disable learning and delete learned statistics.
- Document the limits of application-based exclusion.

Application exclusions are best-effort protections, not a security boundary. The safest mode disables global input capture when sensitive applications are active or when reliable detection is unavailable.

---

## 10. CLI Specification

The CLI is the primary management and debugging interface.

```text
gospan start
gospan stop
gospan status

gospan config validate
gospan config path
gospan config reload

gospan match list
gospan match test ";kt"
gospan match add ";kt" "kiểm tra"
gospan match remove ";kt"
gospan match search "kiểm"          # fuzzy search triggers

gospan search                       # open the search bar (GUI)

gospan vocab list
gospan vocab add ";brse" "Bridge Software Engineer"
gospan vocab export vocabulary.jsonl

gospan doctor
gospan version
```

Command names are provisional and may be adjusted to avoid platform-specific limitations.

### 10.1 CLI Requirements

- Stable exit codes.
- Human-readable output.
- Optional JSON output for scripts.
- Actionable errors.
- No interactive prompt in automation mode.
- `--config` override.
- `--verbose` and `--quiet`.
- Safe handling of paths containing spaces and Unicode.

`start` and `stop` must use an explicit lifecycle mechanism if the application runs as a background process. Do not implement process control using fragile PID assumptions alone.

---

## 11. GUI Strategy

GUI is not required for the first working release.

### 11.1 Preferred Development Order

1. Core engine.
2. Windows keyboard capture.
3. Reliable text insertion.
4. CLI and diagnostics.
5. Configuration reload.
6. Optional system tray.
7. Optional Fyne management interface.

### 11.2 Fyne

Fyne is a candidate for a Go-native cross-platform GUI.

Potential screens:

- Dashboard and engine status.
- Match editor.
- Dictionary browser.
- Vocabulary suggestions.
- Settings.
- Diagnostics and logs.
- Search bar (global hotkey popup).

The GUI must call application services and must not implement a second copy of the matching logic.

If native Windows look and feel becomes a strict requirement, evaluate a direct Windows GUI binding before committing to Fyne.

---

## 12. Concurrency Model

Avoid unnecessary goroutines.

Recommended logical components:

- Main lifecycle controller.
- Keyboard event producer.
- Event processing consumer.
- Configuration reload worker.
- Optional vocabulary persistence worker.
- GUI event loop, if enabled.

### 12.1 Rules

- Keep keyboard callbacks short.
- Use bounded channels where appropriate.
- Do not spawn one goroutine per keystroke.
- Define channel closure and cancellation behavior.
- Avoid blocking the hook callback on storage.
- Protect shared state or publish immutable snapshots.
- Use a single owner for mutable event buffers where possible.
- Run tests with the race detector.

The correct number of goroutines depends on the actual implementation and measured bottlenecks. Goroutine count is not itself a performance objective.

---

## 13. Data and Persistence

### 13.1 Source of Truth

Human-maintained dictionary rules belong in YAML.

Learned vocabulary and optional statistics belong in local application data.

Generated indexes are disposable runtime state.

### 13.2 Storage Options

**JSONL**

Suitable for an MVP with simple append and export operations.

Advantages:

- Easy to inspect.
- Easy to back up.
- No native database server.
- Good interoperability.

Limitations:

- Updates and deletes require rewriting or compaction.
- Concurrent mutation needs explicit coordination.

**SQLite**

Suitable when the application needs efficient queries, indexed statistics, migrations, or reliable transactional updates.

Use a pure-Go driver if it meets stability and performance requirements.

### 13.3 File Safety

- Write temporary files before replacing configuration snapshots.
- Use atomic replacement where supported.
- Validate data before committing changes.
- Maintain backups when destructive migration is involved.
- Never store secrets in the dictionary.
- Keep user data separate from the executable directory.

---

## 14. Performance Targets

These are engineering targets, not verified benchmarks.

| Metric | Initial target |
|---|---:|
| Matcher lookup, in-memory | p95 below 1 ms |
| Regex trigger evaluation | p95 below 5 ms, with timeout |
| Clipboard insertion | p95 below 50 ms for 1 KiB |
| Search bar open | below 200 ms |
| Configuration with 10,000 triggers | Load below 500 ms |
| Idle CPU | Near zero when no relevant events occur |
| Keyboard event callback | Minimal work; no blocking I/O |
| Memory usage | Measure on release builds |
| Startup time | Measure on target Windows hardware |

Benchmarks must distinguish matcher performance from end-to-end keyboard capture and text insertion latency.

Use realistic Unicode triggers and replacement strings. Do not optimize solely against ASCII-only synthetic inputs.

---

## 15. Testing Strategy

### 15.1 Unit Tests

- Trigger matching.
- Duplicate trigger validation.
- Prefix collisions.
- Word boundaries.
- Unicode handling.
- Vietnamese diacritics.
- YAML parsing.
- Template variables.
- Configuration reload.
- Vocabulary filtering.
- Snippet rendering and cursor offsets.
- Regex triggers and ReDoS protection.
- App-specific activation.
- Clipboard variable and insertion.

### 15.2 Integration Tests

- Start and stop lifecycle.
- Config reload while running.
- Dictionary persistence.
- CLI commands.
- Windows hook registration.
- Text insertion.
- Recovery after platform errors.

### 15.3 Manual Compatibility Matrix

Test at minimum:

- Notepad.
- Windows Terminal.
- Browser text fields.
- VS Code.
- Microsoft Word.
- Vietnamese input methods.
- Applications running with elevated privileges, where applicable.

Tests must verify both the resulting text and the absence of duplicate or corrupted input.

### 15.4 Build Validation

```powershell
go test ./...
go vet ./...
go test -race ./...
go test -bench=. ./internal/engine/...
go build -o dist/gospan.exe ./cmd/gospan
```

The race detector and benchmarks are validation tools, not substitutes for end-to-end testing on Windows.

---

## 16. Security and Reliability

Threat model:

- Accidental destructive expansion.
- Malformed configuration.
- Unintended capture in sensitive applications.
- Clipboard corruption.
- Event injection loops.
- Unexpected shutdown.
- Malicious or oversized configuration entries.
- Supply-chain vulnerabilities.

Mitigations:

- Validate all configuration input.
- Limit resource usage.
- Keep logs free of typed content by default.
- Never execute arbitrary shell commands from replacement templates.
- Require explicit opt-in for learning.
- Support disabling expansion globally.
- Provide a reliable shutdown mechanism.
- Pin and audit dependencies.
- Sign release artifacts when feasible.
- Bound regex execution time and complexity.
- Treat clipboard contents as sensitive: cap reads, opt-in only.
- Never execute arbitrary commands from snippet markers.

The application must not attempt to bypass operating-system security boundaries.

---

## 17. Build and Release

### Development

```powershell
go test ./...
go build ./cmd/gospan
```

### Windows release

```powershell
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

go build -trimpath -o dist/gospan.exe ./cmd/gospan
```

The final build settings must be validated against the selected Windows API bindings and GUI dependencies. A dependency requiring CGO may invalidate a CGO-disabled build.

### macOS (planned)

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o dist/gospan-macos ./cmd/gospan
```

### Linux (planned)

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/gospan-linux ./cmd/gospan
```

Release artifacts should include:

- Executable.
- Version information.
- Example configuration.
- README and usage guide.
- SHA-256 checksum.
- License and third-party notices.
- Installation and uninstallation instructions.

---

## 18. Implementation Roadmap

### Phase 0 — Foundation

- Initialize Go module.
- Define package boundaries.
- Implement configuration schema.
- Implement YAML parsing and validation.
- Create CLI diagnostics.
- Add unit tests.

**Exit criteria:** invalid configuration is rejected with actionable errors.

### Phase 1 — Pure-Go Matching Engine

- Implement trigger index.
- Implement suffix buffer.
- Implement boundary rules.
- Implement deterministic matching.
- Add benchmarks and Unicode tests.

**Exit criteria:** engine works without a running GUI or Windows hook.

### Phase 2 — Windows Integration

- Implement keyboard capture.
- Implement input filtering.
- Implement text replacement.
- Handle injected events and lifecycle.
- Test Vietnamese input methods and Unicode.

**Exit criteria:** basic expansion works reliably in selected Windows applications.

### Phase 3 — Runtime Management

- Implement start and stop.
- Implement status and diagnostics.
- Add configuration reload.
- Add robust logging.
- Add persistent settings.

**Exit criteria:** the engine runs as a stable background application.

### Phase 4 — Vocabulary Management

- Add dictionary CRUD operations.
- Add import and export.
- Add optional suggestion candidates.
- Add explicit acceptance workflow.
- Add deletion and privacy controls.

**Exit criteria:** users can maintain and extend vocabulary without editing YAML manually.

### Phase 5 — Optional GUI

- Evaluate Fyne.
- Add system tray if required.
- Build match editor and settings screens.
- Connect GUI to application services.

**Exit criteria:** GUI introduces no duplicate business logic.

### Phase 6 — Production Hardening

- Stress testing.
- Compatibility testing.
- Installer.
- Dependency audit.
- Release documentation.
- Signing and checksums.

**Exit criteria:** reproducible build and documented operational limitations.

### Phase 7 — Clipboard-assisted Insertion

- Insertion strategy selection (backspace / clipboard / auto).
- Safe clipboard save and restore with timeout.
- Large/multiline replacement tests in the compatibility matrix.

**Exit criteria:** 5,000-char multiline replacement inserts reliably without
corrupting the clipboard.

### Phase 8 — Snippet Engine

- Cursor marker `$|$` and tab stops `$0`/`$1`.
- Render to insert + cursor-move operations.
- Platform cursor movement.
- `match test --snippet`.

**Exit criteria:** caret placement and tab stops work in Notepad and VS Code.

### Phase 9 — Clipboard Variable

- `{{clipboard}}` variable (text only, size-capped).
- Privacy toggle and truncation warning.

**Exit criteria:** clipboard content expands correctly; oversized content is
truncated, never crashes.

### Phase 10 — Regex Triggers

- `trigger_type: regex`.
- Capture groups in replacements.
- Compile-time validation and ReDoS protection (timeout).

**Exit criteria:** regex triggers work in the matrix; pathological patterns
time out.

### Phase 11 — App-specific Match Sets

- `apps.include` / `apps.exclude` per rule.
- Match context with foreground process name.
- Reload on focus change (debounced).

**Exit criteria:** a rule tagged for an app expands only there.

### Phase 12 — Search Bar

- Global hotkey → popup with fuzzy search.
- Select to insert.

**Exit criteria:** popup opens in < 200 ms and inserts correctly.

### Phase 13 — Cross-platform

- macOS backend (keyboard capture, insertion, clipboard, foreground process).
- Linux backend.
- Build targets and documented limitations.

**Exit criteria:** basic expansion works on macOS and Linux in a documented
matrix.

---

## 19. Architecture Decision Records

### ADR-001: Go as the primary language

**Decision:** Use Go for core logic and application services.

**Reason:** Simple deployment, strong standard library, concurrency support, and suitable tooling.

### ADR-002: YAML for user-defined matches

**Decision:** YAML is the editable configuration format.

**Reason:** Human-readable, supports structured rules, and works well with version control.

### ADR-003: Modular monolith

**Decision:** Start with one application and internal packages.

**Reason:** Avoid premature distributed architecture and IPC complexity.

### ADR-004: Windows API integration through Go

**Decision:** Use Go bindings for required operating-system APIs.

**Reason:** Preserve a Go-centric codebase while using native capabilities.

### ADR-005: GUI after engine stability

**Decision:** CLI and engine precede GUI.

**Reason:** Core behavior is easier to test without GUI lifecycle and rendering concerns.

### ADR-006: Privacy-first vocabulary learning

**Decision:** Learning is opt-in and suggestions require user approval.

**Reason:** A global keyboard utility handles potentially sensitive input.

### ADR-007: Snippet markers over magic strings

**Decision:** Cursor position and tab stops are expressed with explicit markers
(`$|$`, `$0`/`$1`) rendered into a structured result (text + cursor offset +
tab stops), not embedded as invisible characters.

**Reason:** Keeps rendering testable and lets the platform move the caret
deterministically.

### ADR-008: Regex triggers are opt-in and bounded

**Decision:** Regex triggers require `trigger_type: regex` and are executed with
a timeout and complexity limits.

**Reason:** A global keyboard hook must never hang on pathological input.

### ADR-009: App filtering by process name only

**Decision:** App-specific matching keys off the lower-cased foreground process
base name, never window titles or keystrokes.

**Reason:** Process names are stable and cheap to obtain; window titles are
unreliable and may contain sensitive data.

### ADR-010: Clipboard insertion is a strategy, not a default

**Decision:** Short replacements use backspace + typing; clipboard insertion is
used only above a configurable threshold or for multiline blocks, with safe
save/restore.

**Reason:** Clipboard insertion disrupts the user's clipboard; minimize its use.

### ADR-011: Search bar is a frontend concern

**Decision:** The search bar is implemented in the GUI layer and calls
application services; it does not reimplement matching.

**Reason:** Preserves the rule that business logic lives in the engine.

### ADR-012: Cross-platform via the existing interface

**Decision:** macOS and Linux implement `platform.Platform`; no changes to the
core engine.

**Reason:** The interface boundary already exists (§5.1).

---

## 20. Definition of Done

The MVP is complete when:

- [ ] Go project builds reproducibly.
- [ ] YAML configuration validates correctly.
- [ ] Triggers expand into configured replacements.
- [ ] Unicode and Vietnamese text are tested.
- [ ] Windows keyboard capture works in the supported compatibility matrix.
- [ ] Replacement does not recursively trigger itself.
- [ ] Configuration reload is safe.
- [ ] CLI provides status, diagnostics, and dictionary management.
- [ ] Learning is disabled by default.
- [ ] No network connection is required.
- [ ] Unit and integration tests pass.
- [ ] Known limitations are documented.
- [ ] User data can be exported and backed up.
- [ ] Clipboard-assisted insertion works for large replacements.
- [ ] Snippet cursor and tab stops work in supported apps.
- [ ] `{{clipboard}}` variable works (opt-in).
- [ ] Regex triggers work and time out safely.
- [ ] App-specific match sets activate by process.
- [ ] Search bar opens via hotkey and inserts.
- [ ] macOS and Linux basic expansion is documented.

---

## 21. Final Architectural Principle

Build the smallest reliable engine first.

Do not start with GUI polish, machine learning, cloud synchronization, complex plugin systems, or premature optimization.

The critical path is:

**Go matcher → Windows keyboard integration → safe text insertion → configuration management → vocabulary tools → optional GUI.**

Correctness and predictable typing behavior take priority over feature count.
