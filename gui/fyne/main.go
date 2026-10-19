// Command gospan-gui is the Fyne desktop front end for Gospan.
//
// It lives in its own Go module (gui/fyne) so the core engine keeps its
// CGO-free, pure-Go build. Fyne requires CGO and a C toolchain on Windows; see
// docs/windows-integration.md and ADR-008.
//
// The window is also the on/off switch: keyboard capture and expansion only run
// while the user has pressed "Bật viết tắt".
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	gospanapp "github.com/snzpet/gospan/internal/app"
	"github.com/snzpet/gospan/internal/config"
)

type ui struct {
	fyneApp fyne.App
	window  fyne.Window

	svc  *gospanapp.Service
	path string

	statusLabel  *widget.Label
	toggleBtn    *widget.Button
	pathLabel    *widget.Label
	countLabel   *widget.Label
	messageLabel *widget.Label
	matchList    *widget.List

	configStatus *widget.Label
	pollCancel   context.CancelFunc

	pathFromUser bool

	// System tray, when the platform supports it.
	trayMenu   *fyne.Menu
	trayToggle *fyne.MenuItem

	matches []config.MatchConfig
}

func main() {
	a := app.NewWithID("com.snzpet.gospan")
	a.Settings().SetTheme(newVietnameseTheme())
	a.SetIcon(appIcon())

	w := a.NewWindow("Gospan — Trình gõ tắt")

	u := &ui{
		fyneApp: a,
		window:  w,
		path:    config.ResolvePath(""),
	}
	u.build()
	u.setupTray()
	u.load()

	w.Resize(fyne.NewSize(760, 560))
	w.CenterOnScreen()
	w.ShowAndRun()

	u.stopConfigPoller()
	if u.svc != nil {
		_ = u.svc.Stop()
	}
}

// setupTray installs a system tray icon and menu. When the tray is available,
// closing the window hides it instead of quitting so capture can keep running.
func (u *ui) setupTray() {
	desk, ok := u.fyneApp.(desktop.App)
	if !ok {
		return // platform without tray support: closing quits normally
	}

	show := fyne.NewMenuItem("Hiện cửa sổ", func() {
		u.window.Show()
		u.window.RequestFocus()
	})
	search := fyne.NewMenuItem("Tìm kiếm viết tắt", func() { u.openSearch() })

	u.trayToggle = fyne.NewMenuItem("Bật viết tắt", u.onToggle)

	quit := fyne.NewMenuItem("Thoát", nil)
	quit.IsQuit = true

	u.trayMenu = fyne.NewMenu("Gospan", show, search, u.trayToggle, fyne.NewMenuItemSeparator(), quit)
	desk.SetSystemTrayMenu(u.trayMenu)
	desk.SetSystemTrayIcon(appIcon())
	desk.SetSystemTrayWindow(u.window)

	u.window.SetCloseIntercept(func() {
		u.window.Hide()
	})
}

func (u *ui) build() {
	u.statusLabel = widget.NewLabel("○ Đã tắt")
	u.statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	u.toggleBtn = widget.NewButtonWithIcon("Bật viết tắt", theme.MediaPlayIcon(), u.onToggle)
	u.toggleBtn.Importance = widget.HighImportance

	u.pathLabel = widget.NewLabel(u.path)
	u.pathLabel.Wrapping = fyne.TextTruncate

	u.countLabel = widget.NewLabel("")
	u.messageLabel = widget.NewLabel("")
	u.messageLabel.Wrapping = fyne.TextWrapWord

	u.configStatus = widget.NewLabel("")

	chooseBtn := widget.NewButtonWithIcon("Chọn cấu hình", theme.FolderOpenIcon(), u.onChoose)
	searchBtn := widget.NewButtonWithIcon("Tìm kiếm", theme.SearchIcon(), u.openSearch)
	starterBtn := widget.NewButtonWithIcon("Tạo cấu hình mẫu", theme.DocumentCreateIcon(), u.onCreateStarter)
	reloadBtn := widget.NewButtonWithIcon("Nạp lại", theme.ViewRefreshIcon(), u.onReload)

	u.matchList = widget.NewList(
		func() int { return len(u.matches) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.TextStyle = fyne.TextStyle{Monospace: true}
			return l
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(matchLine(u.matches[i]))
		},
	)

	header := container.NewVBox(
		widget.NewLabelWithStyle("Gospan", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Bật cửa sổ này để kích hoạt gõ tắt toàn hệ thống."),
		widget.NewSeparator(),
		container.NewHBox(u.toggleBtn, u.statusLabel),
		container.NewHBox(widget.NewLabel("Cấu hình:"), u.pathLabel),
		container.NewHBox(chooseBtn, searchBtn, starterBtn, reloadBtn, u.countLabel),
		container.NewHBox(widget.NewLabel("Trạng thái config:"), u.configStatus),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Danh sách viết tắt", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	footer := container.NewVBox(
		widget.NewSeparator(),
		u.messageLabel,
		widget.NewLabel("Thử gõ:  ;dc   ;kt   ;brse   ;date"),
	)

	u.window.SetContent(container.NewBorder(header, footer, nil, nil, u.matchList))
}

// onToggle starts or stops capture. The feature is only active while running.
func (u *ui) onToggle() {
	if u.svc == nil {
		u.setMessage("Chưa nạp được cấu hình. Hãy chọn một file cấu hình hợp lệ.")
		return
	}

	if u.svc.Running() {
		if err := u.svc.Stop(); err != nil {
			u.setMessage("Lỗi khi tắt: " + err.Error())
			return
		}
		u.setRunning(false)
		u.setMessage("Đã tắt tính năng viết tắt.")
		return
	}

	u.toggleBtn.Disable()
	err := u.svc.Start()
	u.toggleBtn.Enable()
	if err != nil {
		dialog.ShowError(err, u.window)
		u.setMessage("Không bật được: " + err.Error())
		u.setRunning(false)
		return
	}
	u.setRunning(true)
	u.setMessage(fmt.Sprintf("Đang bật — %d trigger hoạt động. Nhấn Tắt để dừng.", u.svc.TriggerCount()))
}

func (u *ui) setRunning(running bool) {
	if running {
		u.statusLabel.SetText("● Đang bật")
		u.toggleBtn.SetText("Tắt viết tắt")
		u.toggleBtn.SetIcon(theme.MediaPauseIcon())
	} else {
		u.statusLabel.SetText("○ Đã tắt")
		u.toggleBtn.SetText("Bật viết tắt")
		u.toggleBtn.SetIcon(theme.MediaPlayIcon())
	}

	if u.trayToggle != nil {
		if running {
			u.trayToggle.Label = "Tắt viết tắt"
		} else {
			u.trayToggle.Label = "Bật viết tắt"
		}
		if u.trayMenu != nil {
			u.trayMenu.Refresh()
		}
	}
}

func (u *ui) onChoose() {
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil || rc == nil {
			return
		}
		p := rc.URI().Path()
		_ = rc.Close()
		u.path = p
		u.pathFromUser = true
		u.reload(true)
	}, u.window)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
	if dir, err := storage.ListerForURI(storage.NewFileURI(filepath.Dir(u.path))); err == nil {
		d.SetLocation(dir)
	}
	d.Show()
}

func (u *ui) onReload() {
	u.reload(true)
}

// reload rebuilds the service from the current path, preserving the running
// state when possible.
func (u *ui) reload(restart bool) {
	wasRunning := u.svc != nil && u.svc.Running()
	if u.svc != nil {
		_ = u.svc.Stop()
	}

	u.load()

	if restart && wasRunning && u.svc != nil {
		if err := u.svc.Start(); err == nil {
			u.setRunning(true)
			return
		}
		u.setMessage("Đã nạp lại nhưng không tự bật lại được.")
	}
}

// load opens the service from u.path and refreshes the UI. On a first run with
// the default path, a starter configuration is created automatically.
func (u *ui) load() {
	svc, err := gospanapp.Open(u.path, newLogger())
	created := false

	if err != nil && !u.pathFromUser && config.IsNotExist(err) {
		if werr := config.WriteStarter(u.path); werr == nil {
			if s2, e2 := gospanapp.Open(u.path, newLogger()); e2 == nil {
				svc, err, created = s2, nil, true
			}
		}
	}

	if err != nil {
		u.showLoadError(err)
		return
	}

	u.applyService(svc)
	if created {
		u.setMessage("Đã tạo cấu hình mẫu tại " + svc.Config().App.ConfigPath + ". Nhấn Bật để bắt đầu gõ tắt.")
	} else {
		u.setMessage("Cấu hình đã nạp. Nhấn Bật để kích hoạt viết tắt.")
	}
}

// startConfigPoller surfaces background reload results in the UI.
func (u *ui) startConfigPoller() {
	if u.pollCancel != nil {
		u.pollCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	u.pollCancel = cancel
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if u.svc == nil {
					continue
				}
				loaded := u.svc.ConfigLoaded()
				err := u.svc.LastReloadError()
				count := u.svc.TriggerCount()
				fyne.Do(func() {
					if err != nil {
						u.configStatus.SetText("⚠ Reload lỗi: " + err.Error())
						return
					}
					if loaded {
						u.configStatus.SetText(fmt.Sprintf("✓ Đã nạp (%d trigger)", count))
					} else {
						u.configStatus.SetText("✗ Chưa nạp config")
					}
				})
			}
		}
	}()
}

// stopConfigPoller stops the background poller.
func (u *ui) stopConfigPoller() {
	if u.pollCancel != nil {
		u.pollCancel()
		u.pollCancel = nil
	}
}

// applyService installs a loaded service into the UI.
func (u *ui) applyService(svc *gospanapp.Service) {
	u.svc = svc
	u.matches = svc.Config().Matches
	u.matchList.Refresh()
	u.pathLabel.SetText(svc.Config().App.ConfigPath)
	u.countLabel.SetText(fmt.Sprintf("%d trigger", svc.TriggerCount()))
	if svc.ConfigLoaded() {
		u.toggleBtn.Enable()
		u.configStatus.SetText(fmt.Sprintf("✓ Đã nạp (%d trigger)", svc.TriggerCount()))
	} else {
		u.toggleBtn.Disable()
		u.configStatus.SetText("✗ Chưa nạp config")
	}
	u.setRunning(svc.Running())
	u.startConfigPoller()
}

// showLoadError clears the trigger view and explains the failure.
func (u *ui) showLoadError(err error) {
	u.svc = nil
	u.matches = nil
	u.matchList.Refresh()
	u.pathLabel.SetText(u.path)
	u.countLabel.SetText("")
	u.toggleBtn.Disable()
	u.setRunning(false)
	u.configStatus.SetText("✗ Chưa nạp config")

	if config.IsNotExist(err) {
		u.setMessage("Không tìm thấy file cấu hình: " + u.path + " — nhấn 'Tạo cấu hình mẫu'.")
		return
	}
	u.setMessage("Cấu hình không hợp lệ: " + err.Error())
}

// onCreateStarter writes a starter configuration to the current path.
func (u *ui) onCreateStarter() {
	target := u.path
	if err := config.WriteStarter(target); err != nil {
		dialog.ShowError(err, u.window)
		u.setMessage(err.Error())
		return
	}
	u.pathFromUser = false
	u.reload(false)
	u.setMessage("Đã tạo cấu hình mẫu tại " + target + ". Nhấn Bật để bắt đầu gõ tắt.")
}

func (u *ui) setMessage(s string) {
	u.messageLabel.SetText(s)
}

// newLogger writes structured logs to the user log directory, falling back to
// stderr. Typed content is never logged.
func newLogger() *slog.Logger {
	dir := filepath.Join(config.DefaultDir(), "logs")
	if err := os.MkdirAll(dir, 0o755); err == nil {
		path := filepath.Join(dir, "gospan.log")
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
		}
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func matchLine(m config.MatchConfig) string {
	trigger := m.Trigger
	if !m.IsEnabled() {
		trigger += " (tắt)"
	}
	return fmt.Sprintf("%-18s → %s", trigger, firstLine(m.Replace))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + "…"
	}
	return s
}
