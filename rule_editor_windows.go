//go:build windows

package main

import (
	"errors"
	"os"
	"syscall"
)

func errorCause(err error) string {
	if os.IsPermission(err) {
		return "配置目录无法写入，请检查文件夹权限"
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

func (a *application) persistConfig() {
	previous := a.configSaveError
	a.configSaveError = ""
	if err := saveConfig(a.path, a.cfg); err != nil {
		a.configSaveError = errorCause(err)
	} else if previous != "" {
		a.status = "保存已恢复，当前设置已保存到本机。"
	}
	if a.info != nil && a.info.page == "support" {
		a.updateInformation("support")
	}
	a.layoutMain()
}

func (a *application) footerCopy() string {
	if a.configSaveError != "" {
		return "更改仅本次生效，尚未保存"
	}
	return "配置自动保存在本机"
}
