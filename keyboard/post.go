package keyboard

import (
	"fmt"
	"syscall"
	"time"

	"github.com/rpdg/winput/window"
)

const (
	WM_KEYDOWN = 0x0100
	WM_KEYUP   = 0x0101
	WM_CHAR    = 0x0102

	MAPVK_VSC_TO_VK = 1
)

func MapScanCodeToVK(sc Key) uintptr {
	r, _, _ := window.ProcMapVirtualKeyW.Call(uintptr(sc), MAPVK_VSC_TO_VK)
	return r
}

func post(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) error {
	r, _, e := window.ProcPostMessageW.Call(hwnd, uintptr(msg), wparam, lparam)
	if r == 0 {
		if errno, ok := e.(syscall.Errno); ok && errno != 0 {
			return fmt.Errorf("%w: %v", window.ErrPostMessageFailed, errno)
		}
		return window.ErrPostMessageFailed
	}
	return nil
}

func KeyDown(hwnd uintptr, key Key) error {
	vk := MapScanCodeToVK(key)
	if vk == 0 {
		return fmt.Errorf("unsupported key: %d", key)
	}
	// LParam for WM_KEYDOWN:
	// 0-15: Repeat count (1)
	// 16-23: Scan code
	// 24: Extended key (0 for standard keys, assuming standard for now)
	// 29: Context Code (0)
	// 30: Previous Key State (0 for first press)
	// 31: Transition State (0 for key down)
	lparam := uintptr(1) | (uintptr(key) << 16)
	return post(hwnd, WM_KEYDOWN, vk, lparam)
}

func KeyUp(hwnd uintptr, key Key) error {
	vk := MapScanCodeToVK(key)
	if vk == 0 {
		return fmt.Errorf("unsupported key: %d", key)
	}
	// LParam for WM_KEYUP:
	// 0-15: Repeat count (1)
	// 16-23: Scan code
	// 24: Extended key
	// 29: Context Code (0)
	// 30: Previous Key State (1, always down before up)
	// 31: Transition State (1, key is being released)
	lparam := uintptr(1) | (uintptr(key) << 16) | (1 << 30) | (1 << 31)
	return post(hwnd, WM_KEYUP, vk, lparam)
}

func Press(hwnd uintptr, key Key) error {
	if err := KeyDown(hwnd, key); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)
	return KeyUp(hwnd, key)
}

// Type sends text using WM_CHAR.
// This is the most reliable way for background window text input as it
// bypasses the need for global modifier key states (Shift/Ctrl).
func Type(hwnd uintptr, text string) error {
	for _, r := range text {
		// WM_CHAR accepts UTF-16 code unit.
		if r > 0xFFFF {
			// Handle surrogate pairs
			r -= 0x10000
			high := 0xD800 + (r >> 10)
			low := 0xDC00 + (r & 0x3FF)
			if err := post(hwnd, WM_CHAR, uintptr(high), 1); err != nil {
				return err
			}
			if err := post(hwnd, WM_CHAR, uintptr(low), 1); err != nil {
				return err
			}
		} else {
			if err := post(hwnd, WM_CHAR, uintptr(r), 1); err != nil {
				return err
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	return nil
}
