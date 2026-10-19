# Agents — Gospan

**Project:** Gospan (Go Text Expansion Engine)
**Companion documents:** `architect.md` (source of truth), `skill.md` (how-to)
**Status:** Living document.

This document defines the build-time agents that implement Gospan, their
responsibilities, boundaries, and how they hand off work. It is a working
agreement, not a runtime component: Gospan itself ships **no** autonomous agents.

---

## 0. Ground Rules (all agents)

- `architect.md` is the source of truth. If a task contradicts it, stop and
  raise the conflict instead of silently diverging.
- Respect the architectural rules in `architect.md` §3.1 and the "Do Not Skip"
  order in `skill.md` §12.
- Every change must pass: `gofmt`, `go vet ./...`, `go test ./...`, and
  `go test -race ./...` where concurrency is involved.
- Never commit secrets, typed user content, or generated runtime indexes.
- Keep the core (`internal/engine`, `internal/config`, `internal/expansion`)
  free of OS-specific imports.
- Prefer the standard library; justify any third-party dependency against
  `skill.md` §2 and `architect.md` §2.1.

---

## 1. Agent Roster

| ID | Agent | Owns | Phase |
|---|---|---|---|
| A0 | Architect | architecture decisions, ADRs, interface contracts | all |
| A1 | Config Engineer | `internal/config`, `configs/` | 0 |
| A2 | Engine Engineer | `internal/engine`, `internal/input`, `internal/expansion` | 1 |
| A3 | Platform Engineer | `internal/platform/windows`, text insertion | 2 |
| A4 | Runtime Engineer | `internal/app`, lifecycle, logging | 3 |
| A5 | Vocabulary Engineer | `internal/vocabulary`, `internal/storage` | 4 |
| A6 | CLI Engineer | `cmd/gospan`, `internal/cli` (if added) | 0–4 |
| A7 | GUI Engineer | `gui/fyne`, `internal/tray` | 5 |
| A8 | QA Engineer | `tests/`, benchmarks, compatibility matrix | all |
| A9 | Release Engineer | `scripts/`, packaging, signing, checksums | 6 |

Agents are roles, not separate processes. One contributor may hold several, but
**ownership must be explicit** for each file to avoid conflicting edits.

---

## 2. Agent Charters

### A0 — Architect
**Mission:** keep the system coherent and the contracts stable.
**Responsibilities:**
- Own `architect.md`, ADRs (§19), and cross-package interfaces (§5).
- Approve new dependencies and any change to package boundaries.
- Resolve conflicts between agents; the Architect has final say on structure.
**Deliverables:** ADR entries, interface definitions, boundary reviews.
**Definition of done:** decisions recorded and reflected in `architect.md`.

### A1 — Config Engineer
**Mission:** safe, validated, atomically-reloadable configuration.
**Responsibilities:** schema, loader, validator, watcher; example configs and
match dictionaries.
**Deliverables:** `schema.go`, `loader.go`, `validate.go`, `watcher.go`,
`configs/**`.
**Definition of done:** invalid config rejected with file/line/field messages;
reload never publishes partial state (`architect.md` §6).
**Handoff:** publishes `config.Snapshot` to A2 via the Application Service.

### A2 — Engine Engineer
**Mission:** deterministic, fast, OS-independent matching and expansion.
**Responsibilities:** trigger index, suffix buffer, boundary/case rules,
escaping, precedence, template rendering, reentrancy guard.
**Deliverables:** `engine/*.go`, `input/*.go`, `expansion/*.go`, benchmarks.
**Definition of done:** engine works with synthetic events, no GUI/hook; Unicode
and Vietnamese tests pass; p95 lookup < 1 ms in-memory (`architect.md` §14).
**Handoff:** exposes `Matcher` / expansion pipeline to A4.

### A3 — Platform Engineer
**Mission:** reliable Windows capture and safe text insertion.
**Responsibilities:** low-level keyboard hook, event filtering (injected events),
`SendInput` insertion, clipboard strategy, lifecycle/cleanup.
**Deliverables:** `platform/platform.go`, `platform/windows/*.go`.
**Definition of done:** expansion works in the compatibility matrix; no loops;
no destructive replacement when focus is uncertain (`architect.md` §8).
**Handoff:** implements `platform.Platform` consumed by A4.

### A4 — Runtime Engineer
**Mission:** coordinate everything at runtime without leaking internals.
**Responsibilities:** startup/shutdown, wiring config→engine→platform,
diagnostics, logging, reload orchestration.
**Deliverables:** `internal/app/service.go`, `lifecycle.go`, `internal/logging`.
**Definition of done:** runs as a stable background app; GUI/CLI cannot reach
mutable internals (`architect.md` §5.4).

### A5 — Vocabulary Engineer
**Mission:** privacy-first vocabulary management and optional learning.
**Responsibilities:** CRUD, import/export, candidate generation, acceptance
workflow, retention/deletion controls.
**Deliverables:** `internal/vocabulary/*`, `internal/storage/*`.
**Definition of done:** learning off by default; no full keystroke history
persisted; suggestions require explicit acceptance (`architect.md` §9).

### A6 — CLI Engineer
**Mission:** the primary management and debugging interface.
**Responsibilities:** commands, flags, exit codes, `--json`, actionable errors.
**Deliverables:** `cmd/gospan/main.go` + command wiring.
**Definition of done:** stable exit codes; no interactive prompts in automation;
paths with spaces/Unicode work; `start`/`stop` use an explicit lifecycle
mechanism, not PID guessing (`architect.md` §10).

### A7 — GUI Engineer
**Mission:** optional frontend that adds no business logic.
**Responsibilities:** Fyne dashboard, match editor, dictionary browser,
suggestions, settings, diagnostics; system tray if required.
**Deliverables:** `gui/fyne/main.go`, `internal/tray/*`.
**Definition of done:** GUI calls Application Services only; no duplicate
matching logic (`architect.md` §11).

### A8 — QA Engineer
**Mission:** prove correctness and catch regressions.
**Responsibilities:** unit/integration tests, golden-file error tests,
benchmarks, race tests, manual compatibility matrix.
**Deliverables:** `*_test.go`, `tests/integration/`, `tests/fixtures/`,
compatibility reports in `docs/`.
**Definition of done:** coverage of matching, validation, reload, persistence,
CLI; verified resulting text and absence of duplicate/corrupted input
(`architect.md` §15).
**Independence:** QA must be able to fail a change; QA sign-off is required
before a phase exit.

### A9 — Release Engineer
**Mission:** reproducible, reproducible-verifiable builds.
**Responsibilities:** build/test scripts, versioning, checksums, installer,
signing, third-party notices.
**Deliverables:** `scripts/build.ps1`, `scripts/test.ps1`, `dist/`, release docs.
**Definition of done:** `CGO_ENABLED=0` build reproducible; artifacts include
checksum + install/uninstall docs (`architect.md` §17).

---

## 3. Hand-off Contracts

```text
A1 (config)  --Snapshot-->            A4 (app)
A2 (engine)  --Matcher/pipeline-->    A4 (app)
A3 (platform)--Platform impl-->       A4 (app)
A4 (app)     --Service API-->         A6 (CLI) / A7 (GUI)
A5 (vocab)   --VocabularyStore-->     A4 / A6
A8 (QA)      --verdict-->             all (gate)
A9 (release) --artifacts-->           end users
```

Contract rules:

- Consumers depend on interfaces, not concrete implementations.
- A change to a shared interface requires Architect (A0) approval and a note in
  `architect.md` or an ADR.
- New exported symbols need doc comments (`skill.md` §2.1).

---

## 4. Phase Ownership Map

| Phase | Lead | Supporting | Exit criteria |
|---|---|---|---|
| 0 Foundation | A1 | A6, A8 | invalid config rejected with actionable errors |
| 1 Matching Engine | A2 | A8 | engine works without GUI/hook |
| 2 Windows Integration | A3 | A8 | reliable expansion in selected apps |
| 3 Runtime Management | A4 | A6, A8 | stable background app |
| 4 Vocabulary | A5 | A6, A8 | CRUD/import/export without editing YAML |
| 5 Optional GUI | A7 | A4, A8 | no duplicate business logic |
| 6 Hardening | A9 | A0, A8 | reproducible build, documented limits |

A phase does not start until the previous phase's exit criteria are met and QA
(A8) has signed off.

---

## 5. Escalation & Conflict Resolution

1. Interface or boundary conflict → Architect (A0) decides, records an ADR.
2. Ownership conflict (two agents editing one file) → split the file or assign a
   single owner; A0 arbitrates.
3. Dependency request → A0 evaluates against `architect.md` §2.1.
4. Security/privacy concern → stop the change; require explicit opt-in design
   per `architect.md` §9 and §16.
5. Performance regression → A2 re-benchmarks; no speculative optimization
   without measured evidence (`architect.md` §7.3).

---

## 6. Definition of Done (project, from `architect.md` §20)

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
