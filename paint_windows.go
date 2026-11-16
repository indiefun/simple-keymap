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
	if len(a.cfg.Rules) > 0 || a.inputFailureMessage != "" {
		a.box(dc, 32, 100, w-64, 52, 12, surfaceColor, lineColor)
		a.text(dc, "映射总开关", 48, 114, 104, 26, a.font, ink, 0x20)
		detail, color := a.stateCopy()
		detailWidth := w - 488
		if a.inputFailureMessage != "" {
			detailWidth = 100
		}
		a.text(dc, detail, 164, 116, detailWidth, 22, a.smallFont, color, 0x20|0x8000)
		a.text(dc, "Ctrl + Alt + F12", w-292, 116, 168, 22, a.smallFont, muted, 0x2|0x20)
	}
	if len(a.cfg.Rules) == 0 {
		a.box(dc, w/2-28, 212, 56, 48, 14, accentWash, accentWash)
		a.text(dc, "→", w/2-28, 212, 56, 48, a.titleFont, accentColor, 0x1|0x4|0x20)
		a.text(dc, "先添加你的第一条映射", 32, 279, w-64, 30, a.headingFont, ink, 0x1|0x20)
		a.text(dc, "告诉轻映：你想按哪个键，让它代替哪个键。", 32, 312, w-64, 24, a.font, muted, 0x1|0x20)
		a.text(dc, "例如，按鼠标侧键，代替按下滚轮。", 32, 442, w-64, 24, a.smallFont, muted, 0x1|0x20)
	} else {
		a.text(dc, "我的映射", 32, mainListTop-44, 100, 28, a.headingFont, ink, 0x20)
		a.text(dc, "在其他应用中生效", 140, mainListTop-38, w-350, 22, a.smallFont, muted, 0x20)
	}
	a.line(dc, 32, h-64, w-32, h-64, lineColor, 1)
	fg := muted
	if a.configSaveError != "" {
		fg = warningColor
	}
	available := w - 64
	if a.deletedRule != nil {
		available -= 108
	}
	if a.configSaveError != "" {
		available = w - 280
	}
	a.text(dc, a.footerCopy(), 32, h-48, available, 24, a.smallFont, fg, 0x20|0x8000)
}

func (a *application) drawButton(d *drawItemStruct) {
	if d == nil {
		return
	}
	id := int(d.ID)
	primary := id == idNew || id == idAdd
	bg, fg, border := canvasColor, accentColor, canvasColor
	if primary {
		bg, fg, border = accentColor, surfaceColor, accentColor
	}
	background := a.background
	if id == idToggle {
		bg, border, background = surfaceColor, surfaceColor, a.whiteBrush
	}
	if id >= rowActionBase {
		bg, border = surfaceColor, lineColor
		background, _, _ = createSolidBrush.Call(a.rowBackground((id - rowActionBase) / 3))
		defer deleteObject.Call(background)
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
	fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), background)
	w, h := int(d.Rect.Right)*96/a.dpi, int(d.Rect.Bottom)*96/a.dpi
	if id == idToggle {
		on := a.cfg.Enabled && a.inputFailureMessage == ""
		a.drawSwitch(d.DC, (w-40)/2, (h-22)/2, 40, 22, on, surfaceColor)
		if d.State&0x10 != 0 {
			a.line(d.DC, 8, h-2, w-2, h-2, accentColor, 1)
		}
		return
	}
	if id >= rowActionBase && (id-rowActionBase)%3 == 0 {
		index := (id - rowActionBase) / 3
		if index < len(a.cfg.Rules) {
			a.drawSwitch(d.DC, (w-40)/2, (h-22)/2, 40, 22, a.cfg.Rules[index].Enabled, a.rowBackground(index))
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
			label = "删除"
		}
	}
	a.textRect(d.DC, label, &d.Rect, a.smallFont, fg, 0x1|0x4|0x20)
}

func (a *application) rowBackground(index int) uintptr {
	if send(a.controls[idList], lvmFirst+44, uintptr(index), 2) != 0 {
		return softColor
	}
	return surfaceColor
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
	bg := a.rowBackground(i)
	a.box(d.Draw.DC, 4, top+3, w-26, h-6, 12, bg, lineColor)
	widths := a.listWidths()
	row := a.cfg.Rules[i]
	fg := ink
	if !row.Enabled {
		fg = muted
	}
	sourceX, targetX := widths[0]+8, widths[0]+widths[1]+10
	a.text(d.Draw.DC, keyCaption(row.Source), sourceX, top+17, widths[1]-42, 26, a.font, fg, 0x20|0x8000)
	a.text(d.Draw.DC, keyDetail(row.Source), sourceX, top+47, widths[1]-42, 20, a.smallFont, muted, 0x20|0x8000)
	a.text(d.Draw.DC, "→", targetX-34, top+24, 24, 26, a.font, muted, 0x20)
	a.text(d.Draw.DC, keyCaption(row.Target), targetX, top+17, widths[2]-20, 26, a.font, fg, 0x20|0x8000)
	a.text(d.Draw.DC, keyDetail(row.Target), targetX, top+47, widths[2]-20, 20, a.smallFont, muted, 0x20|0x8000)
	return 0x4
}
