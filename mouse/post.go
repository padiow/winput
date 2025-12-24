package mouse

import (
	"github.com/rpdg/winput/window"
)

const (
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONDOWN   = 0x0204
	WM_RBUTTONUP     = 0x0205
	WM_RBUTTONDBLCLK = 0x0206
	WM_MBUTTONDOWN   = 0x0207
	WM_MBUTTONUP     = 0x0208
	WM_MBUTTONDBLCLK = 0x0209
	WM_MOUSEWHEEL    = 0x020A

	MK_LBUTTON = 0x0001
	MK_RBUTTON = 0x0002
	MK_MBUTTON = 0x0010
)

func makeLParam(x, y int32) uintptr {
	ux := uint32(uint16(x))
	uy := uint32(uint16(y))
	return uintptr(ux | (uy << 16))
}

func post(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) error {
	r, _, _ := window.ProcPostMessageW.Call(hwnd, uintptr(msg), wparam, lparam)
	if r == 0 {
		return nil
	}
	return nil
}

func Move(hwnd uintptr, x, y int32) error {
	lparam := makeLParam(x, y)
	return post(hwnd, WM_MOUSEMOVE, 0, lparam)
}

func Click(hwnd uintptr, x, y int32) error {
	lparam := makeLParam(x, y)
	Move(hwnd, x, y)

	if err := post(hwnd, WM_LBUTTONDOWN, MK_LBUTTON, lparam); err != nil {
		return err
	}
	return post(hwnd, WM_LBUTTONUP, 0, lparam)
}

func ClickRight(hwnd uintptr, x, y int32) error {
	lparam := makeLParam(x, y)
	Move(hwnd, x, y)

	if err := post(hwnd, WM_RBUTTONDOWN, MK_RBUTTON, lparam); err != nil {
		return err
	}
	return post(hwnd, WM_RBUTTONUP, 0, lparam)
}

func ClickMiddle(hwnd uintptr, x, y int32) error {
	lparam := makeLParam(x, y)
	Move(hwnd, x, y)

	if err := post(hwnd, WM_MBUTTONDOWN, MK_MBUTTON, lparam); err != nil {
		return err
	}
	return post(hwnd, WM_MBUTTONUP, 0, lparam)
}

func DoubleClick(hwnd uintptr, x, y int32) error {
	lparam := makeLParam(x, y)
	Move(hwnd, x, y)

	if err := post(hwnd, WM_LBUTTONDBLCLK, MK_LBUTTON, lparam); err != nil {
		return err
	}
	return post(hwnd, WM_LBUTTONUP, 0, lparam)
}

// Scroll sends a vertical scroll message.
// delta is usually a multiple of 120 (WHEEL_DELTA). Positive = forward/up, Negative = backward/down.
// x, y are client coordinates where the scroll happens (usually under cursor).
func Scroll(hwnd uintptr, x, y int32, delta int32) error {
	// WM_MOUSEWHEEL requires Screen Coordinates in LPARAM (low x, high y)
	// BUT PostMessage to a specific window handles client coordinates differently?
	// MSDN says: "The coordinates are relative to the upper-left corner of the screen."
	// So we need Screen coordinates here, even if we are targeting a specific HWND via PostMessage.

	sx, sy, err := window.ClientToScreen(hwnd, x, y)
	if err != nil {
		return err
	}

	// WPARAM: High word = distance, Low word = keys (0)
	// Distance: multiples of 120.
	wparam := uintptr(uint32(uint16(0)) | (uint32(int16(delta)) << 16))
	lparam := makeLParam(sx, sy) // reusing makeLParam but passing Screen X/Y

	return post(hwnd, WM_MOUSEWHEEL, wparam, lparam)
}
