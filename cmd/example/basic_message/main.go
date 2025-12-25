package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/rpdg/winput"
)

func main() {
	fmt.Println("=== winput: Basic Message Backend Example ===")
	fmt.Println("This example uses PostMessageW. It does NOT require focus or mouse movement.")

	// 1. Enable DPI Awareness
	winput.EnablePerMonitorDPI()

	// 2. Find Window
	// Target: Notepad
	w, err := winput.FindByClass("Notepad")
	if err != nil {
		log.Println("❌ Notepad not found. Please open Notepad to run this test.")
		return
	}
	fmt.Printf("✅ Found Notepad handle: %x\n", w.HWND)

	// 3. Input Operations
	// Type text
	fmt.Println("👉 Typing text...")
	if err := w.Type("Hello from winput (Message Backend)!\n"); err != nil {
		if errors.Is(err, winput.ErrWindowNotVisible) {
			log.Fatal("❌ Window is minimized. Please restore it.")
		}
		log.Fatal(err)
	}

	// Press Hotkey (Select All: Ctrl+A)
	fmt.Println("👉 Testing Hotkey (Ctrl+A)...")
	winput.PressHotkey(winput.KeyCtrl, winput.KeyA)
	time.Sleep(500 * time.Millisecond)

	// Mouse Click (Right Click context menu)
	fmt.Println("👉 Testing Right Click...")
	w.ClickRight(100, 100)

	fmt.Println("=== Done ===")
}
