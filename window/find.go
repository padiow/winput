package window

import (
	"fmt"
	"syscall"
	"unsafe"
)

func utf16Ptr(s string) *uint16 {
	ptr, _ := syscall.UTF16PtrFromString(s)
	return ptr
}

func FindByTitle(title string) (uintptr, error) {
	ret, _, _ := ProcFindWindowW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(title))),
	)
	if ret == 0 {
		return 0, fmt.Errorf("window not found with title: %s", title)
	}
	return ret, nil
}

func FindByClass(class string) (uintptr, error) {
	ret, _, _ := ProcFindWindowW.Call(
		uintptr(unsafe.Pointer(utf16Ptr(class))),
		0,
	)
	if ret == 0 {
		return 0, fmt.Errorf("window not found with class: %s", class)
	}
	return ret, nil
}

func FindByPID(targetPid uint32) ([]uintptr, error) {
	var hwnds []uintptr

	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var pid uint32
		ProcGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

		if pid == targetPid {
			// Optional: Check if window is visible? User didn't specify, but usually we want visible ones.
			// But for automation, sometimes invisible ones are used.
			// Let's stick to simple PID match for now.
			hwnds = append(hwnds, hwnd)
		}
		return 1 // Continue enumeration
	})

	ret, _, _ := ProcEnumWindows.Call(cb, 0)
	if ret == 0 {
		// EnumWindows returns 0 if it fails OR if the callback stops it (returns 0).
		// Since we always return 1, 0 means failure or no windows (but EnumWindows rarely fails like that).
		// However, if we found nothing, hwnds is empty.
	}

	if len(hwnds) == 0 {
		return nil, fmt.Errorf("no windows found for PID: %d", targetPid)
	}

	return hwnds, nil
}
