package window

import (
	"fmt"
	"unsafe"
)

// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is (HANDLE)(-4)
var dpiAwarenessPerMonitorV2 = ^uintptr(3)

func EnablePerMonitorDPI() error {
	if ProcSetProcessDpiAwarenessCtx.Find() != nil {
		// Fallback for older windows: SetProcessDPIAware?
		// But prompt focused on modern/robust.
		// If func missing, likely Win7/8. Just ignore or log?
		// For robustness, we return specific error or nil if we don't care.
		return fmt.Errorf("SetProcessDpiAwarenessContext not found")
	}
	r, _, _ := ProcSetProcessDpiAwarenessCtx.Call(dpiAwarenessPerMonitorV2)
	if r == 0 {
		return fmt.Errorf("SetProcessDpiAwarenessContext failed")
	}
	return nil
}

func GetDPI(hwnd uintptr) (uint32, error) {
	// 1. Try GetDpiForWindow (Win10+)
	if ProcGetDpiForWindow.Find() == nil {
		r, _, _ := ProcGetDpiForWindow.Call(hwnd)
		if r != 0 {
			return uint32(r), nil
		}
	}

	// 2. Try GetDpiForMonitor (Win8.1+)
	hMon := MonitorFromWindow(hwnd)
	if hMon != 0 {
		dx, _, err := GetDpiForMonitor(hMon)
		if err == nil {
			return dx, nil
		}
	}

	// 3. Fallback: GetDeviceCaps (Win7/Legacy)
	// LOGPIXELSX = 88
	hdc, _, _ := ProcGetDC.Call(hwnd)
	if hdc != 0 {
		defer ProcReleaseDC.Call(hwnd, hdc)
		dpi, _, _ := ProcGetDeviceCaps.Call(hdc, 88)
		if dpi > 0 {
			return uint32(dpi), nil
		}
	}

	return 96, fmt.Errorf("cannot determine DPI")
}

func MonitorFromWindow(hwnd uintptr) uintptr {
	const MONITOR_DEFAULTTONEAREST = 2
	r, _, _ := ProcMonitorFromWindow.Call(hwnd, MONITOR_DEFAULTTONEAREST)
	return r
}

func GetDpiForMonitor(hmonitor uintptr) (dpiX, dpiY uint32, err error) {
	if ProcGetDpiForMonitor.Find() != nil {
		return 96, 96, fmt.Errorf("GetDpiForMonitor not found")
	}
	var dx, dy uint32
	// MDT_EFFECTIVE_DPI = 0
	r, _, _ := ProcGetDpiForMonitor.Call(hmonitor, 0, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy)))
	if r != 0 {
		return 96, 96, fmt.Errorf("GetDpiForMonitor failed")
	}
	return dx, dy, nil
}
