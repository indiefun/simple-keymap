const {chromium} = require("playwright");
const assert = require("node:assert/strict");
const http = require("node:http");
const fs = require("node:fs");
const path = require("node:path");
const {handler} = require("./serve-site.cjs");

(async () => {
  const server = http.createServer(handler);
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  const base = `http://127.0.0.1:${server.address().port}`;
  const browser = await chromium.launch({headless:true, channel:process.env.BROWSER_CHANNEL || "msedge"});
  const output = path.resolve(__dirname, "../.tools");
  fs.mkdirSync(output, {recursive:true});
  const errors = [];
  try {
    const page = await browser.newPage({viewport:{width:1440,height:980},reducedMotion:"reduce"});
    page.on("pageerror", e => errors.push(e.message));
    page.on("console", message => { if (message.type() === "error" && /Content Security Policy|Refused to/.test(message.text())) errors.push(message.text()); });
    const checkSupportPosition = () => page.waitForFunction(() => {
      const top = document.querySelector("#support").getBoundingClientRect().top;
      return location.hash === "#support" && scrollY > 0 && top >= 0 && top < innerHeight / 3;
    });
    await page.goto(base);
    await page.locator("#demo-press").waitFor();
    assert.match(await page.title(), /Windows.*键盘映射.*鼠标改键/);
    assert.equal(await page.locator("h1").count(), 1);
    assert.equal(await page.locator("[data-download]").getAttribute("href"), "https://github.com/indiefun/simple-keymap/releases/latest");
    assert.equal(await page.locator("[data-download-proxy]").getAttribute("href"), "https://gh-proxy.com/https://github.com/indiefun/simple-keymap/releases/latest/download/SimpleKeymap-windows-latest.zip");
    assert.equal(await page.locator("[data-download-proxy]").isVisible(), true);
    const press = page.locator("#demo-press");
    await press.hover(); await page.mouse.down();
    assert.match(await page.locator("#demo-state").textContent(), /鼠标中键按下/);
    await page.mouse.up();
    assert.match(await page.locator("#demo-state").textContent(), /已抬起.*1 次/);
    await press.focus(); await page.keyboard.down("Space"); await page.keyboard.up("Space");
    assert.match(await page.locator("#demo-state").textContent(), /已抬起.*2 次/);
    await page.locator("#demo-target").selectOption("空格");
    await press.hover(); await page.mouse.down(); await page.mouse.up();
    assert.match(await page.locator("#demo-state").textContent(), /空格已抬起.*1 次/);
    await page.locator("#demo-target").selectOption("鼠标中键");
    await page.locator("summary").first().click();
    assert.equal(await page.locator("details[open]").count(), 1);
    await page.locator("summary").first().click();
    await page.screenshot({path:path.join(output, "website-desktop.png"),fullPage:true});
    await page.locator("#download").screenshot({path:path.join(output, "download-desktop.png")});
    await page.locator('nav a[href="#support"]').click();
    await checkSupportPosition();
    assert.equal(new URL(page.url()).pathname, "/");
    assert.equal(await page.locator("[data-qq-number]").textContent(), "1135778884");
    assert.equal(await page.locator("[data-issues]").getAttribute("href"), "https://github.com/indiefun/simple-keymap/issues");
    assert.equal(await page.locator("[data-issues]").isVisible(), true);
    assert.equal(await page.locator(".donation-code img").evaluate(image => image.complete && image.naturalWidth === 355), true);
    await page.context().grantPermissions(["clipboard-read","clipboard-write"]);
    await page.locator("[data-copy-qq]").click();
    await page.waitForFunction(() => document.querySelector("#toast").textContent.includes("群号已复制"));
    assert.equal(await page.evaluate(() => navigator.clipboard.readText()), "1135778884");
    await page.locator("#copy-feedback").click();
    await page.waitForFunction(() => document.querySelector("#toast").textContent.includes("模板已复制"));
    assert.match(await page.evaluate(() => navigator.clipboard.readText()), /原按键 → 目标按键/);
    await page.goto(base+"/#support");
    await checkSupportPosition();
    await page.locator("#support").screenshot({path:path.join(output, "support-desktop.png")});
    for (const width of [390, 320]) {
      await page.setViewportSize({width,height:844});
      for (const pathname of ["/", "/#support"]) {
        await page.goto(base+pathname);
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `${pathname} overflows at ${width}px`);
        if (pathname === "/#support") await checkSupportPosition();
        if (width === 390) {
          if (pathname === "/") {
            await page.screenshot({path:path.join(output, "website-mobile.png"),fullPage:true});
            await page.locator("#download").screenshot({path:path.join(output, "download-mobile.png")});
          }
          else await page.locator("#support").screenshot({path:path.join(output, "support-mobile.png")});
        }
      }
    }
    const missing = await page.goto(base+"/page-does-not-exist");
    assert.equal(missing.status(),404);
    assert.match(await page.locator('meta[name="robots"]').getAttribute("content"), /noindex/);
    const sitemapResponse = await page.request.get(base+"/sitemap.xml");
    assert.equal(sitemapResponse.status(), 200);
    assert.match(sitemapResponse.headers()["content-type"], /xml/);
    assert.deepEqual(await page.evaluate(xml => {
      const document = new DOMParser().parseFromString(xml, "application/xml");
      if (document.querySelector("parsererror")) throw new Error("Invalid sitemap XML");
      return [...document.getElementsByTagName("loc")].map(element => element.textContent);
    }, await sitemapResponse.text()), ["https://qingying.luming.host/"]);
    const robotsResponse = await page.request.get(base+"/robots.txt");
    assert.equal(robotsResponse.status(), 200);
    assert.match(robotsResponse.headers()["content-type"], /text\/plain/);
    assert.match(await robotsResponse.text(), /Sitemap: https:\/\/qingying\.luming\.host\/sitemap\.xml/);
    const withoutJS = await browser.newContext({javaScriptEnabled:false, viewport:{width:1440,height:980}});
    try {
      const staticPage = await withoutJS.newPage();
      await staticPage.goto(base);
      assert.equal(await staticPage.locator('a[data-repository][href]').isVisible(), true);
      assert.equal(await staticPage.locator('a[data-issues][href]').isVisible(), true);
      assert.equal(await staticPage.locator('[data-download-proxy]').isVisible(), true);
    } finally { await withoutJS.close(); }
    assert.deepEqual(errors, []);
    console.log("Browser checks passed: demo, downloads, support, clipboard, QR, SEO files, real 404, static links without JavaScript, desktop and mobile layouts.");
  } finally { await browser.close(); await new Promise(resolve => server.close(resolve)); }
})().catch(error => { console.error(error); process.exitCode=1; });
