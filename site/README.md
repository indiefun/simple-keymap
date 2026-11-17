# 轻映静态网站

官网：`https://qingying.luming.host/`。纯 HTML / CSS / JavaScript，无框架、无服务端、无统计脚本。支持与捐助集成在首页下方，直接访问 `/#support` 即可定位。首页按键卡片只是交互演示，不发送真实系统按键。

首屏、常见问题和页脚明确说明免费开源（GPL-3.0），主导航在桌面和手机均提供 GitHub 项目入口。源码、问题反馈和许可链接都由 `repositoryURL` 统一生成；无 JavaScript 时保留默认仓库链接。许可证和品牌说明分别见仓库根目录的 `LICENSE`、`TRADEMARKS.md`。

## 修改内容与支持信息

页面在 `site/index.html`，样式在 `site/styles.css`，联系方式统一在 `site/config.js`：

- `repositoryURL`：`https://github.com/indiefun/simple-keymap`，源码和 Issues 使用此仓库。
- `issuesURL`：留空时自动使用仓库的 `/issues`。
- `qqGroup`：轻映专属交流群 `1135778884`。
- `websiteURL`：`https://qingying.luming.host/`。
- `downloadURL`：`https://github.com/indiefun/simple-keymap/releases/latest`，始终跳转到最新程序发布页。
- `proxyDownloadURL`：通过 `gh-proxy.com` 加速下载最新 Release 的 `SimpleKeymap-windows-latest.zip`，作为国内下载主入口；GitHub Release 作为备用。

发布 Action 会保留带版本号的原始 ZIP，同时上传内容相同的固定文件名 `SimpleKeymap-windows-latest.zip` 和对应 SHA256。`releases/latest/download/` 自动跟随新版本，网站无需逐次修改下载链接。

教程与 1.2.2 的操作保持一致：保存结果在左下角状态栏提示，映射左侧开关、右侧直接修改或删除。下载区说明同目录 `config.json` 的迁移与升级保留方式，常见问题说明手动更新和默认关闭的启动检查。后续更新这些流程时须同步官网。

微信收款码原样来自 `../d4-helper/assets/donate.jpg`。未配置的 GitHub / QQ 入口会自动隐藏。

桌面程序顶部「支持与反馈」直接打开 `https://qingying.luming.host/#support`，诊断信息在「关于与许可 → 诊断信息」中复制。「检查更新」通过官网 `/#download` 提供下载入口。新建在独立窗口完成「按哪个键 → 代替哪个键 → 确认效果」，主界面管理已有规则，官网教程与包内说明须保持一致。更新群号、收款码、页面内容后，只需推送 `main`。若更换域名，同时更新 `support_windows.go`、`updates_windows.go`、本站配置、首页 canonical / Open Graph 和 `wrangler.jsonc`。

## 一次性连接 Cloudflare Workers Builds

网站使用 Cloudflare 自带的 Git 自动构建，无需在 GitHub Actions 中保存 Cloudflare API Token。先在 Cloudflare 的 Workers & Pages 中创建或打开 `qingying-keymap`，连接 GitHub 仓库 `indiefun/simple-keymap`，设置：

| 项目 | 值 |
| --- | --- |
| 生产分支 | `main` |
| 根目录 | 仓库根目录 `/` |
| 构建命令 | `npm run site:check` |
| 部署命令 | `npm run site:deploy` |
| Node.js | `24`（例如设置构建环境变量 `NODE_VERSION=24`） |

Cloudflare 会先安装 `package-lock.json` 中的依赖。构建和部署在仓库根目录运行，不要把根目录设为 `site`；`wrangler.jsonc` 的 `assets.directory` 已指向 `./site`。

连接 GitHub 时可能需要在 GitHub 安装或更新 Cloudflare 应用的仓库授权；这一步需用你的账号完成。自定义域名已在 `wrangler.jsonc` 声明为 `qingying.luming.host`，部署账户需要管理 `luming.host` 所属区域。

连接完成后，推送 `main` 就会自动更新网站，不需要版本号或标签，也不需要运行 `build.ps1`。Windows 程序通过版本标签单独发布到 GitHub Releases，官网不上传 exe / ZIP。

## 本地预览与检查

```powershell
npm ci
npm run site:dev
```

访问 `http://127.0.0.1:4173`；支持区为 `http://127.0.0.1:4173/#support`。

```powershell
npm run site:check
npm run site:test
npm run site:deploy:check
```

`site:check` 检查链接、静态资源和支持配置；不依赖任何程序包。`site:test` 使用独立无头 Edge 验证演示、锚点定位、复制、二维码和手机布局。`site:deploy:check` 仅检查 Workers 部署包，不上传。

需模拟 Cloudflare 运行时，可运行 `npm run site:preview`，访问 `http://localhost:8787`，按 Ctrl+C 结束。

## 手动部署（备用）

```powershell
npx wrangler login
npm run site:deploy
```

手动部署也不需要程序版本号。只发布 `site` 中的静态资源；源代码、工具链和本地下载包均不上传。

## 搜索收录与分享

首页标题和摘要明确说明 Windows 键盘映射、鼠标改键与侧键变中键的用途；canonical、Open Graph 和结构化信息统一使用官网域名。`robots.txt` 允许正常抓取并指向 `sitemap.xml`；站点地图只包含首页，不将 `#support` 等页内锚点当成独立页面。404 返回真实 404，并声明 `noindex`。

首页有 `WebSite` 和 `SoftwareApplication` JSON-LD，描述真实功能、系统要求和免费使用，不添加评分或评论。结构化信息不会自动产生特殊搜索结果。修改 JSON-LD 后运行 `npm run site:check`，按提示同步 `_headers` 中的 CSP 哈希，保留严格的脚本来源限制。

分享图为 `assets/social-preview.png`（1200 × 630），由 `../assets/generate_social_preview.py` 使用现有图标生成。网站运行和部署不依赖 Python。

上线后可在 Google Search Console、Bing Webmaster Tools 和百度搜索资源平台中，用域名所有者账号验证网站并提交 `https://qingying.luming.host/sitemap.xml`。验证代码需来自实际账号，工程不预填占位值。Cloudflare 若启用了托管 robots 功能，需确认线上 `/robots.txt` 仍保留 Sitemap 声明并允许正常搜索抓取。收录和排名由搜索引擎决定，部署不会立即更新搜索结果。
