//go:build windows

package windows

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID   = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
)

// processQueryLimitedInformation is the least-privileged access right that
// still allows reading the process image name.
const processQueryLimitedInformation = 0x1000

// ForegroundProcessPath returns the full path of the process that owns the
// foreground window, or an error when it cannot be determined.
func ForegroundProcessPath() (string, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return "", fmt.Errorf("no foreground window")
	}

	var pid uint32
	procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "", fmt.Errorf("no process owns the foreground window")
	}

	handle, _, err := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return "", fmt.Errorf("OpenProcess(%d) failed: %v", pid, err)
	}
	defer procCloseHandle.Call(handle)

	buf := make([]uint16, 512)
	size := uint32(len(buf))
	ok, _, err := procQueryFullProcessImageNameW.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ok == 0 {
		return "", fmt.Errorf("QueryFullProcessImageNameW failed: %v", err)
	}
	return syscall.UTF16ToString(buf[:size]), nil
}

// ForegroundProcessName returns the base name (for example "notepad.exe") of
// the foreground process, lower-cased for case-insensitive comparison.
func ForegroundProcessName() (string, error) {
	path, err := ForegroundProcessPath()
	if err != nil {
		return "", err
	}
	return strings.ToLower(filepath.Base(path)), nil
}
