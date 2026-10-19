package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
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
	statusDot    *canvas.Circle
	toggleBtn    *widget.Button
	pathLabel    *widget.Label
	countLabel   *widget.Label
	configStatus *widget.Label
	messageLabel *widget.Label
	matchList    *widget.List
	matches      []config.MatchConfig

	trayMenu   *fyne.Menu
	trayToggle *fyne.MenuItem
	pollCancel context.CancelFunc

	pathFromUser bool

	// Notification tracking to avoid spamming the OS notification area.
	lastNotifiedCount int
	lastNotifiedErr   error
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

	w.Resize(fyne.NewSize(780, 600))
	w.CenterOnScreen()

	// Keyboard shortcuts.
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		u.openSearch()
	})
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		u.reload(true)
	})
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyB, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		u.onToggle()
	})
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		u.onAddMatch()
	})

	w.ShowAndRun()

	u.stopConfigPoller()
	if u.svc != nil {
		_ = u.svc.Stop()
	}
}

func (u *ui) build() {
	// --- Status row ---
	u.statusDot = canvas.NewCircle(theme.DisabledColor())
	u.statusDot.Resize(fyne.NewSize(12, 12))
	u.statusLabel = widget.NewLabel("○ Đã tắt")
	u.statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	u.toggleBtn = widget.NewButtonWithIcon("Bật viết tắt (Ctrl+B)", theme.MediaPlayIcon(), u.onToggle)
	u.toggleBtn.Importance = widget.HighImportance

	statusRow := container.NewHBox(u.statusDot, u.statusLabel, u.toggleBtn)

	// --- Config row ---
	u.pathLabel = widget.NewLabel(u.path)
	u.pathLabel.Wrapping = fyne.TextTruncate

	u.countLabel = widget.NewLabel("")
	u.configStatus = widget.NewLabel("")

	chooseBtn := widget.NewButtonWithIcon("Chọn", theme.FolderOpenIcon(), u.onChoose)
	searchBtn := widget.NewButtonWithIcon("Tìm kiếm", theme.SearchIcon(), u.openSearch)
	starterBtn := widget.NewButtonWithIcon("Tạo mẫu", theme.DocumentCreateIcon(), u.onCreateStarter)
	reloadBtn := widget.NewButtonWithIcon("Nạp lại", theme.ViewRefreshIcon(), u.onReload)
	addBtn := widget.NewButtonWithIcon("Thêm", theme.ContentAddIcon(), u.onAddMatch)

	configRow := container.NewHBox(widget.NewLabel("Cấu hình:"), u.pathLabel)
	actionRow := container.NewHBox(chooseBtn, searchBtn, starterBtn, reloadBtn, addBtn, u.countLabel)
	statusRow2 := container.NewHBox(widget.NewLabel("Trạng thái:"), u.configStatus)

	// --- Match list ---
	u.matchList = widget.NewList(
		func() int { return len(u.matches) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.TextStyle = fyne.TextStyle{Monospace: true}
			return l
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(u.matches) {
				return
			}
			o.(*widget.Label).SetText(matchLine(u.matches[i]))
		},
	)

	// Empty state
	emptyLabel := widget.NewLabel("Chưa có trigger nào. Nhấn 'Thêm' hoặc 'Tạo mẫu'.")
	emptyLabel.Alignment = fyne.TextAlignCenter

	listContainer := container.NewStack(u.matchList)

	// --- Footer / message ---
	u.messageLabel = widget.NewLabel("")
	u.messageLabel.Wrapping = fyne.TextWrapWord

	footer := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel("Mẹo:"), widget.NewLabel("Ctrl+F tìm kiếm · Ctrl+R nạp lại · Ctrl+B bật/tắt · Ctrl+N thêm trigger")),
		u.messageLabel,
	)

	header := container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("Gospan", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("— trình gõ tắt"),
		),
		widget.NewSeparator(),
		statusRow,
		configRow,
		actionRow,
		statusRow2,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Danh sách viết tắt", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	u.window.SetContent(container.NewBorder(header, footer, nil, nil, listContainer))
}

// onToggle starts or stops capture.
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
		u.statusDot.FillColor = colorGreen()
		u.toggleBtn.SetText("Tắt viết tắt (Ctrl+B)")
		u.toggleBtn.SetIcon(theme.MediaPauseIcon())
		u.window.SetTitle("Gospan — ● Đang bật")
	} else {
		u.statusLabel.SetText("○ Đã tắt")
		u.statusDot.FillColor = theme.DisabledColor()
		u.toggleBtn.SetText("Bật viết tắt (Ctrl+B)")
		u.toggleBtn.SetIcon(theme.MediaPlayIcon())
		u.window.SetTitle("Gospan — ○ Đã tắt")
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
	u.statusDot.Refresh()
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

func (u *ui) onAddMatch() {
	if u.svc == nil {
		u.setMessage("Chưa nạp được cấu hình.")
		return
	}

	triggerEntry := widget.NewEntry()
	triggerEntry.SetPlaceHolder(";kt")
	replaceEntry := widget.NewMultiLineEntry()
	replaceEntry.SetPlaceHolder("kiểm tra")
	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("mô tả (tùy chọn)")

	form := container.NewVBox(
		widget.NewLabel("Trigger:"),
		triggerEntry,
		widget.NewLabel("Replacement:"),
		replaceEntry,
		widget.NewLabel("Description:"),
		descEntry,
	)

	d := dialog.NewCustomConfirm("Thêm trigger", "Thêm", "Hủy", form, func(ok bool) {
		if !ok {
			return
		}
		trig := strings.TrimSpace(triggerEntry.Text)
		repl := replaceEntry.Text
		if trig == "" || repl == "" {
			dialog.ShowError(fmt.Errorf("trigger và replacement không được rỗng"), u.window)
			return
		}
		editor := config.NewEditor(u.svc.Config().App.ConfigPath, u.svc.Config().App.ConfigPath)
		if err := editor.AddMatch(trig, repl, strings.TrimSpace(descEntry.Text)); err != nil {
			dialog.ShowError(err, u.window)
			u.setMessage("Thêm thất bại: " + err.Error())
			return
		}
		u.reload(false)
		u.setMessage(fmt.Sprintf("Đã thêm %s → %s", trig, firstLine(repl)))
	}, u.window)
	d.Resize(fyne.NewSize(400, 320))
	d.Show()
}

func (u *ui) onReload() {
	u.reload(true)
}

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
	u.notifyConfigLoaded(svc.TriggerCount())
}

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
						if err != u.lastNotifiedErr {
							u.lastNotifiedErr = err
							u.notify("Gospan — Lỗi cấu hình", err.Error())
						}
						return
					}
					if loaded {
						u.configStatus.SetText(fmt.Sprintf("✓ Đã nạp (%d trigger)", count))
					} else {
						u.configStatus.SetText("✗ Chưa nạp config")
					}
					if count != u.lastNotifiedCount || u.lastNotifiedErr != nil {
						u.lastNotifiedCount = count
						u.lastNotifiedErr = nil
						u.notify("Gospan", fmt.Sprintf("Đã nạp cấu hình — %d trigger.", count))
					}
				})
			}
		}
	}()
}

// notify sends an OS-level notification.
func (u *ui) notify(title, body string) {
	if u.fyneApp == nil {
		return
	}
	u.fyneApp.SendNotification(&fyne.Notification{
		Title:   title,
		Content: body,
	})
}

// notifyConfigLoaded fires a notification for the initial load.
func (u *ui) notifyConfigLoaded(count int) {
	u.lastNotifiedCount = count
	u.lastNotifiedErr = nil
	u.notify("Gospan", fmt.Sprintf("Đã nạp cấu hình — %d trigger.", count))
}

func (u *ui) stopConfigPoller() {
	if u.pollCancel != nil {
		u.pollCancel()
		u.pollCancel = nil
	}
}

func (u *ui) setMessage(s string) {
	u.messageLabel.SetText(s)
}

func (u *ui) setupTray() {
	desk, ok := u.fyneApp.(desktop.App)
	if !ok {
		return
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

func colorGreen() color.Color {
	return color.RGBA{R: 34, G: 197, B: 94, A: 255}
}
