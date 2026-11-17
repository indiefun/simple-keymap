# 官网维护

官网：<https://qingying.luming.host/>。纯 HTML / CSS / JavaScript 静态站点，使用 Cloudflare Workers Static Assets 部署，下载指向 GitHub Releases。网站构建不依赖 Go 或本地程序包。

## 预览与检查

在仓库根目录使用 Node.js 24：

```powershell
npm ci
npm run site:dev
```

打开 <http://127.0.0.1:4173>，按 Ctrl+C 结束。

```powershell
npm run site:check
npm run site:test
npm run site:deploy:check
```

三项分别检查资源与链接、使用无头 Edge 验证交互和桌面 / 手机布局、预检 Workers 部署包。浏览器截图输出到 `.tools/site-preview/`。使用其他已安装的 Playwright 浏览器通道时，可设置 `BROWSER_CHANNEL`。

需要模拟 Cloudflare 运行环境时，运行 `npm run site:preview`，访问 <http://localhost:8787>。

## 内容与资源

| 文件 | 维护内容 |
| --- | --- |
| `index.html`、`styles.css`、`app.js` | 页面文案、样式和交互演示 |
| `config.js` | 项目、反馈、QQ群、官网与下载链接 |
| `assets/donate.jpg` | 官网支持区收款码 |
| `assets/` 下的图标及分享图 | 按[资源说明](../assets/README.md)更新 |
| `_headers` | 安全响应头与 CSP |
| `robots.txt`、`sitemap.xml` | 搜索抓取与站点地图 |

`issuesURL` 留空时使用项目仓库的 `/issues`；`proxyDownloadURL` 指向最新 Release 的 `SimpleKeymap-windows-latest.zip`，GitHub Release 保留为备用入口。支持区锚点为 `#support`，下载区为 `#download`。

修改软件操作流程时，同步官网教程、常见问题和包内说明。修改首页 JSON-LD 后，运行 `site:check` 并按提示同步 `_headers` 的 CSP 哈希。修改域名时，同时更新 `config.js`、首页 canonical / Open Graph / JSON-LD、`404.html`、`robots.txt`、`sitemap.xml`、`wrangler.jsonc`，以及桌面程序 `support_windows.go`、`updates_windows.go` 中的链接和项目文档。

## 部署

Cloudflare Workers Builds 使用以下设置：

| 设置 | 值 |
| --- | --- |
| 项目 | `qingying-keymap` |
| Git 仓库 | `indiefun/simple-keymap` |
| 生产分支 | `main` |
| 根目录 | 仓库根目录 `/` |
| 构建命令 | `npm run site:check` |
| 部署命令 | `npm run site:deploy` |
| Node.js | `24` |

依赖由 `package-lock.json` 安装，静态目录已在 `wrangler.jsonc` 指向 `./site`。连接仓库后，推送 `main` 自动部署，无需程序标签。部署账号需拥有 `qingying.luming.host` 的域名管理权限。

手动部署备用命令：

```powershell
npx wrangler login
npm run site:deploy
```

部署后检查首页、下载、支持区、`robots.txt`、`sitemap.xml` 和不存在页面的 404 响应。更换域名后，在使用中的搜索站长平台重新验证并提交站点地图。
