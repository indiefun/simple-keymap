//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	windowWidth  = 880
	windowHeight = 680
)

const (
	idSource = 101 + iota
	idTarget
	idRecordSource
	idRecordTarget
	idAdd
	idToggle
	idList
	idDelete
	idHelp
	idSupport
)

type application struct {
	hwnd, instance                          uintptr
	controls                                map[int]uintptr
	font, smallFont, titleFont, headingFont uintptr
	keyFont                                 uintptr
	brandIcon                               uintptr
	background, whiteBrush                  uintptr
	rowImages                               uintptr
	dpi                                     int
	cfg                                     config
	path                                    string
	engine                                  *engine
	sender                                  *inputSender
	pendingOutput                           []output
	outputPosted                            bool
	closing                                 bool
	keyboardHook, mouseHook                 uintptr
	updating                                bool
	capturing                               int
	captureHeld                             string
	capturePending                          bool
	emergencyHeld                           bool
	modifiers                               map[string]bool
	inputFailureMessage                     string
	status                                  string
}

var app *application
var windowCallback = syscall.NewCallback(windowProc)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("Simple Keymap " + appVersion)
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--input-worker" {
		if !strings.EqualFold(filepath.Base(os.Args[0]), "SimpleKeymap.Input.exe") {
			return
		}
		if err := runInputWorker(os.Stdin, os.Stdout, sendNativeInput); err != nil {
			os.Exit(1)
		}
		return
	}
	// The helper file is not a second app entry point.
	if strings.EqualFold(filepath.Base(os.Args[0]), "SimpleKeymap.Input.exe") {
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	setDPIAware.Call()
	mutex, _, mutexErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(utf16("Local\\SimpleKeymap.SingleInstance"))))
	if mutex != 0 {
		defer closeHandle.Call(mutex)
	}
	if mutexErr == syscall.Errno(183) {
		hwnd, _, _ := findWindow.Call(uintptr(unsafe.Pointer(utf16(appClass))), 0)
		if hwnd != 0 {
			showWindow.Call(hwnd, 9)
			setForeground.Call(hwnd)
		}
		return
	}
	path, err := configPath()
	if err != nil {
		showError(0, "无法找到配置目录："+err.Error())
		return
	}
	cfg, loadErr := loadConfig(path)
	app = &application{
		controls: map[int]uintptr{}, cfg: cfg, path: path, engine: newEngine(),
		modifiers: map[string]bool{}, status: "准备好了，添加一条映射就可以开始。", dpi: 96,
	}
	if getDPIForSystem.Find() == nil {
		dpi, _, _ := getDPIForSystem.Call()
		if dpi != 0 {
			app.dpi = int(dpi)
		}
	}
	// Keep every control reachable on smaller screens and large scaling settings.
	sw, _, _ := getSystemMetrics.Call(0)
	sh, _, _ := getSystemMetrics.Call(1)
	for _, limit := range []int{(int(sw) - 48) * 96 / windowWidth, (int(sh) - 100) * 96 / windowHeight} {
		if limit > 0 && limit < app.dpi {
			app.dpi = limit
		}
	}
	if err = app.create(); err != nil {
		showError(0, err.Error())
		return
	}
	defer app.dispose()
	executable, err := os.Executable()
	if err != nil {
		showError(app.hwnd, err.Error())
		return
	}
	cmd, err := inputWorkerCommand(executable)
	if err != nil {
		showError(app.hwnd, err.Error())
		return
	}
	transport, err := newInputTransport(cmd, 750*time.Millisecond)
	if err != nil {
		showError(app.hwnd, err.Error())
		return
	}
	app.sender = newInputSender(transport.deliver, func() {
		postMessage.Call(app.hwnd, wmInputError, uintptr(transport.failure), 0)
	})
	defer app.shutdownInput(transport)
	if err = app.installHooks(); err != nil {
		showError(app.hwnd, err.Error())
		return
	}
	app.apply()
	app.refreshList()
	if len(app.cfg.Rules) != 0 {
		app.status = fmt.Sprintf("已载入 %d 条映射。", len(app.cfg.Rules))
	}
	showWindow.Call(app.hwnd, 5)
	updateWindow.Call(app.hwnd)
	if loadErr != nil {
		showError(app.hwnd, "无法载入配置，已暂停映射。\n"+loadErr.Error()+"\n\n配置位置："+path)
	}
	runMessageLoop(app.hwnd)
}

func inputWorkerCommand(executable string) (*exec.Cmd, error) {
	helper := filepath.Join(filepath.Dir(executable), "SimpleKeymap.Input.exe")
	info, err := os.Stat(helper)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("缺少输入服务文件：%s。请重新运行 build.ps1，并保留 dist 里的两个 exe 文件。", helper)
	}
	cmd := exec.Command(helper, "--input-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd, nil
}

func (a *application) shutdownInput(transport *inputTransport) {
	a.beginShutdown()
	if a.sender != nil {
		a.sender.close()
		select {
		case <-a.sender.done:
		case <-time.After(time.Second):
		}
	}
	transport.close()
}

func (a *application) beginShutdown() {
	if a.closing {
		return
	}
	a.closing = true
	killTimer.Call(a.hwnd, 1)
	a.stopHooks()
}

func runMessageLoop(hwnd uintptr) {
	var msg winMessage
	for {
		result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) <= 0 {
			break
		}
		// IsDialogMessage gives native tab/arrow navigation to our regular window.
		handled, _, _ := isDialogMessage.Call(hwnd, uintptr(unsafe.Pointer(&msg)))
		if handled == 0 {
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func (a *application) px(n int) int { return n * a.dpi / 96 }

func (a *application) makeFont(size, weight int) uintptr {
	f, _, _ := createFont.Call(intArg(-a.px(size)), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf16("Microsoft YaHei UI"))))
	return f
}

func (a *application) create() error {
	a.instance, _, _ = getModuleHandle.Call(0)
	a.font, a.smallFont = a.makeFont(14, 400), a.makeFont(12, 400)
	a.titleFont, a.headingFont = a.makeFont(23, 600), a.makeFont(17, 600)
	a.keyFont = a.makeFont(19, 600)
	a.background, _, _ = createSolidBrush.Call(canvasColor)
	a.whiteBrush, _, _ = createSolidBrush.Call(rgb(255, 255, 255))
	cursor, _, _ := loadCursor.Call(0, 32512)
	icon := loadBrandIcon(a.instance, a.px(32))
	smallIcon := loadBrandIcon(a.instance, a.px(16))
	a.brandIcon = loadBrandIcon(a.instance, a.px(44))
	wc := windowClass{Size: uint32(unsafe.Sizeof(windowClass{})), WndProc: windowCallback,
		Instance: a.instance, Cursor: cursor, Icon: icon, IconSmall: smallIcon,
		Background: a.background, ClassName: utf16(appClass)}
	if r, _, err := registerClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("无法创建窗口：%v", err)
	}
	init := struct{ Size, Classes uint32 }{8, 1}
	initCommonControls.Call(uintptr(unsafe.Pointer(&init)))
	style := uintptr(0x00C80000 | 0x00080000 | 0x00020000 | wsClipChildren) // Caption, menu, minimize.
	r := rect{Right: int32(a.px(windowWidth)), Bottom: int32(a.px(windowHeight))}
	adjustWindowRect.Call(uintptr(unsafe.Pointer(&r)), style, 0, 0)
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	screenW, _, _ := getSystemMetrics.Call(0)
	screenH, _, _ := getSystemMetrics.Call(1)
	x, y := (int(screenW)-w)/2, (int(screenH)-h)/2
	if y < 0 {
		y = 0
	}
	var err error
	a.hwnd, _, err = createWindowEx.Call(0x00010000, uintptr(unsafe.Pointer(utf16(appClass))),
		uintptr(unsafe.Pointer(utf16(appTitle+" · "+appVersion))), style, intArg(x), intArg(y), intArg(w), intArg(h), 0, 0, a.instance, 0)
	if a.hwnd == 0 {
		return fmt.Errorf("无法创建窗口：%v", err)
	}
	if err := a.createControls(); err != nil {
		destroyWindow.Call(a.hwnd)
		return err
	}
	return nil
}

func (a *application) control(class, text string, id int, style uintptr, x, y, w, h int) (uintptr, error) {
	hwnd, _, err := createWindowEx.Call(0, uintptr(unsafe.Pointer(utf16(class))), uintptr(unsafe.Pointer(utf16(text))),
		wsChild|wsVisible|wsTabstop|style, intArg(a.px(x)), intArg(a.px(y)), intArg(a.px(w)), intArg(a.px(h)), a.hwnd, uintptr(id), a.instance, 0)
	if hwnd == 0 {
		return 0, fmt.Errorf("无法创建控件：%v", err)
	}
	a.controls[id] = hwnd
	send(hwnd, wmSetFont, a.font, 1)
	return hwnd, nil
}

func (a *application) createControls() error {
	types := []struct {
		class, text string
		id          int
		style       uintptr
		x, y, w, h  int
	}{
		{"BUTTON", "暂停映射", idToggle, 0xB, 684, 102, 144, 32},
		{"COMBOBOX", "按下这个键", idSource, 0x3 | wsVscroll, 32, 242, 368, 370},
		{"BUTTON", "录入按键", idRecordSource, 0xB, 308, 204, 92, 28},
		{"COMBOBOX", "变为这个键", idTarget, 0x3 | wsVscroll, 480, 242, 368, 370},
		{"BUTTON", "录入按键", idRecordTarget, 0xB, 756, 204, 92, 28},
		{"BUTTON", "+ 添加映射", idAdd, 0xB, 688, 338, 160, 40},
		{"SysListView32", "映射列表", idList, 0x1 | 0x4 | 0x8 | 0x40 | 0x4000, 40, 478, 800, 144},
		{"BUTTON", "删除所选", idDelete, 0xB, 740, 400, 108, 30},
		{"BUTTON", "使用帮助", idHelp, 0xB, 776, 30, 72, 30},
		{"BUTTON", "支持与反馈", idSupport, 0xB, 658, 30, 98, 30},
	}
	for _, c := range types {
		if _, err := a.control(c.class, c.text, c.id, c.style, c.x, c.y, c.w, c.h); err != nil {
			return err
		}
	}
	for _, id := range []int{idSource, idTarget} {
		for _, k := range keys {
			send(a.controls[id], cbAddString, 0, uintptr(unsafe.Pointer(utf16(k.Label))))
		}
		send(a.controls[id], 0x0153, 0, uintptr(a.px(28))) // CB_SETITEMHEIGHT, list items.
		send(a.controls[id], 0x0153, ^uintptr(0), uintptr(a.px(70)))
		send(a.controls[id], 0x0161, 12, 0) // CB_SETMINVISIBLE.
		setWindowTheme.Call(a.controls[id], uintptr(unsafe.Pointer(utf16("Explorer"))), 0)
		setWindowSubclass.Call(a.controls[id], pickerCallback, uintptr(id), 0)
	}
	send(a.controls[idSource], cbSetCurSel, 0, 0)
	send(a.controls[idTarget], cbSetCurSel, 2, 0)
	list := a.controls[idList]
	setWindowTheme.Call(list, uintptr(unsafe.Pointer(utf16("Explorer"))), 0)
	setWindowSubclass.Call(list, listCallback, idList, 0)
	send(list, lvmFirst+54, 0, 0x4|0x20|0x10000) // Checkboxes, full-row select, double buffering.
	send(list, lvmFirst+1, 0, rgb(255, 255, 255))
	send(list, lvmFirst+36, 0, rgb(38, 48, 62))
	send(list, lvmFirst+38, 0, rgb(255, 255, 255))
	for i, c := range []struct {
		text  string
		width int
	}{{"按下这个键", 352}, {"变为", 320}, {"状态", 110}} {
		col := listColumn{Mask: 0x1 | 0x2 | 0x4, Text: utf16(c.text), Width: int32(a.px(c.width))}
		send(list, lvmFirst+97, uintptr(i), uintptr(unsafe.Pointer(&col)))
	}
	a.rowImages, _, _ = imageListCreate.Call(1, uintptr(a.px(47)), 0x20, 1, 1)
	if a.rowImages != 0 {
		send(list, lvmFirst+3, 1, a.rowImages)
	}
	enableWindow.Call(a.controls[idDelete], 0)
	return nil
}

func (a *application) dispose() {
	for _, object := range []uintptr{a.font, a.smallFont, a.titleFont, a.headingFont, a.keyFont, a.background, a.whiteBrush} {
		if object != 0 {
			deleteObject.Call(object)
		}
	}
	if a.rowImages != 0 {
		imageListDestroy.Call(a.rowImages)
	}
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if app != nil {
		if app.closing && (msg == wmInputError || msg == wmSendOutput || msg == wmCaptured || msg == wmHotkey) {
			return 0
		}
		switch msg {
		case wmPaint:
			app.paint(hwnd)
			return 0
		case wmEraseBkgnd:
			return 1
		case wmDrawItem:
			app.drawButton(messageData[drawItemStruct](&lParam))
			return 1
		case wmCommand:
			if wParam>>16 == 0 {
				app.command(int(wParam & 0xFFFF))
			} else if wParam>>16 == 1 && (int(wParam&0xFFFF) == idSource || int(wParam&0xFFFF) == idTarget) {
				app.updateDraft()
			}
			return 0
		case wmNotify:
			h := messageData[notifyHeader](&lParam)
			if h.ID == idList && int32(h.Code) == -12 {
				return app.drawList(messageData[listCustomDraw](&lParam))
			}
			if h.ID == idList && int32(h.Code) == -101 && !app.updating {
				app.readChecks()
				selected := send(app.controls[idList], lvmFirst+12, ^uintptr(0), 0x2)
				enableWindow.Call(app.controls[idDelete], boolArg(int32(selected) >= 0))
			}
		case wmHotkey:
			if app.inputFailureMessage != "" {
				app.status = app.inputFailureMessage
				invalidateRect.Call(app.hwnd, 0, 0)
				return 0
			}
			if app.capturing != 0 {
				app.finishCapture(0)
			}
			app.cfg.Enabled = !app.cfg.Enabled
			app.changed("已切换映射状态。")
			return 0
		case wmCaptured:
			app.finishCapture(int(wParam))
			return 0
		case wmTimer:
			if wParam == 1 {
				app.finishCapture(0)
			}
			return 0
		case wmInputError:
			if app.inputFailureMessage != "" {
				return 0
			}
			app.cfg.Enabled = false
			message := "映射已暂停：无法发送按键。目标程序若以管理员运行，请用相同权限启动轻映。"
			if wParam == inputTimedOut {
				message = "映射已暂停：Windows 输入发送超时。请保存工作并重启 Windows，再开启映射。"
			} else if wParam == inputServiceLost {
				message = "映射已暂停：输入服务已退出，请重新启动轻映。"
			}
			app.inputFailureMessage = message
			app.changed(message)
			return 0
		case wmSendOutput:
			// This posted message runs after the physical hook has returned.
			app.flushOutput()
			return 0
		case wmClose:
			app.beginShutdown()
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

func boolArg(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

func (a *application) command(id int) {
	switch id {
	case idToggle:
		if a.inputFailureMessage != "" {
			showError(a.hwnd, a.inputFailureMessage)
			return
		}
		a.cfg.Enabled = !a.cfg.Enabled
		a.changed("已切换映射状态。")
	case idAdd:
		i, j := int32(send(a.controls[idSource], cbGetCurSel, 0, 0)), int32(send(a.controls[idTarget], cbGetCurSel, 0, 0))
		if i < 0 || j < 0 {
			return
		}
		r := rule{Source: keys[i].ID, Target: keys[j].ID, Enabled: true}
		rules := append(append([]rule{}, a.cfg.Rules...), r)
		if err := validateRules(rules); err != nil {
			showError(a.hwnd, err.Error())
			return
		}
		a.cfg.Rules = rules
		a.changed("已添加：" + keyLabel(r.Source) + " → " + keyLabel(r.Target) + "。")
		a.refreshList()
	case idDelete:
		i := int32(send(a.controls[idList], lvmFirst+12, ^uintptr(0), 0x2))
		if i < 0 || int(i) >= len(a.cfg.Rules) {
			return
		}
		a.cfg.Rules = append(a.cfg.Rules[:i], a.cfg.Rules[i+1:]...)
		a.changed("已删除所选映射。")
		a.refreshList()
	case idRecordSource, idRecordTarget:
		if a.capturing != 0 {
			a.finishCapture(0)
		} else {
			a.startCapture(id)
		}
	case idHelp:
		messageBox.Call(a.hwnd, uintptr(unsafe.Pointer(utf16(
			"轻映 "+appVersion+"\n\n"+
				"1. 选择原按键和目标按键，点击「添加映射」。\n"+
				"2. 也可以点击「录入按键」，再按一下键盘或鼠标按键。Esc 取消录入。\n"+
				"3. 列表左侧勾选启用，取消勾选暂停；选中一行可删除。\n\n"+
				"按住原按键 = 持续按住目标按键，松开即释放。\n"+
				"Ctrl + Alt + F12：全局暂停 / 恢复。\n"+
				"配置窗口内不执行映射，方便随时修改。\n"+
				"最小化后继续生效；关闭窗口即停止映射并退出。\n\n"+
				"只映射单个按键，不支持组合键、宏或滚轮滚动。\n"+
				"目标程序以管理员运行时，轻映也需以管理员运行。\n"+
				"部分游戏可能不接受模拟输入。\n\n"+
				"配置自动保存至：\n"+a.path))), uintptr(unsafe.Pointer(utf16("轻映 · 使用说明"))), 0x40)
	case idSupport:
		a.openSupport()
	}
}

func (a *application) apply() {
	a.emit(a.engine.configure(a.cfg.Enabled && a.capturing == 0 && a.inputFailureMessage == "" && !a.closing, a.cfg.Rules))
	if a.inputFailureMessage != "" {
		setText(a.controls[idToggle], "查看原因")
	} else if a.cfg.Enabled {
		setText(a.controls[idToggle], "暂停映射")
	} else {
		setText(a.controls[idToggle], "开启映射")
	}
	invalidateRect.Call(a.hwnd, 0, 0)
	invalidateRect.Call(a.controls[idToggle], 0, 0)
	// Owner-drawn recording buttons depend on capturing as well as their
	// native text. Repaint them even if EnableWindow did not change state.
	for _, id := range []int{idSource, idTarget, idList, idRecordSource, idRecordTarget} {
		invalidateRect.Call(a.controls[id], 0, 0)
	}
}

func (a *application) draftHint() (string, bool) {
	i, j := int32(send(a.controls[idSource], cbGetCurSel, 0, 0)), int32(send(a.controls[idTarget], cbGetCurSel, 0, 0))
	if i < 0 || j < 0 {
		return "选择两个按键，开始创建映射。", false
	}
	if i == j {
		return "两个按键相同，请换一个目标按键。", false
	}
	for _, r := range a.cfg.Rules {
		if r.Source == keys[i].ID {
			if r.Target == keys[j].ID {
				return "这条映射已在下方列表中。", false
			}
			return "这个原按键已有映射，删除旧规则后可重新设置。", false
		}
	}
	return "按下与抬起同步，长按也一样。", true
}

func (a *application) updateDraft() {
	_, valid := a.draftHint()
	enableWindow.Call(a.controls[idAdd], boolArg(valid && a.capturing == 0))
	for _, id := range []int{idAdd, idSource, idTarget} {
		invalidateRect.Call(a.controls[id], 0, 0)
	}
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *application) changed(message string) {
	a.apply()
	a.status = message
	if a.inputFailureMessage != "" {
		a.status = a.inputFailureMessage
	}
	if err := saveConfig(a.path, a.cfg); err != nil {
		a.status = "当前已生效，但配置保存失败。"
		showError(a.hwnd, "配置保存失败："+err.Error()+"\n\n当前映射仍然生效。")
	}
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *application) refreshList() {
	a.updating = true
	defer func() { a.updating = false }()
	list := a.controls[idList]
	send(list, lvmFirst+9, 0, 0)
	for i, r := range a.cfg.Rules {
		item := listItem{Mask: 0x1 | 0x8, Item: int32(i), Text: utf16(keyCaption(r.Source)), StateMask: 0xF000, State: 0x1000}
		if r.Enabled {
			item.State = 0x2000
		}
		send(list, lvmFirst+77, 0, uintptr(unsafe.Pointer(&item)))
		// LVS_EX_CHECKBOXES initializes inserted rows to unchecked, so apply
		// the saved checkbox state explicitly after insertion.
		send(list, lvmFirst+43, uintptr(i), uintptr(unsafe.Pointer(&item)))
		item = listItem{SubItem: 1, Text: utf16("→    " + keyCaption(r.Target))}
		send(list, lvmFirst+116, uintptr(i), uintptr(unsafe.Pointer(&item)))
		status := "已暂停"
		if r.Enabled {
			status = "已启用"
		}
		item = listItem{SubItem: 2, Text: utf16(status)}
		send(list, lvmFirst+116, uintptr(i), uintptr(unsafe.Pointer(&item)))
	}
	showWindow.Call(list, boolArg(len(a.cfg.Rules) != 0)*5)
	enableWindow.Call(a.controls[idDelete], 0)
	a.updateDraft()
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *application) readChecks() {
	changed := false
	for i := range a.cfg.Rules {
		state := send(a.controls[idList], lvmFirst+44, uintptr(i), 0xF000)
		enabled := state>>12 == 2
		if a.cfg.Rules[i].Enabled != enabled {
			a.cfg.Rules[i].Enabled, changed = enabled, true
		}
	}
	if changed {
		a.changed("规则状态已更新。")
		a.updating = true
		for i, r := range a.cfg.Rules {
			text := "已暂停"
			if r.Enabled {
				text = "已启用"
			}
			item := listItem{SubItem: 2, Text: utf16(text)}
			send(a.controls[idList], lvmFirst+116, uintptr(i), uintptr(unsafe.Pointer(&item)))
		}
		a.updating = false
	}
}

func (a *application) startCapture(id int) {
	a.capturing = id
	a.capturePending = false
	a.apply()
	for _, other := range []int{idSource, idTarget, idRecordSource, idRecordTarget, idAdd, idToggle, idList, idDelete, idHelp, idSupport} {
		enableWindow.Call(a.controls[other], boolArg(other == id))
	}
	setText(a.controls[id], "取消录入")
	a.status = "现在按一下要录入的按键或鼠标按钮。Esc 取消，8 秒后自动结束。"
	setTimer.Call(a.hwnd, 1, 8000, 0)
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *application) finishCapture(index int) {
	if a.capturing == 0 {
		return
	}
	killTimer.Call(a.hwnd, 1)
	recordButton := a.capturing
	combo := idSource
	if recordButton == idRecordTarget {
		combo = idTarget
	}
	// SetWindowText can synchronously request owner drawing. Clear the
	// recording state first so that repaint already sees the normal label.
	a.capturing = 0
	a.capturePending = false
	setText(a.controls[recordButton], "录入按键")
	for _, id := range []int{idSource, idTarget, idRecordSource, idRecordTarget, idAdd, idToggle, idList, idHelp, idSupport} {
		enableWindow.Call(a.controls[id], 1)
	}
	selected := int32(send(a.controls[idList], lvmFirst+12, ^uintptr(0), 0x2))
	enableWindow.Call(a.controls[idDelete], boolArg(selected >= 0))
	if index > 0 && index <= len(keys) {
		send(a.controls[combo], cbSetCurSel, uintptr(index-1), 0)
		a.status = "已录入：" + keys[index-1].Label + "。点击「添加映射」保存。"
	} else {
		a.status = "已取消录入。"
	}
	a.apply()
	a.updateDraft()
}
