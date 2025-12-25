package screen

// Point represents a point in the Virtual Desktop coordinate system.
// Coordinates can be negative (e.g., secondary monitor to the left of primary).
type Point struct {
	X int32
	Y int32
}

// Rect represents a rectangle in the Virtual Desktop coordinate system.
type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// Monitor represents a physical display device.
type Monitor struct {
	Handle  uintptr
	Bounds  Rect
	WorkArea Rect // Excludes taskbar
	Primary bool
	DeviceName string
}
