const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const assert = require("node:assert/strict");
const crypto = require("node:crypto");
const {root} = require("./serve-site.cjs");
const sandbox = {};
vm.runInNewContext(fs.readFileSync(path.join(root, "config.js"), "utf8"), sandbox);
const config = sandbox.KEYMAP_SITE;
for (const field of ["repositoryURL", "issuesURL", "websiteURL", "downloadURL", "proxyDownloadURL"]) {
  if (config[field]) assert.equal(new URL(config[field]).protocol, "https:", `${field} must use HTTPS`);
}
if (config.qqGroup) assert.match(config.qqGroup, /^\d{5,12}$/, "Invalid QQ group number");
const desktopSupport = fs.readFileSync(path.join(root, "../support_windows.go"), "utf8").match(/const supportURL = "([^"]+)"/);
assert.ok(desktopSupport, "Desktop support URL missing");
assert.equal(desktopSupport[1], new URL("#support", config.websiteURL).href, "Desktop and website domains must match");
assert.match(fs.readFileSync(path.join(root, "index.html"), "utf8"), /id="support"/, "Support anchor missing");
for (const file of ["index.html", "404.html"]) {
  const html = fs.readFileSync(path.join(root, file), "utf8");
  for (const [, value] of html.matchAll(/(?:href|src)="([^"]+)"/g)) {
    if (/^(?:https?:|#|data:)/.test(value) || value === "./" || value === "/") continue;
    const target = path.resolve(root, value.split("#")[0].replace(/^\//, ""));
    assert.ok(target.startsWith(root + path.sep), `Path escapes site: ${value}`);
    assert.ok(fs.existsSync(target), `${file}: missing file ${value}. Run build.ps1 to package the download.`);
  }
}
assert.ok(config.downloadURL, "A release download URL is required");
assert.ok(fs.statSync(path.join(root, "assets/donate.jpg")).size > 0, "Donation QR missing");
const index = fs.readFileSync(path.join(root, "index.html"), "utf8").replace(/\r\n/g, "\n");
assert.ok(index.includes(`<link rel="canonical" href="${config.websiteURL}">`), "Canonical URL must match the website");
const structuredText = index.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/)?.[1];
assert.ok(structuredText, "Software structured data missing");
const schema = JSON.parse(structuredText);
const app = schema["@graph"].find(entity => entity["@type"] === "SoftwareApplication");
assert.equal(app.url, config.websiteURL, "Structured data must describe this website");
const schemaHash = "sha256-" + crypto.createHash("sha256").update(structuredText).digest("base64");
const headers = fs.readFileSync(path.join(root, "_headers"), "utf8");
assert.ok(headers.includes(`'${schemaHash}'`), `Update the JSON-LD hash in _headers script-src to '${schemaHash}'`);
assert.ok(fs.readFileSync(path.join(root, "robots.txt"), "utf8").includes(`Sitemap: ${config.websiteURL}sitemap.xml`), "Sitemap must be discoverable");
assert.deepEqual([...fs.readFileSync(path.join(root, "sitemap.xml"), "utf8").matchAll(/<loc>(.*?)<\/loc>/g)].map(match => match[1]), [config.websiteURL], "Only the canonical homepage belongs in this single-page sitemap");
const preview = fs.readFileSync(path.join(root, "assets/social-preview.png"));
assert.equal(preview.readUInt32BE(16), 1200, "Share preview width must match its metadata");
assert.equal(preview.readUInt32BE(20), 630, "Share preview height must match its metadata");
console.log("Static assets, download links, SEO metadata, structured data and sitemap checked; no app build required.");
