package screen

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"

	"github.com/rpdg/winput/window"
)

// GDI Constants & Types
const (
	SRCCOPY        = 0x00CC0020
	DIB_RGB_COLORS = 0
	BI_RGB         = 0
)

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// CaptureVirtualDesktop captures the entire virtual desktop using the efficient CreateDIBSection method.
// Returns an *image.RGBA ready for OpenCV.
//
// Prerequisites:
// The process MUST be Per-Monitor DPI Aware (V1 or V2). Call winput.EnablePerMonitorDPI() first.
func CaptureVirtualDesktop() (*image.RGBA, error) {
	// 1. DPI Awareness Check (Strict)
	if !window.IsPerMonitorDPIAware() {
		return nil, fmt.Errorf("process is not Per-Monitor DPI Aware; call winput.EnablePerMonitorDPI() first to ensure accurate coordinates")
	}

	// 2. Get Virtual Desktop Bounds
	x, _, _ := window.ProcGetSystemMetrics.Call(76) // SM_XVIRTUALSCREEN
	y, _, _ := window.ProcGetSystemMetrics.Call(77) // SM_YVIRTUALSCREEN
	w, _, _ := window.ProcGetSystemMetrics.Call(78) // SM_CXVIRTUALSCREEN
	h, _, _ := window.ProcGetSystemMetrics.Call(79) // SM_CYVIRTUALSCREEN
	
	width := int32(w)
	height := int32(h)

	// Safety check for huge resolutions
	// 4 bytes per pixel. Limit to approx 500MB (e.g. 11000 x 11000)
	if int64(width)*int64(height)*4 > 1024*1024*500 {
		return nil, fmt.Errorf("resolution too large for single capture: %dx%d (exceeds 500MB)", width, height)
	}

	// 3. Create DCs
	// GetDC(0) returns the DC for the entire virtual screen
	hScreenDC, _, _ := window.ProcGetDC.Call(0)
	if hScreenDC == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer window.ProcReleaseDC.Call(0, hScreenDC)

	hMemDC, _, _ := window.ProcCreateCompatibleDC.Call(hScreenDC)
	if hMemDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer window.ProcDeleteDC.Call(hMemDC)

	// 4. Create DIB Section
	// We use a top-down DIB (negative height) so (0,0) is top-left.
	bmi := BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		BiWidth:       width,
		BiHeight:      -height, // Negative for Top-Down
		BiPlanes:      1,
		BiBitCount:    32, // BGRA
		BiCompression: BI_RGB,
	}
	
	var ppvBits uintptr // Pointer to the raw pixel data
	hBitmap, _, _ := window.ProcCreateDIBSection.Call(
		hMemDC,
		uintptr(unsafe.Pointer(&bmi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&ppvBits)),
		0, 0,
	)
	if hBitmap == 0 {
		return nil, fmt.Errorf("CreateDIBSection failed")
	}
	defer window.ProcDeleteObject.Call(hBitmap)

	// 5. Select Bitmap into MemDC
	oldObj, _, _ := window.ProcSelectObject.Call(hMemDC, hBitmap)
	if oldObj == 0 {
		return nil, fmt.Errorf("SelectObject failed")
	}
	// Restore old object before deleting MemDC
	defer window.ProcSelectObject.Call(hMemDC, oldObj)

	// 6. BitBlt: Copy Screen -> Memory -> DIB
	// Because hBitmap is selected in hMemDC, this writes directly to ppvBits!
	ret, _, _ := window.ProcBitBlt.Call(
		hMemDC,
		0, 0, uintptr(width), uintptr(height),
		hScreenDC,
		uintptr(int32(x)), uintptr(int32(y)), // Source coords on virtual screen
		SRCCOPY,
	)
	if ret == 0 {
		return nil, fmt.Errorf("BitBlt failed")
	}

	// 7. Convert to Go Image
	// ppvBits points to the raw BGRA data.
	totalBytes := int(width) * int(height) * 4
	
	// Create a Go slice backed by the C array (Unsafe but efficient for reading)
	// Warning: We must copy this data because hBitmap will be destroyed when we return.
	srcBytes := unsafe.Slice((*byte)(unsafe.Pointer(ppvBits)), totalBytes)
	
	// Alloc new buffer for Go image
	dstBytes := make([]byte, totalBytes)

	// BGRA -> RGBA conversion loop
	for i := 0; i < totalBytes; i += 4 {
		b := srcBytes[i]
		g := srcBytes[i+1]
		r := srcBytes[i+2]
		
		dstBytes[i]   = r
		dstBytes[i+1] = g
		dstBytes[i+2] = b
		// Force Alpha to opaque (255) for OpenCV template matching stability.
		// DWM windows might have transparent pixels, but for automation matching,
		// we usually want solid colors.
		dstBytes[i+3] = 255 
	}

	img := &image.RGBA{
		Pix:    dstBytes,
		Stride: int(width * 4),
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	return img, nil
}
