(() => {
  "use strict";
  const settings = globalThis.KEYMAP_SITE || {};
  const httpURL = value => {
    try { const url = new URL(value); return url.protocol === "https:" ? url.href : ""; } catch { return ""; }
  };
  const repositoryURL = httpURL(settings.repositoryURL);
  const issuesURL = httpURL(settings.issuesURL) || (repositoryURL ? repositoryURL.replace(/\/$/, "") + "/issues" : "");
  for (const [selector, url] of [["[data-repository]", repositoryURL], ["[data-issues]", issuesURL]]) {
    document.querySelectorAll(selector).forEach(link => {
      if (!url) return;
      link.href = url; link.target = "_blank"; link.rel = "noopener noreferrer"; link.hidden = false;
    });
  }
  document.querySelectorAll("[data-issues-container]").forEach(element => { element.hidden = !issuesURL; });
  const qqGroup = /^\d{5,12}$/.test(settings.qqGroup || "") ? settings.qqGroup : "";
  document.querySelectorAll("[data-qq-container]").forEach(element => { element.hidden = !qqGroup; });
  document.querySelectorAll("[data-qq-number]").forEach(element => { element.textContent = qqGroup; });
  for (const [selector, value] of [["[data-download]", settings.downloadURL], ["[data-download-proxy]", settings.proxyDownloadURL]]) {
    const url = httpURL(value);
    document.querySelectorAll(selector).forEach(link => {
      link.hidden = !url;
      if (url) link.href = url;
    });
  }

  let toastTimer;
  function toast(message) {
    const element = document.querySelector("#toast");
    if (!element) return;
    clearTimeout(toastTimer); element.textContent = message; element.classList.add("visible");
    toastTimer = setTimeout(() => element.classList.remove("visible"), 3200);
  }
  async function copyText(text, success) {
    try {
      if (navigator.clipboard && window.isSecureContext) { await navigator.clipboard.writeText(text); }
      else {
        const input = document.createElement("textarea");
        input.value = text; input.style.position = "fixed"; input.style.left = "-9999px";
        document.body.append(input); input.select();
        const copied = document.execCommand("copy"); input.remove();
        if (!copied) throw new Error("clipboard unavailable");
      }
      toast(success);
    } catch { toast(qqGroup && text === qqGroup ? `复制失败，请手动复制群号 ${qqGroup}` : "复制失败，请按页面提示整理反馈。"); }
  }
  document.querySelectorAll("[data-copy-qq]").forEach(button => button.addEventListener("click", () => copyText(qqGroup, "群号已复制，在 QQ 中搜索加群即可。")));
  document.querySelector("#copy-feedback")?.addEventListener("click", () => copyText("【轻映问题反馈】\nWindows 版本：\n轻映版本 / 下载日期：\n目标应用：\n原按键 → 目标按键：\n复现步骤：\n预期结果：\n实际结果 / 错误提示：", "反馈模板已复制，补充信息后发给作者即可。"));

  const press = document.querySelector("#demo-press");
  if (!press) return;
  const source = document.querySelector("#demo-source"), target = document.querySelector("#demo-target");
  const state = document.querySelector("#demo-state");
  let held = false, clicks = 0;
  function setHeld(value) {
    if (held === value) return;
    held = value;
    document.querySelectorAll(".demo-key").forEach(card => card.classList.toggle("is-held", held));
    press.classList.toggle("is-held", held);
    if (held) { state.textContent = `${source.value}按下 → ${target.value}按下`; }
    else { clicks++; state.textContent = `${target.value}已抬起 · 完成 ${clicks} 次映射`; }
  }
  press.addEventListener("pointerdown", event => { if (event.button !== 0) return; press.setPointerCapture(event.pointerId); setHeld(true); });
  for (const event of ["pointerup", "pointercancel", "lostpointercapture"]) press.addEventListener(event, () => setHeld(false));
  press.addEventListener("keydown", event => { if (event.key === " " || event.key === "Enter") { event.preventDefault(); setHeld(true); } });
  press.addEventListener("keyup", event => { if (event.key === " " || event.key === "Enter") { event.preventDefault(); setHeld(false); } });
  press.addEventListener("blur", () => setHeld(false));
  window.addEventListener("blur", () => setHeld(false));
  document.addEventListener("visibilitychange", () => { if (document.hidden) setHeld(false); });
  for (const [select, name] of [[source, "#source-name"], [target, "#target-name"]]) {
    select.addEventListener("change", () => { setHeld(false); clicks = 0; document.querySelector(name).textContent = select.value; state.textContent = "按下、长按、抬起，一一对应。"; });
  }
})();
