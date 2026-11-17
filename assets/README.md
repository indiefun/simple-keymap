# 图片与程序图标

图标由叠放键帽和映射箭头组成。普通程序构建和网站部署直接使用仓库中的资源，无需 Python 或额外图标工具。

| 文件 | 用途 |
| --- | --- |
| `icon.svg` | 可编辑的矢量原稿 |
| `generate_icon.py` | 与 SVG 形状对应的渲染脚本 |
| `icon.png` | 512 × 512 图片，供网站和分享图生成使用 |
| `icon.ico` | Windows 多尺寸图标，包含 16–256 像素九个尺寸 |
| `../rsrc_windows_amd64.syso` | Go 构建自动嵌入的图标资源，资源 ID 为 `1` |
| `generate_social_preview.py` | 生成官网 1200 × 630 分享图 |
| `release-readme.txt` | 便携包使用说明模板，构建时替换 `{{VERSION}}` |

## 更新图标

同时调整 SVG 与渲染脚本中的形状，在仓库根目录运行：

```powershell
python -m pip install Pillow
python assets/generate_icon.py
go install github.com/akavel/rsrc@v0.10.2
$taskGoBin = go env GOBIN
if (!$taskGoBin) { $taskGoBin = Join-Path (go env GOPATH) 'bin' }
& (Join-Path $taskGoBin 'rsrc.exe') -arch amd64 -ico assets/icon.ico -o rsrc_windows_amd64.syso
python assets/generate_social_preview.py
.\build.ps1
npm run site:check
```

图标脚本同时同步网站的 `icon.svg`、`icon.png` 和 `favicon.ico`；提交时将生成资源与原稿一起提交。网站保留资源副本，以便只部署 `site/` 目录。

分享图默认使用 Windows 微软雅黑；其他系统可通过 `--font` 和 `--bold-font` 指定中文字体。只调整分享图时，单独运行 `python assets/generate_social_preview.py` 即可。
