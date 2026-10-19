# Configuration Reference

Gospan reads YAML from, in order of precedence:

1. `--config <path>`
2. `GOSPAN_CONFIG` environment variable
3. `%APPDATA%\gospan\config.yaml`

## Top-level sections

| Section | Purpose |
|---|---|
| `app` | Name, log level, start-with-Windows |
| `engine` | Matching behavior and limits |
| `input` | Event handling and process exclusions |
| `suggestions` | Optional vocabulary learning |
| `storage` | Vocabulary persistence backend |
| `include` | Additional dictionary files to merge |
| `matches` | Inline trigger rules |

## Engine

```yaml
engine:
  case_sensitive: false     # default false
  word_boundary: true       # default true
  escape_char: "\\"         # a single character, or "" to disable
  max_trigger_length: 64    # hard cap 1024
  max_expansion_length: 16384
```

- **word_boundary** applies only to triggers that start with a letter, digit, or
  `_`. Events that begin with punctuation (like `;`) expand anywhere.
- **escape_char** typed immediately before a trigger suppresses expansion and
  removes the escape from the output.

## Matches

```yaml
matches:
  - trigger: ";sig"
    replace: |
      Trân trọng,
      Khoa
    description: optional
    enabled: true          # default true
```

Validation rejects empty triggers, duplicates (case-insensitively unless
`case_sensitive` is true), and oversized triggers/replacements. Errors include
the file and line:

```text
config.yaml:12: trigger: duplicate trigger ";kt"; first defined at config.yaml:8
```

## Regex triggers

A rule can use a regular expression instead of an exact trigger:

```yaml
matches:
  - trigger: '^;d(\d+)$'
    trigger_type: regex
    replace: 'Section $1'
```

- `trigger_type` defaults to `exact`.
- Capture groups are available as `$1`, `$2`, … in the replacement (and `$0`
  for the full match).
- Regexes are compiled and rejected at load time if invalid.
- Go's `regexp` is RE2-based, so matching is linear time; an additional
  timeout guards every evaluation so a pathological pattern can never hang the
  keyboard hook.
- Regex triggers are **greedy**: they fire as soon as the pattern matches the
  current input. Use anchors (`^`, `$`) or minimum lengths (`\d{2,}`) to avoid
  premature expansion.

## Templates

Replacement text may contain variables:

| Variable | Meaning |
|---|---|
| `{{date}}` | Today's date (`2006-01-02`) |
| `{{date:LAYOUT}}` | Date in a Go layout |
| `{{time}}` / `{{time:LAYOUT}}` | Current time |
| `{{datetime}}` / `{{datetime:LAYOUT}}` | Date and time |
| `{{uuid}}` | Random UUID v4 |

Unknown variables are rejected at load time.

## Insertion

Controls how replacements are inserted into the target application:

```yaml
insertion:
  strategy: auto            # auto | backspace | clipboard
  clipboard_threshold: 256  # runes; use clipboard above this in "auto" mode
```

- **backspace** — remove the trigger with backspaces, then type the replacement
  keystroke by keystroke. Safe for every application, but slow for large blocks.
- **clipboard** — remove the trigger with backspaces, paste from the clipboard,
  and restore the user's clipboard afterwards.
- **auto** (default) — backspace for short replacements, clipboard for long or
  multiline ones.

## Clipboard variable

`{{clipboard}}` reads the current clipboard (text only):

```yaml
clipboard:
  enabled: false            # opt-in, disabled by default for privacy
  max_length: 4096          # runes; oversized content is truncated
```

The variable is only resolved when `clipboard.enabled` is true and a clipboard
reader is available (i.e. while the engine is running). Otherwise it expands to
an empty string.

## App-specific match sets

A rule can be restricted to (or excluded from) certain applications by the
foreground process name:

```yaml
matches:
  - trigger: ";code"
    replace: "console.log()"
    apps:
      include: ["code.exe", "windowsterminal.exe"]
      exclude: ["keepass.exe"]
```

- `apps` is optional; absent means "all applications".
- Matching is by lower-cased process base name; glob patterns are supported.
- `input.excluded_processes` still takes precedence and suppresses all
  expansion.

Filter interactively:

```powershell
gospan match list --app code.exe
gospan match test --app code.exe ";code"
```

`configs/matches/` ships ready-made dictionaries. Include only the ones you
need in your configuration:

| File | Language |
|---|---|
| `vietnamese.yaml` | Vietnamese shorthand |
| `english.yaml` | English shorthand |
| `programming.yaml` | Code snippets |
| `japanese.yaml` | Japanese |
| `chinese.yaml` | Chinese |
| `korean.yaml` | Korean |
| `german.yaml` | German |
| `multilang.yaml` | A combined starter set |

See `configs/config.multilang.example.yaml` for a configuration that includes
all of them (49 triggers). All triggers use the `;` prefix by convention, so
they rarely collide with normal typing.

## Storage

Vocabulary entries are persisted by the configured backend:

```yaml
storage:
  backend: jsonl            # jsonl (default) | sqlite
  path: data/vocabulary.jsonl
```

- **jsonl** — append-only JSON Lines. Easy to inspect, back up, and export; no
  native dependency.
- **sqlite** — a local SQLite database via the pure-Go `modernc.org/sqlite`
  driver (no CGO). Better for indexed queries, statistics, and transactional
  updates. Prefer a `.db` path, for example `data/vocabulary.db`.

The path is resolved relative to the configuration file. The backend actually in
use is reported by `gospan doctor`, and it is the backend the CLI really opens —
a value other than `jsonl`/`sqlite` is rejected at load time.

## Managing matches from the CLI

`match add` and `match remove` edit the YAML configuration directly, so the
file stays the single source of truth:

```powershell
gospan match add ";x" "example"
gospan match add ";sig" "Trân trọng,`nKhoa" --description "signature"
gospan match remove ";x"
gospan match add ";x" "example" --file configs/matches/custom.yaml
```

Safety guarantees:

- The rule is fully validated **before** the file is written. Adding a duplicate
  (including one defined in an included file) fails and leaves the file
  untouched.
- Edits preserve unrelated sections and comments; only the `matches` list changes.
- The new file is written to a temporary file in the same directory, then
  replaced atomically.
- A `.bak` backup of the previous file is kept.
- Triggers and descriptions are written quoted, so triggers with special
  characters (`#`, `*`, `123`, `true`, …) round-trip correctly. Multi-line
  replacements use a literal block scalar.

`--file <path>` targets a specific dictionary file; without it, the loaded root
configuration is edited and validated as a whole.
