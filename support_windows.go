//go:build windows

package main

import (
	"fmt"
	"unsafe"
)

const supportURL = "https://qingying.luming.host/#support"
const repositoryURL = "https://github.com/indiefun/simple-keymap"

func (a *application) openSupport() {
	a.openWebPage(supportURL, "已在浏览器打开官网的支持与反馈区域。")
}

func (a *application) openProject() {
	a.openWebPage(repositoryURL, "已在浏览器打开 GitHub 项目，可查看源码、许可与更新。")
}

func (a *application) openWebPage(url, message string) {
	result, _, _ := shellExecute.Call(a.hwnd, uintptr(unsafe.Pointer(utf16("open"))), uintptr(unsafe.Pointer(utf16(url))), 0, 0, 1)
	if result <= 32 {
		showError(a.hwnd, fmt.Sprintf("无法打开浏览器（错误 %d）。\n请手动访问：%s", result, url))
		return
	}
	a.status = message
	invalidateRect.Call(a.hwnd, 0, 0)
}
