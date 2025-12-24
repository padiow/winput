package interception

/*
#cgo LDFLAGS: -L../../ -linterception
#include "../../interception.h"

// Helper to bypass strict array pointer typing in CGO which struggles with typedef array pointers
int interception_send_wrapper(InterceptionContext context, InterceptionDevice device, void* stroke, unsigned int nstroke) {
    return interception_send(context, device, (InterceptionStroke*)stroke, nstroke);
}
*/
import "C"
import "unsafe"

// Types
type Context C.InterceptionContext
type Device C.InterceptionDevice

// Go-friendly structs
type MouseStroke struct {
	State uint16
	Flags uint16
	Rolling int16
	X     int32
	Y     int32
	Information uint32
}

type KeyStroke struct {
	Code  uint16
	State uint16
	Information uint32
}

// Constants for Mouse
const (
	MouseStateLeftDown   = C.INTERCEPTION_MOUSE_LEFT_BUTTON_DOWN
	MouseStateLeftUp     = C.INTERCEPTION_MOUSE_LEFT_BUTTON_UP
	MouseStateRightDown  = C.INTERCEPTION_MOUSE_RIGHT_BUTTON_DOWN
	MouseStateRightUp    = C.INTERCEPTION_MOUSE_RIGHT_BUTTON_UP
	MouseStateMiddleDown = C.INTERCEPTION_MOUSE_MIDDLE_BUTTON_DOWN
	MouseStateMiddleUp   = C.INTERCEPTION_MOUSE_MIDDLE_BUTTON_UP

	MouseFlagMoveRelative = C.INTERCEPTION_MOUSE_MOVE_RELATIVE
	MouseFlagMoveAbsolute = C.INTERCEPTION_MOUSE_MOVE_ABSOLUTE
)

// Constants for Keyboard
const (
	KeyStateDown = C.INTERCEPTION_KEY_DOWN
	KeyStateUp   = C.INTERCEPTION_KEY_UP
	KeyStateE0   = C.INTERCEPTION_KEY_E0
	KeyStateE1   = C.INTERCEPTION_KEY_E1
)

// Functions

func CreateContext() Context {
	return Context(C.interception_create_context())
}

func DestroyContext(ctx Context) {
	C.interception_destroy_context(C.InterceptionContext(ctx))
}

func IsMouse(dev Device) bool {
	return C.interception_is_mouse(C.InterceptionDevice(dev)) != 0
}

func IsKeyboard(dev Device) bool {
	return C.interception_is_keyboard(C.InterceptionDevice(dev)) != 0
}

func SendMouse(ctx Context, dev Device, s *MouseStroke) {
	var cStroke C.InterceptionMouseStroke
	cStroke.state = C.ushort(s.State)
	cStroke.flags = C.ushort(s.Flags)
	cStroke.rolling = C.short(s.Rolling)
	cStroke.x = C.int(s.X)
	cStroke.y = C.int(s.Y)
	cStroke.information = C.uint(s.Information)

	C.interception_send_wrapper(C.InterceptionContext(ctx), C.InterceptionDevice(dev), unsafe.Pointer(&cStroke), 1)
}

func SendKey(ctx Context, dev Device, s *KeyStroke) {
	var cStroke C.InterceptionKeyStroke
	cStroke.code = C.ushort(s.Code)
	cStroke.state = C.ushort(s.State)
	cStroke.information = C.uint(s.Information)

	C.interception_send_wrapper(C.InterceptionContext(ctx), C.InterceptionDevice(dev), unsafe.Pointer(&cStroke), 1)
}