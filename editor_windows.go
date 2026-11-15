//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const editorClass = "SimpleKeymapMappingEditor"
const editorWidth = 680
const editorHeight = 650
const (
	eStep0 = 301 + iota
	eStep1
	eStep2
	eQuestion
	eDescription
	eSelected
	eDetail
	eRecord
	eManual
	eSearch
	eCategory
	eKeyList
	eFeedback
	eNote
	eChangeSource
	eChangeTarget
	eReviewSource
	eReviewTarget
	eExplanation
	eConflict
	eCancel
	eBack
	eNext
)

type mappingEditor struct {
	hwnd            uintptr
	controls        map[int]uintptr
	step            int
	index           int
	draft, original rule
	manual          bool
	filtering       bool
	saveError       string
	recordNotice    string
	returnFocus     uintptr
}

var editorCallback = syscall.NewCallback(editorProc)

func (a *application) registerEditorWindow() error {
	cursor, _, _ := loadCursor.Call(0, 32512)
	wc := windowClass{Size: uint32(unsafe.Sizeof(windowClass{})), WndProc: editorCallback, Instance: a.instance, Cursor: cursor, Background: a.whiteBrush, ClassName: utf16(editorClass), Icon: loadBrandIcon(a.instance, a.px(32)), IconSmall: loadBrandIcon(a.instance, a.px(16))}
	if r, _, err := registerClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("无法创建映射窗口：%v", err)
	}
	return nil
}

func (a *application) openEditor(index int, preset bool) {
	if a.editor != nil {
		setForeground.Call(a.editor.hwnd)
		return
	}
	if index >= len(a.cfg.Rules) {
		return
	}
	a.closeInformation()
	draft := rule{Enabled: true}
	step := 0
	if index >= 0 {
		draft = a.cfg.Rules[index]
		step = 2
	} else if preset {
		draft.Source, draft.Target = "mouse_x1", "mouse_middle"
		step = 2
	}
	focus, _, _ := getFocus.Call()
	ed := &mappingEditor{controls: map[int]uintptr{}, index: index, step: step, draft: draft, original: draft, returnFocus: focus}
	style := uintptr(0x00C80000 | 0x00080000 | wsClipChildren)
	r := rect{Right: int32(a.px(editorWidth)), Bottom: int32(a.px(editorHeight))}
	adjustWindowRect.Call(uintptr(unsafe.Pointer(&r)), style, 0, 0)
	var owner rect
	getWindowRect.Call(a.hwnd, uintptr(unsafe.Pointer(&owner)))
	x := int(owner.Left) + (int(owner.Right-owner.Left)-int(r.Right-r.Left))/2
	y := int(owner.Top) + (int(owner.Bottom-owner.Top)-int(r.Bottom-r.Top))/2
	ed.hwnd, _, _ = createWindowEx.Call(0x10000, uintptr(unsafe.Pointer(utf16(editorClass))), uintptr(unsafe.Pointer(utf16("轻映 · 添加映射"))), style, intArg(max(0, x)), intArg(max(0, y)), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), a.hwnd, 0, a.instance, 0)
	if ed.hwnd == 0 {
		showError(a.hwnd, "无法打开映射窗口，请重试。")
		return
	}
	a.editor = ed
	for _, c := range []struct {
		class, text string
		id          int
		style       uintptr
	}{
		{"BUTTON", "1 · 按哪个键", eStep0, 0xB}, {"BUTTON", "2 · 代替哪个键", eStep1, 0xB}, {"BUTTON", "3 · 确认效果", eStep2, 0xB},
		{"STATIC", "", eQuestion, 0}, {"STATIC", "", eDescription, 0}, {"STATIC", "", eSelected, 0}, {"STATIC", "", eDetail, 0},
		{"BUTTON", "录入按键", eRecord, 0xB}, {"BUTTON", "从列表选择按键", eManual, 0xB},
		{"EDIT", "", eSearch, 0x00800000 | 0x80}, {"COMBOBOX", "按键分类", eCategory, 0x3 | wsVscroll},
		{"LISTBOX", "选择按键；可用方向键和回车", eKeyList, 0x1 | 0x100 | 0x00800000 | wsVscroll},
		{"STATIC", "", eFeedback, 0}, {"STATIC", "", eNote, 0},
		{"BUTTON", "更换", eChangeSource, 0xB}, {"BUTTON", "更换", eChangeTarget, 0xB},
		{"STATIC", "", eReviewSource, 0}, {"STATIC", "", eReviewTarget, 0}, {"STATIC", "", eExplanation, 0},
		{"BUTTON", "修改已有映射", eConflict, 0xB}, {"BUTTON", "取消", eCancel, 0xB}, {"BUTTON", "上一步", eBack, 0xB}, {"BUTTON", "下一步", eNext, 0xB},
	} {
		styles := wsChild | c.style
		if c.class != "STATIC" {
			styles |= wsTabstop
		}
		control, _, _ := createWindowEx.Call(0, uintptr(unsafe.Pointer(utf16(c.class))), uintptr(unsafe.Pointer(utf16(c.text))), styles, 0, 0, 0, 0, ed.hwnd, uintptr(c.id), a.instance, 0)
		if control == 0 {
			a.closeEditor(true)
			showError(a.hwnd, "无法创建按键选择控件，请重试。")
			return
		}
		ed.controls[c.id] = control
		send(control, wmSetFont, a.font, 1)
	}
	for _, id := range []int{eQuestion, eSelected, eReviewSource, eReviewTarget} {
		send(ed.controls[id], wmSetFont, a.headingFont, 1)
	}
	for _, id := range []int{eDescription, eDetail, eFeedback, eNote} {
		send(ed.controls[id], wmSetFont, a.smallFont, 1)
	}
	for _, text := range []string{"全部按键", "鼠标按键", "键盘按键"} {
		send(ed.controls[eCategory], cbAddString, 0, uintptr(unsafe.Pointer(utf16(text))))
	}
	send(ed.controls[eCategory], cbSetCurSel, 0, 0)
	send(ed.controls[eSearch], 0x1501, 1, uintptr(unsafe.Pointer(utf16("搜索按键，如：侧键、空格、F8"))))
	a.renderEditor()
	enableWindow.Call(a.hwnd, 0)
	showWindow.Call(ed.hwnd, 5)
	setForeground.Call(ed.hwnd)
	if step == 2 {
		setFocus.Call(ed.controls[eChangeTarget])
	} else {
		setFocus.Call(ed.controls[eRecord])
	}
}

func (a *application) editorOwns(hwnd uintptr) bool {
	if a.editor == nil {
		return false
	}
	child, _, _ := isChild.Call(a.editor.hwnd, hwnd)
	return hwnd == a.editor.hwnd || child != 0
}
func (a *application) editorDirty() bool {
	return a.editor != nil && a.editor.draft != a.editor.original
}
func (a *application) confirmDiscard() bool {
	if !a.editorDirty() {
		return true
	}
	answer, _, _ := messageBox.Call(a.editor.hwnd, uintptr(unsafe.Pointer(utf16("放弃尚未保存的修改吗？\n选择「否」可继续编辑。原有映射不会改变。"))), uintptr(unsafe.Pointer(utf16("保留还是放弃修改"))), 0x4|0x20|0x100)
	return answer == 6
}
func (a *application) requestCloseEditor() bool {
	if a.capturing != 0 {
		a.finishCapture(0)
		return false
	}
	if !a.confirmDiscard() {
		return false
	}
	a.closeEditor(true)
	return true
}
func (a *application) closeEditor(force bool) {
	if a.editor == nil {
		return
	}
	if !force && !a.confirmDiscard() {
		return
	}
	if a.capturing != 0 {
		a.finishCapture(0)
	}
	ed := a.editor
	a.editor = nil
	enableWindow.Call(a.hwnd, 1)
	destroyWindow.Call(ed.hwnd)
	if !a.closing {
		setForeground.Call(a.hwnd)
		if ed.returnFocus != 0 {
			setFocus.Call(ed.returnFocus)
		}
	}
}

func (a *application) editorConflict() int {
	if a.editor == nil {
		return -1
	}
	for i, r := range a.cfg.Rules {
		if i != a.editor.index && r.Source == a.editor.draft.Source {
			return i
		}
	}
	return -1
}
func (a *application) editorValidation() string {
	ed := a.editor
	if ed == nil {
		return ""
	}
	if ed.saveError != "" {
		return "保存失败：" + ed.saveError + "。原规则未改变，请重试。"
	}
	if ed.draft.Source != "" && ed.draft.Source == ed.draft.Target {
		return "两个按键相同，请换一个要代替的键。"
	}
	if i := a.editorConflict(); i >= 0 {
		r := a.cfg.Rules[i]
		return "已有映射：" + keyCaption(r.Source) + " → " + keyCaption(r.Target)
	}
	return ""
}
func (a *application) editorChoose(id string) {
	if a.editor == nil {
		return
	}
	ed := a.editor
	if ed.step == 0 {
		ed.draft.Source = id
	} else if ed.step == 1 {
		ed.draft.Target = id
	} else {
		return
	}
	ed.manual = false
	ed.saveError = ""
	ed.recordNotice = ""
	a.renderEditor()
	setFocus.Call(ed.controls[eNext])
}
func (a *application) editorSetStep(step int) {
	if a.editor == nil || a.capturing != 0 {
		return
	}
	ed := a.editor
	if step < 0 || step > 2 || step >= 1 && ed.draft.Source == "" || step == 2 && (ed.draft.Target == "" || a.editorConflict() >= 0 || ed.draft.Source == ed.draft.Target) {
		return
	}
	ed.step = step
	ed.manual = false
	ed.filtering = true
	setText(ed.controls[eSearch], "")
	send(ed.controls[eCategory], cbSetCurSel, 0, 0)
	ed.filtering = false
	ed.saveError = ""
	ed.recordNotice = ""
	a.renderEditor()
	if step < 2 {
		setFocus.Call(ed.controls[eRecord])
	} else {
		setFocus.Call(ed.controls[eNext])
	}
}

func (a *application) renderEditor() {
	ed := a.editor
	if ed == nil {
		return
	}
	for _, hwnd := range ed.controls {
		visibleControl(hwnd, false)
		enableWindow.Call(hwnd, 1)
	}
	w, h := a.clientSize(ed.hwnd)
	place := func(id, x, y, width, height int) {
		a.moveControl(ed.controls[id], x, y, width, height)
		visibleControl(ed.controls[id], true)
	}
	for i := 0; i < 3; i++ {
		place(eStep0+i, 28+i*((w-64)/3+4), 24, (w-64)/3, 36)
		enableWindow.Call(ed.controls[eStep0+i], boolArg(a.capturing == 0 && (i == 0 || ed.draft.Source != "") && (i < 2 || ed.draft.Target != "")))
	}
	title := "添加映射"
	if ed.index >= 0 {
		title = "修改映射"
	}
	setText(ed.hwnd, "轻映 · "+title)
	question, desc := "你想按哪个键？", "选择你准备实际按下的键，例如鼠标侧键。"
	if ed.step == 1 {
		question = "让它代替哪个键？"
		desc = "已选「" + keyCaption(ed.draft.Source) + "」，请选择它要代替的键。"
	}
	if ed.step == 2 {
		question = "确认按下后的效果"
		desc = "保存前，可以检查或更换下面的按键。"
	}
	setText(ed.controls[eQuestion], question)
	place(eQuestion, 28, 86, w-56, 32)
	setText(ed.controls[eDescription], desc)
	place(eDescription, 28, 129, w-56, 26)
	if ed.step < 2 {
		selected := ed.draft.Source
		if ed.step == 1 {
			selected = ed.draft.Target
		}
		name, detail := "尚未选择", "录入或从列表选择后，会显示在这里"
		if selected != "" {
			name, detail = keyCaption(selected), keyDetail(selected)
		}
		if a.capturing != 0 {
			name, detail = "等待按键…", "按下并松开一个键，Esc 取消；8 秒后自动结束。"
		}
		setText(ed.controls[eSelected], name)
		place(eSelected, 48, 176, w-96, 30)
		setText(ed.controls[eDetail], detail)
		place(eDetail, 48, 211, w-96, 24)
		setText(ed.controls[eRecord], "点击后，按一下要用的键")
		if selected != "" {
			setText(ed.controls[eRecord], "重新录入按键")
		}
		if a.capturing != 0 {
			setText(ed.controls[eRecord], "取消录入")
		}
		place(eRecord, 28, 251, w-56, 40)
		setText(ed.controls[eManual], "从列表选择按键")
		if ed.manual {
			setText(ed.controls[eManual], "收起按键列表")
		}
		place(eManual, 28, 299, w-56, 30)
		if ed.manual {
			place(eSearch, 28, 341, w-206, 30)
			place(eCategory, w-166, 341, 138, 250)
			place(eKeyList, 28, 381, w-56, h-519)
			a.filterEditorKeys()
		}
		note := "只识别一个按键。Esc 取消录入，也可以从列表选择它。"
		if ed.recordNotice != "" {
			note = ed.recordNotice
		}
		if a.capturing != 0 {
			note = "录入期间暂时暂停映射，完成或取消后恢复此前状态。"
		}
		setText(ed.controls[eNote], note)
		noteWidth := w - 56
		if a.editorConflict() >= 0 {
			noteWidth = w - 240
		}
		place(eNote, 28, h-100, noteWidth, 26)
	} else {
		setText(ed.controls[eReviewSource], "按下："+keyCaption(ed.draft.Source))
		place(eReviewSource, 48, 190, w-210, 32)
		setText(ed.controls[eReviewTarget], "相当于："+keyCaption(ed.draft.Target))
		place(eReviewTarget, 48, 277, w-210, 32)
		place(eChangeSource, w-136, 187, 88, 36)
		place(eChangeTarget, w-136, 274, 88, 36)
		setText(ed.controls[eExplanation], "按下「"+keyCaption(ed.draft.Source)+"」，相当于按下「"+keyCaption(ed.draft.Target)+"」。\r\n一直按住就持续按住，松开即释放。")
		place(eExplanation, 44, 370, w-88, 64)
		note := "原按键的原有动作会被替换，只在其他应用中生效。"
		if !a.cfg.Enabled {
			note += "\r\n全部映射目前已暂停，保存后也保持暂停。"
		} else if !ed.draft.Enabled {
			note += "\r\n这条规则目前已暂停，保存后仍保持暂停。"
		}
		setText(ed.controls[eNote], note)
		place(eNote, 28, 454, w-56, 52)
	}
	feedback := a.editorFeedback()
	setText(ed.controls[eFeedback], feedback)
	place(eFeedback, 28, h-138, w-56, 34)
	if a.editorConflict() >= 0 {
		place(eConflict, w-198, h-103, 170, 30)
	}
	place(eCancel, 28, h-53, 80, 36)
	if ed.step > 0 {
		place(eBack, w-286, h-53, 92, 36)
	}
	text := "下一步"
	if ed.step == 2 {
		if ed.index >= 0 {
			text = "保存修改"
		} else if a.cfg.Enabled {
			text = "添加并启用"
		} else {
			text = "添加并保持暂停"
		}
		if ed.saveError != "" {
			text = "重试保存"
		}
	}
	setText(ed.controls[eNext], text)
	place(eNext, w-182, h-53, 154, 36)
	valid := ed.draft.Source != ""
	if ed.step > 0 {
		valid = valid && ed.draft.Target != "" && ed.draft.Source != ed.draft.Target && a.editorConflict() < 0
	}
	if ed.step == 2 && ed.index >= 0 && ed.draft == ed.original {
		valid = false
	}
	enableWindow.Call(ed.controls[eNext], boolArg(valid && a.capturing == 0))
	if a.capturing != 0 {
		for id, hwnd := range ed.controls {
			if id != eRecord && id != eSelected && id != eDetail && id != eNote && id != eQuestion && id != eDescription {
				enableWindow.Call(hwnd, 0)
			}
		}
	}
	invalidateRect.Call(ed.hwnd, 0, 1)
}

func (a *application) filterEditorKeys() {
	ed := a.editor
	if ed == nil || ed.filtering {
		return
	}
	ed.filtering = true
	defer func() { ed.filtering = false }()
	query := strings.ToLower(strings.TrimSpace(windowText(ed.controls[eSearch])))
	category := send(ed.controls[eCategory], cbGetCurSel, 0, 0)
	list := ed.controls[eKeyList]
	send(list, 0x0184, 0, 0)
	selected := ed.draft.Source
	if ed.step == 1 {
		selected = ed.draft.Target
	}
	for index, k := range keys {
		if category == 1 && !k.Mouse || category == 2 && k.Mouse || query != "" && !strings.Contains(strings.ToLower(k.Label+" "+k.ID), query) {
			continue
		}
		row := send(list, 0x0180, 0, uintptr(unsafe.Pointer(utf16(k.Label))))
		send(list, 0x019A, row, uintptr(index))
		if k.ID == selected {
			send(list, 0x0186, row, 0)
		}
	}
	setText(ed.controls[eFeedback], a.editorFeedback())
}

func (a *application) editorFeedback() string {
	message := a.editorValidation()
	if message == "" && a.editor != nil && a.editor.manual && send(a.editor.controls[eKeyList], 0x018B, 0, 0) == 0 {
		return "没有找到按键，请换一个关键词或分类。"
	}
	return message
}

func (a *application) editorListSelection() {
	if a.editor == nil {
		return
	}
	list := a.editor.controls[eKeyList]
	row := int32(send(list, 0x0188, 0, 0))
	if row < 0 {
		return
	}
	index := int(send(list, 0x0199, uintptr(row), 0))
	if index >= 0 && index < len(keys) {
		ed := a.editor
		if ed.step == 0 {
			ed.draft.Source = keys[index].ID
		} else {
			ed.draft.Target = keys[index].ID
		}
		ed.saveError = ""
		ed.recordNotice = ""
		a.renderEditor()
		setFocus.Call(ed.controls[eKeyList])
	}
}

func (a *application) editorEnter() {
	if a.editor == nil || a.capturing != 0 {
		return
	}
	focus, _, _ := getFocus.Call()
	for _, id := range []int{eNext, eBack, eCancel, eRecord, eManual, eChangeSource, eChangeTarget, eConflict, eStep0, eStep1, eStep2} {
		if a.editor.controls[id] == focus {
			a.editorCommand(id)
			return
		}
	}
	if focus == a.editor.controls[eKeyList] {
		a.editorListSelection()
		return
	}
	setFocus.Call(a.editor.controls[eNext])
}

func (a *application) editorCommand(id int) {
	ed := a.editor
	if ed == nil {
		return
	}
	if a.capturing != 0 {
		if id == eRecord || id == eCancel || id == 2 {
			a.finishCapture(0)
		}
		return
	}
	switch id {
	case eStep0, eStep1, eStep2:
		a.editorSetStep(id - eStep0)
	case eChangeSource:
		a.editorSetStep(0)
	case eChangeTarget:
		a.editorSetStep(1)
	case eCancel, 2:
		a.requestCloseEditor()
	case eBack:
		a.editorSetStep(ed.step - 1)
	case eNext:
		if ed.step < 2 {
			a.editorSetStep(ed.step + 1)
		} else {
			a.saveEditor()
		}
	case eRecord:
		if ed.step < 2 {
			side := idRecordSource
			if ed.step == 1 {
				side = idRecordTarget
			}
			a.startCapture(side)
		}
	case eManual:
		ed.manual = !ed.manual
		a.renderEditor()
		if ed.manual {
			setFocus.Call(ed.controls[eSearch])
		}
	case eConflict:
		i := a.editorConflict()
		if i >= 0 && a.confirmDiscard() {
			a.closeEditor(true)
			a.selectRule(i)
			a.openEditor(i, false)
		}
	}
}

func (a *application) saveEditor() {
	ed := a.editor
	if ed == nil || ed.step != 2 || a.capturing != 0 {
		return
	}
	r := ed.draft
	if r.Source == "" || r.Target == "" || r.Source == r.Target || a.editorConflict() >= 0 || ed.index >= 0 && r == ed.original {
		return
	}
	next := a.cfg
	next.Rules = append([]rule{}, a.cfg.Rules...)
	index := ed.index
	message := "已修改："
	if index < 0 {
		index = len(next.Rules)
		next.Rules = append(next.Rules, r)
		message = "已添加："
	} else {
		if index >= len(next.Rules) {
			return
		}
		next.Rules[index] = r
	}
	if err := saveConfig(a.path, next); err != nil {
		ed.saveError = errorCause(err)
		a.renderEditor()
		return
	}
	a.cfg = next
	a.configSaveError = ""
	a.apply()
	a.successMessage = message + keyCaption(r.Source) + " → " + keyCaption(r.Target)
	a.successActive = a.cfg.Enabled && r.Enabled && a.inputFailureMessage == ""
	a.status = "设置已保存到软件同目录。"
	a.closeEditor(true)
	a.refreshList()
	a.selectRule(index)
	setFocus.Call(a.controls[idList])
}

func (a *application) startCapture(id int) {
	if a.editor == nil || a.captureHeld != "" || a.capturing != 0 {
		return
	}
	a.capturing = id
	a.capturePending = false
	a.captureIndex = 0
	a.captureGeneration++
	a.editor.recordNotice = ""
	a.apply()
	a.renderEditor()
	setTimer.Call(a.hwnd, 1, 8000, 0)
}
func (a *application) finishCapture(index int) {
	if a.capturing == 0 {
		return
	}
	killTimer.Call(a.hwnd, 1)
	a.capturing = 0
	a.capturePending = false
	a.captureIndex = 0
	a.apply()
	if a.editor != nil {
		if index > 0 && index <= len(keys) {
			a.editorChoose(keys[index-1].ID)
		} else {
			a.editor.recordNotice = "录入已结束。可以重新录入，或从列表选择。"
			a.renderEditor()
			setFocus.Call(a.editor.controls[eRecord])
		}
	}
}

func editorProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if app != nil {
		a := app
		switch msg {
		case wmClose:
			a.requestCloseEditor()
			return 0
		case wmDestroy:
			if a.editor != nil && a.editor.hwnd == hwnd {
				a.editor = nil
				enableWindow.Call(a.hwnd, 1)
			}
			return 0
		case wmPaint:
			a.paintEditor(hwnd)
			return 0
		case wmEraseBkgnd:
			return 1
		case wmDrawItem:
			a.drawEditorButton(messageData[drawItemStruct](&lParam))
			return 1
		case 0x0138, 0x0135:
			setTextColor.Call(wParam, ink)
			if a.editor != nil {
				for _, id := range []int{eDescription, eDetail, eNote} {
					if lParam == a.editor.controls[id] {
						setTextColor.Call(wParam, muted)
					}
				}
				if lParam == a.editor.controls[eFeedback] {
					setTextColor.Call(wParam, warningColor)
				}
				if lParam == a.editor.controls[eExplanation] {
					setTextColor.Call(wParam, accentColor)
					setBkMode.Call(wParam, 1)
					return a.accentBrush
				}
			}
			setBkMode.Call(wParam, 1)
			return a.whiteBrush
		case wmCommand:
			id, notification := int(wParam&0xffff), wParam>>16
			if a.editor == nil {
				return 0
			}
			if id == eSearch && notification == 0x300 {
				a.filterEditorKeys()
				return 0
			}
			if id == eCategory && notification == 1 {
				a.filterEditorKeys()
				return 0
			}
			if id == eKeyList && notification == 1 {
				a.editorListSelection()
				return 0
			}
			if notification == 0 {
				a.editorCommand(id)
			}
			return 0
		}
	}
	r, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func (a *application) paintEditor(hwnd uintptr) {
	var ps paintStruct
	dc, _, _ := beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer endPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	var r rect
	getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	fillRect.Call(dc, uintptr(unsafe.Pointer(&r)), a.whiteBrush)
	if a.editor == nil {
		return
	}
	w, h := a.clientSize(hwnd)
	if a.editor.step < 2 {
		a.box(dc, 28, 164, w-56, 78, 12, surfaceColor, lineColor)
	} else {
		a.box(dc, 28, 176, w-56, 64, 12, surfaceColor, lineColor)
		a.box(dc, 28, 263, w-56, 64, 12, surfaceColor, lineColor)
		a.box(dc, 28, 351, w-56, 96, 12, accentWash, accentWash)
	}
	a.line(dc, 28, h-70, w-28, h-70, lineColor, 1)
}
func (a *application) drawEditorButton(d *drawItemStruct) {
	if d == nil {
		return
	}
	id := int(d.ID)
	bg, fg, border := surfaceColor, accentColor, surfaceColor
	if id == eNext {
		bg, fg, border = accentColor, surfaceColor, accentColor
	}
	if id >= eStep0 && id <= eStep2 {
		bg, fg, border = softColor, muted, softColor
		if a.editor != nil && id-eStep0 == a.editor.step {
			bg, fg, border = accentWash, accentColor, accentWash
		}
	}
	if id == eRecord {
		bg, border = accentWash, lineColor
	}
	if d.State&1 != 0 {
		bg, border = softColor, accentColor
	}
	if d.State&4 != 0 {
		fg = rgb(167, 178, 170)
		if id == eNext {
			bg, border = softColor, softColor
		}
	}
	if d.State&0x10 != 0 {
		border = accentColor
	}
	fillRect.Call(d.DC, uintptr(unsafe.Pointer(&d.Rect)), a.whiteBrush)
	w, h := int(d.Rect.Right)*96/a.dpi, int(d.Rect.Bottom)*96/a.dpi
	a.box(d.DC, 0, 0, w, h, 8, bg, border)
	a.textRect(d.DC, windowText(d.Hwnd), &d.Rect, a.smallFont, fg, 0x1|0x4|0x20)
}
