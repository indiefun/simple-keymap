//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var pickerCallback = syscall.NewCallback(pickerProc)
var listCallback = syscall.NewCallback(listProc)

func listProc(hwnd uintptr, msg uint32, wParam, lParam, id, data uintptr) uintptr {
	if app != nil && (msg == 0x0201 || msg == 0x0203) && int(lParam&0xffff) < app.px(32) {
		hit := struct {
			Pt                   point
			Flags                uint32
			Item, SubItem, Group int32
		}{Pt: point{X: int32(lParam & 0xffff), Y: int32(lParam >> 16 & 0xffff)}}
		row := int32(send(hwnd, lvmFirst+18, 0, uintptr(unsafe.Pointer(&hit))))
		if row >= 0 {
			state := send(hwnd, lvmFirst+44, uintptr(row), 0xF000)
			next := uint32(0x2000)
			if state == 0x2000 {
				next = 0x1000
			}
			item := listItem{State: next, StateMask: 0xF000}
			send(hwnd, lvmFirst+43, uintptr(row), uintptr(unsafe.Pointer(&item)))
			return 0
		}
	}
	r, _, _ := defSubclassProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// Retain native dropdown navigation, screen-reader labels and keyboard input,
// while drawing the closed selection as a large, legible key card.
func pickerProc(hwnd uintptr, msg uint32, wParam, lParam, id, data uintptr) uintptr {
	if app != nil {
		switch msg {
		case wmPaint:
			var ps paintStruct
			dc, _, _ := beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
			app.drawPicker(hwnd, int(id), dc)
			endPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
			return 0
		case 0x0318: // WM_PRINTCLIENT, also used for offscreen layout previews.
			app.drawPicker(hwnd, int(id), wParam)
			return 0
		case wmEraseBkgnd:
			return 1
		case 0x0007, 0x0008, 0x000A: // Focus or enable state changed.
			invalidateRect.Call(hwnd, 0, 0)
		}
	}
	r, _, _ := defSubclassProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func keyCaption(id string) string {
	switch id {
	case "mouse_x1":
		return "鼠标侧键 1"
	case "mouse_x2":
		return "鼠标侧键 2"
	case "mouse_middle":
		return "鼠标中键"
	}
	return keyLabel(id)
}

func keyDetail(id string) string {
	switch id {
	case "mouse_x1":
		return "后退按钮"
	case "mouse_x2":
		return "前进按钮"
	case "mouse_middle":
		return "按下滚轮"
	case "mouse_left", "mouse_right":
		return "鼠标按键"
	}
	return "键盘按键"
}

func (a *application) drawPicker(hwnd uintptr, id int, dc uintptr) {
	var r rect
	getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right)*96/a.dpi, int(r.Bottom)*96/a.dpi
	fillRect.Call(dc, uintptr(unsafe.Pointer(&r)), a.background)
	border, bg := lineColor, surfaceColor
	focus, _, _ := getFocus.Call()
	capturing := a.capturing == idRecordSource && id == idSource || a.capturing == idRecordTarget && id == idTarget
	if focus == hwnd || capturing {
		border = accentColor
	}
	if capturing {
		bg = accentWash
	}
	a.box(dc, 0, 0, w, h, 14, bg, border)
	index := int32(send(hwnd, cbGetCurSel, 0, 0))
	if index < 0 || int(index) >= len(keys) {
		return
	}
	k := keys[index]
	a.box(dc, 16, (h-40)/2, 40, 40, 12, softColor, softColor)
	a.keyIcon(dc, 25, (h-24)/2, k.Mouse, accentColor)
	name, detail := keyCaption(k.ID), keyDetail(k.ID)
	if capturing {
		name, detail = "等待按键…", "现在按一下要使用的键"
	}
	a.text(dc, name, 72, h/2-24, w-116, 28, a.keyFont, ink, 0x20|0x8000)
	a.text(dc, detail, 72, h/2+7, w-116, 22, a.smallFont, muted, 0x20)
	a.line(dc, w-30, h/2-2, w-25, h/2+3, muted, 1)
	a.line(dc, w-25, h/2+3, w-20, h/2-2, muted, 1)
}
