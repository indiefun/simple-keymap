//go:build windows

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	unicode16 "unicode/utf16"
	"unsafe"
)

func TestNativeRuleAndCaptureFlow(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previous := app
	a := &application{controls: map[int]uintptr{}, modifiers: map[string]bool{}, dpi: 96, path: filepath.Join(t.TempDir(), "config.json"), cfg: config{Enabled: true}, engine: newEngine()}
	if value := os.Getenv("SIMPLE_KEYMAP_TEST_DPI"); value != "" {
		dpi, err := strconv.Atoi(value)
		if err != nil || dpi < 72 || dpi > 192 {
			t.Fatal("invalid test DPI")
		}
		a.dpi = dpi
	}
	app = a
	defer func() { app = previous }()
	if err := a.create(); err != nil {
		t.Fatal(err)
	}
	defer a.dispose()
	defer destroyWindow.Call(a.hwnd)
	a.apply()
	a.refreshList()
	if a.editor != nil || len(a.cfg.Rules) != 0 {
		t.Fatal("first launch must be an empty list without an automatic dialog")
	}
	captureInterface(t, a.hwnd, "v2-first-open.png")
	a.command(idAdd)
	if a.editor == nil || a.editor.step != 0 || a.editor.draft.Source != "" {
		t.Fatal("first add must ask for the physical source")
	}
	if enabled, _, _ := user32.NewProc("IsWindowEnabled").Call(a.hwnd); enabled != 0 {
		t.Fatal("editor is not modal")
	}
	captureInterface(t, a.editor.hwnd, "v2-source-step.png")
	firstWindow := a.editor.hwnd
	a.openEditor(-1, false)
	if a.editor.hwnd != firstWindow {
		t.Fatal("second editor created")
	}
	// Native list supports search, categories and every supported key, including
	// damaged keys and Escape which cannot be recorded using the cancel key.
	a.editorCommand(eManual)
	setText(a.editor.controls[eSearch], "侧键")
	if count := send(a.editor.controls[eKeyList], 0x018B, 0, 0); count != 2 {
		t.Fatalf("side key search: %d", count)
	}
	setText(a.editor.controls[eSearch], "")
	send(a.editor.controls[eCategory], cbSetCurSel, 2, 0)
	a.filterEditorKeys()
	if count := send(a.editor.controls[eKeyList], 0x018B, 0, 0); count != uintptr(len(keys)-5) {
		t.Fatal("keyboard category omitted keys")
	}
	setText(a.editor.controls[eSearch], "not-a-supported-key")
	if !strings.Contains(windowText(a.editor.controls[eFeedback]), "没有找到") {
		t.Fatal("empty search lacks next step")
	}
	setText(a.editor.controls[eSearch], "")
	captureInterface(t, a.editor.hwnd, "v2-key-list.png")
	a.editorChoose("mouse_x1")
	a.editorCommand(eNext)
	if a.editor.step != 1 || !strings.Contains(windowText(a.editor.controls[eDescription]), "鼠标侧键 1") {
		t.Fatal("target step lacks source context")
	}
	a.editorChoose("mouse_x1")
	if !strings.Contains(a.editorValidation(), "相同") {
		t.Fatal("same key allowed")
	}
	a.editorCommand(eNext)
	if a.editor.step != 1 {
		t.Fatal("invalid draft advanced")
	}
	a.editorChoose("mouse_middle")
	a.editorCommand(eNext)
	if a.editor.step != 2 || windowText(a.editor.controls[eNext]) != "添加并启用" {
		t.Fatal("confirmation action unclear")
	}
	captureInterface(t, a.editor.hwnd, "v2-add-confirm.png")
	a.editorCommand(eNext)
	if a.editor != nil || len(a.cfg.Rules) != 1 || a.cfg.Rules[0] != (rule{"mouse_x1", "mouse_middle", true}) {
		t.Fatal("new rule not created")
	}
	captureInterface(t, a.hwnd, "v2-first-success.png")
	testNativeRowRepaint(t, a)
	// Inline toggle pauses only this rule and edit opens directly at confirmation.
	a.command(rowActionBase)
	if a.cfg.Rules[0].Enabled {
		t.Fatal("row toggle did not pause")
	}
	a.command(rowActionBase + 1)
	if a.editor.step != 2 || a.editor.draft.Enabled {
		t.Fatal("edit lost paused rule state")
	}
	if enabled, _, _ := user32.NewProc("IsWindowEnabled").Call(a.editor.controls[eNext]); enabled != 0 {
		t.Fatal("unchanged edit can be saved")
	}
	a.editorCommand(eChangeTarget)
	a.editorChoose("space")
	if a.cfg.Rules[0].Target != "mouse_middle" {
		t.Fatal("draft changed live rule")
	}
	// Exercise both choices of the real discard prompt without injecting input.
	answerDiscardPrompt(t, a, 7)
	if a.editor == nil {
		t.Fatal("keep editing choice discarded draft")
	}
	answerDiscardPrompt(t, a, 6)
	if a.editor != nil || a.cfg.Rules[0].Target != "mouse_middle" {
		t.Fatal("cancel changed original")
	}
	a.command(rowActionBase + 1)
	a.editorCommand(eChangeTarget)
	a.editorChoose("space")
	a.editorCommand(eNext)
	a.editorCommand(eNext)
	if a.editor != nil || a.cfg.Rules[0].Target != "space" || a.cfg.Rules[0].Enabled {
		t.Fatal("saved edit did not retain paused state")
	}
	// Adding never releases an existing global pause.
	a.command(idToggle)
	a.command(idNew)
	a.editorChoose("A")
	a.editorCommand(eNext)
	a.editorChoose("B")
	a.editorCommand(eNext)
	if windowText(a.editor.controls[eNext]) != "添加并保持暂停" {
		t.Fatal("global pause not explained")
	}
	a.editorCommand(eNext)
	if a.cfg.Enabled || a.engine.enabled || len(a.cfg.Rules) != 2 {
		t.Fatal("add changed global pause")
	}
	// Conflict identifies the existing mapping and can switch directly to edit.
	a.command(idNew)
	a.editorChoose("A")
	if !strings.Contains(a.editorValidation(), "已有映射") {
		t.Fatal("duplicate source unrecognized")
	}
	if i := a.editorConflict(); i != 1 {
		t.Fatal(i)
	}
	answerNativePrompt(t, a, 6, func() { a.editorCommand(eConflict) })
	if a.editor.index != 1 || a.editor.step != 2 {
		t.Fatal("row edit lost identity")
	}
	a.closeEditor(true)
	testWizardCapture(t, a)
	testTransactionalSave(t, a)
	testNativeLayout(t, a)
	// Deletion and undo preserve the actual paused rule and persist the result.
	a.selectRule(0)
	removed := a.cfg.Rules[0]
	a.command(idDelete)
	a.command(idUndo)
	if a.cfg.Rules[0] != removed {
		t.Fatal("undo changed rule state")
	}
	saved, err := loadConfig(a.path)
	if err != nil || !reflect.DeepEqual(saved, a.cfg) {
		t.Fatal("settings round trip failed")
	}
	a.successMessage = ""
	a.layoutMain()
	captureInterface(t, a.hwnd, "v2-main.png")
	testInformationFlow(t, a)
	// UI changes cannot re-enable a failed input service, and output handoff
	// remains deferred until the hook returns, just as before the redesign.
	received := make(chan []output, 1)
	a.sender = newInputSender(func(batch []output) bool { received <- append([]output(nil), batch...); return true }, nil)
	defer func() { a.sender.close(); awaitSender(t, a.sender.done) }()
	a.emit([]output{{"mouse_middle", true}, {"mouse_middle", false}})
	if len(a.pendingOutput) != 2 {
		t.Fatal("output no longer deferred")
	}
	windowProc(a.hwnd, wmSendOutput, 0, 0)
	select {
	case batch := <-received:
		if len(batch) != 2 || !batch[0].Down || batch[1].Down {
			t.Fatal(batch)
		}
	case <-time.After(time.Second):
		t.Fatal("output handoff blocked")
	}
	windowProc(a.hwnd, wmInputError, inputTimedOut, 0)
	windowProc(a.hwnd, wmHotkey, 0, 0)
	if a.cfg.Enabled || a.engine.enabled || !strings.Contains(a.inputFailureMessage, "超时") {
		t.Fatal("input fault hidden")
	}
	if enabled, _, _ := user32.NewProc("IsWindowEnabled").Call(a.controls[idToggle]); enabled != 0 {
		t.Fatal("fault leaves the switch looking usable")
	}
	send(a.controls[idFaultDetails], 0x00f5, 0, 0)
	if a.info == nil || a.info.page != "support" {
		t.Fatal("fault has no actionable details")
	}
	a.closeInformation()
	captureInterface(t, a.hwnd, "v2-input-failure.png")
	a.command(idNew)
	a.editorChoose("F8")
	a.editorCommand(eNext)
	a.editorChoose("enter")
	a.editorCommand(eNext)
	a.editorCommand(eNext)
	if a.engine.enabled || a.cfg.Enabled || a.successActive {
		t.Fatal("saving enabled failed input")
	}
}

func answerDiscardPrompt(t *testing.T, a *application, button uintptr) {
	answerNativePrompt(t, a, button, func() { a.requestCloseEditor() })
}

func answerNativePrompt(t *testing.T, a *application, button uintptr, action func()) {
	t.Helper()
	done := make(chan bool, 1)
	go func() {
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			hwnd, _, _ := findWindow.Call(uintptr(unsafe.Pointer(utf16("#32770"))), uintptr(unsafe.Pointer(utf16("保留还是放弃修改"))))
			if hwnd != 0 && a.ownWindow(hwnd) {
				postMessage.Call(hwnd, wmCommand, button, 0)
				done <- true
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		done <- false
	}()
	action()
	if !<-done {
		t.Fatal("discard confirmation not found")
	}
}

func testNativeLayout(t *testing.T, a *application) {
	t.Helper()
	previous := a.cfg
	a.cfg.Rules = nil
	for index := 5; index < 25; index++ {
		a.cfg.Rules = append(a.cfg.Rules, rule{keys[index].ID, "mouse_middle", index%2 == 0})
	}
	a.successMessage = ""
	a.refreshList()
	moveWindow.Call(a.hwnd, 0, 0, uintptr(a.px(760)), uintptr(a.px(590)), 1)
	a.selectRule(19)
	style := user32.NewProc("GetWindowLongPtrW")
	visible := func(hwnd uintptr) bool { bits, _, _ := style.Call(hwnd, intArg(-16)); return bits&wsVisible != 0 }
	if !visible(a.rowActions[19][1]) || visible(a.rowActions[0][1]) {
		t.Fatal("row buttons failed to follow scroll position")
	}
	w, h := a.clientSize(a.hwnd)
	mapPoints := user32.NewProc("MapWindowPoints")
	for _, hwnd := range []uintptr{a.controls[idAbout], a.controls[idSupport], a.controls[idHelp], a.controls[idNew], a.controls[idToggle], a.controls[idList], a.rowActions[19][0], a.rowActions[19][1], a.rowActions[19][2]} {
		var r rect
		getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		mapPoints.Call(0, a.hwnd, uintptr(unsafe.Pointer(&r)), 2)
		if r.Left < 0 || r.Top < 0 || int(r.Right) > a.px(w) || int(r.Bottom) > a.px(h) {
			t.Fatal("main action clipped at minimum size")
		}
	}
	captureInterface(t, a.hwnd, "v2-main-small-scrolled.png")
	// Native scrolling moves child windows as well. After repeated scrolling,
	// the actions must still align with and open the corresponding rule.
	for _, index := range []int{0, 19, 8, 18} {
		a.selectRule(index)
		if !visible(a.rowActions[index][1]) {
			t.Fatalf("scrolled rule %d lost edit button", index)
		}
		send(a.rowActions[index][1], 0x00f5, 0, 0) // BM_CLICK, including WM_COMMAND routing.
		if a.editor == nil || a.editor.index != index {
			t.Fatal("scrolled button opened a different rule")
		}
		a.closeEditor(true)
	}
	a.selectRule(19)
	send(a.controls[idList], 0x0100, 0x71, 0)
	if a.editor == nil || a.editor.index != 19 {
		t.Fatal("F2 did not open selected rule")
	}
	a.closeEditor(true)
	a.cfg = previous
	a.refreshList()
	moveWindow.Call(a.hwnd, 0, 0, uintptr(a.px(windowWidth+16)), uintptr(a.px(windowHeight+40)), 1)
}

func testNativeRowRepaint(t *testing.T, a *application) {
	t.Helper()
	list := a.controls[idList]
	getParent := user32.NewProc("GetParent")
	for _, button := range a.rowActions[0] {
		parent, _, _ := getParent.Call(button)
		if parent != list {
			t.Fatal("row actions can be overpainted by the list")
		}
	}
	// Exercise selection using the real list mouse messages. Only repaint the
	// list afterwards: repainting each button would mask the reported bug.
	preview := os.Getenv("SIMPLE_KEYMAP_UI_PREVIEW_DIR") != ""
	if preview {
		showWindow.Call(a.hwnd, 4)
		user32.NewProc("RedrawWindow").Call(a.hwnd, 0, 0, 0x185) // Invalidate, erase, all children, update now.
		defer showWindow.Call(a.hwnd, 0)
	}
	for _, selected := range []bool{false, true, false, true} {
		item := listItem{StateMask: 3}
		send(list, lvmFirst+43, ^uintptr(0), uintptr(unsafe.Pointer(&item)))
		if selected {
			position := uintptr(a.px(24)) | uintptr(a.px(40))<<16
			// The native list tracks selection in a nested message loop until
			// release; queue that release before sending the button-down.
			postMessage.Call(list, 0x0202, 0, position)
			send(list, 0x0201, 1, position)
			if a.selectedRule() != 0 {
				t.Fatal("native click failed to select rule")
			}
		}
		invalidateRect.Call(list, 0, 1)
		// Paint pending updates, including children invalidated by selection;
		// do not force invalidation of otherwise unchanged row buttons.
		user32.NewProc("RedrawWindow").Call(list, 0, 0, 0x180)
		for _, button := range a.rowActions[0] {
			style, _, _ := user32.NewProc("GetWindowLongPtrW").Call(button, intArg(-16))
			if style&wsVisible == 0 {
				t.Fatal("selection or repaint hid a row action")
			}
		}
		if selected {
			captureScreenInterface(t, a.hwnd, "v2-row-selected.png")
		} else {
			captureScreenInterface(t, a.hwnd, "v2-row-unselected.png")
		}
	}
	// Tab traversal must reach nested actions, and a native click must still
	// open the correct editor after selection, deselection and repainting.
	setFocus.Call(list)
	msg := winMessage{Hwnd: list, Message: 0x0100, WParam: 9}
	if !a.enterRowActions(&msg) {
		t.Fatal("Tab failed to enter row actions")
	}
	focus, _, _ := getFocus.Call()
	if focus != a.rowActions[0][0] {
		t.Fatalf("Tab cannot reach the row switch: focus %x (%s), list %x, switch %x", focus, windowText(focus), list, a.rowActions[0][0])
	}
	for _, next := range []uintptr{a.rowActions[0][1], a.rowActions[0][2]} {
		msg.Hwnd = focus
		isDialogMessage.Call(a.hwnd, uintptr(unsafe.Pointer(&msg)))
		focus, _, _ = getFocus.Call()
		if focus != next {
			t.Fatal("Tab failed to traverse row actions")
		}
	}
	send(a.rowActions[0][1], 0x00f5, 0, 0)
	if a.editor == nil || a.editor.index != 0 {
		t.Fatal("edit button stopped working after repaint")
	}
	a.closeEditor(true)
	// The global switch retains each rule's individual choice across both
	// clicks and the global shortcut. Its accessible name follows the state.
	before := append([]rule(nil), a.cfg.Rules...)
	for _, command := range []func(){func() { send(a.controls[idToggle], 0x00f5, 0, 0) }, func() { send(a.hwnd, wmHotkey, 0, 0) }} {
		command()
		if !reflect.DeepEqual(a.cfg.Rules, before) {
			t.Fatal("global switch changed individual rule choices")
		}
		state := "已暂停"
		if a.cfg.Enabled {
			state = "已开启"
		}
		if !strings.Contains(windowText(a.controls[idToggle]), state) {
			t.Fatal("global switch accessibility text is out of date")
		}
	}
}

func testWizardCapture(t *testing.T, a *application) {
	t.Helper()
	a.command(idNew)
	for _, enabled := range []bool{false, true} {
		a.cfg.Enabled = enabled
		a.apply()
		for _, step := range []int{0, 1} {
			if step == 1 {
				a.editor.draft.Source = "C"
			}
			a.editor.step = step
			a.renderEditor()
			for _, exit := range []string{"captured", "escape", "button", "timeout", "close"} {
				a.editorCommand(eRecord)
				if a.capturing == 0 || a.engine.enabled {
					t.Fatal("capture did not pause input")
				}
				if step == 0 && enabled && exit == "captured" {
					captureInterface(t, a.editor.hwnd, "v2-recording.png")
				}
				switch exit {
				case "captured":
					if !a.handleInput("F9", true) || a.captureIndex == 0 {
						t.Fatal("capture down not consumed")
					}
					if !a.capturePending || a.capturing == 0 {
						t.Fatal("capture must wait for release")
					}
					a.handleInput("F9", false)
					send(a.hwnd, wmCaptured, uintptr(a.captureIndex), a.captureGeneration)
				case "escape":
					a.handleInput("escape", true)
					a.handleInput("escape", false)
					send(a.hwnd, wmCaptured, 0, a.captureGeneration)
				case "button":
					a.editorCommand(eRecord)
				case "timeout":
					send(a.hwnd, wmTimer, 1, 0)
				case "close":
					if a.requestCloseEditor() {
						t.Fatal("recording Escape/close discarded dialog")
					}
				}
				if a.editor == nil || a.capturing != 0 || a.capturePending || a.engine.enabled != enabled {
					t.Fatalf("capture did not restore state: %s", exit)
				}
				if windowText(a.editor.controls[eRecord]) == "取消录入" {
					t.Fatal("record label stuck")
				}
			}
		}
	}
	// Late capture messages cannot replace a selection in a later session.
	a.editor.step = 0
	a.startCapture(idRecordSource)
	old := a.captureGeneration
	a.finishCapture(0)
	a.startCapture(idRecordSource)
	a.capturePending = true
	send(a.hwnd, wmCaptured, 6, old)
	if a.capturing == 0 {
		t.Fatal("stale capture completed new session")
	}
	a.finishCapture(0)
	// Recording the same physical source that was held before capture must
	// clear its old engine hold when the recorded release is swallowed.
	a.cfg.Enabled = true
	a.cfg.Rules[0].Enabled = true
	a.apply()
	source := a.cfg.Rules[0].Source
	a.engine.process(source, true)
	a.startCapture(idRecordSource)
	a.handleInput(source, true)
	a.handleInput(source, false)
	send(a.hwnd, wmCaptured, uintptr(a.captureIndex), a.captureGeneration)
	if _, held := a.engine.held[source]; held {
		t.Fatal("recording swallowed the old engine release")
	}
	a.closeEditor(true)
}

func testTransactionalSave(t *testing.T, a *application) {
	t.Helper()
	a.cfg.Enabled = true
	a.cfg.Rules[0].Enabled = true
	a.apply()
	a.selectRule(0)
	a.command(idEdit)
	a.editorCommand(eChangeTarget)
	a.editorChoose("F10")
	a.editorCommand(eNext)
	before := append([]rule(nil), a.cfg.Rules...)
	originalPath := a.path
	// Saving a mid-press edit releases the old target, while swallowing the
	// later physical release. The helper still receives no native test input.
	a.engine.process(a.cfg.Rules[0].Source, true)
	blocked := filepath.Join(t.TempDir(), "file-parent")
	if err := os.WriteFile(blocked, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	a.path = filepath.Join(blocked, "config.json")
	a.editorCommand(eNext)
	if a.editor == nil || a.editor.saveError == "" || !reflect.DeepEqual(before, a.cfg.Rules) {
		t.Fatal("save failure committed draft or closed editor")
	}
	captureInterface(t, a.editor.hwnd, "v2-save-failure.png")
	a.path = originalPath
	a.editorCommand(eNext)
	if a.editor != nil || a.cfg.Rules[0].Target != "F10" {
		t.Fatal("save retry failed")
	}
	if len(a.engine.refs) != 0 || !a.engine.held[a.cfg.Rules[0].Source].released {
		t.Fatal("save left old simulated key held")
	}
	if consumed, out := a.engine.process(a.cfg.Rules[0].Source, false); !consumed || len(out) != 0 {
		t.Fatal("save leaked a matching source release")
	}
	a.path = filepath.Join(blocked, "config.json")
	a.command(idToggle)
	if a.configSaveError == "" {
		t.Fatal("immediate change lost save warning")
	}
	a.command(idNew)
	a.startCapture(idRecordSource)
	a.finishCapture(0)
	a.closeEditor(true)
	if !strings.Contains(a.footerCopy(), "尚未保存") {
		t.Fatal("ordinary operation hid save failure")
	}
	diagnostic := a.diagnosticText()
	for _, sensitive := range []string{a.path, blocked, "mouse_x1", "config.json"} {
		if strings.Contains(diagnostic, sensitive) {
			t.Fatal("diagnostic leaks paths or rule details")
		}
	}
	a.path = originalPath
	a.command(idRetrySave)
	if a.configSaveError != "" {
		t.Fatal("retry did not clear warning")
	}
}

func testInformationFlow(t *testing.T, a *application) {
	t.Helper()
	a.command(idAbout)
	if a.info == nil || a.info.page != "about" {
		t.Fatal("about missing")
	}
	body := windowText(a.info.controls[infoBody])
	for _, s := range []string{"GPL-3.0-only", "不提供任何担保", "再分发"} {
		if !strings.Contains(body, s) {
			t.Fatal("legal notice missing")
		}
	}
	send(a.info.hwnd, wmCommand, infoLicense, 0)
	if windowText(a.info.controls[infoBody]) != normalizeNoticeText(licenseText) {
		t.Fatal("offline license truncated")
	}
	send(a.info.controls[infoBody], 0x00B6, 0, 10000)
	if send(a.info.controls[infoBody], 0x00CE, 0, 0) == 0 {
		t.Fatal("license not scrollable")
	}
	send(a.info.hwnd, wmCommand, infoBack, 0)
	send(a.info.hwnd, wmCommand, infoBrand, 0)
	if windowText(a.info.controls[infoBody]) != normalizeNoticeText(trademarksText) {
		t.Fatal("brand notice missing")
	}
	a.command(idHelp)
	if a.info.page != "help" {
		t.Fatal("help missing")
	}
	a.command(idSupport)
	testDiagnosticClipboard(t, a)
	a.closeInformation()
}

func TestVersionedSourceLinks(t *testing.T) {
	if versionSourceURL("1.0.1") != repositoryURL+"/tree/v1.0.1" {
		t.Fatal("source does not match release tag")
	}
	for _, version := range []string{"dev", "", "../main", "1.0.1/other"} {
		if versionSourceURL(version) != repositoryURL {
			t.Fatal("development source URL unsafe")
		}
	}
}

func testDiagnosticClipboard(t *testing.T, a *application) {
	t.Helper()
	if os.Getenv("SIMPLE_KEYMAP_TEST_CLIPBOARD") != "1" {
		return // Opt in on a test desktop; avoid replacing the user's clipboard.
	}
	getData := user32.NewProc("GetClipboardData")
	readText := func() string {
		opened, _, _ := openClipboard.Call(a.hwnd)
		if opened == 0 {
			t.Fatal("cannot open clipboard for round trip")
		}
		defer closeClipboard.Call()
		handle, _, _ := getData.Call(13)
		if handle == 0 {
			return ""
		}
		pointer, _, _ := globalLock.Call(handle)
		if pointer == 0 {
			t.Fatal("cannot lock clipboard text")
		}
		defer globalUnlock.Call(handle)
		size, _, _ := kernel32.NewProc("GlobalSize").Call(handle)
		units := unsafe.Slice(messageData[uint16](&pointer), int(size/2))
		for i, unit := range units {
			if unit == 0 {
				units = units[:i]
				break
			}
		}
		return string(unicode16.Decode(units))
	}
	previous := readText()
	defer func() { _ = writeClipboardText(a.hwnd, previous) }()
	send(a.info.hwnd, wmCommand, infoCopy, 0)
	if readText() != a.diagnosticText() {
		t.Fatal("diagnostic clipboard round trip failed")
	}
	if !strings.Contains(windowText(a.info.controls[infoCopy]), "已复制") {
		t.Fatal("copy success not visible")
	}
}

// Captures native control rendering only when explicitly requested locally.
func captureInterface(t *testing.T, hwnd uintptr, filename string) {
	captureNativeInterface(t, hwnd, filename, false)
}

// Screen captures deliberately avoid PrintWindow, which repaints all children
// and can hide the overlay bug seen after a list-only selection repaint.
func captureScreenInterface(t *testing.T, hwnd uintptr, filename string) {
	captureNativeInterface(t, hwnd, filename, true)
}

func captureNativeInterface(t *testing.T, hwnd uintptr, filename string, screen bool) {
	t.Helper()
	directory := os.Getenv("SIMPLE_KEYMAP_UI_PREVIEW_DIR")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	visible, _, _ := user32.NewProc("IsWindowVisible").Call(hwnd)
	if visible == 0 {
		showWindow.Call(hwnd, 4) // Show without activation, only for requested previews.
		defer showWindow.Call(hwnd, 0)
	}
	updateWindow.Call(hwnd)
	var bounds rect
	getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds)))
	w, h := int(bounds.Right-bounds.Left), int(bounds.Bottom-bounds.Top)
	dc, _, _ := user32.NewProc("GetWindowDC").Call(hwnd)
	defer user32.NewProc("ReleaseDC").Call(hwnd, dc)
	buffer, _, _ := createCompatibleDC.Call(dc)
	defer deleteDC.Call(buffer)
	bitmap, _, _ := createCompatibleBitmap.Call(dc, uintptr(w), uintptr(h))
	defer deleteObject.Call(bitmap)
	old, _, _ := selectObject.Call(buffer, bitmap)
	if screen {
		gdi32.NewProc("GdiFlush").Call()
		bitBlt.Call(buffer, 0, 0, uintptr(w), uintptr(h), dc, 0, 0, 0x00CC0020)
	} else {
		user32.NewProc("PrintWindow").Call(hwnd, buffer, 0)
	}
	selectObject.Call(buffer, old)
	header := struct {
		Size                   uint32
		Width, Height          int32
		Planes, BitCount       uint16
		Compression, SizeImage uint32
		XPels, YPels           int32
		Used, Important        uint32
	}{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}
	pixels := make([]byte, w*h*4)
	rows, _, _ := gdi32.NewProc("GetDIBits").Call(dc, bitmap, 0, uintptr(h), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), 0)
	if int(rows) != h {
		t.Fatal("native preview capture failed")
	}
	for offset := 0; offset < len(pixels); offset += 4 {
		pixels[offset], pixels[offset+2], pixels[offset+3] = pixels[offset+2], pixels[offset], 255
	}
	output, err := os.Create(filepath.Join(directory, filename))
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	if err := png.Encode(output, &image.RGBA{Pix: pixels, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}); err != nil {
		t.Fatal(err)
	}
}
