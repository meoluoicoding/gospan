//go:build windows

package windows

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/snzpet/gospan/internal/platform"
)

var procSendInput = user32.NewProc("SendInput")

// SendInput constants.
const (
	inputKeyboard    = 1
	keyeventfKeyUp   = 0x0002
	keyeventfUnicode = 0x0004
	vkBack           = 0x08
	vkControl        = 0x11
	vkLeft           = 0x25
)

// keybdInput mirrors the Win32 KEYBDINPUT structure.
type keyboardInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// winInput mirrors the Win32 INPUT structure. The trailing padding keeps the
// union the size of MOUSEINPUT so the structure matches the C layout on amd64.
type winInput struct {
	inputType uint32
	_         uint32
	ki        keyboardInput
	_         [8]byte
}

func keyEvent(vk, scan, flags uint32) winInput {
	return winInput{
		inputType: inputKeyboard,
		ki: keyboardInput{
			wVk:         uint16(vk),
			wScan:       uint16(scan),
			dwFlags:     flags,
			dwExtraInfo: injectedMagic,
		},
	}
}

func (p *Platform) sendInputs(inputs []winInput) error {
	if len(inputs) == 0 {
		return nil
	}
	n, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(winInput{}),
	)
	if int(n) != len(inputs) {
		return fmt.Errorf("SendInput inserted %d of %d events: %v", n, len(inputs), err)
	}
	return nil
}

// ReplacePreviousText deletes len([]rune(typed)) characters with backspaces and
// then inserts replacement using the requested strategy.
func (p *Platform) ReplacePreviousText(typed, replacement string, strategy platform.InsertionStrategy) error {
	switch strategy {
	case platform.InsertClipboard:
		return p.insertViaClipboard(typed, replacement)
	case platform.InsertBackspace:
		return p.insertViaBackspace(typed, replacement)
	default: // InsertAuto
		if len([]rune(replacement)) > 0 && (len([]rune(replacement)) >= 256 || strings.Contains(replacement, "\n")) {
			if err := p.insertViaClipboard(typed, replacement); err == nil {
				return nil
			}
		}
		return p.insertViaBackspace(typed, replacement)
	}
}

// MoveCursorLeft moves the caret left by n positions.
func (p *Platform) MoveCursorLeft(n int) error {
	if n <= 0 {
		return nil
	}
	inputs := make([]winInput, 0, n*2)
	for i := 0; i < n; i++ {
		inputs = append(inputs, keyEvent(vkLeft, 0, 0))
		inputs = append(inputs, keyEvent(vkLeft, 0, keyeventfKeyUp))
	}
	return p.sendInputs(inputs)
}

// replacement keystroke by keystroke. This is the dependency-free strategy.
func (p *Platform) insertViaBackspace(typed, replacement string) error {
	var inputs []winInput

	for i := 0; i < len([]rune(typed)); i++ {
		inputs = append(inputs, keyEvent(vkBack, 0, 0))
		inputs = append(inputs, keyEvent(vkBack, 0, keyeventfKeyUp))
	}

	for _, unit := range utf16.Encode([]rune(replacement)) {
		inputs = append(inputs, keyEvent(0, uint32(unit), keyeventfUnicode))
		inputs = append(inputs, keyEvent(0, uint32(unit), keyeventfUnicode|keyeventfKeyUp))
	}

	return p.sendInputs(inputs)
}

// insertViaClipboard removes the trigger with backspaces and pastes the
// replacement from the clipboard, saving and restoring the user's clipboard.
func (p *Platform) insertViaClipboard(typed, replacement string) error {
	saved, _ := ReadClipboardText()

	var inputs []winInput
	for i := 0; i < len([]rune(typed)); i++ {
		inputs = append(inputs, keyEvent(vkBack, 0, 0))
		inputs = append(inputs, keyEvent(vkBack, 0, keyeventfKeyUp))
	}
	if err := p.sendInputs(inputs); err != nil {
		return err
	}

	if err := WriteClipboardText(replacement); err != nil {
		_ = WriteClipboardText(saved)
		return err
	}

	paste := []winInput{
		keyEvent(vkControl, 0, 0),
		keyEvent(0x56, 0, 0), // 'V'
		keyEvent(0x56, 0, keyeventfKeyUp),
		keyEvent(vkControl, 0, keyeventfKeyUp),
	}
	if err := p.sendInputs(paste); err != nil {
		_ = WriteClipboardText(saved)
		return err
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = WriteClipboardText(saved)
	}()
	return nil
}
