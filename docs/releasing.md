# 发布与维护

桌面程序和官网独立发布：程序由版本标签触发 GitHub Release，官网跟随 `main` 通过 Cloudflare Workers Builds 自动部署。

## 发布桌面程序

1. 更新根目录 `VERSION`，使用 `主版本.次版本.修订号`，并在 `.github/release-notes/v版本号.md` 写明用户可见的变化、升级方式及必要限制。
2. 核对程序、[官网教程](../site/index.html)和[包内说明](../assets/release-readme.txt)一致，运行 `build.ps1`。涉及官网时同时运行官网检查。
3. 提交并推送 `main`，确认该提交的 **Check app and website** 工作流通过后，为同一提交创建尚未使用的标签：

```powershell
$taskVersion = (Get-Content -LiteralPath VERSION -Raw).Trim()
$taskTag = 'v' + $taskVersion
git tag $taskTag
git push origin $taskTag
```

`.github/workflows/release.yml` 校验标签与 `VERSION` 一致，构建、测试并生成与标签对应的源码包，然后发布以下六个附件：

| 附件 | 用途 |
| --- | --- |
| `SimpleKeymap-v版本号-windows-x64.zip` 及 `.sha256` | 带版本号的便携程序包和校验值 |
| `SimpleKeymap-v版本号-source.zip` 及 `.sha256` | 对应完整源码和校验值 |
| `SimpleKeymap-windows-latest.zip` 及 `.sha256` | 与便携包内容相同、供官网固定链接下载的附件 |

便携包只包含两个 EXE、`README.txt`、`LICENSE`、`NOTICE`、`TRADEMARKS.md`，不包含用户的 `config.json`。包内说明链接到同版本标签源码；源码包保留构建脚本、图标资源和开发说明。

发布后确认工作流成功、六个附件齐全且校验值匹配，再检查官网 GitHub 下载入口、加速下载和程序更新检查。下载的新包应能独立解压启动，覆盖升级应保留已有配置。

已发布的标签和附件保持不变；需要修正程序或发布包时使用新版本，避免同一版本对应不同内容。仅整理开发文档或维护官网时，无需发布新的程序版本。

## 官网与支持信息

[官网维护说明](../site/README.md)包含本地检查、Cloudflare 设置和域名变更清单。官网下载固定跟随 `releases/latest`，通常无需随程序版本修改链接。

维护问题反馈时，先收集版本、Windows 版本、映射关系、目标应用和复现步骤；需要时请用户从程序复制诊断摘要。不要要求上传个人配置、完整路径或无关的按键记录。已确认的修复应补充到对应版本发布说明。

发行时保留 [LICENSE](../LICENSE)、[NOTICE](../NOTICE) 和 [TRADEMARKS.md](../TRADEMARKS.md)，并提供完整对应源码。历史发布说明保留在 `.github/release-notes/`，作为发布记录。
