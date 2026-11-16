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
	windowWidth  = 900
	windowHeight = 650
)

const (
	idRecordSource = 101 + iota
	idRecordTarget
	idAdd
	idToggle
	idList
	idDelete
	idHelp
	idSupport
	idAbout
	idEdit
	idRetrySave
	idNew
	idPreset
	idFaultDetails
	idUndo
	idUpdates
)

type application struct {
	hwnd, instance                          uintptr
	controls                                map[int]uintptr
	font, smallFont, titleFont, headingFont uintptr
	keyFont                                 uintptr
	brandIcon                               uintptr
	background, whiteBrush                  uintptr
	accentBrush                             uintptr
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
	configSaveError                         string
	configurationActive                     bool
	info                                    *informationWindow
	editor                                  *mappingEditor
	captureIndex                            int
	captureGeneration                       uintptr
	deletedRule                             *rule
	deletedIndex                            int
	rowActions                              map[int][3]uintptr
	updates                                 *updateState
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
	cfg, loadErr := loadPortableConfig(path)
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
	if loadErr != nil {
		app.status = "配置未能载入，映射已暂停。"
	}
	showWindow.Call(app.hwnd, 5)
	updateWindow.Call(app.hwnd)
	if app.cfg.CheckUpdatesOnStartup {
		app.startUpdateCheck()
	}
	if loadErr != nil {
		showError(app.hwnd, "无法载入配置，已暂停映射。\n"+loadErr.Error()+"\n\n配置位置："+path)
	}
	runMessageLoop(app.hwnd)
}

func inputWorkerCommand(executable string) (*exec.Cmd, error) {
	helper := filepath.Join(filepath.Dir(executable), "SimpleKeymap.Input.exe")
	info, err := os.Stat(helper)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("缺少配套文件 SimpleKeymap.Input.exe。\n请重新下载完整压缩包，全部解压到同一文件夹后，再运行 SimpleKeymap.exe。\n\n未找到文件：%s", helper)
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
	if a.updates != nil && a.updates.cancel != nil {
		a.updates.cancel()
	}
	a.closeEditor(true)
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
		var handled uintptr
		if app != nil && app.enterRowActions(&msg) {
			continue
		}
		if app != nil && app.editor != nil && app.editorOwns(msg.Hwnd) {
			if msg.Message == 0x0100 && msg.WParam == 0x1B {
				app.requestCloseEditor()
				continue
			}
			// The recorder swallows both edges before posting completion. Never
			// let its key-up become a dialog action (in particular Enter).
			if msg.Message == 0x0100 && msg.WParam == 0x0D {
				app.editorEnter()
				continue
			}
			handled, _, _ = isDialogMessage.Call(app.editor.hwnd, uintptr(unsafe.Pointer(&msg)))
		}
		if app != nil && app.info != nil {
			if msg.Message == 0x0100 && msg.WParam == 0x1B && app.ownInformationWindow(msg.Hwnd) {
				app.closeInformation()
				continue
			}
			handled, _, _ = isDialogMessage.Call(app.info.hwnd, uintptr(unsafe.Pointer(&msg)))
		}
		if handled == 0 {
			handled, _, _ = isDialogMessage.Call(hwnd, uintptr(unsafe.Pointer(&msg)))
		}
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
	a.accentBrush, _, _ = createSolidBrush.Call(accentWash)
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
	if err := a.registerInformationWindow(); err != nil {
		return err
	}
	if err := a.registerEditorWindow(); err != nil {
		return err
	}
	init := struct{ Size, Classes uint32 }{8, 1}
	initCommonControls.Call(uintptr(unsafe.Pointer(&init)))
	style := uintptr(0x00CC0000 | 0x00080000 | 0x00020000 | wsClipChildren)
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
