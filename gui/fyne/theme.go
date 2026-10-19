package main

import (
	"image/color"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// vietnameseTheme wraps the default theme and, when a Vietnamese-capable system
// font is available, uses it for regular and bold text. Fyne's bundled font may
// lack Latin Extended Additional glyphs, which would render Vietnamese
// diacritics as boxes.
type vietnameseTheme struct {
	fallback fyne.Theme
	regular  fyne.Resource
	bold     fyne.Resource
}

// newVietnameseTheme returns the default theme with Vietnamese font support
// when a suitable system font can be loaded.
func newVietnameseTheme() fyne.Theme {
	t := vietnameseTheme{fallback: theme.DefaultTheme()}
	t.regular = loadFirstFont(
		`C:\Windows\Fonts\segoeui.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
		`C:\Windows\Fonts\tahoma.ttf`,
	)
	t.bold = loadFirstFont(
		`C:\Windows\Fonts\segoeuib.ttf`,
		`C:\Windows\Fonts\arialbd.ttf`,
		`C:\Windows\Fonts\tahomabd.ttf`,
	)
	return t
}

func loadFirstFont(paths ...string) fyne.Resource {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if r, err := fyne.LoadResourceFromPath(p); err == nil {
			return r
		}
	}
	return nil
}

func (t vietnameseTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.fallback.Color(name, variant)
}

func (t vietnameseTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.fallback.Icon(name)
}

func (t vietnameseTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.fallback.Size(name)
}

func (t vietnameseTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Monospace {
		return t.fallback.Font(style) // keep the default monospace font
	}
	if style.Bold && t.bold != nil {
		return t.bold
	}
	if t.regular != nil {
		return t.regular
	}
	return t.fallback.Font(style)
}
