# Plan — Gospan toward Espanso parity (v2)

**Companion:** `architect.md` (source of truth), `agents.md`, `skill.md`
**Status:** Draft — expands the MVP roadmap with features 1–7 from the parity gap analysis.
**Scope:** Post-MVP work only. Items already shipped (core engine, Windows capture/insertion, CLI, YAML config, optional Fyne GUI/tray, JSONL+SQLite vocabulary) are assumed stable.

---

## 1. Goal

Bring Gospan from "core text expander" to a feature set competitive with
Espanso's everyday use, while keeping the product principles from
`architect.md` §1.2: Go-first, local-first, deterministic, privacy-friendly,
no cloud, no arbitrary shell execution.

The seven work items (in priority order):

1. **Clipboard-assisted insertion** for large/multiline replacements.
2. **Snippet engine**: cursor position marker (`$|$`) and tab stops (`$1`/`$0`).
3. **Clipboard variable** `{{clipboard}}`.
4. **Regex triggers**.
5. **App-specific match sets** (per-application dictionaries).
6. **Search bar** (popup to pick a trigger).
7. **Cross-platform**: macOS and Linux backends.

---

## 2. Non-goals (unchanged)

- Cloud sync, accounts, plugin marketplace, OCR/speech/LLM, custom keyboard
  driver, arbitrary shell execution from templates, guaranteed universal app
  compatibility.

---

## 3. Phases

Each phase has a lead agent (see `agents.md`), supporting agents, deliverables,
and exit criteria. A phase starts only after the previous phase exits and QA
signs off.

### Phase 6 — Production hardening (A9, A0, A8)  *[already planned]*
- Stress/compatibility testing, installer, signing, checksums, dependency audit,
  release docs.
- **Exit:** reproducible `CGO_ENABLED=0` build, documented limitations.

### Phase 7 — Clipboard-assisted insertion (A3, A2, A8)
- **Scope:** strategy selection in `platform` (backspace vs clipboard), safe
  clipboard save/restore, configurable threshold, fallback on failure.
- **Deliverables:** `platform/clipboard.go` interface; Windows impl; config
  `insertion.strategy`, `insertion.clipboard_threshold`; tests against the
  compatibility matrix with large/multiline replacements.
- **Exit:** a 5,000-char multiline replacement inserts reliably in Notepad,
  VS Code, Word, Terminal without corrupting the clipboard.

### Phase 8 — Snippet engine (A2, A6, A7, A8)
- **Scope:** cursor marker `$|$` and tab stops `$1`…`$0` in replacements; render
  into a sequence of insert + cursor-move operations instead of one string.
- **Deliverables:** `expansion/snippet.go`; new `MatchResult` carrying
  `Text` + `CursorOffset` (or `[]InsertOp`); platform `MoveCursor`/`SendKey`;
  `match test --snippet` CLI; GUI preview.
- **Exit:** `$|$` places the caret; `$1`/`$0` tab between fields in at least
  Notepad and VS Code.

### Phase 9 — Clipboard variable (A2, A3, A8)
- **Scope:** `{{clipboard}}` variable reading the current clipboard (text only),
  with a size cap and a privacy toggle.
- **Deliverables:** `platform.ClipboardReader`; `expansion` variable; config
  `clipboard.enabled`, `clipboard.max_length`; tests.
- **Exit:** `{{clipboard}}` expands to the current clipboard text; large
  clipboards are truncated with a warning, never crash.

### Phase 10 — Regex triggers (A2, A1, A8)
- **Scope:** optional regex match per rule (`trigger_type: regex`), with
  capture groups available as `$1`… in the replacement; ReDoS protection
  (timeout + complexity limits).
- **Deliverables:** `engine/regex.go`; config `trigger_type`; validation of
  regex compile + complexity; benchmarks.
- **Exit:** regex triggers work in the matrix; pathological patterns time out
  instead of hanging the hook.

### Phase 11 — App-specific match sets (A1, A2, A3, A4, A8)
- **Scope:** per-application dictionaries activated by foreground process name
  (include/exclude lists, glob on process name).
- **Deliverables:** config `app_filters`; `engine` receives a `MatchContext`
  (process name); platform foreground name already available; reload on focus
  change (debounced); CLI `match list --app`.
- **Exit:** a rule tagged `apps: [code.exe]` expands only in VS Code.

### Phase 12 — Search bar (A7, A4, A6, A8)
- **Scope:** global hotkey opens a small popup listing triggers; fuzzy search;
  selecting one inserts its replacement.
- **Deliverables:** `gui/fyne/search.go` (or a separate `search/` package);
  hotkey registration in platform; fuzzy matcher; tests.
- **Exit:** hotkey opens popup in < 200 ms; selecting inserts correctly.

### Phase 13 — Cross-platform (A3, A9, A8)
- **Scope:** macOS and Linux backends implementing `platform.Platform`:
  keyboard capture, text insertion, clipboard, foreground process.
- **Deliverables:** `internal/platform/darwin/*`, `internal/platform/linux/*`;
  build targets; CI matrix; documented limitations per OS.
- **Exit:** basic expansion works on macOS and Linux in a documented matrix.

### Phase 14 — Production (A9, A0, A8)
- Installer (MSI/DMG/deb), signing, auto-update (optional), final docs.

---

## 4. Dependencies & risks

| Item | Risk | Mitigation |
|---|---|---|
| Clipboard insertion | Overwriting user clipboard | Save/restore with timeout; abort if clipboard changes mid-operation |
| Snippet cursor moves | App-specific caret handling | Limit to supported matrix; document limitations |
| Regex | ReDoS hangs the hook | Compile-time validation + runtime timeout + bounded input |
| Search bar | Global hotkey conflicts | Configurable hotkey; graceful fallback |
| Cross-platform | CGO for macOS (CGEvent) | Evaluate pure-Go options (e.g. `robotgo` needs CGO); document |
| Binary size | modernc + Fyne + clipboard libs | Keep core lean; GUI remains separate module |

---

## 5. Definition of done (v2)

In addition to `architect.md` §20:

- [ ] Clipboard-assisted insertion works for ≥ 5,000-char replacements.
- [ ] `$|$` and `$0`/`$1` tab stops work in Notepad and VS Code.
- [ ] `{{clipboard}}` expands current clipboard (text, capped).
- [ ] Regex triggers work and time out safely.
- [ ] App-specific match sets activate by foreground process.
- [ ] Search bar opens via hotkey and inserts.
- [ ] macOS and Linux basic expansion documented.
- [ ] All new features have unit + integration tests; race detector clean.

---

## 6. Ownership

See `agents.md`. New work items map to existing agents; no new agent roles are
introduced. A0 (Architect) approves interface changes (clipboard, snippet
renderer, app filter, search provider).
