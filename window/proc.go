package window

import (
	"syscall"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	shcore                       = syscall.NewLazyDLL("shcore.dll")
	
	ProcFindWindowW              = user32.NewProc("FindWindowW")
	ProcFindWindowExW            = user32.NewProc("FindWindowExW")
	ProcGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	ProcEnumWindows              = user32.NewProc("EnumWindows")
	ProcIsWindowVisible          = user32.NewProc("IsWindowVisible")
	ProcGetClassNameW            = user32.NewProc("GetClassNameW")
	
	ProcScreenToClient           = user32.NewProc("ScreenToClient")
	ProcClientToScreen           = user32.NewProc("ClientToScreen")
	ProcGetClientRect            = user32.NewProc("GetClientRect")
	ProcGetCursorPos             = user32.NewProc("GetCursorPos")
	ProcMonitorFromPoint         = user32.NewProc("MonitorFromPoint")
	ProcMonitorFromWindow        = user32.NewProc("MonitorFromWindow")
	
	ProcGetDpiForWindow          = user32.NewProc("GetDpiForWindow") // Win10+
	ProcSetProcessDpiAwarenessCtx = user32.NewProc("SetProcessDpiAwarenessContext")
	
	ProcGetDpiForMonitor         = shcore.NewProc("GetDpiForMonitor")
	
	ProcPostMessageW             = user32.NewProc("PostMessageW")
	ProcMapVirtualKeyW           = user32.NewProc("MapVirtualKeyW")
)
