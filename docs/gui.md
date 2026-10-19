# GUI (Fyne)

Gospan ships an optional desktop front end built with
[Fyne](https://fyne.io). The window doubles as the on/off switch: **keyboard
capture and text expansion run only while the app is open and enabled.**

## Why a separate module

Fyne requires **CGO** and a C toolchain on Windows. To keep the core engine
pure-Go and buildable with `CGO_ENABLED=0`, the GUI lives in its own Go module:

```text
gui/fyne/
├── go.mod        module github.com/snzpet/gospan/gui
├── main.go       window, controls, service wiring
└── theme.go      Vietnamese-capable font override
```

The GUI imports the application service (`internal/app`) and never reimplements
matching logic. The core module never imports `gui/`.

## Requirements

- Go 1.26+
- A C toolchain (`gcc`), for example via `scoop install mingw`
- `CGO_ENABLED=1`
- Windows 10/11 with OpenGL

## Build

```powershell
./scripts/build-gui.ps1
# or
cd gui/fyne
go build -trimpath -ldflags "-s -w -H=windowsgui" -o ../../dist/gospan-gui.exe .
```

The default build is a **Windows GUI application** (`-H=windowsgui`), so it opens
**without a console window**. For troubleshooting, build with a console:

```powershell
./scripts/build-gui.ps1 -Console
```

## Run

```powershell
./dist/gospan-gui.exe
```

**First run:** if no configuration exists yet at the default path, the app
creates a starter configuration automatically and says so. It also never
overwrites an existing file.

1. The app loads the configuration from `%APPDATA%\gospan\config.yaml` (or the
   file chosen with **Chọn cấu hình**).
2. Press **Bật viết tắt** to install the keyboard hook and start expanding.
3. Press **Tắt viết tắt** (or use the tray menu) to stop capture completely.

The status label shows `● Đang bật` / `○ Đã tắt`. The trigger list reflects the
active configuration; **Nạp lại** re-reads the file and restores the previous
running state.

## System tray

Gospan adds a tray icon with a menu:

| Item | Action |
|---|---|
| Hiện cửa sổ | Show and focus the window |
| Tìm kiếm viết tắt | Open the search popup |
| Bật viết tắt / Tắt viết tắt | Toggle capture (label follows state) |
| Thoát | Quit completely |

Left-clicking the tray icon shows the window. Closing the window with **X**
**hides it to the tray** instead of quitting, so expansion keeps running in the
background. Quit for real from the tray menu (this also stops capture).

If the platform has no tray support, closing the window quits normally.

## Behaviour and safety

- Capture is installed only on **Bật**; nothing listens to the keyboard while the
  app is off.
- Closing the window hides it to the tray (when supported); capture keeps
  running. Quit from the tray menu to stop.
- Quitting the app stops capture (`Service.Stop`).
- A configuration that fails validation is reported in the message area and the
  toggle stays disabled.
- Logs go to `%APPDATA%\gospan\logs\gospan.log` and never contain typed content.

## Vietnamese text

Fyne's bundled font may not include Latin Extended Additional glyphs. The GUI
installs a theme that substitutes a Vietnamese-capable system font (Segoe UI,
Arial, or Tahoma) when one is found, so diacritics render correctly.

## Known limitations

- The toggle blocks briefly (well under a second) while the hook is installed.
- Capture cannot reach elevated or secure-desktop windows; see
  [`windows-integration.md`](windows-integration.md).
- Insertion of very large multi-line replacements is still SendInput-based and
  not yet clipboard-assisted.

## Troubleshooting

**"Cấu hình không hợp lệ" / "Không tìm thấy file cấu hình"** — the file at the
shown path is missing or invalid. Press **Tạo cấu hình mẫu** to create a starter
at that path, or **Chọn cấu hình** to pick another file. Note that a
configuration using `include:` needs those included files to exist next to it.

**Nothing expands** — press **Bật viết tắt**; capture is off until then. Check
the status shows `● Đang bật`.
