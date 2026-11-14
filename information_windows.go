//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"syscall"
	"unsafe"
)

// Legal notices travel with the executable and remain readable offline.
//
//go:embed LICENSE
var licenseText string

//go:embed TRADEMARKS.md
var trademarksText string

const informationClass = "SimpleKeymapInformation"

const (
	infoTitle = 201 + iota
	infoBody
	infoLicense
	infoSource
	infoBrand
	infoProject
	infoDetails
	infoBack
	infoCopy
	infoWebsite
	infoClose
)

type informationWindow struct {
	hwnd     uintptr
	controls map[int]uintptr
	page     string
	actions  []int
}

var informationCallback = syscall.NewCallback(informationProc)

func (a *application) registerInformationWindow() error {
	cursor, _, _ := loadCursor.Call(0, 32512)
	wc := windowClass{Size: uint32(unsafe.Sizeof(windowClass{})), WndProc: informationCallback,
		Instance: a.instance, Cursor: cursor, Background: a.whiteBrush, ClassName: utf16(informationClass)}
	if r, _, err := registerClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("无法创建帮助窗口：%v", err)
	}
	return nil
}

func (a *application) openInformation(page string) {
	if a.capturing != 0 {
		return
	}
	if a.info == nil {
		style := uintptr(0x00CC0000 | 0x00080000 | wsClipChildren) // Caption, resize, system menu.
		r := rect{Right: int32(a.px(700)), Bottom: int32(a.px(560))}
		adjustWindowRect.Call(uintptr(unsafe.Pointer(&r)), style, 0, 0)
		var owner rect
		getWindowRect.Call(a.hwnd, uintptr(unsafe.Pointer(&owner)))
		x := int(owner.Left) + (int(owner.Right-owner.Left)-int(r.Right-r.Left))/2
		y := int(owner.Top) + a.px(40)
		hwnd, _, err := createWindowEx.Call(0x00010000, uintptr(unsafe.Pointer(utf16(informationClass))), uintptr(unsafe.Pointer(utf16("轻映"))),
			style, intArg(max(0, x)), intArg(max(0, y)), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), a.hwnd, 0, a.instance, 0)
		if hwnd == 0 {
			showError(a.hwnd, fmt.Sprintf("无法打开帮助窗口：%v", err))
			return
		}
		a.info = &informationWindow{hwnd: hwnd, controls: map[int]uintptr{}}
		for _, c := range []struct {
			class string
			id    int
			style uintptr
		}{
			{"STATIC", infoTitle, 0}, {"EDIT", infoBody, 0x4 | 0x40 | 0x800 | wsVscroll},
		} {
			style := wsChild | wsVisible | c.style
			if c.id == infoBody {
				style |= wsTabstop
			}
			control, _, _ := createWindowEx.Call(0, uintptr(unsafe.Pointer(utf16(c.class))), 0, style,
				0, 0, 0, 0, hwnd, uintptr(c.id), a.instance, 0)
			if control == 0 {
				a.closeInformation()
				showError(a.hwnd, "无法创建帮助内容，请重新启动轻映后再试。")
				return
			}
			a.info.controls[c.id] = control
			send(control, wmSetFont, a.font, 1)
		}
		send(a.info.controls[infoTitle], wmSetFont, a.headingFont, 1)
		send(a.info.controls[infoBody], 0x00C5, 1024*1024, 0) // EM_LIMITTEXT.
		for _, id := range []int{infoLicense, infoSource, infoBrand, infoProject, infoDetails, infoBack, infoCopy, infoWebsite, infoClose} {
			control, _, _ := createWindowEx.Call(0, uintptr(unsafe.Pointer(utf16("BUTTON"))), 0, wsChild|wsTabstop|0xB,
				0, 0, 0, 0, hwnd, uintptr(id), a.instance, 0)
			if control == 0 {
				a.closeInformation()
				showError(a.hwnd, "无法创建帮助按钮，请重新启动轻映后再试。")
				return
			}
			a.info.controls[id] = control
			send(control, wmSetFont, a.smallFont, 1)
		}
	}
	a.updateInformation(page)
	showWindow.Call(a.info.hwnd, 5)
	setForeground.Call(a.info.hwnd)
	setFocus.Call(a.info.controls[infoBody])
}

func (a *application) informationContent(page string) (string, string, []int) {
	switch page {
	case "about":
		return "关于轻映与开源许可", "轻映 Simple Keymap · " + appVersion + "\nCopyright © 2026 Kevin Loo\n\n免费开源 · GNU GPL 第 3 版（GPL-3.0-only）\n\n你可以依 GPL-3.0 使用、修改和再分发本软件。分发时须遵守许可，包括保留版权与许可声明，并按要求提供对应源码。\n\n本软件不提供任何担保，包括适销性或适合特定用途的默示担保；详见许可全文。\n\n项目名称与图标用于识别官方项目。发布修改版时须明确区分来源，不得冒充官方。\n\n许可全文和品牌说明均可离线查看。", []int{infoLicense, infoSource, infoBrand, infoProject, infoClose}
	case "license":
		return "GNU GPL 第 3 版 · 许可全文", licenseText, []int{infoBack, infoClose}
	case "brand":
		return "项目名称与图标使用说明", trademarksText, []int{infoBack, infoClose}
	case "help":
		return "三步，开始使用", "1  选原按键\n在「按下这个键」中选择要替换的键。\n\n2  选目标按键\n在「变为这个键」中选择新的用途。\n\n3  添加，切到其他应用\n点击「添加映射」，切换窗口或最小化轻映后试按。\n首次打开已选好「鼠标侧键 1 → 鼠标中键」。\n\n按住原按键就是按住目标按键，松开即释放。\n轻映自己的窗口内暂不执行映射，方便修改配置。\n\nCtrl + Alt + F12：随时暂停 / 恢复全部映射。\n最小化继续运行；关闭主窗口停止映射并退出。", []int{infoDetails, infoCopy, infoClose}
	case "details":
		return "详细使用说明", "录入按键\n点击按键卡片展开选择，或点击「录入按键」后按一下。Esc 取消，8 秒后自动结束。\n\n编辑与暂停\n选中规则后点击「编辑所选」，也可以双击规则；点击「保存修改」应用，或「取消修改」保留原规则。取消勾选只暂停该条规则，总开关控制全部规则。\n\n适用范围\n支持单个键盘或鼠标按键之间映射。原按键的原有动作会被替换。暂不支持组合键、宏或滚轮滚动。\n\n管理员权限与游戏\n目标程序以管理员运行时，轻映也需要相同权限。部分游戏或受保护程序可能不接受模拟输入。\n\n保存与隐私\n规则自动保存在本机，不上传按键记录。保存失败会持续提示，可点击「重试保存」。\n配置位置：\n" + a.path + "\n\n故障反馈\n遇到问题可在「支持与反馈」中复制诊断信息，再补充目标应用和复现步骤。输入发送超时时，请先保存工作并重启 Windows。", []int{infoBack, infoCopy, infoClose}
	case "support":
		body := "遇到问题？先复制诊断信息，再到官网反馈。\n\n复制内容包含轻映版本、Windows 版本、总开关和规则数量、错误提示；不包含按键记录、规则明细或个人文件路径。\n\n请另行补充：目标应用、原按键 → 目标按键、复现步骤、预期和实际结果。"
		if a.inputFailureMessage != "" {
			body += "\n\n当前输入问题：\n" + a.inputFailureMessage
		}
		if a.configSaveError != "" {
			body += "\n\n当前设置尚未保存，重启后可能丢失。请检查配置目录是否可写后，点击主窗口的「重试保存」。\n保存错误：" + a.configSaveError
		}
		body += "\n\n官网提供 GitHub Issues、QQ 交流群与自愿捐助入口。全部功能免费，捐助不影响使用。"
		return "支持与反馈", body, []int{infoCopy, infoWebsite, infoClose}
	}
	return "轻映", "", []int{infoClose}
}

func (a *application) updateInformation(page string) {
	if a.info == nil {
		return
	}
	title, body, actions := a.informationContent(page)
	a.info.page, a.info.actions = page, actions
	setText(a.info.hwnd, "轻映 · "+title)
	setText(a.info.controls[infoTitle], title)
	body = normalizeNoticeText(body)
	setText(a.info.controls[infoBody], body)
	send(a.info.controls[infoBody], 0x00B1, 0, 0) // EM_SETSEL: top of document.
	send(a.info.controls[infoBody], 0x00B7, 0, 0) // EM_SCROLLCARET.
	for _, id := range []int{infoLicense, infoSource, infoBrand, infoProject, infoDetails, infoBack, infoCopy, infoWebsite, infoClose} {
		showWindow.Call(a.info.controls[id], 0)
		setText(a.info.controls[id], informationActionText(id))
	}
	a.layoutInformation()
}

func informationActionText(id int) string {
	return map[int]string{infoLicense: "查看许可全文", infoSource: "本版本源码 ↗", infoBrand: "品牌说明", infoProject: "GitHub 项目 ↗", infoDetails: "详细说明", infoBack: "返回", infoCopy: "复制诊断信息", infoWebsite: "官网反馈 ↗", infoClose: "关闭"}[id]
}

func (a *application) layoutInformation() {
	if a.info == nil {
		return
	}
	var r rect
	getClientRect.Call(a.info.hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right)*96/a.dpi, int(r.Bottom)*96/a.dpi
	move := func(id, x, y, width, height int) {
		moveWindow.Call(a.info.controls[id], intArg(a.px(x)), intArg(a.px(y)), intArg(a.px(width)), intArg(a.px(height)), 1)
	}
	move(infoTitle, 24, 22, max(100, w-48), 30)
	rows := (len(a.info.actions) + 2) / 3
	actionsY := h - 24 - rows*44
	move(infoBody, 24, 70, max(100, w-48), max(40, actionsY-86))
	width := max(50, (w-64)/3)
	for index, id := range a.info.actions {
		move(id, 24+(index%3)*(width+8), actionsY+(index/3)*44, width, 36)
		showWindow.Call(a.info.controls[id], 5)
	}
	invalidateRect.Call(a.info.hwnd, 0, 1)
}

func (a *application) ownInformationWindow(hwnd uintptr) bool {
	if a.info == nil {
		return false
	}
	if a.info.hwnd == hwnd {
		return true
	}
	for _, control := range a.info.controls {
		if control == hwnd {
			return true
		}
	}
	return false
}

func (a *application) closeInformation() {
	if a.info != nil {
		hwnd := a.info.hwnd
		a.info = nil
		destroyWindow.Call(hwnd)
	}
}

func informationProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if app != nil {
		switch msg {
		case wmActivateApp:
			app.configurationActive = wParam != 0
			app.refreshRuleStatuses()
			invalidateRect.Call(app.hwnd, 0, 0)
			invalidateRect.Call(app.controls[idList], 0, 0)
			return 0
		case wmClose:
			app.closeInformation()
			return 0
		case wmDestroy:
			if app.info != nil && app.info.hwnd == hwnd {
				app.info = nil
			}
			return 0
		case wmSize:
			app.layoutInformation()
			return 0
		case 0x0024: // WM_GETMINMAXINFO: keep controls reachable while resizing.
			limits := messageData[struct{ Reserved, MaxSize, MaxPosition, MinTrack, MaxTrack point }](&lParam)
			limits.MinTrack = point{int32(app.px(520)), int32(app.px(440))}
			return 0
		case 0x0138, 0x0135: // Static labels and the read-only editor share a white surface.
			setTextColor.Call(wParam, ink)
			setBkMode.Call(wParam, 1)
			return app.whiteBrush
		case wmDrawItem:
			d := messageData[drawItemStruct](&lParam)
			if d != nil {
				fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), app.whiteBrush)
				w, h := int(d.Rect.Right)*96/app.dpi, int(d.Rect.Bottom)*96/app.dpi
				bg := softColor
				if d.State&1 != 0 {
					bg = accentWash
				}
				border := bg
				if d.State&0x10 != 0 {
					border = accentColor
				}
				app.box(d.DC, 0, 0, w, h, 8, bg, border)
				app.textRect(d.DC, windowText(d.Hwnd), &d.Rect, app.smallFont, accentColor, 0x1|0x4|0x20)
			}
			return 1
		case wmCommand:
			if wParam>>16 != 0 {
				break
			}
			switch int(wParam & 0xFFFF) {
			case 2, infoClose:
				app.closeInformation() // IDCANCEL from keyboard navigation.
			case infoLicense:
				app.updateInformation("license")
			case infoBrand:
				app.updateInformation("brand")
			case infoDetails:
				app.updateInformation("details")
			case infoBack:
				if app.info != nil && app.info.page == "details" {
					app.updateInformation("help")
				} else {
					app.updateInformation("about")
				}
			case infoSource:
				app.openWebPage(versionSourceURL(appVersion), "已在浏览器打开本版本源码。")
			case infoProject:
				app.openProject()
			case infoWebsite:
				app.openSupport()
			case infoCopy:
				app.copyDiagnostics()
			}
			return 0
		}
	}
	r, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}
