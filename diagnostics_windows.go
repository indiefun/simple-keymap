//go:build windows

package main

import (
	"fmt"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var releaseVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func versionSourceURL(version string) string {
	if releaseVersion.MatchString(version) {
		return repositoryURL + "/tree/" + url.PathEscape("v"+version)
	}
	return repositoryURL
}

func windowsDescription() string {
	version := struct {
		Size, Major, Minor, Build, Platform uint32
		ServicePack                         [128]uint16
		ServiceMajor, ServiceMinor, Suite   uint16
		Product, Reserved                   byte
	}{}
	version.Size = uint32(unsafe.Sizeof(version))
	proc := syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")
	if proc.Find() != nil {
		return "Windows（版本读取失败）"
	}
	result, _, _ := proc.Call(uintptr(unsafe.Pointer(&version)))
	if result != 0 {
		return "Windows（版本读取失败）"
	}
	name := fmt.Sprintf("Windows %d.%d", version.Major, version.Minor)
	if version.Major == 10 {
		name = "Windows 10"
		if version.Build >= 22000 {
			name = "Windows 11"
		}
	}
	return fmt.Sprintf("%s · 内部版本 %d · %s", name, version.Build, runtime.GOARCH)
}

func (a *application) diagnosticText() string {
	enabled := 0
	for _, r := range a.cfg.Rules {
		if r.Enabled {
			enabled++
		}
	}
	toggle := "暂停"
	if a.cfg.Enabled {
		toggle = "开启"
	}
	input := "未报告故障"
	if a.inputFailureMessage != "" {
		input = a.inputFailureMessage
	}
	saved := "当前未报告保存错误"
	if a.configSaveError != "" {
		saved = "更改未保存：" + a.configSaveError
	}
	return fmt.Sprintf("【轻映诊断信息】\r\n轻映版本：%s\r\n系统：%s\r\n总开关：%s\r\n规则数量：%d（勾选 %d）\r\n输入状态：%s\r\n保存状态：%s\r\n\r\n【请补充】\r\n目标应用及版本：\r\n原按键 → 目标按键：\r\n复现步骤：\r\n预期结果：\r\n实际结果：\r\n", appVersion, windowsDescription(), toggle, len(a.cfg.Rules), enabled, input, saved)
}

func writeClipboardText(owner uintptr, text string) error {
	data, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}
	opened, _, _ := openClipboard.Call(owner)
	if opened == 0 {
		return fmt.Errorf("剪贴板正被其他程序占用，请稍后重试")
	}
	defer closeClipboard.Call()
	memory, _, _ := globalAlloc.Call(0x0002, uintptr(len(data)*2)) // GMEM_MOVEABLE.
	if memory == 0 {
		return fmt.Errorf("无法分配剪贴板内存")
	}
	transferred := false
	defer func() {
		if !transferred {
			globalFree.Call(memory)
		}
	}()
	pointer, _, _ := globalLock.Call(memory)
	if pointer == 0 {
		return fmt.Errorf("无法写入剪贴板")
	}
	// GlobalLock returns memory owned by Windows, not a pointer into the Go heap.
	buffer := unsafe.Slice(messageData[uint16](&pointer), len(data))
	copy(buffer, data)
	globalUnlock.Call(memory)
	if ok, _, _ := emptyClipboard.Call(); ok == 0 {
		return fmt.Errorf("无法清空剪贴板，请稍后重试")
	}
	if handle, _, _ := setClipboardData.Call(13, memory); handle == 0 {
		return fmt.Errorf("无法写入剪贴板，请稍后重试")
	}
	transferred = true // Windows owns memory after SetClipboardData succeeds.
	return nil
}

func (a *application) copyDiagnostics() {
	if err := writeClipboardText(a.hwnd, a.diagnosticText()); err != nil {
		a.status = err.Error()
		showError(a.hwnd, err.Error())
	} else {
		a.status = "诊断信息已复制，请补充复现步骤后反馈。"
		if a.info != nil {
			setText(a.info.controls[infoCopy], "已复制，可再次复制")
			invalidateRect.Call(a.info.controls[infoCopy], 0, 0)
		}
	}
	invalidateRect.Call(a.hwnd, 0, 0)
}

func normalizeNoticeText(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
}
