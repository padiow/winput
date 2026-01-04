package window

import (
	"fmt"
	"unsafe"
)

// DPI Awareness Contexts (Pseudo-Handles)
// Win10 1607+
var (
	DPI_AWARENESS_CONTEXT_UNAWARE              = uintptr(0xFFFFFFFF) // -1
	DPI_AWARENESS_CONTEXT_SYSTEM_AWARE         = uintptr(0xFFFFFFFE) // -2
	DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE    = uintptr(0xFFFFFFFD) // -3 (V1)
	DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = uintptr(0xFFFFFFFC) // -4 (V2)
	DPI_AWARENESS_CONTEXT_UNAWARE_GDISCALED    = uintptr(0xFFFFFFFB) // -5
)

// EnablePerMonitorDPI attempts to set the process to Per-Monitor DPI Aware (V2).
// It falls back to V1 or System Aware on older systems if V2 is unavailable.
func EnablePerMonitorDPI() error {
	// Try SetProcessDpiAwarenessContext (Win10 1607+)
	// Prefer V2
	r, _, _ := ProcSetProcessDpiAwarenessCtx.Call(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)
	if r != 0 {
		return nil
	}
	// Fallback to V1
	r, _, _ = ProcSetProcessDpiAwarenessCtx.Call(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE)
	if r != 0 {
		return nil
	}

	// Fallback for older Windows (8.1/10 early) - Shcore.dll
	// SetProcessDpiAwareness(PROCESS_PER_MONITOR_DPI_AWARE = 2)
	procSetProcessDpiAwareness := shcore.NewProc("SetProcessDpiAwareness")
	if procSetProcessDpiAwareness.Find() == nil {
		r, _, _ := procSetProcessDpiAwareness.Call(2)
		if r == 0 { // S_OK
			return nil
		}
	}

	// Fallback for Vista/8 - User32.dll
	// SetProcessDPIAware()
	procSetProcessDPIAware := user32.NewProc("SetProcessDPIAware")
	if procSetProcessDPIAware.Find() == nil {
		r, _, _ := procSetProcessDPIAware.Call()
		if r != 0 {
			return nil
		}
	}

	return fmt.Errorf("failed to set DPI awareness")
}

// GetDPI returns the DPI for the specified window.
// It tries to use GetDpiForWindow (Win10 1607+), falling back to System DPI.
func GetDPI(hwnd uintptr) (uint32, uint32, error) {
	if ProcGetDpiForWindow.Find() == nil {
		dpi, _, _ := ProcGetDpiForWindow.Call(hwnd)
		if dpi != 0 {
			return uint32(dpi), uint32(dpi), nil
		}
	}

	// Fallback: GetDC -> GetDeviceCaps
	// This returns the System DPI (or Per-Monitor if the process is aware, but less reliable for mixed setups)
	hdc, _, _ := ProcGetDC.Call(hwnd)
	if hdc == 0 {
		return 96, 96, fmt.Errorf("GetDC failed")
	}
	defer ProcReleaseDC.Call(hwnd, hdc)

	const LOGPIXELSX = 88
	const LOGPIXELSY = 90

	dpiX, _, _ := ProcGetDeviceCaps.Call(hdc, LOGPIXELSX)
	dpiY, _, _ := ProcGetDeviceCaps.Call(hdc, LOGPIXELSY)

	if dpiX == 0 {
		dpiX = 96
	}
	if dpiY == 0 {
		dpiY = 96
	}

	return uint32(dpiX), uint32(dpiY), nil
}

// IsPerMonitorDPIAware checks if the current process is Per-Monitor DPI Aware (V1 or V2).
// This is critical for ensuring that screen coordinates (GetSystemMetrics, BitBlt) are exact
// pixels and not virtualized/scaled by the OS.
func IsPerMonitorDPIAware() bool {
	// API only available on Win10 1607+
	if ProcGetProcessDpiAwarenessCtx.Find() != nil || ProcAreDpiAwarenessContextsEqual.Find() != nil {
		// If API missing, we can't strictly check.
		// We could fallback to GetProcessDpiAwareness from Shcore, but for simplicity
		// and considering the target audience, we might assume false or true.
		// Safest is false to warn user.
		return false
	}

	ctx, _, _ := ProcGetProcessDpiAwarenessCtx.Call(0) // 0 = Current Process
	if ctx == 0 {
		return false
	}

	// Helper to check equality
	areEqual := func(a, b uintptr) bool {
		r, _, _ := ProcAreDpiAwarenessContextsEqual.Call(a, b)
		return r != 0
	}

	// Accept V2 (Best)
	if areEqual(ctx, DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2) {
		return true
	}
	// Accept V1 (Good enough for coordinates)
	if areEqual(ctx, DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE) {
		return true
	}

	return false
}