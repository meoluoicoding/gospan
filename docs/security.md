# Security and Privacy

Gospan is a global keyboard utility, so it treats input handling carefully. This
document summarizes the current posture; see [`../architect.md`](../architect.md)
§16 for the threat model.

## Principles

- **Local-first.** No telemetry, no network calls, no account.
- **No typed content in logs.** Logging is structured and excludes keystrokes.
- **No arbitrary execution.** Replacement templates render a fixed set of
  variables; they never execute shell commands or evaluate expressions.
- **Learning is opt-in.** `suggestions.learn_from_typing` defaults to false.
  When enabled, only phrase occurrence counts are collected — never full
  keystroke histories or complete sentences.
- **Explicit acceptance.** Observed phrases become *candidates* that the user
  must accept; they are never auto-activated.

## Resource limits

- `engine.max_trigger_length` (hard cap 1024) and `engine.max_expansion_length`
  (hard cap 1 MiB) bound rule sizes.
- The input buffer is bounded (`input.buffer_size`).
- The event channel is bounded; overflow drops events instead of blocking.

## Configuration safety

- All rules are validated before activation.
- Reload is atomic: a failed reload keeps the last valid configuration.
- The user's configuration is never silently overwritten.

## Controls

- Disable expansion globally by setting `engine.enabled: false`.
- Clear learned statistics via the vocabulary service (`Collector.Reset`).
- Exclude sensitive processes with `input.excluded_processes`.

## Honest limitations

- Application-based exclusion is best-effort and cannot be a security boundary;
  a determined or unusual application may evade detection.
- The safest configuration for sensitive environments is to keep learning off
  and, where necessary, disable global capture entirely.
- Gospan does not attempt to bypass operating-system security boundaries.
