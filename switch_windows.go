//go:build windows

package main

import (
	"math"
	"unsafe"
)

// Render the capsule and circular thumb at the actual display resolution.
// Coverage at their edges avoids the jagged outline of small GDI RoundRects.
func (a *application) drawSwitch(dc uintptr, x, y, w, h int, on bool, background uintptr) {
	width, height := a.px(w), a.px(h)
	if width <= 0 || height <= 0 {
		return
	}
	track := rgb(174, 187, 178)
	if on {
		track = accentColor
	}
	radius := float64(height) / 2
	inset := 3 * float64(a.dpi) / 96
	thumbRadius := radius - inset
	thumbX := radius
	if on {
		thumbX = float64(width) - radius
	}
	pixels := make([]uint32, width*height)
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			cx, cy := float64(px)+0.5, float64(py)+0.5
			nearestX := max(radius, min(float64(width)-radius, cx))
			coverage := max(0, min(1, radius+0.5-math.Hypot(cx-nearestX, cy-radius)))
			thumb := max(0, min(1, thumbRadius+0.5-math.Hypot(cx-thumbX, cy-radius)))
			var color uint32
			for channel := 0; channel < 3; channel++ {
				bg := float64(background >> (channel * 8) & 255)
				fg := float64(track >> (channel * 8) & 255)
				value := bg + (fg-bg)*coverage
				value += (255 - value) * thumb
				color |= uint32(math.Round(value)) << ((2 - channel) * 8) // DIB uses BGR.
			}
			pixels[py*width+px] = color
		}
	}
	info := struct {
		Size                   uint32
		Width, Height          int32
		Planes, BitCount       uint16
		Compression, SizeImage uint32
		XPels, YPels           int32
		Used, Important        uint32
	}{Size: 40, Width: int32(width), Height: -int32(height), Planes: 1, BitCount: 32}
	setDIBitsToDevice.Call(dc, intArg(a.px(x)), intArg(a.px(y)), uintptr(width), uintptr(height), 0, 0, 0, uintptr(height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&info)), 0)
}
