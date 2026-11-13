const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const assert = require("node:assert/strict");
const {root} = require("./serve-site.cjs");
const sandbox = {};
vm.runInNewContext(fs.readFileSync(path.join(root, "config.js"), "utf8"), sandbox);
const config = sandbox.KEYMAP_SITE;
for (const field of ["repositoryURL", "issuesURL", "websiteURL", "downloadURL"]) {
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
console.log("Static pages, local assets, contact configuration and release download link checked; no app build required.");
