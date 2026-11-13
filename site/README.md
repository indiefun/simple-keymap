# 轻映静态网站

官网：`https://qingying.luming.host/`。纯 HTML / CSS / JavaScript，无框架、无服务端、无统计脚本。支持与捐助集成在首页下方，直接访问 `/#support` 即可定位。首页按键卡片只是交互演示，不发送真实系统按键。

## 修改内容与支持信息

页面在 `site/index.html`，样式在 `site/styles.css`，联系方式统一在 `site/config.js`：

- `repositoryURL`：`https://github.com/indiefun/simple-keymap`，源码和 Issues 使用此仓库。
- `issuesURL`：留空时自动使用仓库的 `/issues`。
- `qqGroup`：轻映专属交流群 `1135778884`。
- `websiteURL`：`https://qingying.luming.host/`。
- `downloadURL`：`https://github.com/indiefun/simple-keymap/releases/latest`，始终跳转到最新程序发布页。

微信收款码原样来自 `../d4-helper/assets/donate.jpg`。未配置的 GitHub / QQ 入口会自动隐藏。

桌面程序直接打开 `https://qingying.luming.host/#support`，不嵌入本地网站。更新群号、收款码、页面内容后，只需推送 `main`。若更换域名，同时更新 `support_windows.go`、本站配置、首页 canonical / Open Graph 和 `wrangler.jsonc`。

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
