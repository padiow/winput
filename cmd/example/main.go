package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/rpdg/winput"
	"github.com/rpdg/winput/screen"
)

func main() {
	fmt.Println("=== winput Library Example ===")

	// 1. Enable DPI Awareness (Critical for correct coordinates)
	if err := winput.EnablePerMonitorDPI(); err != nil {
		log.Printf("Warning: Failed to enable DPI awareness: %v", err)
	}

	// 2. Find Window
	// Try finding Notepad by Process Name first, then Class
	windows, err := winput.FindByProcessName("notepad.exe")
	var w *winput.Window
	if err == nil && len(windows) > 0 {
		w = windows[0]
		fmt.Println("✅ Found Notepad via Process Name")
	} else {
		w, err = winput.FindByClass("Notepad")
		if err != nil {
			log.Println("❌ 未找到记事本窗口，请先打开记事本运行此示例。")
			return
		}
		fmt.Println("✅ Found Notepad via Window Class")
	}

	// 3. Check Visibility
	// New safety feature: operations fail if window is minimized
	// Let's bring it to front (User manual action required usually, but we check state)
	// winput doesn't provide "ShowWindow" yet to keep API clean, but we warn user.
	
	// 4. Basic Input (Message Backend)
	fmt.Println("👉 Testing Message Backend (Click & Type)...")
	if err := w.Click(100, 100); err != nil {
		if errors.Is(err, winput.ErrWindowNotVisible) {
			log.Fatal("❌ Window is minimized or hidden. Please restore it.")
		}
		log.Fatal(err)
	}

	w.Type("Hello from winput! ")
	w.PressHotkey(winput.KeyShift, winput.Key1) // Prints '!'
	w.Press(winput.KeyEnter)

	// 5. Global Input & Screen Geometry (New in v1.1)
	fmt.Println("👉 Testing Global Input & Screen Geometry...")
	bounds := screen.VirtualBounds()
	fmt.Printf("   Virtual Desktop Bounds: %d, %d, %d, %d\n", bounds.Left, bounds.Top, bounds.Right, bounds.Bottom)
	
	monitors, _ := screen.Monitors()
	fmt.Printf("   Found %d monitors\n", len(monitors))

	// Move mouse to center of primary monitor (using Global API)
	if len(monitors) > 0 {
		center := monitors[0].Bounds
		cx := (center.Left + center.Right) / 2
		cy := (center.Top + center.Bottom) / 2
		fmt.Printf("   Moving to primary monitor center: %d, %d\n", cx, cy)
		winput.MoveMouseTo(cx, cy)
	}

	// 6. HID Backend (Optional)
	fmt.Println("👉 Testing HID Backend (Mouse Move)...")
	// Note: interception.dll must be present for this to work
	winput.SetHIDLibraryPath("interception.dll") // Default, strictly optional call
	winput.SetBackend(winput.BackendHID)

	// MoveRel is a good test for HID
	err = w.MoveRel(50, 50)
	if err != nil {
		if errors.Is(err, winput.ErrDriverNotInstalled) {
			fmt.Println("⚠️ Interception driver not installed. Skipping HID tests.")
		} else if errors.Is(err, winput.ErrDLLLoadFailed) {
			fmt.Println("⚠️ interception.dll not found. Skipping HID tests.")
		} else {
			log.Printf("❌ HID Error: %v", err)
		}
		// Fallback
		winput.SetBackend(winput.BackendMessage)
	} else {
		fmt.Println("✅ HID Move successful")
		w.Type("(HID Input)")
	}

	time.Sleep(1 * time.Second)
	fmt.Println("=== Done ===")
}
