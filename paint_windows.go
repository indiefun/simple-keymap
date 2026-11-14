//go:build windows

package main

import (
	"fmt"
	"unsafe"
)

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

func (a *application) stateCopy() (string, string, uintptr, uintptr) {
	if a.inputFailureMessage != "" {
		return "输入已暂停", "输入服务遇到问题，点击右侧查看原因。", warningColor, rgb(251, 240, 224)
	}
	if a.capturing != 0 {
		name := "原按键"
		if a.capturing == idRecordTarget {
			name = "目标按键"
		}
		return "正在录入" + name, "按一下键盘或鼠标按键，Esc 取消。", accentColor, accentWash
	}
	if !a.cfg.Enabled {
		return "映射已暂停", "规则已保留，打开总开关即可恢复。", muted, rgb(237, 240, 238)
	}
	count := 0
	for _, r := range a.cfg.Rules {
		if r.Enabled {
			count++
		}
	}
	if count == 0 {
		if len(a.cfg.Rules) == 0 {
			return "添加第一条映射", "选好两个按键，点击「添加映射」。", accentColor, accentWash
		}
		return "规则均已暂停", "勾选下方规则，即可恢复对应映射。", muted, rgb(237, 240, 238)
	}
	if a.configurationActive {
		return "配置窗口内暂不执行", fmt.Sprintf("%d 条规则待用 · 切到其他应用自动生效", count), accentColor, accentWash
	}
	return "映射已开启", fmt.Sprintf("%d 条规则已启用", count), accentColor, accentWash
}

func (a *application) paintContent(dc uintptr) {
	drawIconEx.Call(dc, uintptr(a.px(30)), uintptr(a.px(24)), a.brandIcon, uintptr(a.px(44)), uintptr(a.px(44)), 0, 0, 3)
	a.text(dc, "轻映", 86, 22, 180, 32, a.titleFont, ink, 0x20)
	a.text(dc, "免费开源 · 键盘与鼠标改键", 87, 56, 320, 22, a.smallFont, muted, 0x20)
	title, detail, color, wash := a.stateCopy()
	a.box(dc, 32, 88, 816, 60, 14, wash, wash)
	a.box(dc, 50, 108, 7, 7, 7, color, color)
	a.text(dc, title, 68, 98, 570, 25, a.font, color, 0x20)
	a.text(dc, detail, 68, 124, 600, 21, a.smallFont, muted, 0x20|0x8000)
	heading := "新建映射"
	if a.editing {
		heading = "修改映射"
	}
	a.text(dc, heading, 32, 174, 240, 28, a.headingFont, ink, 0x20)
	a.text(dc, "按下这个键", 32, 212, 270, 22, a.font, muted, 0x20)
	a.text(dc, "变为这个键", 480, 212, 270, 22, a.font, muted, 0x20)
	a.box(dc, 423, 264, 34, 34, 34, softColor, softColor)
	a.line(dc, 433, 281, 448, 281, accentColor, 1)
	a.line(dc, 443, 276, 448, 281, accentColor, 1)
	a.line(dc, 443, 286, 448, 281, accentColor, 1)
	hint, _ := a.draftHint()
	a.text(dc, hint, 32, 350, 512, 24, a.smallFont, muted, 0x20|0x8000)
	a.line(dc, 32, 393, 848, 393, lineColor, 1)
	a.text(dc, "我的映射", 32, 406, 140, 28, a.headingFont, ink, 0x20)
	a.box(dc, 120, 408, 30, 22, 10, rgb(232, 237, 233), rgb(232, 237, 233))
	a.text(dc, fmt.Sprint(len(a.cfg.Rules)), 120, 408, 30, 22, a.smallFont, muted, 0x1|0x4|0x20)
	a.box(dc, 32, 446, 816, 186, 14, surfaceColor, lineColor)
	if len(a.cfg.Rules) == 0 {
		a.box(dc, 416, 480, 48, 42, 12, softColor, softColor)
		a.keyIcon(dc, 428, 489, false, muted)
		a.text(dc, "选两个按键，添加你的第一条映射", 64, 536, 752, 28, a.font, ink, 0x1|0x20)
		a.text(dc, "点击「添加映射」，再切换到其他应用试按。", 64, 570, 752, 24, a.smallFont, muted, 0x1|0x20)
	} else {
		a.text(dc, "启用 / 原按键", 58, 455, 320, 22, a.smallFont, muted, 0x20)
		a.text(dc, "映射为", 432, 455, 280, 22, a.smallFont, muted, 0x20)
		a.text(dc, "状态", 736, 455, 90, 22, a.smallFont, muted, 0x20)
		a.line(dc, 48, 478, 832, 478, lineColor, 1)
	}
	footerColor := muted
	if a.configSaveError != "" {
		footerColor = warningColor
	}
	footerWidth := 408
	if a.configSaveError != "" {
		footerWidth = 318
	}
	a.text(dc, a.footerCopy(), 32, 648, footerWidth, 22, a.smallFont, footerColor, 0x20|0x8000)
	a.text(dc, "Ctrl + Alt + F12   暂停 / 恢复", 534, 648, 314, 22, a.smallFont, muted, 0x2|0x20)
}

func (a *application) drawButton(d *drawItemStruct) {
	if d == nil {
		return
	}
	id := int(d.ID)
	text := windowText(d.Hwnd)
	if id == a.capturing {
		text = "取消录入"
	}
	if id == idAdd && a.editing {
		text = "保存修改"
	}
	bg, fg, border := canvasColor, muted, canvasColor
	if id == idAdd {
		bg, fg, border = accentColor, surfaceColor, accentColor
	}
	if id == idRecordSource || id == idRecordTarget {
		fg = accentColor
	}
	if d.State&0x1 != 0 {
		if id == idAdd {
			bg, border = rgb(16, 83, 66), rgb(16, 83, 66)
		} else {
			bg, border = softColor, softColor
		}
	}
	if d.State&0x4 != 0 {
		bg, fg, border = canvasColor, rgb(174, 182, 177), canvasColor
	}
	if id == idAdd && d.State&0x4 != 0 {
		bg, border = rgb(229, 234, 230), rgb(229, 234, 230)
	}
	if d.State&0x10 != 0 {
		border = accentColor
	}
	if id == idToggle {
		_, _, _, wash := a.stateCopy()
		b, _, _ := createSolidBrush.Call(wash)
		fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), b)
		deleteObject.Call(b)
		if a.inputFailureMessage != "" {
			bg, border = rgb(255, 249, 240), rgb(243, 222, 193)
		}
	} else {
		fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), a.background)
	}
	w, h := int(d.Rect.Right)*96/a.dpi, int(d.Rect.Bottom)*96/a.dpi
	if id == idToggle && a.inputFailureMessage == "" {
		_, _, _, wash := a.stateCopy()
		b, _, _ := createSolidBrush.Call(wash)
		fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), b)
		deleteObject.Call(b)
		a.text(d.DC, "总开关", 0, 0, w-54, h, a.smallFont, muted, 0x4|0x20)
		track, knob := rgb(182, 192, 186), w-44
		if a.cfg.Enabled {
			track, knob = accentColor, w-27
		}
		if d.State&0x4 != 0 {
			track = rgb(186, 203, 193)
		}
		a.box(d.DC, w-47, (h-24)/2, 44, 24, 24, track, track)
		a.box(d.DC, knob, (h-18)/2, 18, 18, 18, surfaceColor, surfaceColor)
		if d.State&0x10 != 0 {
			a.line(d.DC, 0, h-1, w-54, h-1, accentColor, 1)
		}
		return
	}
	if id == idToggle {
		text, fg = "查看原因", warningColor
	}
	a.box(d.DC, 0, 0, w, h, 10, bg, border)
	font := a.font
	if id != idAdd {
		font = a.smallFont
	}
	a.textRect(d.DC, text, &d.Rect, font, fg, 0x1|0x4|0x20)
}

func (a *application) drawList(d *listCustomDraw) uintptr {
	const itemPrepaint = 0x10001
	switch d.Draw.Stage {
	case 1:
		return 0x20
	case itemPrepaint:
		i := int(d.Draw.Item)
		if i < 0 || i >= len(a.cfg.Rules) {
			return 0
		}
		// Classic common controls may omit rc in an item draw notification.
		r := rect{}
		send(a.controls[idList], lvmFirst+14, uintptr(i), uintptr(unsafe.Pointer(&r)))
		bg := surfaceColor
		selected := send(a.controls[idList], lvmFirst+44, uintptr(i), 2) != 0
		if selected {
			bg = softColor
		}
		b, _, _ := createSolidBrush.Call(bg)
		fillRect.Call(d.Draw.DC, uintptr(unsafe.Pointer(&r)), b)
		deleteObject.Call(b)
		x, top := int(r.Left)*96/a.dpi, int(r.Top)*96/a.dpi
		h := int(r.Bottom-r.Top) * 96 / a.dpi
		row := a.cfg.Rules[i]
		checkBG, checkBorder := surfaceColor, rgb(171, 185, 176)
		if row.Enabled {
			checkBG, checkBorder = accentColor, accentColor
		}
		a.box(d.Draw.DC, x+6, top+(h-14)/2, 14, 14, 4, checkBG, checkBorder)
		if row.Enabled {
			y := top + (h-14)/2
			a.line(d.Draw.DC, x+9, y+7, x+12, y+10, surfaceColor, 1)
			a.line(d.Draw.DC, x+12, y+10, x+17, y+4, surfaceColor, 1)
		}
		a.text(d.Draw.DC, keyCaption(row.Source), x+34, top, 310, h, a.font, ink, 0x4|0x20|0x8000)
		a.text(d.Draw.DC, "→", x+356, top, 30, h, a.font, muted, 0x4|0x20)
		a.text(d.Draw.DC, keyCaption(row.Target), x+392, top, 270, h, a.font, ink, 0x4|0x20|0x8000)
		pillX, pillY := x+680, top+(h-24)/2
		label, fg, wash := a.ruleStateLabel(row), muted, softColor
		if row.Enabled {
			if label == "已启用" {
				fg, wash = accentColor, accentWash
			}
		}
		a.box(d.Draw.DC, pillX, pillY, 82, 24, 10, wash, wash)
		a.text(d.Draw.DC, label, pillX, pillY, 82, 24, a.smallFont, fg, 0x1|0x4|0x20)
		if i < len(a.cfg.Rules)-1 {
			a.line(d.Draw.DC, x+6, top+h-1, x+782, top+h-1, softColor, 1)
		}
		return 0x4
	}
	return 0
}

func (a *application) ruleStateLabel(r rule) string {
	if !r.Enabled {
		return "未勾选"
	}
	if a.inputFailureMessage != "" {
		return "输入故障"
	}
	if a.capturing != 0 {
		return "录入中"
	}
	if !a.cfg.Enabled {
		return "全局暂停"
	}
	if a.configurationActive {
		return "切换后生效"
	}
	return "已启用"
}
