//go:build windows

// Package windows implements the Windows platform integration: low-level
// keyboard capture and text insertion via the Win32 API. All code here is
// guarded by the windows build tag.
package windows

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/snzpet/gospan/internal/platform"
)

// Win32 constants.
const (
	whKeyboardLL = 13

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105
	wmQuit       = 0x0012

	hcAction = 0

	llkhfInjected = 0x00000010
)

// injectedMagic tags events synthesised by Gospan so they can be ignored.
const injectedMagic = 0x606F7370 // "gosp"

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procGetKeyboardState    = user32.NewProc("GetKeyboardState")
	procToUnicodeEx         = user32.NewProc("ToUnicodeEx")
	procGetKeyboardLayout   = user32.NewProc("GetKeyboardLayout")
	procGetCurrentThreadID  = kernel32.NewProc("GetCurrentThreadId")
)

type kbdLLHookStruct struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

// Platform is the Windows implementation of platform.Platform.
type Platform struct {
	events chan platform.KeyboardEvent

	ignoreInjected bool
	eventQueueSize int

	mu       sync.Mutex
	hook     uintptr
	threadID uint32
	callback uintptr
	started  bool
	stopOnce sync.Once
}

// New returns a Windows platform integration. It does not start capturing until
// Start is called.
func New() *Platform {
	return &Platform{
		events:         make(chan platform.KeyboardEvent, 1024),
		ignoreInjected: true,
	}
}

// SetIgnoreInjected controls whether events synthesised by other applications
// (including this one) are filtered out.
func (p *Platform) SetIgnoreInjected(ignore bool) {
	p.ignoreInjected = ignore
}

// Events returns the captured event channel.
func (p *Platform) Events() <-chan platform.KeyboardEvent {
	return p.events
}

// ForegroundProcessName returns the lower-cased base name of the process that
// owns the foreground window. It implements the optional process-namer role
// used for application exclusion.
func (p *Platform) ForegroundProcessName() (string, error) {
	return ForegroundProcessName()
}

// Start installs the low-level keyboard hook on a dedicated OS thread and pumps
// its message loop there.
func (p *Platform) Start() error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return fmt.Errorf("windows platform already started")
	}
	p.mu.Unlock()

	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		threadID, _, _ := procGetCurrentThreadID.Call()

		p.callback = syscall.NewCallback(p.hookProc)
		hook, _, err := procSetWindowsHookExW.Call(
			uintptr(whKeyboardLL),
			p.callback,
			0, // hMod: 0 is valid for WH_KEYBOARD_LL
			0, // dwThreadId: 0 attaches to all threads
		)
		if hook == 0 {
			ready <- fmt.Errorf("SetWindowsHookExW failed: %v", err)
			return
		}

		p.mu.Lock()
		p.hook = hook
		p.threadID = uint32(threadID)
		p.started = true
		p.mu.Unlock()

		ready <- nil

		var m msg
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(ret) <= 0 { // WM_QUIT or error
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}

		p.mu.Lock()
		if p.hook != 0 {
			procUnhookWindowsHookEx.Call(p.hook)
			p.hook = 0
		}
		p.started = false
		p.mu.Unlock()
		close(p.events)
	}()

	return <-ready
}

// Stop unhooks and terminates the message loop.
func (p *Platform) Stop() error {
	var stopErr error
	p.stopOnce.Do(func() {
		p.mu.Lock()
		threadID := p.threadID
		p.mu.Unlock()
		if threadID != 0 {
			procPostThreadMessageW.Call(uintptr(threadID), uintptr(wmQuit), 0, 0)
		} else {
			stopErr = fmt.Errorf("windows platform is not running")
		}
	})
	return stopErr
}

// hookProc is the low-level keyboard hook callback. It must stay short: it
// translates the key to a rune and forwards the event without blocking.
//
// lParam is typed as unsafe.Pointer rather than uintptr so the pointer
// conversion is direct; this avoids a uintptr-to-Pointer conversion, which the
// vet unsafeptr check (correctly) flags elsewhere.
func (p *Platform) hookProc(nCode int32, wParam uintptr, lParam unsafe.Pointer) uintptr {
	if nCode == hcAction {
		if kb := (*kbdLLHookStruct)(lParam); kb != nil {
			p.emit(kb, wParam)
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, uintptr(lParam))
	return ret
}

func (p *Platform) emit(kb *kbdLLHookStruct, wParam uintptr) {
	injected := kb.flags&llkhfInjected != 0 || kb.dwExtraInfo == injectedMagic
	if injected && p.ignoreInjected {
		return
	}

	down := wParam == wmKeyDown || wParam == wmSysKeyDown

	ev := platform.KeyboardEvent{
		KeyCode:   kb.vkCode,
		IsDown:    down,
		Timestamp: int64(kb.time),
		Injected:  injected,
	}
	if down {
		if r, ok := keyToRune(kb.vkCode, kb.scanCode); ok {
			ev.Rune = r
		}
	}

	select {
	case p.events <- ev:
	default: // bounded queue: drop events rather than block the hook
	}
}

// keyToRune translates a virtual key + scan code into a rune using the current
// keyboard layout. It returns false for non-character keys.
func keyToRune(vkCode, scanCode uint32) (rune, bool) {
	var state [256]byte
	ok, _, _ := procGetKeyboardState.Call(uintptr(unsafe.Pointer(&state[0])))
	if ok == 0 {
		return 0, false
	}

	layout, _, _ := procGetKeyboardLayout.Call(0) // current thread layout

	var buf [8]uint16
	ret, _, _ := procToUnicodeEx.Call(
		uintptr(vkCode),
		uintptr(scanCode),
		uintptr(unsafe.Pointer(&state[0])),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
		layout,
	)
	if int32(ret) <= 0 {
		return 0, false
	}
	return rune(buf[0]), true
}
