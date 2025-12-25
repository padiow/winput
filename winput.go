package winput

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rpdg/winput/hid"
	"github.com/rpdg/winput/keyboard"
	"github.com/rpdg/winput/mouse"
	"github.com/rpdg/winput/window"
)

type Window struct {
	HWND uintptr
}

// -----------------------------------------------------------------------------
// Window Discovery
// -----------------------------------------------------------------------------

func FindByTitle(title string) (*Window, error) {
	hwnd, err := window.FindByTitle(title)
	if err != nil {
		return nil, ErrWindowNotFound
	}
	return &Window{HWND: hwnd}, nil
}

func FindByClass(class string) (*Window, error) {
	hwnd, err := window.FindByClass(class)
	if err != nil {
		return nil, ErrWindowNotFound
	}
	return &Window{HWND: hwnd}, nil
}

func FindByPID(pid uint32) ([]*Window, error) {
	hwnds, err := window.FindByPID(pid)
	if err != nil {
		return nil, ErrWindowNotFound
	}
	windows := make([]*Window, len(hwnds))
	for i, h := range hwnds {
		windows[i] = &Window{HWND: h}
	}
	return windows, nil
}

// FindByProcessName searches for all top-level windows belonging to a process with the given executable name.
// name: e.g. "notepad.exe"
func FindByProcessName(name string) ([]*Window, error) {
	pid, err := window.FindPIDByName(name)
	if err != nil {
		return nil, err
	}
	return FindByPID(pid)
}

// -----------------------------------------------------------------------------
// Window State
// -----------------------------------------------------------------------------

func (w *Window) IsValid() bool {
	return window.IsValid(w.HWND)
}

func (w *Window) IsVisible() bool {
	return window.IsVisible(w.HWND) && !window.IsIconic(w.HWND)
}

func (w *Window) checkReady() error {
	if !w.IsValid() {
		return ErrWindowGone
	}
	if !w.IsVisible() {
		return ErrWindowNotVisible
	}
	return nil
}

// -----------------------------------------------------------------------------
// Backend Configuration
// -----------------------------------------------------------------------------

type Backend int

const (
	BackendMessage Backend = iota
	BackendHID
)

var (
	currentBackend Backend = BackendMessage
	backendMutex   sync.RWMutex
)

func SetBackend(b Backend) {
	backendMutex.Lock()
	defer backendMutex.Unlock()
	currentBackend = b
}

func SetHIDLibraryPath(path string) {
	hid.SetLibraryPath(path)
}

// -----------------------------------------------------------------------------
// Global Input API (Screen Coordinates)
// -----------------------------------------------------------------------------

// MoveMouseTo moves the mouse cursor to the absolute screen coordinates (Virtual Desktop).
// x, y: Virtual Screen Coordinates (can be negative).
// BackendMessage: Uses SetCursorPos (Instant).
// BackendHID: Uses human-like trajectory.
func MoveMouseTo(x, y int32) error {
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		return hid.Move(x, y)
	}
	
	// Fallback to User32 SetCursorPos
	r, _, _ := window.ProcSetCursorPos.Call(uintptr(x), uintptr(y))
	if r == 0 {
		return fmt.Errorf("SetCursorPos failed")
	}
	return nil
}

// ClickMouseAt moves to the specified screen coordinates and performs a left click.
func ClickMouseAt(x, y int32) error {
	if err := MoveMouseTo(x, y); err != nil {
		return err
	}
	
	if getBackend() == BackendHID {
		// hid.Click relies on current cursor position, which MoveMouseTo just set.
		// However, hid.Click takes "target" coords and moves again.
		// We can just call hid.Click(x, y) as it handles movement internally too.
		return hid.Click(x, y)
	}

	// BackendMessage Fallback: mouse_event
	// MOUSEEVENTF_LEFTDOWN = 0x0002
	// MOUSEEVENTF_LEFTUP   = 0x0004
	time.Sleep(30 * time.Millisecond)
	window.ProcMouseEvent.Call(0x0002, 0, 0, 0, 0)
	window.ProcMouseEvent.Call(0x0004, 0, 0, 0, 0)
	return nil
}

// KeyDown simulates a global key down event.
// Does not require a target window.
func KeyDown(k Key) error {
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		return hid.KeyDown(uint16(k))
	}

	// Message Backend Fallback: keybd_event
	// KEYEVENTF_SCANCODE = 0x0008
	// We map ScanCode to VK because keybd_event expects VK usually, but can take ScanCode.
	// Let's use VK for better compatibility if we don't have Extended Key flag logic.
	// Actually, keybd_event(bVk, bScan, dwFlags, dwExtraInfo)
	// We can use keyboard.MapScanCodeToVK
	vk := keyboard.MapScanCodeToVK(k)
	window.ProcKeybdEvent.Call(vk, 0, 0, 0)
	return nil
}

// KeyUp simulates a global key up event.
func KeyUp(k Key) error {
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		return hid.KeyUp(uint16(k))
	}

	// KEYEVENTF_KEYUP = 0x0002
	vk := keyboard.MapScanCodeToVK(k)
	window.ProcKeybdEvent.Call(vk, 0, 0x0002, 0)
	return nil
}

// Press simulates a global key press (Down + Up).
func Press(k Key) error {
	if err := KeyDown(k); err != nil {
		return err
	}
	// Default delay for key press to be registered by most apps
	time.Sleep(30 * time.Millisecond)
	return KeyUp(k)
}

// PressHotkey simulates a global combination of keys.
func PressHotkey(keys ...Key) error {
	if len(keys) == 0 {
		return nil
	}
	
	// No Window checkReady here, as it's global
	if err := checkBackend(); err != nil {
		return err
	}

	for _, k := range keys {
		if err := KeyDown(k); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Hold the combination briefly
	time.Sleep(30 * time.Millisecond)
	for i := len(keys) - 1; i >= 0; i-- {
		if err := KeyUp(keys[i]); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// Type simulates global text input.
func Type(text string) error {
	if err := checkBackend(); err != nil {
		return err
	}

	// For global input, we don't have the luxury of WM_CHAR bypassing layout.
	// We MUST simulate keystrokes.
	// So we use the same logic as HID backend (LookupKey -> Shift -> Press).
	
	for _, r := range text {
		k, shifted, ok := keyboard.LookupKey(r)
		if !ok {
			return ErrUnsupportedKey
		}

		if shifted {
			if err := KeyDown(KeyShift); err != nil { return err }
			time.Sleep(10 * time.Millisecond)
			if err := Press(k); err != nil { 
				KeyUp(KeyShift)
				return err 
			}
			if err := KeyUp(KeyShift); err != nil { return err }
		} else {
			if err := Press(k); err != nil { return err }
		}
		// Delay between characters
		time.Sleep(30 * time.Millisecond)
	}
	return nil
}

func checkBackend() error {
	backendMutex.RLock()
	cb := currentBackend
	backendMutex.RUnlock()

	if cb == BackendHID {
		if err := hid.Init(); err != nil {
			if errors.Is(err, hid.ErrDriverNotInstalled) {
				return ErrDriverNotInstalled
			}
			return fmt.Errorf("%w: %v", ErrDLLLoadFailed, err)
		}
	}
	return nil
}

func getBackend() Backend {
	backendMutex.RLock()
	defer backendMutex.RUnlock()
	return currentBackend
}

// -----------------------------------------------------------------------------
// Input API (Mouse)
// -----------------------------------------------------------------------------

func (w *Window) Move(x, y int32) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		sx, sy, err := window.ClientToScreen(w.HWND, x, y)
		if err != nil {
			return err
		}
		return hid.Move(sx, sy)
	}
	return mouse.Move(w.HWND, x, y)
}

func (w *Window) MoveRel(dx, dy int32) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		cx, cy, err := window.GetCursorPos()
		if err != nil {
			return err
		}
		return hid.Move(cx+dx, cy+dy)
	}

	sx, sy, err := window.GetCursorPos()
	if err != nil {
		return err
	}
	tx, ty := sx+dx, sy+dy
	cx, cy, err := window.ScreenToClient(w.HWND, tx, ty)
	if err != nil {
		return err
	}
	return mouse.Move(w.HWND, cx, cy)
}

func (w *Window) Click(x, y int32) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		sx, sy, err := window.ClientToScreen(w.HWND, x, y)
		if err != nil {
			return err
		}
		return hid.Click(sx, sy)
	}
	return mouse.Click(w.HWND, x, y)
}

func (w *Window) ClickRight(x, y int32) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		sx, sy, err := window.ClientToScreen(w.HWND, x, y)
		if err != nil {
			return err
		}
		return hid.ClickRight(sx, sy)
	}
	return mouse.ClickRight(w.HWND, x, y)
}

func (w *Window) ClickMiddle(x, y int32) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		sx, sy, err := window.ClientToScreen(w.HWND, x, y)
		if err != nil {
			return err
		}
		return hid.ClickMiddle(sx, sy)
	}
	return mouse.ClickMiddle(w.HWND, x, y)
}

func (w *Window) DoubleClick(x, y int32) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		sx, sy, err := window.ClientToScreen(w.HWND, x, y)
		if err != nil {
			return err
		}
		return hid.DoubleClick(sx, sy)
	}
	return mouse.DoubleClick(w.HWND, x, y)
}

func (w *Window) Scroll(x, y int32, delta int32) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		return hid.Scroll(delta)
	}
	return mouse.Scroll(w.HWND, x, y, delta)
}

// -----------------------------------------------------------------------------
// Input API (Keyboard)
// -----------------------------------------------------------------------------

type Key = keyboard.Key

const (
	KeyEsc       = keyboard.KeyEsc
	Key1         = keyboard.Key1
	Key2         = keyboard.Key2
	Key3         = keyboard.Key3
	Key4         = keyboard.Key4
	Key5         = keyboard.Key5
	Key6         = keyboard.Key6
	Key7         = keyboard.Key7
	Key8         = keyboard.Key8
	Key9         = keyboard.Key9
	Key0         = keyboard.Key0
	KeyMinus     = keyboard.KeyMinus
	KeyEqual     = keyboard.KeyEqual
	KeyBkSp      = keyboard.KeyBkSp
	KeyTab       = keyboard.KeyTab
	KeyQ         = keyboard.KeyQ
	KeyW         = keyboard.KeyW
	KeyE         = keyboard.KeyE
	KeyR         = keyboard.KeyR
	KeyT         = keyboard.KeyT
	KeyY         = keyboard.KeyY
	KeyU         = keyboard.KeyU
	KeyI         = keyboard.KeyI
	KeyO         = keyboard.KeyO
	KeyP         = keyboard.KeyP
	KeyLBr       = keyboard.KeyLBr
	KeyRBr       = keyboard.KeyRBr
	KeyEnter     = keyboard.KeyEnter
	KeyCtrl      = keyboard.KeyCtrl
	KeyA         = keyboard.KeyA
	KeyS         = keyboard.KeyS
	KeyD         = keyboard.KeyD
	KeyF         = keyboard.KeyF
	KeyG         = keyboard.KeyG
	KeyH         = keyboard.KeyH
	KeyJ         = keyboard.KeyJ
	KeyK         = keyboard.KeyK
	KeyL         = keyboard.KeyL
	KeySemi      = keyboard.KeySemi
	KeyQuot      = keyboard.KeyQuot
	KeyTick      = keyboard.KeyTick
	KeyShift     = keyboard.KeyShift
	KeyBackslash = keyboard.KeyBackslash
	KeyZ         = keyboard.KeyZ
	KeyX         = keyboard.KeyX
	KeyC         = keyboard.KeyC
	KeyV         = keyboard.KeyV
	KeyB         = keyboard.KeyB
	KeyN         = keyboard.KeyN
	KeyM         = keyboard.KeyM
	KeyComma     = keyboard.KeyComma
	KeyDot       = keyboard.KeyDot
	KeySlash     = keyboard.KeySlash
	KeySpace     = keyboard.KeySpace
	KeyAlt       = keyboard.KeyAlt
	KeyCaps      = keyboard.KeyCaps
	KeyF1        = keyboard.KeyF1
	KeyF2        = keyboard.KeyF2
	KeyF3        = keyboard.KeyF3
	KeyF4        = keyboard.KeyF4
	KeyF5        = keyboard.KeyF5
	KeyF6        = keyboard.KeyF6
	KeyF7        = keyboard.KeyF7
	KeyF8        = keyboard.KeyF8
	KeyF9        = keyboard.KeyF9
	KeyF10       = keyboard.KeyF10
	KeyF11       = keyboard.KeyF11
	KeyF12       = keyboard.KeyF12
	KeyNumLock   = keyboard.KeyNumLock
	KeyScroll    = keyboard.KeyScroll
	
	KeyHome      = keyboard.KeyHome
	KeyArrowUp   = keyboard.KeyArrowUp
	KeyPageUp    = keyboard.KeyPageUp
	KeyLeft      = keyboard.KeyLeft
	KeyRight     = keyboard.KeyRight
	KeyEnd       = keyboard.KeyEnd
	KeyArrowDown = keyboard.KeyArrowDown
	KeyPageDown  = keyboard.KeyPageDown
	KeyInsert    = keyboard.KeyInsert
	KeyDelete    = keyboard.KeyDelete
)

func KeyFromRune(r rune) (Key, bool) {
	k, _, ok := keyboard.LookupKey(r)
	return k, ok
}

func (w *Window) KeyDown(key Key) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		return hid.KeyDown(uint16(key))
	}
	return keyboard.KeyDown(w.HWND, key)
}

func (w *Window) KeyUp(key Key) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		return hid.KeyUp(uint16(key))
	}
	return keyboard.KeyUp(w.HWND, key)
}

func (w *Window) Press(key Key) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	if getBackend() == BackendHID {
		return hid.Press(uint16(key))
	}
	return keyboard.Press(w.HWND, key)
}

func (w *Window) PressHotkey(keys ...Key) error {
	if len(keys) == 0 {
		return nil
	}
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	for _, k := range keys {
		if err := w.KeyDown(k); err != nil {
			return err
		}
	}
	for i := len(keys) - 1; i >= 0; i-- {
		if err := w.KeyUp(keys[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w *Window) Type(text string) error {
	if err := w.checkReady(); err != nil {
		return err
	}
	if err := checkBackend(); err != nil {
		return err
	}

	cb := getBackend()

	for _, r := range text {
		k, shifted, ok := keyboard.LookupKey(r)
		if !ok {
			return ErrUnsupportedKey
		}

		if shifted {
			if cb == BackendHID {
				if err := hid.KeyDown(uint16(KeyShift)); err != nil {
					return err
				}
				time.Sleep(10 * time.Millisecond)
				// Defer cleanup not applicable in loop, must manual check
				if err := hid.Press(uint16(k)); err != nil {
					hid.KeyUp(uint16(KeyShift)) // Try cleanup
					return err
				}
				if err := hid.KeyUp(uint16(KeyShift)); err != nil {
					return err
				}
			} else {
				if err := keyboard.KeyDown(w.HWND, KeyShift); err != nil {
					return err
				}
				time.Sleep(10 * time.Millisecond)
				if err := keyboard.Press(w.HWND, k); err != nil {
					keyboard.KeyUp(w.HWND, KeyShift) // Try cleanup
					return err
				}
				if err := keyboard.KeyUp(w.HWND, KeyShift); err != nil {
					return err
				}
			}
		} else {
			if cb == BackendHID {
				if err := hid.Press(uint16(k)); err != nil {
					return err
				}
			} else {
				if err := keyboard.Press(w.HWND, k); err != nil {
					return err
				}
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	return nil
}
// -----------------------------------------------------------------------------
// Coordinate & DPI
// -----------------------------------------------------------------------------

func EnablePerMonitorDPI() error {
	return window.EnablePerMonitorDPI()
}

func (w *Window) DPI() (uint32, uint32, error) {
	return window.GetDPI(w.HWND)
}

func (w *Window) ClientRect() (width, height int32, err error) {
	return window.GetClientRect(w.HWND)
}

func (w *Window) ScreenToClient(x, y int32) (cx, cy int32, err error) {
	return window.ScreenToClient(w.HWND, x, y)
}

func (w *Window) ClientToScreen(x, y int32) (sx, sy int32, err error) {
	return window.ClientToScreen(w.HWND, x, y)
}
