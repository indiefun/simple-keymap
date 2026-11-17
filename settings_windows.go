//go:build windows

package main

import (
	"errors"
	"os"
	"syscall"
)

func errorCause(err error) string {
	if os.IsPermission(err) {
		return "软件目录无法写入，请移动整个文件夹到可写位置"
	}
	if os.IsNotExist(err) {
		return "配置目录不可用，请检查目录或磁盘后重试"
	}
	if errors.Is(err, syscall.Errno(32)) {
		return "配置文件正被其他程序占用，请关闭占用后重试"
	}
	if errors.Is(err, syscall.Errno(112)) {
		return "磁盘空间不足，请清理空间后重试"
	}
	for errors.Unwrap(err) != nil {
		err = errors.Unwrap(err)
	}
	return err.Error()
}

// Keep save failures visible until the portable configuration is written.
func (a *application) persistConfig() {
	previous := a.configSaveError
	a.configSaveError = ""
	if err := saveConfig(a.path, a.cfg); err != nil {
		a.configSaveError = errorCause(err)
	} else if previous != "" {
		a.status = "保存已恢复，设置已保存到软件同目录。"
	}
	if a.info != nil && a.info.page == "support" {
		a.updateInformation("support")
	}
	a.layoutMain()
}

func (a *application) footerCopy() string {
	if a.configSaveError != "" {
		return "尚未保存 · " + a.configSaveError
	}
	if a.status != "" {
		return a.status
	}
	return "配置保存在软件同目录 · config.json"
}
