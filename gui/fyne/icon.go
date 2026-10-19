package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/gospan.png
var iconPNG []byte

// appIconResource is the window and system-tray icon.
var appIconResource = fyne.NewStaticResource("gospan.png", iconPNG)

func appIcon() fyne.Resource {
	return appIconResource
}
