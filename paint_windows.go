//go:build windows

package main

import (
	"fmt"
	"unsafe"
)

var (
	ink   = rgb(30, 42, 56)
	muted = rgb(108, 121, 137)
	teal  = rgb(14, 116, 108)
)

func (a *application) box(dc uintptr, x, y, w, h, radius int, color, border uintptr) {
	b, _, _ := createSolidBrush.Call(color)
	p, _, _ := createPen.Call(0, uintptr(a.px(1)), border)
	oldB, _, _ := selectObject.Call(dc, b)
	oldP, _, _ := selectObject.Call(dc, p)
	roundRect.Call(dc, intArg(a.px(x)), intArg(a.px(y)), intArg(a.px(x+w)), intArg(a.px(y+h)), intArg(a.px(radius)), intArg(a.px(radius)))
	selectObject.Call(dc, oldP)
	selectObject.Call(dc, oldB)
	deleteObject.Call(p)
	deleteObject.Call(b)
}

func (a *application) text(dc uintptr, text string, x, y, w, h int, font, color uintptr, flags uintptr) {
	r := rect{int32(a.px(x)), int32(a.px(y)), int32(a.px(x + w)), int32(a.px(y + h))}
	a.textRect(dc, text, &r, font, color, flags)
}

func (a *application) textRect(dc uintptr, text string, r *rect, font, color, flags uintptr) {
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, color)
	setBkMode.Call(dc, 1)
	drawText.Call(dc, uintptr(unsafe.Pointer(utf16(text))), ^uintptr(0), uintptr(unsafe.Pointer(r)), flags|0x800) // DT_NOPREFIX.
	selectObject.Call(dc, old)
}

func (a *application) paint(hwnd uintptr) {
	var ps paintStruct
	dc, _, _ := beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer endPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	var client rect
	getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	buffer, _, _ := createCompatibleDC.Call(dc)
	bmp, _, _ := createCompatibleBitmap.Call(dc, uintptr(client.Right), uintptr(client.Bottom))
	old, _, _ := selectObject.Call(buffer, bmp)
	fillRect.Call(buffer, uintptr(unsafe.Pointer(&client)), a.background)
	a.paintContent(buffer)
	bitBlt.Call(dc, 0, 0, uintptr(client.Right), uintptr(client.Bottom), buffer, 0, 0, 0x00CC0020)
	selectObject.Call(buffer, old)
	deleteObject.Call(bmp)
	deleteDC.Call(buffer)
}

func (a *application) paintContent(dc uintptr) {
	a.box(dc, 36, 30, 48, 48, 14, teal, teal)
	a.text(dc, "映", 36, 30, 48, 48, a.headingFont, rgb(255, 255, 255), 0x1|0x4|0x20)
	a.text(dc, "轻映", 100, 25, 160, 38, a.titleFont, ink, 0x20)
	a.text(dc, "让常用操作，更顺手。", 101, 67, 300, 22, a.smallFont, muted, 0x20)
	status, color := "映射已暂停", muted
	if a.cfg.Enabled {
		status, color = "映射已开启", teal
	}
	if a.capturing != 0 {
		status, color = "正在录入按键", teal
	}
	a.text(dc, status, 504, 48, 150, 24, a.font, color, 0x2|0x20)
	a.box(dc, 28, 120, 804, 232, 16, rgb(255, 255, 255), rgb(226, 232, 238))
	a.text(dc, "添加映射", 48, 138, 180, 27, a.headingFont, ink, 0x20)
	a.text(dc, "选好两个按键，让它们换一种用法。", 48, 171, 600, 22, a.smallFont, muted, 0x20)
	a.text(dc, "原按键", 48, 196, 340, 23, a.smallFont, muted, 0x20)
	a.text(dc, "映射为", 464, 196, 340, 23, a.smallFont, muted, 0x20)
	a.text(dc, "→", 404, 224, 48, 30, a.headingFont, muted, 0x1|0x20)
	a.text(dc, "长按会保持按下，松开后自动释放。", 48, 299, 590, 24, a.smallFont, muted, 0x20)
	a.text(dc, fmt.Sprintf("我的映射  ·  %d", len(a.cfg.Rules)), 36, 374, 330, 28, a.headingFont, ink, 0x20)
	a.text(dc, "勾选启用，取消勾选暂停", 500, 380, 324, 22, a.smallFont, muted, 0x2|0x20)
	a.box(dc, 28, 412, 804, 196, 16, rgb(255, 255, 255), rgb(226, 232, 238))
	if len(a.cfg.Rules) == 0 {
		a.text(dc, "还没有映射", 48, 468, 764, 30, a.headingFont, muted, 0x1|0x20)
		a.text(dc, "从上方添加第一条，立即开始使用。", 48, 508, 764, 26, a.smallFont, muted, 0x1|0x20)
	}
	a.text(dc, "自动保存 · 配置窗口内不执行映射", 170, 630, 520, 22, a.smallFont, muted, 0x20)
	a.text(dc, a.status, 36, 670, 788, 22, a.smallFont, ink, 0x20|0x8000)
	a.text(dc, "Ctrl + Alt + F12 暂停 / 恢复     ·     最小化后继续生效，关闭窗口即退出", 36, 698, 788, 20, a.smallFont, muted, 0x20)
}

func (a *application) drawButton(d *drawItemStruct) {
	if d == nil {
		return
	}
	id := int(d.ID)
	text := map[int]string{idToggle: "暂停映射", idRecordSource: "录入", idRecordTarget: "录入", idAdd: "+ 添加映射", idDelete: "删除所选", idHelp: "使用说明"}[id]
	if id == idToggle && !a.cfg.Enabled {
		text = "开启映射"
	}
	if id == a.capturing {
		text = "取消"
	}
	bg, fg, border := rgb(255, 255, 255), ink, rgb(215, 223, 232)
	primary := id == idAdd || id == idToggle && !a.cfg.Enabled
	if primary {
		bg, fg, border = teal, rgb(255, 255, 255), teal
	}
	if d.State&0x1 != 0 { // ODS_SELECTED.
		if primary {
			bg, border = rgb(9, 88, 82), rgb(9, 88, 82)
		} else {
			bg = rgb(230, 237, 242)
		}
	}
	if d.State&0x4 != 0 {
		bg, fg, border = rgb(240, 243, 246), rgb(164, 173, 184), rgb(228, 233, 238)
	}
	if d.State&0x10 != 0 {
		border = teal
	} // Visible keyboard focus.
	fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), a.background)
	b, _, _ := createSolidBrush.Call(bg)
	p, _, _ := createPen.Call(0, uintptr(a.px(1)), border)
	oldB, _, _ := selectObject.Call(d.DC, b)
	oldP, _, _ := selectObject.Call(d.DC, p)
	roundRect.Call(d.DC, uintptr(d.Rect.Left), uintptr(d.Rect.Top), uintptr(d.Rect.Right), uintptr(d.Rect.Bottom), uintptr(a.px(10)), uintptr(a.px(10)))
	selectObject.Call(d.DC, oldP)
	selectObject.Call(d.DC, oldB)
	deleteObject.Call(p)
	deleteObject.Call(b)
	a.textRect(d.DC, text, &d.Rect, a.font, fg, 0x1|0x4|0x20)
}
