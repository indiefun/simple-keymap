//go:build windows

package main

import "unsafe"

var (
	ink          = rgb(31, 43, 47)
	muted        = rgb(112, 123, 127)
	canvasColor  = rgb(247, 248, 247)
	surfaceColor = rgb(255, 255, 255)
	softColor    = rgb(240, 244, 242)
	lineColor    = rgb(223, 229, 225)
	accentColor  = rgb(23, 108, 88)
	accentWash   = rgb(231, 242, 235)
	warningColor = rgb(154, 92, 32)
)

func (a *application) box(dc uintptr, x, y, w, h, radius int, color, border uintptr) {
	b, _, _ := createSolidBrush.Call(color)
	p, _, _ := createPen.Call(0, uintptr(max(1, a.px(1))), border)
	oldB, _, _ := selectObject.Call(dc, b)
	oldP, _, _ := selectObject.Call(dc, p)
	roundRect.Call(dc, intArg(a.px(x)), intArg(a.px(y)), intArg(a.px(x+w)), intArg(a.px(y+h)), intArg(a.px(radius)), intArg(a.px(radius)))
	selectObject.Call(dc, oldP)
	selectObject.Call(dc, oldB)
	deleteObject.Call(p)
	deleteObject.Call(b)
}

func (a *application) line(dc uintptr, x1, y1, x2, y2 int, color uintptr, width int) {
	p, _, _ := createPen.Call(0, uintptr(max(1, a.px(width))), color)
	old, _, _ := selectObject.Call(dc, p)
	moveTo.Call(dc, intArg(a.px(x1)), intArg(a.px(y1)), 0)
	lineTo.Call(dc, intArg(a.px(x2)), intArg(a.px(y2)))
	selectObject.Call(dc, old)
	deleteObject.Call(p)
}

func (a *application) text(dc uintptr, text string, x, y, w, h int, font, color uintptr, flags uintptr) {
	r := rect{int32(a.px(x)), int32(a.px(y)), int32(a.px(x + w)), int32(a.px(y + h))}
	a.textRect(dc, text, &r, font, color, flags)
}

func (a *application) textRect(dc uintptr, text string, r *rect, font, color, flags uintptr) {
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, color)
	setBkMode.Call(dc, 1)
	drawText.Call(dc, uintptr(unsafe.Pointer(utf16(text))), ^uintptr(0), uintptr(unsafe.Pointer(r)), flags|0x800)
	selectObject.Call(dc, old)
}

func (a *application) keyIcon(dc uintptr, x, y int, mouse bool, color uintptr) {
	if mouse {
		a.box(dc, x+4, y, 16, 24, 12, surfaceColor, color)
		a.line(dc, x+12, y+4, x+12, y+10, color, 2)
		a.line(dc, x+4, y+12, x+20, y+12, color, 1)
	} else {
		a.box(dc, x, y+3, 24, 18, 5, surfaceColor, color)
		for row := 0; row < 2; row++ {
			for col := 0; col < 3; col++ {
				a.line(dc, x+5+col*6, y+8+row*5, x+7+col*6, y+8+row*5, color, 1)
			}
		}
		a.line(dc, x+7, y+17, x+17, y+17, color, 1)
	}
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
	w, h := a.clientSize(a.hwnd)
	drawIconEx.Call(dc, uintptr(a.px(32)), uintptr(a.px(24)), a.brandIcon, uintptr(a.px(40)), uintptr(a.px(40)), 0, 0, 3)
	a.text(dc, "轻映", 84, 22, 200, 30, a.titleFont, ink, 0x20)
	a.text(dc, "免费开源 · 键盘与鼠标改键", 85, 55, 350, 22, a.smallFont, muted, 0x20)
	a.line(dc, 32, 86, w-32, 86, lineColor, 1)
	title, detail, color, _ := a.stateCopy()
	a.text(dc, title, 32, 104, w-220, 30, a.headingFont, color, 0x20)
	a.text(dc, detail, 32, 140, w-64, 24, a.smallFont, muted, 0x20|0x8000)
	if len(a.cfg.Rules) == 0 {
		a.box(dc, w/2-28, 212, 56, 48, 14, accentWash, accentWash)
		a.text(dc, "→", w/2-28, 212, 56, 48, a.titleFont, accentColor, 0x1|0x4|0x20)
		a.text(dc, "先添加你的第一条映射", 32, 279, w-64, 30, a.headingFont, ink, 0x1|0x20)
		a.text(dc, "告诉轻映：你想按哪个键，让它代替哪个键。", 32, 312, w-64, 24, a.font, muted, 0x1|0x20)
		a.text(dc, "例如，按鼠标侧键，代替按下滚轮。", 32, 442, w-64, 24, a.smallFont, muted, 0x1|0x20)
	} else {
		a.text(dc, "轻映自己的窗口内不改键，方便你修改配置。", 32, 170, w-64, 22, a.smallFont, muted, 0x20)
		if a.successMessage != "" {
			a.box(dc, 32, 198, w-64, 66, 12, accentWash, accentWash)
			a.text(dc, a.successMessage, 48, 207, w-340, 23, a.font, accentColor, 0x20|0x8000)
			note := "切换到其他应用试按；最小化后仍会运行。"
			if !a.successActive {
				note = "规则已保存，恢复全部或开启该条规则后再试按。"
			}
			a.text(dc, note, 48, 236, w-96, 22, a.smallFont, muted, 0x20|0x8000)
		}
		a.text(dc, "我的映射", 32, a.listTop()-44, 300, 28, a.headingFont, ink, 0x20)
	}
	if a.configSaveError != "" {
		a.text(dc, "当前更改尚未保存，可重试。", 32, h-84, 260, 24, a.smallFont, warningColor, 0x20)
	} else if a.status != "" {
		a.text(dc, a.status, 32, h-84, w-180, 24, a.smallFont, muted, 0x20|0x8000)
	}
	a.line(dc, 32, h-54, w-32, h-54, lineColor, 1)
	footer := "配置自动保存在本机"
	fg := muted
	if a.configSaveError != "" {
		footer = "更改仅本次生效，尚未保存"
		fg = warningColor
	}
	a.text(dc, footer, 32, h-42, 260, 24, a.smallFont, fg, 0x20)
	a.text(dc, "Ctrl + Alt + F12 · 暂停 / 恢复", w-470, h-42, 250, 24, a.smallFont, muted, 0x20)
	a.text(dc, "最小化继续 · 关闭退出", w-224, h-42, 192, 24, a.smallFont, muted, 0x2|0x20)
}

func (a *application) drawButton(d *drawItemStruct) {
	if d == nil {
		return
	}
	id := int(d.ID)
	primary := id == idNew || id == idAdd || id == idTryNow
	bg, fg, border := canvasColor, accentColor, canvasColor
	if primary {
		bg, fg, border = accentColor, surfaceColor, accentColor
	}
	if id == idToggle {
		bg, border = surfaceColor, lineColor
	}
	if id >= rowActionBase {
		bg, border = surfaceColor, surfaceColor
		index, action := (id-rowActionBase)/3, (id-rowActionBase)%3
		if index < len(a.cfg.Rules) && action == 0 {
			if a.cfg.Rules[index].Enabled {
				bg, border = accentWash, accentWash
			} else {
				bg, fg, border = softColor, muted, softColor
			}
		}
	}
	if d.State&1 != 0 {
		if primary {
			bg, border = rgb(16, 83, 66), rgb(16, 83, 66)
		} else {
			bg, border = accentWash, accentWash
		}
	}
	if d.State&4 != 0 {
		bg, fg, border = canvasColor, rgb(174, 182, 177), canvasColor
	}
	if d.State&0x10 != 0 {
		border = accentColor
	}
	fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), a.background)
	w, h := int(d.Rect.Right)*96/a.dpi, int(d.Rect.Bottom)*96/a.dpi
	if id >= rowActionBase && (id-rowActionBase)%3 == 0 {
		index := (id - rowActionBase) / 3
		if index < len(a.cfg.Rules) {
			on := a.cfg.Rules[index].Enabled
			track, knob := rgb(174, 187, 178), 5
			if on {
				track, knob = accentColor, 17
			}
			a.box(d.DC, 2, (h-18)/2, 30, 18, 18, track, track)
			a.box(d.DC, knob, (h-12)/2, 12, 12, 12, surfaceColor, surfaceColor)
			a.text(d.DC, ruleEnabledText(a.cfg.Rules[index]), 38, 0, w-38, h, a.smallFont, fg, 0x4|0x20)
			if d.State&0x10 != 0 {
				a.line(d.DC, 2, h-2, w-2, h-2, accentColor, 1)
			}
			return
		}
	}
	a.box(d.DC, 0, 0, w, h, 9, bg, border)
	label := windowText(d.Hwnd)
	if id >= rowActionBase {
		if (id-rowActionBase)%3 == 1 {
			label = "修改"
		} else {
			label = "⋯"
		}
	}
	a.textRect(d.DC, label, &d.Rect, a.smallFont, fg, 0x1|0x4|0x20)
}

func (a *application) drawList(d *listCustomDraw) uintptr {
	if d.Draw.Stage == 1 {
		return 0x20
	}
	if d.Draw.Stage != 0x10001 {
		return 0
	}
	i := int(d.Draw.Item)
	if i < 0 || i >= len(a.cfg.Rules) {
		return 0
	}
	var r rect
	send(a.controls[idList], lvmFirst+14, uintptr(i), uintptr(unsafe.Pointer(&r)))
	w, _ := a.clientSize(a.controls[idList])
	top := int(r.Top) * 96 / a.dpi
	h := int(r.Bottom-r.Top) * 96 / a.dpi
	fillRect.Call(d.Draw.DC, uintptr(unsafe.Pointer(&r)), a.whiteBrush)
	bg := surfaceColor
	if send(a.controls[idList], lvmFirst+44, uintptr(i), 2) != 0 {
		bg = softColor
	}
	a.box(d.Draw.DC, 4, top+3, w-26, h-6, 12, bg, lineColor)
	widths := a.listWidths()
	row := a.cfg.Rules[i]
	fg := ink
	if !row.Enabled {
		fg = muted
	}
	a.text(d.Draw.DC, keyCaption(row.Source), 20, top+17, widths[0]-50, 26, a.font, fg, 0x20|0x8000)
	a.text(d.Draw.DC, keyDetail(row.Source), 20, top+47, widths[0]-50, 20, a.smallFont, muted, 0x20|0x8000)
	a.text(d.Draw.DC, "→", widths[0]-24, top+24, 24, 26, a.font, muted, 0x20)
	a.text(d.Draw.DC, keyCaption(row.Target), widths[0]+10, top+17, widths[1]-20, 26, a.font, fg, 0x20|0x8000)
	a.text(d.Draw.DC, keyDetail(row.Target), widths[0]+10, top+47, widths[1]-20, 20, a.smallFont, muted, 0x20|0x8000)
	return 0x4
}
