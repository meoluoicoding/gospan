package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/platform"
)

// openSearch opens a small popup that lets the user fuzzy-search the configured
// triggers and insert one with a single click or Enter.
func (u *ui) openSearch() {
	if u.svc == nil {
		u.setMessage("Chưa nạp được cấu hình.")
		return
	}

	w := u.fyneApp.NewWindow("Tìm kiếm viết tắt")
	w.Resize(fyne.NewSize(460, 420))

	entry := widget.NewEntry()
	entry.SetPlaceHolder("Nhập từ khóa (trigger hoặc nội dung)...")

	var list *widget.List
	var filtered []config.MatchConfig

	refresh := func() {
		q := strings.ToLower(strings.TrimSpace(entry.Text))
		filtered = nil
		if u.svc != nil {
			for _, m := range u.svc.Config().Matches {
				if !m.IsEnabled() {
					continue
				}
				if q == "" ||
					strings.Contains(strings.ToLower(m.Trigger), q) ||
					strings.Contains(strings.ToLower(m.Replace), q) ||
					strings.Contains(strings.ToLower(m.Description), q) {
					filtered = append(filtered, m)
				}
			}
		}
		if list != nil {
			list.Refresh()
		}
	}

	list = widget.NewList(
		func() int { return len(filtered) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.TextStyle = fyne.TextStyle{Monospace: true}
			return l
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(filtered) {
				return
			}
			o.(*widget.Label).SetText(matchLine(filtered[i]))
		},
	)

	entry.OnChanged = func(string) { refresh() }
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filtered) {
			return
		}
		u.insertMatch(filtered[id])
		w.Close()
	}
	entry.OnSubmitted = func(string) {
		if len(filtered) > 0 {
			u.insertMatch(filtered[0])
			w.Close()
		}
	}

	refresh()

	content := container.NewBorder(entry, nil, nil, nil, list)
	w.SetContent(content)
	w.CenterOnScreen()
	w.Show()
}

// insertMatch inserts (or copies) a match's replacement.
func (u *ui) insertMatch(m config.MatchConfig) {
	if u.svc == nil {
		return
	}
	text := m.Replace
	if rendered, err := u.svc.RenderTemplate(text); err == nil && rendered != "" {
		text = rendered
	}

	plat, ok := u.svc.Platform()
	if ok {
		strategy := platform.InsertBackspace
		if err := plat.ReplacePreviousText("", text, strategy); err != nil {
			_ = plat.WriteClipboard(text)
			u.setMessage("Đã copy vào clipboard — dán bằng Ctrl+V")
			return
		}
		u.setMessage("Đã chèn: " + m.Trigger)
		return
	}

	u.fyneApp.Clipboard().SetContent(text)
	u.setMessage("Đã copy vào clipboard — dán bằng Ctrl+V")
}
