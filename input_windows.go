//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const injectionTag = 0x534B4D50

type keyboardHookData struct {
	VK, Scan, Flags, Time uint32
	Extra                 uintptr
}
type mouseHookData struct {
	Pt                point
	Data, Flags, Time uint32
	Extra             uintptr
}

// INPUT is 40 bytes on Windows amd64, and 28 bytes on 386.
type winInput struct {
	Type uint32
	_    [unsafe.Sizeof(uintptr(0)) - 4]byte
	Data [unsafe.Sizeof(mouseInput{})]byte
}
type mouseInput struct {
	DX, DY            int32
	Data, Flags, Time uint32
	Extra             uintptr
}
type keyboardInput struct {
	VK, Scan    uint16
	Flags, Time uint32
	Extra       uintptr
}

var keyboardCallback = syscall.NewCallback(keyboardProc)
var mouseCallback = syscall.NewCallback(mouseProc)

func (a *application) installHooks() error {
	module, _, _ := getModuleHandle.Call(0)
	var err error
	a.keyboardHook, _, err = setWindowsHookEx.Call(13, keyboardCallback, module, 0)
	if a.keyboardHook == 0 {
		return fmt.Errorf("无法监听键盘：%v", err)
	}
	a.mouseHook, _, err = setWindowsHookEx.Call(14, mouseCallback, module, 0)
	if a.mouseHook == 0 {
		unhookWindowsHook.Call(a.keyboardHook)
		a.keyboardHook = 0
		return fmt.Errorf("无法监听鼠标：%v", err)
	}
	return nil
}

func (a *application) stopHooks() {
	a.emit(a.engine.releaseAll())
	if a.keyboardHook != 0 {
		unhookWindowsHook.Call(a.keyboardHook)
	}
	if a.mouseHook != 0 {
		unhookWindowsHook.Call(a.mouseHook)
	}
	a.keyboardHook, a.mouseHook = 0, 0
	a.flushOutput()
}

func keyboardProc(code int32, wParam uintptr, d *keyboardHookData) uintptr {
	if code >= 0 && app != nil {
		if d.Extra != injectionTag {
			down := wParam == 0x0100 || wParam == 0x0104
			up := wParam == 0x0101 || wParam == 0x0105
			id := keyboardKey(d.VK, d.Scan, d.Flags)
			if id == "lctrl" || id == "rctrl" || id == "lalt" || id == "ralt" {
				app.modifiers[id] = down
			}
			// Reserve an escape hatch even if Ctrl, Alt or F12 itself is mapped.
			ctrl := app.modifiers["lctrl"] || app.modifiers["rctrl"]
			alt := app.modifiers["lalt"] || app.modifiers["ralt"]
			if d.VK == 0x7B && app.emergencyHeld {
				if up {
					app.emergencyHeld = false
					app.releaseReservedF12()
				}
				return 1
			}
			if d.VK == 0x7B && ctrl && alt {
				if down && !app.emergencyHeld {
					app.emergencyHeld = true
					app.releaseReservedF12()
					postMessage.Call(app.hwnd, wmHotkey, 1, 0)
				}
				if up {
					app.emergencyHeld = false
				}
				return 1
			}
			if (down || up) && id != "" && app.handleInput(id, down) {
				return 1
			}
		}
	}
	r, _, _ := callNextHook.Call(0, uintptr(code), wParam, uintptr(unsafe.Pointer(d)))
	return r
}

func (a *application) releaseReservedF12() {
	_, out := a.engine.process("F12", false)
	a.emit(out)
}

func mouseProc(code int32, wParam uintptr, d *mouseHookData) uintptr {
	if code >= 0 && app != nil {
		if d.Extra != injectionTag {
			id, down := mouseButtonEvent(wParam, d.Data)
			if id != "" && app.handleInput(id, down) {
				return 1
			}
		}
	}
	r, _, _ := callNextHook.Call(0, uintptr(code), wParam, uintptr(unsafe.Pointer(d)))
	return r
}

func mouseButtonEvent(message uintptr, data uint32) (string, bool) {
	switch message {
	case 0x0201:
		return "mouse_left", true
	case 0x0202:
		return "mouse_left", false
	case 0x0204:
		return "mouse_right", true
	case 0x0205:
		return "mouse_right", false
	case 0x0207:
		return "mouse_middle", true
	case 0x0208:
		return "mouse_middle", false
	case 0x020B, 0x020C:
		switch data >> 16 {
		case 1:
			return "mouse_x1", message == 0x020B
		case 2:
			return "mouse_x2", message == 0x020B
		}
	}
	return "", false
}

func (a *application) handleInput(id string, down bool) bool {
	if a.closing {
		return false
	}
	// Swallow both edges of a recorded button, including repeated key-downs.
	if a.captureHeld == id {
		if !down {
			a.captureHeld = ""
		}
		return true
	}
	if a.capturing != 0 && !a.capturePending && down {
		// Clicking the visible Cancel button must cancel instead of recording
		// that click as a left-button binding. Other left clicks can be recorded.
		if id == "mouse_left" {
			var cursor point
			var bounds rect
			getCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
			getWindowRect.Call(a.controls[a.capturing], uintptr(unsafe.Pointer(&bounds)))
			if cursor.X >= bounds.Left && cursor.X < bounds.Right && cursor.Y >= bounds.Top && cursor.Y < bounds.Bottom {
				return false
			}
		}
		if id == "escape" {
			a.captureHeld = id
			a.capturePending = true
			postMessage.Call(a.hwnd, wmCaptured, 0, 0)
			return true
		}
		for i, k := range keys {
			if k.ID == id {
				a.captureHeld = id
				a.capturePending = true
				postMessage.Call(a.hwnd, wmCaptured, uintptr(i+1), 0)
				return true
			}
		}
	}
	// Always allow normal interaction with our own configuration window.
	// A previously swallowed source must still receive its matching release.
	if _, held := a.engine.held[id]; !held {
		foreground, _, _ := getForeground.Call()
		// The first click into an inactive configuration window must remain
		// usable even if the left mouse button has been remapped globally.
		if id == "mouse_left" || id == "mouse_right" {
			var cursor point
			getCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
			foregroundAtPoint, _, _ := windowFromPoint.Call(uintptr(uint64(uint32(cursor.X)) | uint64(uint32(cursor.Y))<<32))
			if a.ownWindow(foregroundAtPoint) {
				return false
			}
		}
		if a.ownWindow(foreground) {
			return false
		}
	}
	consume, out := a.engine.process(id, down)
	a.emit(out)
	return consume
}

func (a *application) ownWindow(hwnd uintptr) bool {
	var pid uint32
	getWindowProcess.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	self, _, _ := getCurrentProcessID.Call()
	return uintptr(pid) == self
}

func inputFor(o output) winInput {
	k, _ := findKey(o.Key)
	var input winInput
	if k.Mouse {
		m := (*mouseInput)(unsafe.Pointer(&input.Data[0]))
		m.Extra = injectionTag
		switch k.ID {
		case "mouse_left":
			m.Flags = 0x0002
		case "mouse_right":
			m.Flags = 0x0008
		case "mouse_middle":
			m.Flags = 0x0020
		case "mouse_x1", "mouse_x2":
			m.Flags, m.Data = 0x0080, 1
			if k.ID == "mouse_x2" {
				m.Data = 2
			}
		}
		if !o.Down {
			m.Flags <<= 1
		}
	} else {
		input.Type = 1
		kb := (*keyboardInput)(unsafe.Pointer(&input.Data[0]))
		kb.VK, kb.Extra = k.VK, injectionTag
		// Scan codes work with ordinary applications and most DirectInput clients.
		scan, _, _ := mapVirtualKey.Call(uintptr(k.VK), 0)
		if scan != 0 {
			kb.VK, kb.Scan, kb.Flags = 0, uint16(scan&0xff), 0x0008
		}
		if k.Extended {
			kb.Flags |= 0x0001
		}
		if !o.Down {
			kb.Flags |= 0x0002
		}
	}
	return input
}

func (a *application) emit(out []output) {
	if a.sender == nil || len(out) == 0 {
		return
	}
	if a.hwnd == 0 { // A controller without a window is used by callback tests.
		a.sender.enqueue(out)
		return
	}
	a.pendingOutput = append(a.pendingOutput, out...)
	if !a.outputPosted {
		a.outputPosted = true
		postMessage.Call(a.hwnd, wmSendOutput, 0, 0)
	}
}

func (a *application) flushOutput() {
	a.outputPosted = false
	if a.sender != nil {
		a.sender.enqueue(a.pendingOutput)
	}
	a.pendingOutput = nil
}

// Runs only on inputSender's worker, never on the window / hook thread.
func sendNativeInput(out []output) bool {
	for _, o := range out {
		input := inputFor(o)
		n, _, _ := sendInput.Call(1, uintptr(unsafe.Pointer(&input)), unsafe.Sizeof(input))
		if n != 1 {
			return false
		}
	}
	return true
}
