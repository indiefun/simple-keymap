//go:build windows

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	unicode16 "unicode/utf16"
	"unsafe"
)

// Called on the same locked UI thread as TestNativeRuleAndCaptureFlow. These
// use real controls and persisted settings, with no input hooks or SendInput.
func testAdvancedInterface(t *testing.T, a *application) {
	t.Helper()
	selectKey := func(control int, id string) {
		for index, k := range keys {
			if k.ID == id {
				send(a.controls[control], cbSetCurSel, uintptr(index), 0)
				send(a.hwnd, wmCommand, uintptr(control)|1<<16, a.controls[control])
				return
			}
		}
		t.Fatalf("key missing: %s", id)
	}
	a.cfg.Enabled = true
	selectKey(idSource, "mouse_x1")
	selectKey(idTarget, "mouse_middle")
	a.command(idAdd)
	a.command(idEdit)
	if !a.editing || windowText(a.controls[idAdd]) != "保存修改" {
		t.Fatal("selected rule did not enter edit mode")
	}
	selectKey(idTarget, "space")
	if a.cfg.Rules[0].Target != "mouse_middle" {
		t.Fatal("draft changed rule before saving")
	}
	a.command(idCancelEdit)
	if a.editing || a.cfg.Rules[0].Target != "mouse_middle" {
		t.Fatal("cancel changed original rule")
	}
	activation := struct {
		Header        notifyHeader
		Item, SubItem int32
	}{Header: notifyHeader{Hwnd: a.controls[idList], ID: idList, Code: ^uint32(2)}, Item: 0}
	send(a.hwnd, wmNotify, idList, uintptr(unsafe.Pointer(&activation)))
	if !a.editing {
		t.Fatal("double click did not edit the clicked rule")
	}
	captureInterface(t, a.hwnd, "app-edit.png")
	a.command(idCancelEdit)
	a.cfg.Rules[0].Enabled = false
	a.refreshList()
	a.selectRule(0)
	a.command(idEdit)
	selectKey(idTarget, "space")
	a.command(idAdd)
	if a.editing || len(a.cfg.Rules) != 1 || a.cfg.Rules[0].Target != "space" || a.cfg.Rules[0].Enabled {
		t.Fatalf("edit lost rule identity or enabled state: %#v", a.cfg.Rules)
	}
	saved, err := loadConfig(a.path)
	if err != nil || !reflect.DeepEqual(saved, a.cfg) {
		t.Fatal("edited rule was not persisted")
	}
	selectKey(idSource, "A")
	selectKey(idTarget, "B")
	a.command(idAdd)
	a.selectRule(0)
	a.command(idEdit)
	selectKey(idSource, "A")
	if _, valid := a.draftHint(); valid {
		t.Fatal("edit accepted duplicate source")
	}
	before := append([]rule(nil), a.cfg.Rules...)
	a.command(idAdd)
	if !reflect.DeepEqual(before, a.cfg.Rules) {
		t.Fatal("invalid edit changed rules")
	}
	a.command(idCancelEdit)
	selectKey(idSource, "A")
	a.command(idLocateRule)
	if a.selectedRule() != 1 {
		t.Fatal("duplicate source did not locate existing rule")
	}
	// Save failure persists through other UI actions and clears only on success.
	originalPath := a.path
	blocked := filepath.Join(t.TempDir(), "file-as-parent")
	if err := os.WriteFile(blocked, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	a.path = filepath.Join(blocked, "config.json")
	a.command(idToggle)
	if a.configSaveError == "" || !strings.Contains(a.footerCopy(), "未保存") {
		t.Fatal("save failure not visible")
	}
	a.status = "普通操作提示"
	a.startCapture(idRecordTarget)
	a.finishCapture(0)
	if !strings.Contains(a.footerCopy(), "未保存") {
		t.Fatal("ordinary operation hid save failure")
	}
	diagnostic := a.diagnosticText()
	for _, required := range []string{appVersion, "Windows", "总开关", "规则数量", "更改未保存", "复现步骤"} {
		if !strings.Contains(diagnostic, required) {
			t.Errorf("diagnostic lacks %s", required)
		}
	}
	for _, sensitive := range []string{a.path, blocked, "mouse_x1", "config.json"} {
		if strings.Contains(diagnostic, sensitive) {
			t.Errorf("diagnostic contains private detail %q", sensitive)
		}
	}
	a.status = ""
	captureInterface(t, a.hwnd, "app-save-failure.png")
	a.path = originalPath
	a.command(idRetrySave)
	if a.configSaveError != "" || strings.Contains(a.footerCopy(), "未保存") {
		t.Fatal("successful retry did not clear failure")
	}
	saved, err = loadConfig(a.path)
	if err != nil || !reflect.DeepEqual(saved, a.cfg) {
		t.Fatal("save retry lost current settings")
	}
	// Enabled state, foreground and each pause reason have distinct descriptions.
	a.cfg.Enabled = true
	a.cfg.Rules[0].Enabled = true
	a.apply()
	send(a.hwnd, wmActivateApp, 1, 0)
	if title, _, _, _ := a.stateCopy(); title != "配置窗口内暂不执行" {
		t.Fatal(title)
	}
	if a.ruleStateLabel(a.cfg.Rules[0]) != "切换后生效" {
		t.Fatal("row does not explain foreground suppression")
	}
	captureInterface(t, a.hwnd, "app-main.png")
	send(a.hwnd, wmActivateApp, 0, 0)
	if title, _, _, _ := a.stateCopy(); title != "映射已开启" {
		t.Fatal(title)
	}
	a.cfg.Enabled = false
	if a.ruleStateLabel(a.cfg.Rules[0]) != "全局暂停" {
		t.Fatal("global pause reason missing")
	}
	a.cfg.Enabled = true
	a.capturing = idRecordSource
	if a.ruleStateLabel(a.cfg.Rules[0]) != "录入中" {
		t.Fatal("capture pause reason missing")
	}
	a.capturing = 0
	a.inputFailureMessage = "受控测试故障"
	if a.ruleStateLabel(a.cfg.Rules[0]) != "输入故障" {
		t.Fatal("input failure reason missing")
	}
	a.inputFailureMessage = ""
	// The full license is embedded and displayed by a native scrolling editor.
	a.command(idAbout)
	if a.info == nil || a.info.page != "about" {
		t.Fatal("about entry did not open")
	}
	if text := windowText(a.info.controls[infoBody]); !strings.Contains(text, "GPL-3.0-only") || !strings.Contains(text, "不提供任何担保") || !strings.Contains(text, "再分发") {
		t.Fatal("about lacks legal notices")
	}
	captureInterface(t, a.info.hwnd, "app-about.png")
	moveWindow.Call(a.info.hwnd, 0, 0, uintptr(a.px(520)), uintptr(a.px(440)), 1)
	var client rect
	getClientRect.Call(a.info.hwnd, uintptr(unsafe.Pointer(&client)))
	toClient := user32.NewProc("MapWindowPoints")
	for _, id := range append([]int{infoTitle, infoBody}, a.info.actions...) {
		var bounds rect
		getWindowRect.Call(a.info.controls[id], uintptr(unsafe.Pointer(&bounds)))
		toClient.Call(0, a.info.hwnd, uintptr(unsafe.Pointer(&bounds)), 2)
		if bounds.Left < 0 || bounds.Top < 0 || bounds.Right > client.Right || bounds.Bottom > client.Bottom {
			t.Fatalf("information control %d clips at minimum size", id)
		}
	}
	captureInterface(t, a.info.hwnd, "app-about-small.png")
	send(a.info.hwnd, wmCommand, infoLicense, 0)
	if windowText(a.info.controls[infoBody]) != normalizeNoticeText(licenseText) {
		t.Fatal("offline license missing or truncated")
	}
	send(a.info.controls[infoBody], 0x00B6, 0, 10000) // EM_LINESCROLL to the end.
	if send(a.info.controls[infoBody], 0x00CE, 0, 0) == 0 {
		t.Fatal("license cannot be scrolled")
	}
	captureInterface(t, a.info.hwnd, "app-license.png")
	send(a.info.hwnd, wmCommand, infoBack, 0)
	send(a.info.hwnd, wmCommand, infoBrand, 0)
	if windowText(a.info.controls[infoBody]) != normalizeNoticeText(trademarksText) {
		t.Fatal("offline brand notice missing")
	}
	a.command(idHelp)
	if a.info.page != "help" || !strings.Contains(windowText(a.info.controls[infoBody]), "切到其他应用") {
		t.Fatal("quick help missing")
	}
	captureInterface(t, a.info.hwnd, "app-help.png")
	send(a.info.hwnd, wmCommand, infoDetails, 0)
	if a.info.page != "details" || !strings.Contains(windowText(a.info.controls[infoBody]), a.path) {
		t.Fatal("detailed help missing")
	}
	send(a.info.hwnd, wmCommand, infoBack, 0)
	if a.info.page != "help" {
		t.Fatal("details back navigation broken")
	}
	a.command(idSupport)
	if a.info.page != "support" {
		t.Fatal("support entry missing")
	}
	testDiagnosticClipboard(t, a)
	a.closeInformation()
	if a.info != nil {
		t.Fatal("closing info window kept stale handle")
	}
	// Editing releases an already held target and waits for the source key-up.
	a.cfg.Rules = []rule{{"mouse_x1", "mouse_middle", true}}
	a.refreshList()
	a.apply()
	a.engine.process("mouse_x1", true)
	a.selectRule(0)
	a.command(idEdit)
	selectKey(idTarget, "space")
	a.command(idAdd)
	if len(a.engine.refs) != 0 || !a.engine.held["mouse_x1"].released {
		t.Fatal("editing left a simulated key held")
	}
	if consumed, out := a.engine.process("mouse_x1", false); !consumed || len(out) != 0 {
		t.Fatal("mid-press edit leaked a source release")
	}
	a.selectRule(0)
	a.command(idDelete)
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
	user32.NewProc("PrintWindow").Call(hwnd, buffer, 0)
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
