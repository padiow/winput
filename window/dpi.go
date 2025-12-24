package window

import (
	"fmt"
	"unsafe"
)

// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is (HANDLE)(-4)
var dpiAwarenessPerMonitorV2 = ^uintptr(3)

func EnablePerMonitorDPI() error {
	if ProcSetProcessDpiAwarenessCtx.Find() != nil {
		return fmt.Errorf("SetProcessDpiAwarenessContext not found")
	}
	r, _, _ := ProcSetProcessDpiAwarenessCtx.Call(dpiAwarenessPerMonitorV2)
	// If the return value is NULL (0), it might have failed, but documentation says:
	// "If the function succeeds, the return value is TRUE. Otherwise, it is FALSE."
	// Actually SetProcessDpiAwarenessContext returns TRUE/FALSE.
	if r == 0 {
		return fmt.Errorf("SetProcessDpiAwarenessContext failed")
	}
	return nil
}

func GetDPI(hwnd uintptr) (uint32, error) {
	if ProcGetDpiForWindow.Find() == nil {
		r, _, _ := ProcGetDpiForWindow.Call(hwnd)
		if r != 0 {
			return uint32(r), nil
		}
	}
	
	// Fallback or explicit failure? Prompt says "No silent degradation". 
	// But getting DPI is critical. If GetDpiForWindow is missing (Win8.1 or older), 
	// we might want to try GetDpiForMonitor.
	
	hMon := MonitorFromWindow(hwnd)
	if hMon != 0 {
		dx, _, err := GetDpiForMonitor(hMon)
		if err == nil {
			return dx, nil
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
