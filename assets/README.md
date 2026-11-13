# 轻映图标

两个错位叠放的键帽，一枚向右箭头，表达「一个键变成另一个键」。深绿背景与配置界面的绿色保持一致，前景用浅色提高小尺寸识别度。

- `icon.svg`：可编辑的矢量原稿。
- `icon.png`：512 × 512 透明背景图片。
- `icon.ico`：Windows 图标，包含 16、20、24、32、40、48、64、128、256 像素九个尺寸。
- `generate_icon.py`：与 SVG 几何形状一致的渲染脚本。需要 Python 和 Pillow；仅重新制作图标时使用。

程序图标已嵌入根目录的 `rsrc_windows_amd64.syso`，正常运行 `build.ps1` 即可打包，无需安装额外依赖。

需要修改图标时，调整 SVG 和渲染脚本的对应形状，然后执行：

```powershell
python assets/generate_icon.py
.\.tools\bin\rsrc.exe -arch amd64 -ico assets/icon.ico -o rsrc_windows_amd64.syso
.\build.ps1
```

资源工具为 `github.com/akavel/rsrc@v0.10.2`，本次放在 `.tools/bin/rsrc.exe`。代码中资源 ID 为 `1`，用于 exe、标题栏和界面品牌图标。

`generate_social_preview.py` 使用现有图标生成官网的 1200 × 630 分享预览图，输出到 `site/assets/social-preview.png`。同样需要 Pillow，默认使用 Windows 微软雅黑；其他系统可通过 `--font` 和 `--bold-font` 指定中文字体。预览图只随网站发布，不影响程序打包。
