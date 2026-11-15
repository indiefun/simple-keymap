//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const rowActionBase = 1000

var listCallback = syscall.NewCallback(listProc)

func (a *application) createControls() error {
	for _, c := range []struct {
		text string
		id   int
	}{
		{"＋ 添加映射", idNew}, {"映射总开关", idToggle}, {"查看原因", idFaultDetails},
		{"关于与许可", idAbout}, {"支持与反馈 ↗", idSupport}, {"检查更新", idUpdates},
		{"添加第一条映射", idAdd}, {"用「鼠标侧键 → 中键」示例开始", idPreset},
		{"撤销删除", idUndo}, {"重试保存", idRetrySave}, {"最小化，去试试", idTryNow}, {"知道了", idDismissSuccess},
	} {
		if _, err := a.control("BUTTON", c.text, c.id, 0xB, 0, 0, 100, 36); err != nil {
			return err
		}
	}
	list, err := a.control("SysListView32", "已有映射；方向键选择，F2 修改，Delete 删除", idList, 0x1|0x4|0x8|0x40|0x4000|wsClipChildren, 32, 214, 836, 300)
	if err != nil {
		return err
	}
	setWindowTheme.Call(list, uintptr(unsafe.Pointer(utf16("Explorer"))), 0)
	setWindowSubclass.Call(list, listCallback, idList, 0)
	// Let native dialog navigation reach the row buttons inside the list.
	setWindowLong.Call(list, intArg(-20), 0x10000) // WS_EX_CONTROLPARENT.
	send(list, lvmFirst+54, 0, 0x20|0x10000)
	send(list, lvmFirst+1, 0, surfaceColor)
	send(list, lvmFirst+38, 0, surfaceColor)
	for index, text := range []string{"启用", "按下这个键", "代替这个键", "修改", "删除"} {
		column := listColumn{Mask: 0x1 | 0x2 | 0x4, Text: utf16(text), Width: int32(a.px(100))}
		send(list, lvmFirst+97, uintptr(index), uintptr(unsafe.Pointer(&column)))
	}
	a.rowImages, _, _ = imageListCreate.Call(1, uintptr(a.px(80)), 0x20, 1, 1)
	if a.rowImages != 0 {
		send(list, lvmFirst+3, 1, a.rowImages)
	}
	a.layoutMain()
	return nil
}

func (a *application) clientSize(hwnd uintptr) (int, int) {
	var r rect
	getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return int(r.Right) * 96 / a.dpi, int(r.Bottom) * 96 / a.dpi
}
func (a *application) moveControl(hwnd uintptr, x, y, w, h int) {
	moveWindow.Call(hwnd, intArg(a.px(x)), intArg(a.px(y)), intArg(a.px(w)), intArg(a.px(h)), 1)
}
func visibleControl(hwnd uintptr, show bool) { showWindow.Call(hwnd, boolArg(show)*5) }

func (a *application) listTop() int {
	if a.successMessage != "" {
		return 298
	}
	return 214
}
func (a *application) layoutMain() {
	if a.controls[idList] == 0 {
		return
	}
	w, h := a.clientSize(a.hwnd)
	has := len(a.cfg.Rules) > 0
	place := func(id, x, y, width, height int, visible bool) {
		a.moveControl(a.controls[id], x, y, width, height)
		visibleControl(a.controls[id], visible)
	}
	place(idAbout, w-366, 30, 116, 32, true)
	place(idSupport, w-242, 30, 118, 32, true)
	place(idUpdates, w-116, 30, 84, 32, true)
	place(idToggle, w-112, 108, 64, 36, has || a.inputFailureMessage != "")
	place(idFaultDetails, 270, 108, 92, 36, a.inputFailureMessage != "")
	enableWindow.Call(a.controls[idToggle], boolArg(a.inputFailureMessage == ""))
	place(idNew, w-180, a.listTop()-52, 148, 38, has)
	place(idAdd, (w-200)/2, 340, 200, 44, !has)
	place(idPreset, (w-360)/2, 394, 360, 34, !has)
	place(idUndo, w-132, h-86, 100, 32, a.deletedRule != nil)
	place(idRetrySave, 300, h-86, 90, 30, a.configSaveError != "")
	place(idTryNow, w-268, 166, 152, 38, a.successMessage != "" && a.successActive)
	place(idDismissSuccess, w-108, 166, 76, 38, a.successMessage != "")
	a.moveControl(a.controls[idList], 32, a.listTop(), w-64, max(80, h-a.listTop()-110))
	visibleControl(a.controls[idList], has)
	widths := a.listWidths()
	for index, width := range widths {
		send(a.controls[idList], lvmFirst+30, uintptr(index), uintptr(a.px(width)))
	}
	a.layoutRowActions()
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *application) listWidths() [5]int {
	w, _ := a.clientSize(a.controls[idList])
	usable := w - 20
	return [5]int{72, (usable - 216) / 2, (usable - 216) / 2, 72, 72}
}

func (a *application) layoutRowActions() {
	if a.rowActions == nil {
		return
	}
	w, h := a.clientSize(a.controls[idList])
	widths := a.listWidths()
	x := widths[0] + widths[1] + widths[2]
	for index, buttons := range a.rowActions {
		var r rect
		send(a.controls[idList], lvmFirst+14, uintptr(index), uintptr(unsafe.Pointer(&r)))
		top, bottom := int(r.Top)*96/a.dpi, int(r.Bottom)*96/a.dpi
		shown := len(a.cfg.Rules) > 0 && bottom > 0 && top >= 0 && bottom <= h && w > 0
		for action, button := range buttons {
			left, width := 12, 48
			if action > 0 {
				left, width = x+(action-1)*widths[3], widths[action+2]-6
			}
			a.moveControl(button, left, top+22, width, 36)
			visibleControl(button, shown)
		}
	}
}

func (a *application) refreshList() {
	a.updating = true
	defer func() { a.updating = false }()
	for _, buttons := range a.rowActions {
		for _, button := range buttons {
			destroyWindow.Call(button)
		}
	}
	a.rowActions = map[int][3]uintptr{}
	list := a.controls[idList]
	send(list, lvmFirst+9, 0, 0)
	for index, r := range a.cfg.Rules {
		item := listItem{Mask: 1, Item: int32(index), Text: utf16(ruleEnabledText(r))}
		send(list, lvmFirst+77, 0, uintptr(unsafe.Pointer(&item)))
		for column, text := range []string{keyLabel(r.Source), keyLabel(r.Target), "修改", "删除"} {
			item = listItem{SubItem: int32(column + 1), Text: utf16(text)}
			send(list, lvmFirst+116, uintptr(index), uintptr(unsafe.Pointer(&item)))
		}
		var buttons [3]uintptr
		for action, text := range []string{ruleEnabledText(r), "修改", "删除"} {
			id := rowActionBase + index*3 + action
			accessible := text
			if action == 0 {
				accessible = keyCaption(r.Source) + "，" + text + "；点击切换"
			} else if action == 1 {
				accessible = "修改 " + keyCaption(r.Source) + " 到 " + keyCaption(r.Target)
			} else {
				accessible = "删除 " + keyCaption(r.Source) + " 到 " + keyCaption(r.Target)
			}
			// The list owns and clips its buttons. Sibling overlays are painted
			// over when the native list redraws selection or keyboard focus.
			hwnd, _, _ := createWindowEx.Call(0, uintptr(unsafe.Pointer(utf16("BUTTON"))), uintptr(unsafe.Pointer(utf16(text))), wsChild|wsVisible|wsTabstop|0xB, 0, 0, 0, 0, list, uintptr(id), a.instance, 0)
			setText(hwnd, accessible)
			buttons[action] = hwnd
			send(hwnd, wmSetFont, a.smallFont, 1)
		}
		a.rowActions[index] = buttons
	}
	a.layoutMain()
}

func ruleEnabledText(r rule) string {
	if r.Enabled {
		return "已启用"
	}
	return "已暂停"
}
func (a *application) selectedRule() int {
	return int(int32(send(a.controls[idList], lvmFirst+12, ^uintptr(0), 0x2)))
}

func (a *application) enterRowActions(msg *winMessage) bool {
	if msg.Hwnd != a.controls[idList] || msg.Message != 0x0100 || msg.WParam != 9 {
		return false
	}
	shift, _, _ := getAsyncKeyState.Call(0x10)
	if shift&0x8000 != 0 {
		return false
	}
	index := a.selectedRule()
	if index < 0 && len(a.cfg.Rules) > 0 {
		index = int(send(a.controls[idList], lvmFirst+39, 0, 0)) // Top visible row.
	}
	if buttons, ok := a.rowActions[index]; ok {
		a.selectRule(index)
		setFocus.Call(buttons[0])
		return true
	}
	return false
}
func (a *application) selectRule(index int) {
	item := listItem{State: 3, StateMask: 3}
	send(a.controls[idList], lvmFirst+43, uintptr(index), uintptr(unsafe.Pointer(&item)))
	send(a.controls[idList], lvmFirst+19, uintptr(index), 0)
	a.layoutRowActions()
}
func (a *application) refreshRuleStatuses() { invalidateRect.Call(a.controls[idList], 0, 0) }

func (a *application) popup(owner uintptr, entries []struct {
	id   int
	text string
}) int {
	menu, _, _ := createPopupMenu.Call()
	if menu == 0 {
		return 0
	}
	defer destroyMenu.Call(menu)
	for _, entry := range entries {
		appendMenu.Call(menu, 0, uintptr(entry.id), uintptr(unsafe.Pointer(utf16(entry.text))))
	}
	var p point
	getCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	chosen, _, _ := trackPopupMenu.Call(menu, 0x100|0x2, uintptr(p.X), uintptr(p.Y), 0, owner, 0)
	return int(chosen)
}

func (a *application) ruleMenu(index int) {
	if index < 0 || index >= len(a.cfg.Rules) {
		return
	}
	a.selectRule(index)
	chosen := a.popup(a.hwnd, []struct {
		id   int
		text string
	}{{idEdit, "修改映射\tF2"}, {idDelete, "删除映射"}})
	if chosen != 0 {
		a.command(chosen)
	}
}

func (a *application) command(id int) {
	if a.editor != nil {
		return
	}
	if id >= rowActionBase {
		index, action := (id-rowActionBase)/3, (id-rowActionBase)%3
		if index >= len(a.cfg.Rules) {
			return
		}
		a.selectRule(index)
		switch action {
		case 0:
			a.cfg.Rules[index].Enabled = !a.cfg.Rules[index].Enabled
			a.successMessage = ""
			a.changed("规则开关已更新。")
			a.refreshList()
		case 1:
			a.openEditor(index, false)
		case 2:
			a.deleteSelected()
		}
		return
	}
	switch id {
	case idNew, idAdd:
		a.openEditor(-1, false)
	case idPreset:
		a.openEditor(-1, true)
	case idEdit:
		if i := a.selectedRule(); i >= 0 {
			a.openEditor(i, false)
		}
	case idDelete:
		a.deleteSelected()
	case idUndo:
		a.undoDelete()
	case idToggle:
		if a.inputFailureMessage != "" {
			return
		}
		a.cfg.Enabled = !a.cfg.Enabled
		a.successMessage = ""
		a.changed("已切换全部映射的开关。")
		a.layoutMain()
	case idFaultDetails:
		a.openInformation("support")
	case idHelp:
		a.openInformation("help")
	case idSupport:
		a.openSupport()
	case idUpdates:
		a.openInformation("updates")
		a.startUpdateCheck()
	case idAbout:
		a.openInformation("about")
	case idRetrySave:
		a.persistConfig()
	case idTryNow:
		showWindow.Call(a.hwnd, 6)
	case idDismissSuccess:
		a.successMessage = ""
		a.layoutMain()
	}
}

func (a *application) deleteSelected() {
	i := a.selectedRule()
	if i < 0 || i >= len(a.cfg.Rules) {
		return
	}
	r := a.cfg.Rules[i]
	a.deletedRule = &r
	a.deletedIndex = i
	a.cfg.Rules = append(a.cfg.Rules[:i], a.cfg.Rules[i+1:]...)
	a.successMessage = ""
	a.changed("已删除：" + keyCaption(r.Source) + " → " + keyCaption(r.Target) + "。可撤销。")
	a.refreshList()
}
func (a *application) undoDelete() {
	if a.deletedRule == nil {
		return
	}
	rules := append([]rule{}, a.cfg.Rules...)
	index := min(a.deletedIndex, len(rules))
	rules = append(rules, rule{})
	copy(rules[index+1:], rules[index:])
	rules[index] = *a.deletedRule
	if err := validateRules(rules); err != nil {
		a.status = "原按键已有新映射，无法撤销删除。"
		invalidateRect.Call(a.hwnd, 0, 0)
		return
	}
	a.cfg.Rules = rules
	a.deletedRule = nil
	a.changed("已恢复删除的映射。")
	a.refreshList()
	a.selectRule(index)
}

func (a *application) apply() {
	a.emit(a.engine.configure(a.cfg.Enabled && a.capturing == 0 && a.inputFailureMessage == "" && !a.closing, a.cfg.Rules))
	text := "映射总开关，已开启；点击暂停全部"
	if !a.cfg.Enabled {
		text = "映射总开关，已暂停；点击恢复全部"
	}
	if a.inputFailureMessage != "" {
		text = "映射总开关，故障暂停；请查看原因"
	}
	setText(a.controls[idToggle], text)
	enableWindow.Call(a.controls[idToggle], boolArg(a.inputFailureMessage == ""))
	invalidateRect.Call(a.hwnd, 0, 0)
	invalidateRect.Call(a.controls[idToggle], 0, 0)
}

func (a *application) changed(message string) {
	a.apply()
	a.status = message
	a.persistConfig()
	if a.editor != nil {
		a.renderEditor()
	}
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *application) dispose() {
	if a.updates != nil && a.updates.cancel != nil {
		a.updates.cancel()
	}
	a.closeEditor(true)
	a.closeInformation()
	for _, object := range []uintptr{a.font, a.smallFont, a.titleFont, a.headingFont, a.keyFont, a.background, a.whiteBrush, a.accentBrush} {
		if object != 0 {
			deleteObject.Call(object)
		}
	}
	if a.rowImages != 0 {
		imageListDestroy.Call(a.rowImages)
	}
}
func boolArg(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if app != nil {
		a := app
		if a.closing && (msg == wmInputError || msg == wmSendOutput || msg == wmCaptured || msg == wmHotkey) {
			return 0
		}
		switch msg {
		case wmActivateApp:
			a.configurationActive = wParam != 0
			invalidateRect.Call(a.hwnd, 0, 0)
			return 0
		case wmSize:
			a.layoutMain()
			return 0
		case 0x0024:
			limits := messageData[struct{ Reserved, MaxSize, MaxPosition, MinTrack, MaxTrack point }](&lParam)
			limits.MinTrack = point{int32(a.px(760)), int32(a.px(590))}
			return 0
		case wmPaint:
			a.paint(hwnd)
			return 0
		case wmEraseBkgnd:
			return 1
		case wmDrawItem:
			a.drawButton(messageData[drawItemStruct](&lParam))
			return 1
		case wmCommand:
			if wParam>>16 == 0 {
				a.command(int(wParam & 0xffff))
			}
			return 0
		case wmNotify:
			h := messageData[notifyHeader](&lParam)
			if h.ID == idList && int32(h.Code) == -101 && !a.updating {
				change := messageData[struct {
					Header        notifyHeader
					Item, SubItem int32
				}](&lParam)
				if buttons, ok := a.rowActions[int(change.Item)]; ok {
					for _, button := range buttons {
						invalidateRect.Call(button, 0, 1)
					}
				}
			}
			if h.ID == idList && int32(h.Code) == -12 {
				return a.drawList(messageData[listCustomDraw](&lParam))
			}
			if h.ID == idList && int32(h.Code) == -3 {
				a.command(idEdit)
			}
		case wmHotkey:
			if a.inputFailureMessage != "" {
				return 0
			}
			if a.capturing != 0 {
				a.finishCapture(0)
			}
			a.cfg.Enabled = !a.cfg.Enabled
			a.successMessage = ""
			a.changed("已切换全部映射的开关。")
			a.layoutMain()
			return 0
		case wmCaptured:
			if lParam == a.captureGeneration && a.capturePending {
				a.finishCapture(int(wParam))
			}
			return 0
		case wmTimer:
			if wParam == 1 {
				a.finishCapture(0)
			}
			return 0
		case wmInputError:
			if a.inputFailureMessage != "" {
				return 0
			}
			a.cfg.Enabled = false
			message := "映射已暂停：无法发送按键。目标程序若以管理员运行，请用相同权限启动轻映。"
			if wParam == inputTimedOut {
				message = "映射已暂停：Windows 输入发送超时。请保存工作并重启 Windows，再开启映射。"
			} else if wParam == inputServiceLost {
				message = "映射已暂停：输入服务已退出，请重新启动轻映。"
			}
			a.inputFailureMessage = message
			a.successMessage = ""
			a.changed(message)
			a.layoutMain()
			return 0
		case wmSendOutput:
			a.flushOutput()
			return 0
		case wmUpdateResult:
			a.receiveUpdateResult()
			return 0
		case wmClose:
			if a.editor != nil && !a.requestCloseEditor() {
				return 0
			}
			a.beginShutdown()
			destroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			postQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func listProc(hwnd uintptr, msg uint32, wParam, lParam, id, data uintptr) uintptr {
	if app != nil {
		if msg == wmCommand && wParam>>16 == 0 && int(wParam&0xffff) >= rowActionBase {
			app.command(int(wParam & 0xffff))
			return 0
		}
		if msg == wmDrawItem {
			app.drawButton(messageData[drawItemStruct](&lParam))
			return 1
		}
		if msg == 0x0100 {
			switch wParam {
			case 0x71:
				app.command(idEdit)
				return 0
			case 0x2e:
				app.command(idDelete)
				return 0
			case 0x5d:
				app.ruleMenu(app.selectedRule())
				return 0
			case 0x79:
				shift, _, _ := getAsyncKeyState.Call(0x10)
				if shift&0x8000 != 0 {
					app.ruleMenu(app.selectedRule())
					return 0
				}
			}
		}
		if msg == 0x007b {
			app.ruleMenu(app.selectedRule())
			return 0
		}
	}
	r, _, _ := defSubclassProc.Call(hwnd, uintptr(msg), wParam, lParam)
	if app != nil && (msg == 0x0115 || msg == 0x020a || msg == 0x0100 || msg == wmSize || msg == lvmFirst+19 || msg == lvmFirst+20) {
		app.layoutRowActions()
	}
	return r
}

func (a *application) stateCopy() (string, uintptr) {
	if a.inputFailureMessage != "" {
		return "输入服务异常", warningColor
	}
	if !a.cfg.Enabled {
		return "规则已保留", muted
	}
	count := 0
	for _, r := range a.cfg.Rules {
		if r.Enabled {
			count++
		}
	}
	if count == 0 {
		return "请开启下方规则", muted
	}
	return fmt.Sprintf("%d / %d 条规则已启用", count, len(a.cfg.Rules)), muted
}
