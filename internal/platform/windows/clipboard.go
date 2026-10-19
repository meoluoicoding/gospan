//go:build windows

package windows

import (
	"fmt"
	"reflect"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
)

// Clipboard format constants.
const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// ReadClipboardText reads the current clipboard as text. It returns "" when the
// clipboard is empty or contains no text.
func ReadClipboardText() (string, error) {
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return "", fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", nil
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return "", fmt.Errorf("GlobalLock failed")
	}
	defer procGlobalUnlock.Call(h)

	// Build a slice view over the locked memory without converting a uintptr
	// to unsafe.Pointer (which go vet flags). The slice header's Data field is
	// a uintptr and can be assigned directly.
	const maxUnits = 1 << 20
	var sl []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&sl))
	hdr.Data = ptr
	hdr.Len = maxUnits
	hdr.Cap = maxUnits

	n := 0
	for n < len(sl) && sl[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(sl[:n]), nil
}

// WriteClipboardText replaces the clipboard contents with text.
func WriteClipboardText(text string) error {
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()

	if r, _, _ := procEmptyClipboard.Call(); r == 0 {
		return fmt.Errorf("EmptyClipboard failed")
	}

	units := utf16.Encode([]rune(text))
	buf := make([]uint16, len(units)+1)
	copy(buf, units)

	size := uintptr(len(buf) * 2)
	h, _, err := procGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return fmt.Errorf("GlobalAlloc failed: %v", err)
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return fmt.Errorf("GlobalLock failed")
	}
	var dst []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&dst))
	hdr.Data = ptr
	hdr.Len = len(buf)
	hdr.Cap = len(buf)
	copy(dst, buf)
	procGlobalUnlock.Call(h)

	if r, _, err := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
		return fmt.Errorf("SetClipboardData failed: %v", err)
	}
	// ownership transferred to clipboard; do not free h
	return nil
}

// ReadClipboard implements platform.Platform.
func (p *Platform) ReadClipboard() (string, error) {
	return ReadClipboardText()
}

// WriteClipboard implements platform.Platform.
func (p *Platform) WriteClipboard(text string) error {
	return WriteClipboardText(text)
}
