// Package winput provides a Windows input automation library focused on background operation.
// It supports window discovery, coordinate conversion (screen/client), DPI awareness,
// and input injection using Window Messages (PostMessage).
//
// Key Features:
// - Object-centric API (Window struct)
// - Background mouse and keyboard input
// - DPI aware coordinate handling
// - Explicit error handling
//
// Example:
//
//  w, err := winput.FindByTitle("Untitled - Notepad")
//  if err != nil {
//      panic(err)
//  }
//
//  w.Click(100, 100)
//  w.Type("Hello World")
//  w.Press(winput.KeyEnter)
//
package winput
