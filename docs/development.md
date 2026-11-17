# 开发指南

## 环境与构建

桌面程序面向 Windows x64，需要 Go 1.23 或更新版本，不需要 C 编译器。官网维护需要 Node.js 24；Python 和 Pillow 仅用于重新生成图片资源。

在仓库根目录运行：

```powershell
.\build.ps1 -TestOnly
.\build.ps1
```

第一条执行 Go 测试和 `go vet`；第二条通过检查后编译并打包，版本读取根目录的 `VERSION`。构建前会结束**本工程目录内**运行的轻映及输入辅助进程，再替换 `dist/` 中的两个 EXE。其他目录中的程序不受影响，已有 `dist/config.json` 会保留。

打包脚本校验 ZIP 文件清单、许可文件、对应版本源码链接、内置版本及 SHA256。也可单独运行：

```powershell
.\scripts\verify-package.ps1
```

## 工程结构

Go 源码保持单一 `main` 包，Windows 专用文件以 `_windows.go` 结尾，测试与对应模块放在一起。

| 文件或目录 | 职责 |
| --- | --- |
| `main_windows.go`、`main_ui_windows.go` | 程序入口、主窗口与映射列表 |
| `editor_windows.go`、`information_windows.go` | 映射引导窗口、关于与许可等信息窗口 |
| `paint_windows.go`、`switch_windows.go`、`icon_windows.go` | 绘制、开关与图标 |
| `config.go`、`settings_windows.go`、`keys.go` | 配置读写、保存反馈与按键定义 |
| `engine.go` | 映射状态及按下 / 抬起逻辑 |
| `input_windows.go`、`sender.go`、`transport.go`、`job_windows.go` | 输入监听、发送队列、辅助进程通信及生命周期 |
| `updates_windows.go`、`support_windows.go`、`diagnostics_windows.go` | 更新、项目链接与诊断摘要 |
| `win32_windows.go`、`version.go` | Windows API 声明与程序版本 |
| `assets/`、`rsrc_windows_amd64.syso` | 图片原稿、生成脚本、包内说明模板与已编译图标资源 |
| `site/`、`scripts/` | 官网与检查、预览、打包校验脚本 |
| `.github/` | 持续检查、发布流程与各版本发布说明 |
| `dist/`、`.tools/`、`.wrangler/`、`node_modules/` | 本地构建结果、预览图片、工具缓存与前端依赖，均不提交 |

`rsrc_windows_amd64.syso` 需要保留在仓库中，普通构建会自动嵌入程序图标。网站的图标副本供静态部署使用，更新方式见[资源说明](../assets/README.md)。

## 维护约定

- 界面以「看懂映射、添加或修改、到其他应用试用」为主线；新建和修改在独立引导窗口完成，操作反馈集中在主窗口状态栏。
- 按键 ID 是持久化配置值，修改显示名称时保持 ID 稳定。配置只存于 EXE 同目录的 `config.json`，保存采用临时文件与原子替换。
- 新建与编辑先保存草稿再应用；保存失败时保留原规则和草稿。即时开关等操作保存失败时，持续提示并保留重试入口。
- 映射跟随源键的按下和抬起，模拟输入不递归触发映射。暂停、改规则和退出时释放已模拟按住的键。
- `SendInput` 仅在独立辅助进程执行。通信和发送有时限，故障后停止输出；退出丢弃积压事件，辅助进程由 Windows Job 管理。
- 更新检查默认关闭，不上传规则或按键记录；诊断摘要不包含规则明细和个人文件路径。

## 验证

`build.ps1 -TestOnly` 覆盖配置、映射状态、按键识别、通信故障、进程退出、原生界面和更新检查。输入测试使用受控替身；目标应用兼容性需要另外实测。

高 DPI 和界面预览可按需运行：

```powershell
$env:SIMPLE_KEYMAP_TEST_DPI = '144'
$env:SIMPLE_KEYMAP_UI_PREVIEW_DIR = Join-Path $PWD '.tools\ui-preview'
try {
    go test -timeout 30s -run TestNativeRuleAndCaptureFlow -count=1 ./...
    if ($LASTEXITCODE -ne 0) { throw 'Interface checks failed.' }
} finally {
    Remove-Item Env:SIMPLE_KEYMAP_TEST_DPI, Env:SIMPLE_KEYMAP_UI_PREVIEW_DIR -ErrorAction SilentlyContinue
}
```

CI 运行默认及 144 DPI 检查，并设置 `SIMPLE_KEYMAP_TEST_CLIPBOARD=1` 验证诊断摘要的剪贴板往返；本地默认不改动剪贴板。官网检查见 [site/README.md](../site/README.md)。

提交前运行受影响模块的检查。修改交互时同步官网教程、包内说明和测试；提交 PR 时说明用户能观察到的变化及验证结果。临时截图、下载副本和用户配置不加入版本控制。
