//go:build windows

package main

import (
	"fmt"
	"unsafe"
)

const supportURL = "https://qingying.luming.host/#support"

func (a *application) openSupport() {
	result, _, _ := shellExecute.Call(a.hwnd, uintptr(unsafe.Pointer(utf16("open"))), uintptr(unsafe.Pointer(utf16(supportURL))), 0, 0, 1)
	if result <= 32 {
		showError(a.hwnd, fmt.Sprintf("无法打开浏览器（错误 %d）。\n请手动访问：%s", result, supportURL))
		return
	}
	a.status = "已在浏览器打开官网的支持与反馈区域。"
	invalidateRect.Call(a.hwnd, 0, 0)
}
