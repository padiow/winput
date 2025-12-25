package main

import (
	"fmt"

	"github.com/rpdg/winput"
	"github.com/rpdg/winput/screen"
)

func main() {
	fmt.Println("=== winput: Global Vision Example ===")
	fmt.Println("This mode uses absolute screen coordinates, ideal for Electron apps or games.")

	// 1. Screen Geometry
	bounds := screen.VirtualBounds()
	fmt.Printf("🖥️  Virtual Desktop: [%d, %d, %d, %d]\n", bounds.Left, bounds.Top, bounds.Right, bounds.Bottom)

	monitors, _ := screen.Monitors()
	for i, m := range monitors {
		fmt.Printf("   Monitor %d: %+v (Primary: %v)\n", i, m.Bounds, m.Primary)
	}

	if len(monitors) == 0 {
		return
	}

	// 2. Global Input
	// Move to center of primary monitor
	center := monitors[0].Bounds
	cx := (center.Left + center.Right) / 2
	cy := (center.Top + center.Bottom) / 2

	fmt.Printf("👉 Moving mouse to center of primary monitor (%d, %d)...\n", cx, cy)
	winput.MoveMouseTo(cx, cy)

	// Simulate typing "globally" (goes to active window)
	fmt.Println("👉 Typing globally...")
	winput.Type("Global Input")

	fmt.Println("=== Done ===")
}
