# Windows Integration

Gospan targets Windows 10/11 x64 first. All Windows code lives in
`internal/platform/windows` behind `//go:build windows` and the
`platform.Platform` interface, so the core engine stays portable.

## Keyboard capture

- `SetWindowsHookExW(WH_KEYBOARD_LL, ...)` installs a low-level keyboard hook.
- The hook is installed and pumped on a dedicated, `runtime.LockOSThread`-ed
  goroutine that runs the message loop.
- The callback (`hookProc`) does the minimum work: translate the key to a rune
  via `ToUnicodeEx`, then push to a bounded channel. It never blocks, never
  parses YAML, and never touches storage.
- If the channel is full, events are dropped rather than blocking the hook.

## Text insertion

- `SendInput` performs backspaces to remove the trigger and then types the
  replacement as Unicode (`KEYEVENTF_UNICODE`).
- Injected events carry a magic `dwExtraInfo` tag and are filtered by the hook,
  preventing expansion loops.
- `LLKHF_INJECTED` events are ignored by default (`input.ignore_injected_events`).

## Process exclusion

`ForegroundProcessPath` / `ForegroundProcessName` resolve the process owning the
foreground window using `GetForegroundWindow` +
`QueryFullProcessImageNameW`. Matching processes in `input.excluded_processes`
suppress expansion.

## Known limitations

- Secure desktop and elevated applications may not receive injected input.
- IME composition, dead keys, and surrogate pairs need per-application testing.
- Application exclusion is best-effort, not a security boundary.
- Clipboard-assisted insertion for very large replacements is not yet
  implemented.

The compatibility matrix (Notepad, Windows Terminal, browsers, VS Code, Word,
Vietnamese IMEs) is defined in [`../architect.md`](../architect.md) §15.3.
